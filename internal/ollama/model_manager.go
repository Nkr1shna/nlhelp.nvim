package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const (
	// Default model for neovim keybinding search
	defaultNeovimModel = "llama3.2:3b"
	// Custom model from HuggingFace (placeholder for future custom model)
	customNeovimModel = ""                // Will be set when custom model is available
	pullTimeout       = 600 * time.Second // 10 minutes for model download
)

// ModelManager handles model downloading and verification
type ModelManager struct {
	client  *Client
	baseURL string
}

// NewModelManager creates a new model manager
func NewModelManager(client *Client) *ModelManager {
	return &ModelManager{
		client:  client,
		baseURL: client.baseURL,
	}
}

// OllamaPullRequest represents a request to pull a model
type OllamaPullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream"`
}

// OllamaPullResponse represents a response from pulling a model
type OllamaPullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
	Error     string `json:"error,omitempty"`
}

// DownloadModel downloads a model from HuggingFace via Ollama
func (m *ModelManager) DownloadModel(modelName string) error {
	fmt.Printf("Downloading model: %s\n", modelName)

	// Check if model already exists
	if m.client.isModelAvailable(modelName) {
		fmt.Printf("Model %s already exists locally\n", modelName)
		return nil
	}

	// Use Ollama CLI for pulling models as it handles HuggingFace integration
	return m.pullModelViaCLI(modelName)
}

// pullModelViaCLI pulls a model using Ollama CLI
func (m *ModelManager) pullModelViaCLI(modelName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), pullTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ollamaBinary, "pull", modelName)

	// Create pipes for real-time output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start model pull: %w", err)
	}

	// Read output in goroutines
	go func() {
		scanner := make([]byte, 1024)
		for {
			n, err := stdout.Read(scanner)
			if err != nil {
				break
			}
			fmt.Print(string(scanner[:n]))
		}
	}()

	go func() {
		scanner := make([]byte, 1024)
		for {
			n, err := stderr.Read(scanner)
			if err != nil {
				break
			}
			fmt.Print(string(scanner[:n]))
		}
	}()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("model pull failed: %w", err)
	}

	fmt.Printf("\nModel %s downloaded successfully\n", modelName)
	return nil
}

// VerifyModel verifies model integrity and loading capability
func (m *ModelManager) VerifyModel(modelName string) error {
	fmt.Printf("Verifying model: %s\n", modelName)

	// Check if model exists in local registry
	if !m.client.isModelAvailable(modelName) {
		return fmt.Errorf("model %s not found locally", modelName)
	}

	// Test model loading by attempting to load it
	originalModel := m.client.modelName
	if err := m.client.LoadModel(modelName); err != nil {
		return fmt.Errorf("failed to load model %s: %w", modelName, err)
	}

	// Test basic inference to ensure model works
	if err := m.testModelInference(modelName); err != nil {
		// Restore original model if test fails
		if originalModel != "" {
			m.client.LoadModel(originalModel)
		}
		return fmt.Errorf("model inference test failed for %s: %w", modelName, err)
	}

	// Restore original model
	if originalModel != "" {
		m.client.LoadModel(originalModel)
	}

	fmt.Printf("Model %s verified successfully\n", modelName)
	return nil
}

// testModelInference tests basic inference capability of a model
func (m *ModelManager) testModelInference(modelName string) error {
	testPrompt := "Hello, this is a test prompt."

	ollamaReq := OllamaGenerateRequest{
		Model:  modelName,
		Prompt: testPrompt,
		Stream: false,
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("failed to marshal test request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", m.baseURL+"/api/generate", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send test request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read test response: %w", err)
	}

	var ollamaResp OllamaGenerateResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return fmt.Errorf("failed to unmarshal test response: %w", err)
	}

	if ollamaResp.Error != "" {
		return fmt.Errorf("model test inference error: %s", ollamaResp.Error)
	}

	if strings.TrimSpace(ollamaResp.Response) == "" {
		return fmt.Errorf("model returned empty response")
	}

	return nil
}

// DownloadNeovimModel downloads the default or custom neovim model
func (m *ModelManager) DownloadNeovimModel() error {
	// Use default model for now (llama3.2:3b)
	// Custom model will be added when available on HuggingFace
	if customNeovimModel != "" {
		// Try custom model first, fallback to default
		if err := m.DownloadModelWithRetry(customNeovimModel, 2); err != nil {
			fmt.Printf("Custom model not available, falling back to default: %s\n", defaultNeovimModel)
			return m.DownloadModelWithRetry(defaultNeovimModel, 3)
		}
		return nil
	}

	// Download default model
	return m.DownloadModelWithRetry(defaultNeovimModel, 3)
}

// DownloadModelWithRetry downloads a model with retry logic
func (m *ModelManager) DownloadModelWithRetry(modelName string, maxRetries int) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Download attempt %d/%d for model: %s\n", attempt, maxRetries, modelName)

		if err := m.DownloadModel(modelName); err != nil {
			lastErr = err
			fmt.Printf("Download attempt %d failed: %v\n", attempt, err)

			if attempt < maxRetries {
				// Exponential backoff
				waitTime := time.Duration(attempt*attempt) * 10 * time.Second
				fmt.Printf("Retrying in %v...\n", waitTime)
				time.Sleep(waitTime)
			}
			continue
		}

		// Verify the downloaded model
		if err := m.VerifyModel(modelName); err != nil {
			lastErr = err
			fmt.Printf("Model verification failed on attempt %d: %v\n", attempt, err)

			if attempt < maxRetries {
				// Remove corrupted model and retry
				m.removeModel(modelName)
				waitTime := time.Duration(attempt*attempt) * 5 * time.Second
				fmt.Printf("Retrying in %v...\n", waitTime)
				time.Sleep(waitTime)
			}
			continue
		}

		// Success
		return nil
	}

	return fmt.Errorf("failed to download and verify model %s after %d attempts: %w", modelName, maxRetries, lastErr)
}

// removeModel removes a model from local storage
func (m *ModelManager) removeModel(modelName string) error {
	cmd := exec.Command(ollamaBinary, "rm", modelName)
	return cmd.Run()
}

// GetRecommendedModel returns the recommended model for neovim keybinding search
func (m *ModelManager) GetRecommendedModel() string {
	// Check if custom model is available and configured
	if customNeovimModel != "" && m.client.isModelAvailable(customNeovimModel) {
		return customNeovimModel
	}

	// Check if default model is available
	if m.client.isModelAvailable(defaultNeovimModel) {
		return defaultNeovimModel
	}

	// Return default model name for download
	return defaultNeovimModel
}

// ListAvailableModels returns a list of locally available models
func (m *ModelManager) ListAvailableModels() ([]string, error) {
	models, err := m.client.listModels()
	if err != nil {
		return nil, err
	}

	var modelNames []string
	for _, model := range models {
		modelNames = append(modelNames, model.Name)
	}

	return modelNames, nil
}

// SetCustomModel sets the custom model name for future use
func (m *ModelManager) SetCustomModel(modelName string) {
	// This would be used when the custom model becomes available
	// For now, we'll keep using the default model
	fmt.Printf("Custom model will be set to: %s (when available)\n", modelName)
}

// GetDefaultModel returns the default model name
func (m *ModelManager) GetDefaultModel() string {
	return defaultNeovimModel
}
