package server

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"nvim-smart-keybind-search/internal/interfaces"
)

// RPCService implements the JSON-RPC service for the keybinding search
type RPCService struct {
	ragAgent      interfaces.RAGAgent
	vectorDB      interfaces.VectorDB
	llmClient     interfaces.LLMClient
	healthMonitor *HealthMonitor
}

// NewRPCService creates a new RPC service instance
func NewRPCService(ragAgent interfaces.RAGAgent, vectorDB interfaces.VectorDB, llmClient interfaces.LLMClient) *RPCService {
	return &RPCService{
		ragAgent:      ragAgent,
		vectorDB:      vectorDB,
		llmClient:     llmClient,
		healthMonitor: NewHealthMonitor(),
	}
}

// QueryArgs represents the arguments for the Query RPC method
type QueryArgs struct {
	Query   string            `json:"query"`
	Context map[string]string `json:"context,omitempty"`
	Limit   int               `json:"limit,omitempty"`
}

// QueryResult represents the result of a query operation for RPC
type QueryResult struct {
	Results   []SearchResult `json:"results"`
	Reasoning string         `json:"reasoning"`
	Error     string         `json:"error,omitempty"`
}

// SearchResult represents a single keybinding search result for RPC
type SearchResult struct {
	Keybinding  Keybinding `json:"keybinding"`
	Relevance   float64    `json:"relevance"`
	Explanation string     `json:"explanation"`
}

// Keybinding represents a vim keybinding for RPC
type Keybinding struct {
	ID          string            `json:"id"`
	Keys        string            `json:"keys"`
	Command     string            `json:"command"`
	Description string            `json:"description"`
	Mode        string            `json:"mode"`
	Plugin      string            `json:"plugin,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Query processes a natural language query and returns keybinding suggestions
func (s *RPCService) Query(args *QueryArgs, result *QueryResult) error {
	// Start timing for performance metrics
	start := time.Now()
	success := false

	// Panic recovery
	defer func() {
		if recoverErr := RecoverFromPanic(); recoverErr != nil {
			result.Error = recoverErr.Error()
		}

		// Record metrics
		if s.healthMonitor != nil {
			duration := time.Since(start)
			s.healthMonitor.GetMetricsCollector().RecordQuery(duration, success)
		}
	}()

	// Validate RAG agent availability
	if s.ragAgent == nil {
		rpcErr := NewRPCError(ErrorCodeServiceUnavailable, "RAG agent not initialized")
		result.Error = rpcErr.Message
		LogError(rpcErr, "Query")
		return rpcErr
	}

	// Validate input
	if args == nil {
		rpcErr := NewRPCError(ErrorCodeInvalidRequest, "arguments cannot be nil")
		result.Error = rpcErr.Message
		LogError(rpcErr, "Query")
		return rpcErr
	}

	if args.Query == "" {
		rpcErr := NewRPCError(ErrorCodeInvalidQuery, "query cannot be empty")
		result.Error = rpcErr.Message
		LogError(rpcErr, "Query")
		return rpcErr
	}

	// Sanitize and validate query
	query := sanitizeQuery(args.Query)
	if len(query) > 1000 { // Reasonable limit for query length
		rpcErr := NewRPCError(ErrorCodeQueryTooLong, "query too long (max 1000 characters)")
		result.Error = rpcErr.Message
		LogError(rpcErr, "Query")
		return rpcErr
	}

	// Set default limit if not specified
	if args.Limit <= 0 {
		args.Limit = 10
	} else if args.Limit > 50 { // Reasonable upper limit
		args.Limit = 50
	}

	// Process query through RAG agent
	queryResult, err := s.ragAgent.ProcessQuery(query)
	if err != nil {
		rpcErr := WrapError(err, ErrorCodeRAGAgentError, "failed to process query")
		result.Error = rpcErr.Message
		LogError(rpcErr, "Query")
		return rpcErr
	}

	// Ensure we don't return more results than requested
	if len(queryResult.Results) > args.Limit {
		queryResult.Results = queryResult.Results[:args.Limit]
	}

	// Convert interface types to RPC types
	*result = convertToRPCQueryResult(queryResult)
	success = true
	return err
}

// SyncKeybindingsArgs represents the arguments for bulk keybinding synchronization
type SyncKeybindingsArgs struct {
	Keybindings   []Keybinding `json:"keybindings"`
	ClearExisting bool         `json:"clear_existing,omitempty"`
}

// SyncKeybindingsResult represents the result of bulk synchronization
type SyncKeybindingsResult struct {
	Success        bool   `json:"success"`
	ProcessedCount int    `json:"processed_count"`
	Error          string `json:"error,omitempty"`
}

// SyncKeybindings performs bulk initialization of keybindings in the vector database
func (s *RPCService) SyncKeybindings(args *SyncKeybindingsArgs, result *SyncKeybindingsResult) error {
	var err error
	// Panic recovery
	defer func() {
		if recoverErr := RecoverFromPanic(); recoverErr != nil {
			err = recoverErr
			result.Success = false
			result.Error = recoverErr.Error()
		}
	}()

	if s.ragAgent == nil {
		rpcErr := NewRPCError(ErrorCodeServiceUnavailable, "RAG agent not initialized")
		result.Success = false
		result.Error = rpcErr.Message
		LogError(rpcErr, "SyncKeybindings")
		return rpcErr
	}

	// Validate input
	if args == nil {
		rpcErr := NewRPCError(ErrorCodeInvalidRequest, "arguments cannot be nil")
		result.Success = false
		result.Error = rpcErr.Message
		LogError(rpcErr, "SyncKeybindings")
		return rpcErr
	}

	if args.Keybindings == nil {
		args.Keybindings = []Keybinding{}
	}

	// Validate keybindings count (reasonable limit for bulk operations)
	if len(args.Keybindings) > 10000 {
		rpcErr := NewRPCError(ErrorCodeInvalidRequest, "too many keybindings in bulk sync (max 10000)")
		result.Success = false
		result.Error = rpcErr.Message
		LogError(rpcErr, "SyncKeybindings")
		return rpcErr
	}

	// If clearing existing, we could add logic here to clear the database
	// For now, we'll just update with the new keybindings
	if len(args.Keybindings) > 0 {
		// Convert RPC keybindings to interface keybindings
		interfaceKeybindings := make([]interfaces.Keybinding, len(args.Keybindings))
		for i, rpcKeybinding := range args.Keybindings {
			interfaceKeybindings[i] = convertFromRPCKeybinding(rpcKeybinding)
		}
		err := s.ragAgent.UpdateVectorDB(interfaceKeybindings)
		if err != nil {
			rpcErr := WrapError(err, ErrorCodeVectorDBError, "failed to sync keybindings")
			result.Success = false
			result.Error = rpcErr.Message
			LogError(rpcErr, "SyncKeybindings")
			return rpcErr
		}
	}

	result.Success = true
	result.ProcessedCount = len(args.Keybindings)
	return err
}

// UpdateKeybindingsArgs represents the arguments for updating keybindings
type UpdateKeybindingsArgs struct {
	Keybindings []Keybinding `json:"keybindings"`
}

// UpdateKeybindingsResult represents the result of incremental updates
type UpdateKeybindingsResult struct {
	Success      bool   `json:"success"`
	UpdatedCount int    `json:"updated_count"`
	Error        string `json:"error,omitempty"`
}

// UpdateKeybindings updates the vector database with new keybindings incrementally
func (s *RPCService) UpdateKeybindings(args *UpdateKeybindingsArgs, result *UpdateKeybindingsResult) error {
	var err error
	// Panic recovery
	defer func() {
		if recoverErr := RecoverFromPanic(); recoverErr != nil {
			err = recoverErr
			result.Success = false
			result.Error = recoverErr.Error()
		}
	}()

	if s.ragAgent == nil {
		rpcErr := NewRPCError(ErrorCodeServiceUnavailable, "RAG agent not initialized")
		result.Success = false
		result.Error = rpcErr.Message
		LogError(rpcErr, "UpdateKeybindings")
		return rpcErr
	}

	// Validate input
	if args == nil {
		rpcErr := NewRPCError(ErrorCodeInvalidRequest, "arguments cannot be nil")
		result.Success = false
		result.Error = rpcErr.Message
		LogError(rpcErr, "UpdateKeybindings")
		return rpcErr
	}

	if len(args.Keybindings) == 0 {
		result.Success = true
		result.UpdatedCount = 0
		return nil
	}

	// Validate keybindings count (reasonable limit for incremental updates)
	if len(args.Keybindings) > 1000 {
		rpcErr := NewRPCError(ErrorCodeInvalidRequest, "too many keybindings in incremental update (max 1000)")
		result.Success = false
		result.Error = rpcErr.Message
		LogError(rpcErr, "UpdateKeybindings")
		return rpcErr
	}

	// Convert RPC keybindings to interface keybindings
	interfaceKeybindings := make([]interfaces.Keybinding, len(args.Keybindings))
	for i, rpcKeybinding := range args.Keybindings {
		interfaceKeybindings[i] = convertFromRPCKeybinding(rpcKeybinding)
	}
	err = s.ragAgent.UpdateVectorDB(interfaceKeybindings)
	if err != nil {
		rpcErr := WrapError(err, ErrorCodeVectorDBError, "failed to update keybindings")
		result.Success = false
		result.Error = rpcErr.Message
		LogError(rpcErr, "UpdateKeybindings")
		return rpcErr
	}

	result.Success = true
	result.UpdatedCount = len(args.Keybindings)
	return err
}

// DetailedHealthCheckArgs represents the arguments for detailed health check
type DetailedHealthCheckArgs struct{}

// DetailedHealthCheck performs a comprehensive health check with performance metrics
func (s *RPCService) DetailedHealthCheck(args *DetailedHealthCheckArgs, result *DetailedHealthStatus) error {
	var err error
	// Panic recovery
	defer func() {
		if recoverErr := RecoverFromPanic(); recoverErr != nil {
			err = recoverErr
			result.Status = "error"
			result.Error = recoverErr.Error()
		}
	}()

	// Get detailed health status from health monitor
	if s.healthMonitor != nil {
		detailedStatus := s.healthMonitor.GetDetailedHealthStatus(s)
		*result = *detailedStatus
	} else {
		// Fallback to basic health check if monitor not available
		var basicHealth HealthStatus
		s.HealthCheck(&HealthCheckArgs{}, &basicHealth)

		result.HealthStatus = basicHealth
		result.Metrics = PerformanceMetrics{}
		result.Dependencies = make(map[string]DependencyStatus)
		result.SystemInfo = SystemInfo{
			Version:   "1.0.0",
			StartTime: time.Now(),
		}
		result.Uptime = time.Since(time.Now())
	}

	return err
}

// GetMetricsArgs represents the arguments for getting metrics
type GetMetricsArgs struct{}

// GetMetrics returns current performance metrics
func (s *RPCService) GetMetrics(args *GetMetricsArgs, result *PerformanceMetrics) error {
	var err error
	// Panic recovery
	defer func() {
		if recoverErr := RecoverFromPanic(); recoverErr != nil {
			err = recoverErr
		}
	}()

	if s.healthMonitor != nil {
		metrics := s.healthMonitor.GetMetricsCollector().GetMetrics()
		*result = metrics
	} else {
		// Return empty metrics if monitor not available
		*result = PerformanceMetrics{
			StartTime: time.Now(),
		}
	}

	return err
}

// HealthStatus represents the health status of the service
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Error     string            `json:"error,omitempty"`
}

// HealthCheckArgs represents the arguments for health check
type HealthCheckArgs struct{}

// HealthCheck performs a comprehensive health check of all service components
func (s *RPCService) HealthCheck(args *HealthCheckArgs, result *HealthStatus) error {
	var err error
	// Panic recovery
	defer func() {
		if recoverErr := RecoverFromPanic(); recoverErr != nil {
			err = recoverErr
			result.Status = "error"
			result.Error = recoverErr.Error()
		}
	}()

	status := &HealthStatus{
		Timestamp: time.Now(),
		Services:  make(map[string]string),
	}

	allHealthy := true

	// Check RAG Agent with response time measurement
	start := time.Now()
	if s.ragAgent != nil {
		if err := s.ragAgent.HealthCheck(); err != nil {
			status.Services["rag_agent"] = fmt.Sprintf("unhealthy: %v (checked in %v)", err, time.Since(start))
			allHealthy = false
		} else {
			status.Services["rag_agent"] = fmt.Sprintf("healthy (checked in %v)", time.Since(start))
		}
	} else {
		status.Services["rag_agent"] = "not initialized"
		allHealthy = false
	}

	// Check Vector Database (ChromaDB) with response time measurement
	start = time.Now()
	if s.vectorDB != nil {
		if err := s.vectorDB.HealthCheck(); err != nil {
			status.Services["vector_db"] = fmt.Sprintf("unhealthy: %v (checked in %v)", err, time.Since(start))
			allHealthy = false
		} else {
			status.Services["vector_db"] = fmt.Sprintf("healthy (checked in %v)", time.Since(start))
		}
	} else {
		status.Services["vector_db"] = "not initialized"
		allHealthy = false
	}

	// Check LLM Client (Ollama) with response time measurement and model availability
	start = time.Now()
	if s.llmClient != nil {
		if err := s.llmClient.HealthCheck(); err != nil {
			status.Services["llm_client"] = fmt.Sprintf("unhealthy: %v (checked in %v)", err, time.Since(start))
			allHealthy = false
		} else {
			// Check model availability
			if modelInfo, err := s.llmClient.GetModelInfo(); err == nil {
				status.Services["llm_client"] = fmt.Sprintf("healthy - model: %s (checked in %v)", modelInfo.Name, time.Since(start))
			} else {
				status.Services["llm_client"] = fmt.Sprintf("healthy - no model loaded (checked in %v)", time.Since(start))
			}
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
	return err
}

// sanitizeQuery cleans and validates the input query
func sanitizeQuery(query string) string {
	// Trim whitespace
	query = strings.TrimSpace(query)

	// Remove control characters but keep printable characters
	var cleaned strings.Builder
	for _, r := range query {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			cleaned.WriteRune(r)
		}
	}

	// Normalize multiple spaces to single spaces
	result := strings.Join(strings.Fields(cleaned.String()), " ")

	return result
}

// convertToRPCQueryResult converts interface QueryResult to RPC QueryResult
func convertToRPCQueryResult(interfaceResult *interfaces.QueryResult) QueryResult {
	rpcResult := QueryResult{
		Reasoning: interfaceResult.Reasoning,
		Error:     interfaceResult.Error,
		Results:   make([]SearchResult, len(interfaceResult.Results)),
	}

	for i, interfaceSearchResult := range interfaceResult.Results {
		rpcResult.Results[i] = SearchResult{
			Keybinding:  convertToRPCKeybinding(interfaceSearchResult.Keybinding),
			Relevance:   interfaceSearchResult.Relevance,
			Explanation: interfaceSearchResult.Explanation,
		}
	}

	return rpcResult
}

// convertToRPCKeybinding converts interface Keybinding to RPC Keybinding
func convertToRPCKeybinding(interfaceKeybinding interfaces.Keybinding) Keybinding {
	return Keybinding{
		ID:          interfaceKeybinding.ID,
		Keys:        interfaceKeybinding.Keys,
		Command:     interfaceKeybinding.Command,
		Description: interfaceKeybinding.Description,
		Mode:        interfaceKeybinding.Mode,
		Plugin:      interfaceKeybinding.Plugin,
		Metadata:    interfaceKeybinding.Metadata,
	}
}

// convertFromRPCKeybinding converts RPC Keybinding to interface Keybinding
func convertFromRPCKeybinding(rpcKeybinding Keybinding) interfaces.Keybinding {
	return interfaces.Keybinding{
		ID:          rpcKeybinding.ID,
		Keys:        rpcKeybinding.Keys,
		Command:     rpcKeybinding.Command,
		Description: rpcKeybinding.Description,
		Mode:        rpcKeybinding.Mode,
		Plugin:      rpcKeybinding.Plugin,
		Metadata:    rpcKeybinding.Metadata,
	}
}
