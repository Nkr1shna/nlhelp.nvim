package rag

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"nvim-smart-keybind-search/internal/interfaces"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// Agent implements the RAGAgent interface for keybinding search
type Agent struct {
	vectorDB          interfaces.VectorDB
	llmClient         interfaces.LLMClient
	queryProcessor    *QueryProcessor
	responseGenerator *ResponseGenerator
	config            *AgentConfig
	mu                sync.RWMutex
}

// AgentConfig holds configuration for the RAG agent
type AgentConfig struct {
	MaxSearchResults    int
	SimilarityThreshold float64
	ContextWindowSize   int
	MaxResponseTokens   int
	Temperature         float64
	QueryExpansion      bool
	UserBoostFactor     float64
	ResponseTimeout     time.Duration
}

// DefaultAgentConfig returns default configuration for the RAG agent
func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		MaxSearchResults:    10,
		SimilarityThreshold: 0.3,
		ContextWindowSize:   2000,
		MaxResponseTokens:   500,
		Temperature:         0.1, // Low temperature for consistent results
		QueryExpansion:      true,
		UserBoostFactor:     0.2, // Boost user keybindings by 20%
		ResponseTimeout:     30 * time.Second,
	}
}

// NewAgent creates a new RAG agent
func NewAgent(vectorDB interfaces.VectorDB, llmClient interfaces.LLMClient, config *AgentConfig) *Agent {
	if config == nil {
		config = DefaultAgentConfig()
	}

	agent := &Agent{
		vectorDB:  vectorDB,
		llmClient: llmClient,
		config:    config,
	}

	// Initialize query processor
	agent.queryProcessor = NewQueryProcessor(llmClient, vectorDB, DefaultQueryProcessorConfig())

	// Initialize response generator
	agent.responseGenerator = NewResponseGenerator(llmClient, DefaultResponseGeneratorConfig())

	return agent
}

// Initialize sets up the RAG agent with required dependencies
func (a *Agent) Initialize() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Validate dependencies
	if a.vectorDB == nil {
		return fmt.Errorf("vector database is required")
	}

	if a.llmClient == nil {
		return fmt.Errorf("LLM client is required")
	}

	// Initialize vector database
	if err := a.vectorDB.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize vector database: %w", err)
	}

	// Initialize LLM client
	if err := a.llmClient.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	log.Println("RAG agent initialized successfully")
	return nil
}

// ProcessQuery processes a natural language query and returns relevant keybindings
func (a *Agent) ProcessQuery(query string) (*interfaces.QueryResult, error) {
	if strings.TrimSpace(query) == "" {
		return &interfaces.QueryResult{
			Results:   []interfaces.SearchResult{},
			Reasoning: "Empty query provided",
			Error:     "Query cannot be empty",
		}, nil
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	log.Printf("Processing query: %s", query)
	start := time.Now()

	// Step 1: Process query with intelligent understanding
	processedQuery, err := a.queryProcessor.ProcessQuery(query)
	if err != nil {
		log.Printf("Query processing failed: %v", err)
		// Continue with original query
		processedQuery = &ProcessedQuery{
			Original: query,
			Expanded: query,
		}
	}

	// Step 2: Perform vector search with processed query
	searchResults, err := a.vectorDB.Search(processedQuery.Expanded, a.config.MaxSearchResults)
	if err != nil {
		return &interfaces.QueryResult{
			Results:   []interfaces.SearchResult{},
			Reasoning: "Vector search failed",
			Error:     fmt.Sprintf("Failed to search vector database: %v", err),
		}, nil
	}

	log.Printf("Found %d vector search results", len(searchResults))

	// Step 3: Filter by similarity threshold
	filteredResults := a.filterBySimilarity(searchResults)
	log.Printf("Filtered to %d results above similarity threshold", len(filteredResults))

	if len(filteredResults) == 0 {
		return &interfaces.QueryResult{
			Results:   []interfaces.SearchResult{},
			Reasoning: "No keybindings found matching the query criteria",
			Error:     "",
		}, nil
	}

	// Step 4: Build context for LLM using intelligent context building
	context := a.queryProcessor.BuildContextFromResults(query, filteredResults, processedQuery.Intent)

	// Step 5: Generate and parse LLM response
	llmAnalysis, err := a.responseGenerator.GenerateAndParseResponse(query, context)
	if err != nil {
		log.Printf("LLM response generation and parsing failed: %v", err)
		// Fallback to basic results without LLM enhancement
		return a.createFallbackResult(query, filteredResults), nil
	}

	// Step 6: Rank and combine results
	finalResults, reasoning, err := a.responseGenerator.RankAndCombineResults(query, filteredResults, llmAnalysis, processedQuery)
	if err != nil {
		log.Printf("Result ranking and combination failed: %v", err)
		// Fallback to basic results
		return a.createFallbackResult(query, filteredResults), nil
	}

	duration := time.Since(start)
	log.Printf("Query processed in %v, returning %d results", duration, len(finalResults))

	return &interfaces.QueryResult{
		Results:   finalResults,
		Reasoning: reasoning,
		Error:     "",
	}, nil
}

// filterBySimilarity filters search results by similarity threshold
func (a *Agent) filterBySimilarity(results []interfaces.VectorSearchResult) []interfaces.VectorSearchResult {
	filtered := make([]interfaces.VectorSearchResult, 0, len(results))

	for _, result := range results {
		if result.Score >= a.config.SimilarityThreshold {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// createFallbackResult creates a basic result when LLM processing fails
func (a *Agent) createFallbackResult(query string, vectorResults []interfaces.VectorSearchResult) *interfaces.QueryResult {
	var results []interfaces.SearchResult

	// Convert top vector results to search results
	maxResults := a.config.MaxSearchResults
	if len(vectorResults) < maxResults {
		maxResults = len(vectorResults)
	}

	for i := 0; i < maxResults; i++ {
		result := vectorResults[i]
		keybinding := a.vectorResultToKeybinding(result)

		searchResult := interfaces.SearchResult{
			Keybinding:  keybinding,
			Relevance:   result.Score,
			Explanation: a.generateBasicExplanation(query, keybinding),
		}

		results = append(results, searchResult)
	}

	// Apply user boost
	results = a.applyUserBoost(results)

	// Sort by relevance
	sort.Slice(results, func(i, j int) bool {
		return results[i].Relevance > results[j].Relevance
	})

	return &interfaces.QueryResult{
		Results:   results,
		Reasoning: fmt.Sprintf("Found %d keybindings using vector similarity search (LLM enhancement unavailable)", len(results)),
		Error:     "",
	}
}

// vectorResultToKeybinding converts a vector search result to a keybinding
func (a *Agent) vectorResultToKeybinding(result interfaces.VectorSearchResult) interfaces.Keybinding {
	metadata := result.Document.Metadata

	keybinding := interfaces.Keybinding{
		Metadata: make(map[string]string),
	}

	// Extract metadata using v2 API methods
	if metadata != nil {
		if id, ok := metadata.GetString("keybinding_id"); ok {
			keybinding.ID = id
		}
		if keys, ok := metadata.GetString("keys"); ok {
			keybinding.Keys = keys
		}
		if command, ok := metadata.GetString("command"); ok {
			keybinding.Command = command
		}
		if description, ok := metadata.GetString("description"); ok {
			keybinding.Description = description
		}
		if mode, ok := metadata.GetString("mode"); ok {
			keybinding.Mode = mode
		}
		if plugin, ok := metadata.GetString("plugin"); ok {
			keybinding.Plugin = plugin
		}

		// Note: We can't easily iterate over all metadata keys with the v2 API
		// For now, we'll only copy the known fields
	}

	return keybinding
}

// generateBasicExplanation generates a basic explanation for keybindings not detailed by LLM
func (a *Agent) generateBasicExplanation(query string, keybinding interfaces.Keybinding) string {
	explanation := fmt.Sprintf("Keybinding '%s' matches your query", keybinding.Keys)

	if keybinding.Description != "" {
		explanation += fmt.Sprintf(": %s", keybinding.Description)
	}

	if keybinding.Mode != "" && keybinding.Mode != "normal" {
		explanation += fmt.Sprintf(" (in %s mode)", keybinding.Mode)
	}

	return explanation
}

// applyUserBoost applies boost factor to user-configured keybindings
func (a *Agent) applyUserBoost(results []interfaces.SearchResult) []interfaces.SearchResult {
	for i := range results {
		if source, exists := results[i].Keybinding.Metadata["source"]; exists && source == "user" {
			results[i].Relevance += a.config.UserBoostFactor
			// Ensure relevance doesn't exceed 1.0
			if results[i].Relevance > 1.0 {
				results[i].Relevance = 1.0
			}
		}
	}

	return results
}

// UpdateVectorDB updates the vector database with new keybindings
func (a *Agent) UpdateVectorDB(keybindings []interfaces.Keybinding) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(keybindings) == 0 {
		return nil
	}

	log.Printf("Updating vector database with %d keybindings", len(keybindings))

	// Convert keybindings to documents
	documents := make([]interfaces.Document, len(keybindings))
	for i, kb := range keybindings {
		// Generate content for vectorization
		content := a.generateKeybindingContent(kb)

		// Create metadata
		metadata := make(map[string]string)
		metadata["keybinding_id"] = kb.ID
		metadata["keys"] = kb.Keys
		metadata["command"] = kb.Command
		metadata["description"] = kb.Description
		metadata["mode"] = kb.Mode
		metadata["plugin"] = kb.Plugin
		metadata["source"] = "user" // Mark as user keybinding
		metadata["updated_at"] = time.Now().Format(time.RFC3339)

		// Copy original metadata
		for k, v := range kb.Metadata {
			if _, exists := metadata[k]; !exists {
				metadata[k] = v
			}
		}

		// Convert metadata map to DocumentMetadata
		chromaMetadata, err := chroma.NewDocumentMetadataFromMap(convertStringMapToInterface(metadata))
		if err != nil {
			return fmt.Errorf("failed to create metadata for keybinding %s: %w", kb.ID, err)
		}

		documents[i] = interfaces.Document{
			ID:       chroma.DocumentID(kb.ID),
			Content:  content,
			Metadata: chromaMetadata,
		}
	}

	// Store documents in vector database
	if err := a.vectorDB.Store(documents); err != nil {
		return fmt.Errorf("failed to store keybindings in vector database: %w", err)
	}

	log.Printf("Successfully updated vector database with %d keybindings", len(keybindings))
	return nil
}

// generateKeybindingContent generates searchable content for a keybinding
func (a *Agent) generateKeybindingContent(kb interfaces.Keybinding) string {
	var parts []string

	// Always include keys and command
	parts = append(parts, kb.Keys)
	if kb.Command != "" {
		parts = append(parts, kb.Command)
	}

	// Include description if available
	if kb.Description != "" {
		parts = append(parts, kb.Description)
	}

	// Include mode information
	if kb.Mode != "" {
		parts = append(parts, fmt.Sprintf("mode:%s", kb.Mode))
	}

	// Include plugin information
	if kb.Plugin != "" {
		parts = append(parts, fmt.Sprintf("plugin:%s", kb.Plugin))
	}

	// Join all parts
	content := strings.Join(parts, " ")

	// Clean up content
	content = strings.TrimSpace(content)
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\t", " ")

	// Remove multiple spaces
	words := strings.Fields(content)
	content = strings.Join(words, " ")

	return content
}

// HealthCheck verifies the agent is functioning properly
func (a *Agent) HealthCheck() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check vector database
	if err := a.vectorDB.HealthCheck(); err != nil {
		return fmt.Errorf("vector database health check failed: %w", err)
	}

	// Check LLM client
	if err := a.llmClient.HealthCheck(); err != nil {
		return fmt.Errorf("LLM client health check failed: %w", err)
	}

	// Test basic functionality with a simple query
	testQuery := "test"
	_, err := a.vectorDB.Search(testQuery, 1)
	if err != nil {
		return fmt.Errorf("vector search test failed: %w", err)
	}

	return nil
}

// GetStats returns statistics about the RAG agent
func (a *Agent) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["max_search_results"] = a.config.MaxSearchResults
	stats["similarity_threshold"] = a.config.SimilarityThreshold
	stats["context_window_size"] = a.config.ContextWindowSize
	stats["query_expansion_enabled"] = a.config.QueryExpansion
	stats["user_boost_factor"] = a.config.UserBoostFactor
	stats["response_timeout"] = a.config.ResponseTimeout.String()

	return stats
}

// convertStringMapToInterface converts map[string]string to map[string]interface{}
func convertStringMapToInterface(stringMap map[string]string) map[string]interface{} {
	interfaceMap := make(map[string]interface{})
	for k, v := range stringMap {
		interfaceMap[k] = v
	}
	return interfaceMap
}
