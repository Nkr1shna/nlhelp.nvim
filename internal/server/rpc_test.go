package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"nvim-smart-keybind-search/internal/interfaces"
)

// MockRAGAgent implements the RAGAgent interface for testing
type MockRAGAgent struct {
	shouldError bool
}

func (m *MockRAGAgent) ProcessQuery(query string) (*interfaces.QueryResult, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	return &interfaces.QueryResult{
		Results: []interfaces.SearchResult{
			{
				Keybinding: interfaces.Keybinding{
					ID:          "test1",
					Keys:        "dd",
					Command:     "delete line",
					Description: "Delete current line",
					Mode:        "normal",
				},
				Relevance:   0.95,
				Explanation: "This deletes the current line",
			},
		},
		Reasoning: "Found matching keybinding for delete operation",
	}, nil
}

func (m *MockRAGAgent) UpdateVectorDB(keybindings []interfaces.Keybinding) error {
	if m.shouldError {
		return fmt.Errorf("mock update error")
	}
	return nil
}

func (m *MockRAGAgent) Initialize() error {
	return nil
}

func (m *MockRAGAgent) HealthCheck() error {
	// Simulate some processing time
	time.Sleep(1 * time.Millisecond)
	if m.shouldError {
		return fmt.Errorf("mock health check error")
	}
	return nil
}

func (m *MockRAGAgent) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"max_search_results":      10,
		"similarity_threshold":    0.3,
		"query_expansion_enabled": true,
	}
}

// MockVectorDB implements the VectorDB interface for testing
type MockVectorDB struct {
	shouldError bool
}

func (m *MockVectorDB) Store(documents []interfaces.Document) error {
	if m.shouldError {
		return fmt.Errorf("mock store error")
	}
	return nil
}

func (m *MockVectorDB) Search(query string, limit int) ([]interfaces.VectorSearchResult, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock search error")
	}
	return []interfaces.VectorSearchResult{}, nil
}

func (m *MockVectorDB) Delete(ids []string) error {
	return nil
}

func (m *MockVectorDB) Initialize() error {
	return nil
}

func (m *MockVectorDB) HealthCheck() error {
	// Simulate some processing time
	time.Sleep(1 * time.Millisecond)
	if m.shouldError {
		return fmt.Errorf("mock vector db health check error")
	}
	return nil
}

func (m *MockVectorDB) Close() error {
	return nil
}

// MockLLMClient implements the LLMClient interface for testing
type MockLLMClient struct {
	shouldError bool
}

func (m *MockLLMClient) Generate(request interfaces.LLMRequest) (*interfaces.LLMResponse, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock generate error")
	}
	return &interfaces.LLMResponse{Text: "mock response"}, nil
}

func (m *MockLLMClient) Embed(text string) ([]float64, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock embed error")
	}
	return []float64{0.1, 0.2, 0.3}, nil
}

func (m *MockLLMClient) LoadModel(modelName string) error {
	return nil
}

func (m *MockLLMClient) GetModelInfo() (*interfaces.ModelInfo, error) {
	return &interfaces.ModelInfo{Name: "mock-model", Status: "loaded"}, nil
}

func (m *MockLLMClient) Initialize() error {
	return nil
}

func (m *MockLLMClient) HealthCheck() error {
	// Simulate some processing time
	time.Sleep(1 * time.Millisecond)
	if m.shouldError {
		return fmt.Errorf("mock llm health check error")
	}
	return nil
}

func (m *MockLLMClient) Close() error {
	return nil
}

func TestRPCService_Query(t *testing.T) {
	tests := []struct {
		name        string
		args        *QueryArgs
		shouldError bool
		expectError string
	}{
		{
			name: "valid query",
			args: &QueryArgs{
				Query: "delete line",
				Limit: 5,
			},
			shouldError: false,
		},
		{
			name: "empty query",
			args: &QueryArgs{
				Query: "",
			},
			shouldError: true,
			expectError: "query cannot be empty",
		},
		{
			name:        "nil args",
			args:        nil,
			shouldError: true,
			expectError: "arguments cannot be nil",
		},
		{
			name: "query too long",
			args: &QueryArgs{
				Query: strings.Repeat("a", 1001),
			},
			shouldError: true,
			expectError: "query too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewRPCService(&MockRAGAgent{}, &MockVectorDB{}, &MockLLMClient{})
			var result QueryResult

			err := service.Query(tt.args, &result)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.expectError != "" && !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("expected error containing '%s', got '%s'", tt.expectError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(result.Results) == 0 {
					t.Errorf("expected results but got none")
				}
			}
		})
	}
}

func TestRPCService_SyncKeybindings(t *testing.T) {
	tests := []struct {
		name        string
		args        *SyncKeybindingsArgs
		shouldError bool
	}{
		{
			name: "valid sync",
			args: &SyncKeybindingsArgs{
				Keybindings: []Keybinding{
					{ID: "1", Keys: "dd", Command: "delete", Description: "Delete line"},
				},
			},
			shouldError: false,
		},
		{
			name: "empty keybindings",
			args: &SyncKeybindingsArgs{
				Keybindings: []Keybinding{},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewRPCService(&MockRAGAgent{}, &MockVectorDB{}, &MockLLMClient{})
			var result SyncKeybindingsResult

			err := service.SyncKeybindings(tt.args, &result)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !result.Success {
					t.Errorf("expected success but got failure: %s", result.Error)
				}
			}
		})
	}
}

func TestRPCService_UpdateKeybindings(t *testing.T) {
	service := NewRPCService(&MockRAGAgent{}, &MockVectorDB{}, &MockLLMClient{})

	args := &UpdateKeybindingsArgs{
		Keybindings: []Keybinding{
			{ID: "1", Keys: "yy", Command: "yank", Description: "Yank line"},
		},
	}

	var result UpdateKeybindingsResult
	err := service.UpdateKeybindings(args, &result)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success but got failure: %s", result.Error)
	}

	if result.UpdatedCount != 1 {
		t.Errorf("expected updated count 1, got %d", result.UpdatedCount)
	}
}

func TestRPCService_HealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		ragError       bool
		vectorDBError  bool
		llmError       bool
		expectedStatus string
	}{
		{
			name:           "all healthy",
			expectedStatus: "healthy",
		},
		{
			name:           "rag agent unhealthy",
			ragError:       true,
			expectedStatus: "unhealthy",
		},
		{
			name:           "vector db unhealthy",
			vectorDBError:  true,
			expectedStatus: "unhealthy",
		},
		{
			name:           "llm client unhealthy",
			llmError:       true,
			expectedStatus: "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewRPCService(
				&MockRAGAgent{shouldError: tt.ragError},
				&MockVectorDB{shouldError: tt.vectorDBError},
				&MockLLMClient{shouldError: tt.llmError},
			)

			var result HealthStatus
			err := service.HealthCheck(&HealthCheckArgs{}, &result)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, result.Status)
			}

			if result.Timestamp.IsZero() {
				t.Errorf("expected timestamp to be set")
			}
		})
	}
}

func TestRPCService_DetailedHealthCheck(t *testing.T) {
	service := NewRPCService(&MockRAGAgent{}, &MockVectorDB{}, &MockLLMClient{})

	var result DetailedHealthStatus
	err := service.DetailedHealthCheck(&DetailedHealthCheckArgs{}, &result)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", result.Status)
	}

	// Check that dependencies are populated
	if len(result.Dependencies) == 0 {
		t.Errorf("expected dependencies to be populated")
	}

	// Check system info
	if result.SystemInfo.Version == "" {
		t.Errorf("expected system version to be set")
	}

	// Check uptime
	if result.Uptime <= 0 {
		t.Errorf("expected positive uptime")
	}
}

func TestRPCService_GetMetrics(t *testing.T) {
	service := NewRPCService(&MockRAGAgent{}, &MockVectorDB{}, &MockLLMClient{})

	var result PerformanceMetrics
	err := service.GetMetrics(&GetMetricsArgs{}, &result)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check that start time is set
	if result.StartTime.IsZero() {
		t.Errorf("expected start time to be set")
	}
}

func TestSanitizeQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			input:    "multiple   spaces   here",
			expected: "multiple spaces here",
		},
		{
			input:    "",
			expected: "",
		},
		{
			input:    "normal query",
			expected: "normal query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeQuery(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
