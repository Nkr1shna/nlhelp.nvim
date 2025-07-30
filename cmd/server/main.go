package main

import (
	"bufio"
	"encoding/json"
	"log"
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

	// Create collection manager
	collectionManager := chromadb.NewCollectionManager(vectorDB)

	llmClient := ollama.NewClient("")

	// Initialize the clients (this will auto-install and start services if needed)
	log.Println("Initializing ChromaDB...")
	if err := vectorDB.Initialize(); err != nil {
		log.Fatalf("Failed to initialize ChromaDB: %v", err)
	}

	log.Println("Initializing Collection Manager...")
	if err := collectionManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize Collection Manager: %v", err)
	}

	log.Println("Initializing Ollama...")
	if err := llmClient.Initialize(); err != nil {
		log.Fatalf("Failed to initialize Ollama: %v", err)
	}

	// Create RAG agent with collection manager
	ragAgent := rag.NewAgent(vectorDB, collectionManager, llmClient, rag.DefaultAgentConfig())

	// Create RPC service with actual dependencies
	rpcService := server.NewRPCService(ragAgent, vectorDB, llmClient)

	log.Println("Starting JSON-RPC server on stdin/stdout")

	// Read from stdin and write to stdout for JSON-RPC communication
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse JSON-RPC request
		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			sendErrorResponse(-32700, "Parse error", req.ID)
			continue
		}

		// Validate JSON-RPC version
		if req.JSONRPC != "2.0" {
			sendErrorResponse(-32600, "Invalid Request", req.ID)
			continue
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

		responseBytes, err := json.Marshal(response)
		if err != nil {
			log.Printf("Failed to marshal response: %v", err)
			continue
		}

		os.Stdout.Write(responseBytes)
		os.Stdout.Write([]byte("\n"))
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading from stdin: %v", err)
	}
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

func sendErrorResponse(code int, message string, id interface{}) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal error response: %v", err)
		return
	}

	os.Stdout.Write(responseBytes)
	os.Stdout.Write([]byte("\n"))
}
