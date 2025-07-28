package main

import (
	"log"
	"reflect"

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

	// Create RAG agent
	ragAgent := rag.NewAgent(vectorDB, llmClient, rag.DefaultAgentConfig())

	// Create service with actual dependencies
	service := server.NewRPCService(ragAgent, vectorDB, llmClient)

	// Check the service type and methods
	serviceType := reflect.TypeOf(service)
	log.Printf("Service type: %v", serviceType)

	for i := 0; i < serviceType.NumMethod(); i++ {
		method := serviceType.Method(i)
		log.Printf("Method %d: %s, Type: %v", i, method.Name, method.Type)
	}

	// Test the RPC methods directly
	testRPCMethods(service)

	log.Println("RPC service test completed successfully!")
}

func testRPCMethods(service *server.RPCService) {
	// Test Query method
	queryArgs := &server.QueryArgs{Query: "test query", Limit: 5}
	var queryResult server.QueryResult
	err := service.Query(queryArgs, &queryResult)
	if err != nil {
		log.Printf("Query method test failed: %v", err)
	} else {
		log.Printf("Query method test passed")
	}

	// Test HealthCheck method
	var healthResult server.HealthStatus
	err = service.HealthCheck(&server.HealthCheckArgs{}, &healthResult)
	if err != nil {
		log.Printf("HealthCheck method test failed: %v", err)
	} else {
		log.Printf("HealthCheck method test passed, status: %s", healthResult.Status)
	}

	// Test GetMetrics method
	var metricsResult server.PerformanceMetrics
	err = service.GetMetrics(&server.GetMetricsArgs{}, &metricsResult)
	if err != nil {
		log.Printf("GetMetrics method test failed: %v", err)
	} else {
		log.Printf("GetMetrics method test passed")
	}

	// Test DetailedHealthCheck method
	var detailedHealthResult server.DetailedHealthStatus
	err = service.DetailedHealthCheck(&server.DetailedHealthCheckArgs{}, &detailedHealthResult)
	if err != nil {
		log.Printf("DetailedHealthCheck method test failed: %v", err)
	} else {
		log.Printf("DetailedHealthCheck method test passed, status: %s", detailedHealthResult.Status)
	}

	// Test SyncKeybindings method
	syncArgs := &server.SyncKeybindingsArgs{
		Keybindings: []server.Keybinding{
			{ID: "1", Keys: "dd", Command: "delete", Description: "Delete line"},
		},
	}
	var syncResult server.SyncKeybindingsResult
	err = service.SyncKeybindings(syncArgs, &syncResult)
	if err != nil {
		log.Printf("SyncKeybindings method test failed: %v", err)
	} else {
		log.Printf("SyncKeybindings method test passed, success: %v", syncResult.Success)
	}

	// Test UpdateKeybindings method
	updateArgs := &server.UpdateKeybindingsArgs{
		Keybindings: []server.Keybinding{
			{ID: "1", Keys: "yy", Command: "yank", Description: "Yank line"},
		},
	}
	var updateResult server.UpdateKeybindingsResult
	err = service.UpdateKeybindings(updateArgs, &updateResult)
	if err != nil {
		log.Printf("UpdateKeybindings method test failed: %v", err)
	} else {
		log.Printf("UpdateKeybindings method test passed, success: %v", updateResult.Success)
	}
}
