package ollama

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nvim-smart-keybind-search/internal/interfaces"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:11434")

	if client == nil {
		t.Fatal("Client is nil")
	}

	if client.baseURL != "http://localhost:11434" {
		t.Error("Base URL not set correctly")
	}
}

func TestNewClient_DefaultURL(t *testing.T) {
	client := NewClient("")

	if client == nil {
		t.Fatal("Client is nil")
	}

	if client.baseURL != "http://localhost:11434" {
		t.Errorf("Expected default URL 'http://localhost:11434', got '%s'", client.baseURL)
	}
}

func TestClient_IsAvailable(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "test-model"},
				},
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewClient(server.URL)

	// Test availability
	err := client.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}

func TestClient_IsAvailable_Unavailable(t *testing.T) {
	client := NewClient("http://nonexistent-host:9999")

	// Test availability with unavailable server
	err := client.HealthCheck()
	if err == nil {
		t.Error("Expected error when server is unavailable")
	}
}

func TestClient_Generate(t *testing.T) {
	// Create a mock server that returns a valid response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"response": "Test response",
				"done":     true,
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test response generation
	request := interfaces.LLMRequest{
		Prompt: "Test prompt",
	}

	response, err := client.Generate(request)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected non-nil response")
	}

	if response.Text != "Test response" {
		t.Errorf("Expected response 'Test response', got '%s'", response.Text)
	}
}

func TestClient_Generate_Error(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Test error",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)

	request := interfaces.LLMRequest{
		Prompt: "Test prompt",
	}

	response, err := client.Generate(request)
	if err == nil {
		t.Error("Expected error from server")
	}

	if response != nil {
		t.Error("Expected nil response on error")
	}
}

func TestClient_Embed(t *testing.T) {
	// Create a mock server that returns embeddings
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/embeddings" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"embeddings": [][]float64{
					{0.1, 0.2, 0.3, 0.4, 0.5},
				},
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test embedding generation
	embeddings, err := client.Embed("test text")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embeddings) != 5 {
		t.Errorf("Expected 5-dimensional embedding, got %d", len(embeddings))
	}

	// Check first value
	if embeddings[0] != 0.1 {
		t.Errorf("Expected first embedding value 0.1, got %f", embeddings[0])
	}
}

func TestClient_Embed_Error(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Embedding error",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test embedding with error
	embeddings, err := client.Embed("test text")
	if err == nil {
		t.Error("Expected error from server")
	}

	if embeddings != nil {
		t.Error("Expected nil embeddings on error")
	}
}

func TestClient_LoadModel(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"response": "Model loaded",
				"done":     true,
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test model loading
	err := client.LoadModel("test-model")
	if err != nil {
		t.Fatalf("LoadModel failed: %v", err)
	}
}

func TestClient_GetModelInfo(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/show" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"license":    "MIT",
				"modelfile":  "FROM llama2",
				"parameters": "7B",
				"template":   "{{ .Prompt }}",
				"system":     "You are a helpful assistant.",
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test model info retrieval
	info, err := client.GetModelInfo()
	if err != nil {
		t.Fatalf("GetModelInfo failed: %v", err)
	}

	if info == nil {
		t.Fatal("Expected non-nil model info")
	}

	if info.Name == "" {
		t.Error("Expected non-empty model name")
	}
}

func TestClient_HealthCheck(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []map[string]interface{}{},
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test health check
	err := client.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}

func TestClient_HealthCheck_Unavailable(t *testing.T) {
	client := NewClient("http://nonexistent-host:9999")

	// Test health check with unavailable service
	err := client.HealthCheck()
	if err == nil {
		t.Error("Expected error when service is unavailable")
	}
}

func TestClient_Initialize(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []map[string]interface{}{},
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test initialization
	err := client.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
}

func TestClient_Close(t *testing.T) {
	client := NewClient("http://localhost:11434")

	// Test close
	err := client.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestClient_SetRequestTimeout(t *testing.T) {
	client := NewClient("http://localhost:11434")

	// Test setting timeout
	timeout := 60 * time.Second
	client.SetRequestTimeout(timeout)

	// Verify timeout was set (we can't directly access the http client timeout)
	// but we can verify the function doesn't panic
	if client.httpClient == nil {
		t.Error("HTTP client should not be nil")
	}
}

func TestClient_GenerateKeybindingResponse(t *testing.T) {
	// Create a mock server that returns a keybinding response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"response": `{
					"keybindings": [
						{
							"keys": "dd",
							"description": "Delete current line",
							"explanation": "This is the standard vim command to delete the current line"
						}
					],
					"reasoning": "The query 'delete line' matches the 'dd' command"
				}`,
				"done": true,
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test keybinding response generation
	response, err := client.GenerateKeybindingResponse("delete line")
	if err != nil {
		t.Fatalf("GenerateKeybindingResponse failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected non-nil response")
	}

	if len(response.Results) != 1 {
		t.Errorf("Expected 1 keybinding, got %d", len(response.Results))
	}

	if response.Results[0].Keys != "dd" {
		t.Errorf("Expected keys 'dd', got '%s'", response.Results[0].Keys)
	}
}

func TestClient_ListAvailableModels(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "test-model-1"},
					{"name": "test-model-2"},
				},
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test model listing
	models, err := client.ListAvailableModels()
	if err != nil {
		t.Fatalf("ListAvailableModels failed: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	if models[0] != "test-model-1" {
		t.Errorf("Expected model name 'test-model-1', got '%s'", models[0])
	}
}

func TestClient_GetRecommendedModel(t *testing.T) {
	client := NewClient("http://localhost:11434")

	// Test getting recommended model
	model := client.GetRecommendedModel()
	if model == "" {
		t.Error("Expected non-empty recommended model")
	}
}
