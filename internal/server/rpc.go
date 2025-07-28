package server

import (
	"fmt"
	"time"

	"nvim-smart-keybind-search/internal/interfaces"
)

// RPCService implements the JSON-RPC service for the keybinding search
type RPCService struct {
	ragAgent  interfaces.RAGAgent
	vectorDB  interfaces.VectorDB
	llmClient interfaces.LLMClient
}

// NewRPCService creates a new RPC service instance
func NewRPCService(ragAgent interfaces.RAGAgent, vectorDB interfaces.VectorDB, llmClient interfaces.LLMClient) *RPCService {
	return &RPCService{
		ragAgent:  ragAgent,
		vectorDB:  vectorDB,
		llmClient: llmClient,
	}
}

// QueryArgs represents the arguments for the Query RPC method
type QueryArgs struct {
	Query   string            `json:"query"`
	Context map[string]string `json:"context,omitempty"`
	Limit   int               `json:"limit,omitempty"`
}

// Query processes a natural language query and returns keybinding suggestions
func (s *RPCService) Query(args *QueryArgs, result *interfaces.QueryResult) error {
	if args.Query == "" {
		return fmt.Errorf("query cannot be empty")
	}

	// Set default limit if not specified
	if args.Limit <= 0 {
		args.Limit = 10
	}

	// Process query through RAG agent
	queryResult, err := s.ragAgent.ProcessQuery(args.Query)
	if err != nil {
		return fmt.Errorf("failed to process query: %w", err)
	}

	*result = *queryResult
	return nil
}

// UpdateKeybindingsArgs represents the arguments for updating keybindings
type UpdateKeybindingsArgs struct {
	Keybindings []interfaces.Keybinding `json:"keybindings"`
}

// UpdateKeybindings updates the vector database with new keybindings
func (s *RPCService) UpdateKeybindings(args *UpdateKeybindingsArgs, result *bool) error {
	if len(args.Keybindings) == 0 {
		*result = true
		return nil
	}

	err := s.ragAgent.UpdateVectorDB(args.Keybindings)
	if err != nil {
		return fmt.Errorf("failed to update keybindings: %w", err)
	}

	*result = true
	return nil
}

// HealthStatus represents the health status of the service
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Error     string            `json:"error,omitempty"`
}

// HealthCheck performs a comprehensive health check of all service components
func (s *RPCService) HealthCheck(args interface{}, result *HealthStatus) error {
	status := &HealthStatus{
		Timestamp: time.Now(),
		Services:  make(map[string]string),
	}

	allHealthy := true

	// Check RAG Agent
	if s.ragAgent != nil {
		if err := s.ragAgent.HealthCheck(); err != nil {
			status.Services["rag_agent"] = fmt.Sprintf("unhealthy: %v", err)
			allHealthy = false
		} else {
			status.Services["rag_agent"] = "healthy"
		}
	} else {
		status.Services["rag_agent"] = "not initialized"
		allHealthy = false
	}

	// Check Vector Database
	if s.vectorDB != nil {
		if err := s.vectorDB.HealthCheck(); err != nil {
			status.Services["vector_db"] = fmt.Sprintf("unhealthy: %v", err)
			allHealthy = false
		} else {
			status.Services["vector_db"] = "healthy"
		}
	} else {
		status.Services["vector_db"] = "not initialized"
		allHealthy = false
	}

	// Check LLM Client
	if s.llmClient != nil {
		if err := s.llmClient.HealthCheck(); err != nil {
			status.Services["llm_client"] = fmt.Sprintf("unhealthy: %v", err)
			allHealthy = false
		} else {
			status.Services["llm_client"] = "healthy"
		}
	} else {
		status.Services["llm_client"] = "not initialized"
		allHealthy = false
	}

	if allHealthy {
		status.Status = "healthy"
	} else {
		status.Status = "unhealthy"
	}

	*result = *status
	return nil
}