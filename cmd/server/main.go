package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"nvim-smart-keybind-search/internal/chromadb"
	"nvim-smart-keybind-search/internal/ollama"
	"nvim-smart-keybind-search/internal/rag"
	"nvim-smart-keybind-search/internal/server"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      interface{} `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	// Initialize dependencies
	vectorDB, err := chromadb.NewClient(chromadb.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to create ChromaDB client: %v", err)
	}

	llmClient := ollama.NewClient("")
	if err != nil {
		log.Fatalf("Failed to create Ollama client: %v", err)
	}

	// Initialize the clients (this will auto-install and start services if needed)
	log.Println("Initializing ChromaDB...")
	if err := vectorDB.Initialize(); err != nil {
		log.Fatalf("Failed to initialize ChromaDB: %v", err)
	}

	log.Println("Initializing Ollama...")
	if err := llmClient.Initialize(); err != nil {
		log.Fatalf("Failed to initialize Ollama: %v", err)
	}

	// Create RAG agent
	ragAgent := rag.NewAgent(vectorDB, llmClient, rag.DefaultAgentConfig())

	// Create RPC service with actual dependencies
	rpcService := server.NewRPCService(ragAgent, vectorDB, llmClient)

	// Set up HTTP handler for JSON-RPC
	http.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		// Only allow POST requests
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// Parse JSON-RPC request
		var req JSONRPCRequest
		if err := json.Unmarshal(body, &req); err != nil {
			sendErrorResponse(w, -32700, "Parse error", req.ID)
			return
		}

		// Validate JSON-RPC version
		if req.JSONRPC != "2.0" {
			sendErrorResponse(w, -32600, "Invalid Request", req.ID)
			return
		}

		// Handle different methods
		var result interface{}
		var rpcErr *RPCError

		switch req.Method {
		case "Query":
			result, rpcErr = handleQuery(rpcService, req.Params)
		case "SyncKeybindings":
			result, rpcErr = handleSyncKeybindings(rpcService, req.Params)
		case "UpdateKeybindings":
			result, rpcErr = handleUpdateKeybindings(rpcService, req.Params)
		case "HealthCheck":
			result, rpcErr = handleHealthCheck(rpcService, req.Params)
		case "DetailedHealthCheck":
			result, rpcErr = handleDetailedHealthCheck(rpcService, req.Params)
		case "GetMetrics":
			result, rpcErr = handleGetMetrics(rpcService, req.Params)
		default:
			rpcErr = &RPCError{
				Code:    -32601,
				Message: "Method not found",
			}
		}

		// Send response
		response := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
		}

		if rpcErr != nil {
			response.Error = rpcErr
		} else {
			response.Result = result
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting JSON-RPC server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleQuery(service *server.RPCService, params interface{}) (interface{}, *RPCError) {
	// Parse params
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params"}
	}

	var args server.QueryArgs
	if err := json.Unmarshal(paramsBytes, &args); err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params"}
	}

	var result server.QueryResult
	if err := service.Query(&args, &result); err != nil {
		return nil, &RPCError{Code: -32603, Message: err.Error()}
	}

	return result, nil
}

func handleSyncKeybindings(service *server.RPCService, params interface{}) (interface{}, *RPCError) {
	// Parse params
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params"}
	}

	var args server.SyncKeybindingsArgs
	if err := json.Unmarshal(paramsBytes, &args); err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params"}
	}

	var result server.SyncKeybindingsResult
	if err := service.SyncKeybindings(&args, &result); err != nil {
		return nil, &RPCError{Code: -32603, Message: err.Error()}
	}

	return result, nil
}

func handleUpdateKeybindings(service *server.RPCService, params interface{}) (interface{}, *RPCError) {
	// Parse params
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params"}
	}

	var args server.UpdateKeybindingsArgs
	if err := json.Unmarshal(paramsBytes, &args); err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params"}
	}

	var result server.UpdateKeybindingsResult
	if err := service.UpdateKeybindings(&args, &result); err != nil {
		return nil, &RPCError{Code: -32603, Message: err.Error()}
	}

	return result, nil
}

func handleHealthCheck(service *server.RPCService, params interface{}) (interface{}, *RPCError) {
	var result server.HealthStatus
	if err := service.HealthCheck(&server.HealthCheckArgs{}, &result); err != nil {
		return nil, &RPCError{Code: -32603, Message: err.Error()}
	}

	return result, nil
}

func handleDetailedHealthCheck(service *server.RPCService, params interface{}) (interface{}, *RPCError) {
	var result server.DetailedHealthStatus
	if err := service.DetailedHealthCheck(&server.DetailedHealthCheckArgs{}, &result); err != nil {
		return nil, &RPCError{Code: -32603, Message: err.Error()}
	}

	return result, nil
}

func handleGetMetrics(service *server.RPCService, params interface{}) (interface{}, *RPCError) {
	var result server.PerformanceMetrics
	if err := service.GetMetrics(&server.GetMetricsArgs{}, &result); err != nil {
		return nil, &RPCError{Code: -32603, Message: err.Error()}
	}

	return result, nil
}

func sendErrorResponse(w http.ResponseWriter, code int, message string, id interface{}) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
