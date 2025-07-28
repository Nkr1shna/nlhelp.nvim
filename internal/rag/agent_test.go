package rag

import (
	"testing"

	"nvim-smart-keybind-search/internal/interfaces"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// MockVectorDB is a mock implementation of VectorDB for testing
type MockVectorDB struct {
	searchResults []interfaces.VectorSearchResult
	searchError   error
}

func (m *MockVectorDB) Store(documents []interfaces.Document) error {
	return nil
}

func (m *MockVectorDB) Search(query string, limit int) ([]interfaces.VectorSearchResult, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResults, nil
}

func (m *MockVectorDB) Delete(ids []string) error {
	return nil
}

func (m *MockVectorDB) Initialize() error {
	return nil
}

func (m *MockVectorDB) HealthCheck() error {
	return nil
}

func (m *MockVectorDB) Close() error {
	return nil
}

// MockLLMClient is a mock implementation of LLMClient for testing
type MockLLMClient struct {
	generateResponse *interfaces.LLMResponse
	generateError    error
	embedVector      []float64
	embedError       error
}

func (m *MockLLMClient) Generate(request interfaces.LLMRequest) (*interfaces.LLMResponse, error) {
	if m.generateError != nil {
		return nil, m.generateError
	}
	if m.generateResponse != nil {
		return m.generateResponse, nil
	}
	return &interfaces.LLMResponse{
		Text:   "Test response",
		Tokens: 10,
	}, nil
}

func (m *MockLLMClient) Embed(text string) ([]float64, error) {
	if m.embedError != nil {
		return nil, m.embedError
	}
	if m.embedVector != nil {
		return m.embedVector, nil
	}
	return []float64{0.1, 0.2, 0.3}, nil
}

func (m *MockLLMClient) LoadModel(modelName string) error {
	return nil
}

func (m *MockLLMClient) GetModelInfo() (*interfaces.ModelInfo, error) {
	return &interfaces.ModelInfo{
		Name:    "test-model",
		Version: "1.0",
		Size:    "1GB",
		Status:  "loaded",
	}, nil
}

func (m *MockLLMClient) Initialize() error {
	return nil
}

func (m *MockLLMClient) HealthCheck() error {
	return nil
}

func (m *MockLLMClient) Close() error {
	return nil
}

func TestNewAgent(t *testing.T) {
	mockVectorDB := &MockVectorDB{}
	mockLLMClient := &MockLLMClient{}

	agent := NewAgent(mockVectorDB, mockLLMClient, nil)

	if agent == nil {
		t.Fatal("Expected agent to be created, got nil")
	}

	if agent.vectorDB != mockVectorDB {
		t.Error("Expected vectorDB to be set correctly")
	}

	if agent.llmClient != mockLLMClient {
		t.Error("Expected llmClient to be set correctly")
	}

	if agent.queryProcessor == nil {
		t.Error("Expected queryProcessor to be initialized")
	}

	if agent.responseGenerator == nil {
		t.Error("Expected responseGenerator to be initialized")
	}

	if agent.config == nil {
		t.Error("Expected config to be initialized with defaults")
	}
}

func TestAgentInitialize(t *testing.T) {
	mockVectorDB := &MockVectorDB{}
	mockLLMClient := &MockLLMClient{}

	agent := NewAgent(mockVectorDB, mockLLMClient, nil)

	err := agent.Initialize()
	if err != nil {
		t.Fatalf("Expected initialization to succeed, got error: %v", err)
	}
}

func TestAgentProcessQueryEmptyQuery(t *testing.T) {
	mockVectorDB := &MockVectorDB{}
	mockLLMClient := &MockLLMClient{}

	agent := NewAgent(mockVectorDB, mockLLMClient, nil)

	result, err := agent.ProcessQuery("")
	if err != nil {
		t.Fatalf("Expected no error for empty query, got: %v", err)
	}

	if result.Error != "Query cannot be empty" {
		t.Errorf("Expected error message for empty query, got: %s", result.Error)
	}

	if len(result.Results) != 0 {
		t.Errorf("Expected no results for empty query, got %d results", len(result.Results))
	}
}

func TestAgentProcessQueryWithResults(t *testing.T) {
	// Setup mock vector search results
	metadata, _ := chroma.NewDocumentMetadataFromMap(map[string]interface{}{
		"keybinding_id": "test1",
		"keys":          "dd",
		"command":       "delete",
		"description":   "Delete current line",
		"mode":          "normal",
		"source":        "builtin",
	})

	mockResults := []interfaces.VectorSearchResult{
		{
			Document: interfaces.Document{
				ID:       chroma.DocumentID("test1"),
				Content:  "delete line dd",
				Metadata: metadata,
			},
			Score:    0.9,
			Distance: 0.1,
		},
	}

	mockVectorDB := &MockVectorDB{
		searchResults: mockResults,
	}

	// Setup mock LLM response
	mockLLMResponse := &interfaces.LLMResponse{
		Text: `ANALYSIS:
The user wants to delete a line.

RECOMMENDATIONS:
1. Keys: dd | Command: delete | Description: Delete current line | Mode: normal | Score: 0.9 | Explanation: This deletes the entire line where the cursor is positioned.

REASONING:
The dd command is the most direct way to delete a line in vim.

ALTERNATIVES:
You could also use D to delete from cursor to end of line.`,
		Tokens: 50,
	}

	mockLLMClient := &MockLLMClient{
		generateResponse: mockLLMResponse,
	}

	agent := NewAgent(mockVectorDB, mockLLMClient, nil)

	result, err := agent.ProcessQuery("delete line")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Error != "" {
		t.Errorf("Expected no error in result, got: %s", result.Error)
	}

	if len(result.Results) == 0 {
		t.Error("Expected at least one result")
	}

	if len(result.Results) > 0 {
		firstResult := result.Results[0]
		if firstResult.Keybinding.Keys != "dd" {
			t.Errorf("Expected first result keys to be 'dd', got: %s", firstResult.Keybinding.Keys)
		}

		if firstResult.Relevance <= 0 {
			t.Errorf("Expected positive relevance score, got: %f", firstResult.Relevance)
		}

		if firstResult.Explanation == "" {
			t.Error("Expected explanation to be provided")
		}
	}

	if result.Reasoning == "" {
		t.Error("Expected reasoning to be provided")
	}
}

func TestAgentHealthCheck(t *testing.T) {
	mockVectorDB := &MockVectorDB{}
	mockLLMClient := &MockLLMClient{}

	agent := NewAgent(mockVectorDB, mockLLMClient, nil)

	err := agent.HealthCheck()
	if err != nil {
		t.Fatalf("Expected health check to pass, got error: %v", err)
	}
}

func TestAgentGetStats(t *testing.T) {
	mockVectorDB := &MockVectorDB{}
	mockLLMClient := &MockLLMClient{}

	agent := NewAgent(mockVectorDB, mockLLMClient, nil)

	stats := agent.GetStats()
	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	expectedKeys := []string{
		"max_search_results",
		"similarity_threshold",
		"context_window_size",
		"query_expansion_enabled",
		"user_boost_factor",
		"response_timeout",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stats to contain key: %s", key)
		}
	}
}

func TestAgentUpdateVectorDB(t *testing.T) {
	mockVectorDB := &MockVectorDB{}
	mockLLMClient := &MockLLMClient{}

	agent := NewAgent(mockVectorDB, mockLLMClient, nil)

	keybindings := []interfaces.Keybinding{
		{
			ID:          "test1",
			Keys:        "dd",
			Command:     "delete",
			Description: "Delete current line",
			Mode:        "normal",
		},
	}

	err := agent.UpdateVectorDB(keybindings)
	if err != nil {
		t.Fatalf("Expected UpdateVectorDB to succeed, got error: %v", err)
	}
}

func TestAgentFilterBySimilarity(t *testing.T) {
	mockVectorDB := &MockVectorDB{}
	mockLLMClient := &MockLLMClient{}

	config := DefaultAgentConfig()
	config.SimilarityThreshold = 0.5

	agent := NewAgent(mockVectorDB, mockLLMClient, config)

	results := []interfaces.VectorSearchResult{
		{Score: 0.9}, // Should be included
		{Score: 0.6}, // Should be included
		{Score: 0.4}, // Should be filtered out
		{Score: 0.3}, // Should be filtered out
	}

	filtered := agent.filterBySimilarity(results)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered results, got %d", len(filtered))
	}

	for _, result := range filtered {
		if result.Score < config.SimilarityThreshold {
			t.Errorf("Expected all filtered results to have score >= %f, got %f",
				config.SimilarityThreshold, result.Score)
		}
	}
}
