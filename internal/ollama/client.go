package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"nvim-smart-keybind-search/internal/interfaces"
)

const (
	defaultOllamaURL = "http://localhost:11434"
	ollamaBinary     = "ollama"
	installTimeout   = 300 * time.Second
	requestTimeout   = 30 * time.Second
)

// Client implements the LLMClient interface for Ollama
type Client struct {
	baseURL        string
	httpClient     *http.Client
	modelName      string
	modelManager   *ModelManager
	responseParser *ResponseParser
}

// NewClient creates a new Ollama client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultOllamaURL
	}

	client := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}

	client.modelManager = NewModelManager(client)
	client.responseParser = NewResponseParser()
	return client
}

// Initialize sets up the Ollama client and verifies connectivity
func (c *Client) Initialize() error {
	// Check if Ollama is installed
	if !c.isOllamaInstalled() {
		if err := c.installOllama(); err != nil {
			return fmt.Errorf("failed to install Ollama: %w", err)
		}
	}

	// Check if Ollama service is running
	if !c.isServiceRunning() {
		if err := c.startService(); err != nil {
			return fmt.Errorf("failed to start Ollama service: %w", err)
		}
	}

	return c.HealthCheck()
}

// isOllamaInstalled checks if Ollama binary exists and is executable
func (c *Client) isOllamaInstalled() bool {
	_, err := exec.LookPath(ollamaBinary)
	return err == nil
}

// installOllama automatically installs Ollama for macOS/Linux systems
func (c *Client) installOllama() error {
	switch runtime.GOOS {
	case "darwin":
		return c.installOllamaMacOS()
	case "linux":
		return c.installOllamaLinux()
	default:
		return fmt.Errorf("automatic installation not supported for %s", runtime.GOOS)
	}
}

// installOllamaMacOS installs Ollama on macOS using the official installer
func (c *Client) installOllamaMacOS() error {
	fmt.Println("Installing Ollama on macOS...")

	// Download and run the official installer
	cmd := exec.Command("curl", "-fsSL", "https://ollama.ai/install.sh")
	installScript, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to download Ollama installer: %w", err)
	}

	// Execute the install script
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()

	cmd = exec.CommandContext(ctx, "sh", "-c", string(installScript))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute Ollama installer: %w", err)
	}

	// Verify installation
	if !c.isOllamaInstalled() {
		return fmt.Errorf("Ollama installation verification failed")
	}

	fmt.Println("Ollama installed successfully")
	return nil
}

// installOllamaLinux installs Ollama on Linux using the official installer
func (c *Client) installOllamaLinux() error {
	fmt.Println("Installing Ollama on Linux...")

	// Download and run the official installer
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "curl", "-fsSL", "https://ollama.ai/install.sh")
	installScript, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to download Ollama installer: %w", err)
	}

	// Execute the install script
	cmd = exec.CommandContext(ctx, "sh", "-c", string(installScript))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute Ollama installer: %w", err)
	}

	// Verify installation
	if !c.isOllamaInstalled() {
		return fmt.Errorf("Ollama installation verification failed")
	}

	fmt.Println("Ollama installed successfully")
	return nil
}

// isServiceRunning checks if Ollama service is running
func (c *Client) isServiceRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/version", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// startService starts the Ollama daemon
func (c *Client) startService() error {
	fmt.Println("Starting Ollama service...")

	// Start Ollama serve in background
	cmd := exec.Command(ollamaBinary, "serve")

	// Start the service in background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Ollama service: %w", err)
	}

	// Wait for service to be ready
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		if c.isServiceRunning() {
			fmt.Println("Ollama service started successfully")
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("Ollama service failed to start within timeout")
}

// stopService stops the Ollama daemon
func (c *Client) stopService() error {
	// Find and kill Ollama processes
	cmd := exec.Command("pkill", "-f", "ollama serve")
	_ = cmd.Run() // Ignore error if no process found

	// Wait a moment for graceful shutdown
	time.Sleep(2 * time.Second)

	return nil
}

// HealthCheck verifies the Ollama service is available and responsive
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/version", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama service is not responding: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama service returned status %d", resp.StatusCode)
	}

	return nil
}

// Close closes the Ollama client connection
func (c *Client) Close() error {
	// No persistent connections to close for HTTP client
	return nil
}

// OllamaGenerateRequest represents a request to Ollama's generate API
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaGenerateResponse represents a response from Ollama's generate API
type OllamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

// OllamaEmbedRequest represents a request to Ollama's embed API
type OllamaEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// OllamaEmbedResponse represents a response from Ollama's embed API
type OllamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
	Error      string      `json:"error,omitempty"`
}

// OllamaModelInfo represents model information from Ollama
type OllamaModelInfo struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
}

// OllamaListResponse represents the response from listing models
type OllamaListResponse struct {
	Models []OllamaModelInfo `json:"models"`
}

// Generate generates text based on the given request
func (c *Client) Generate(request interfaces.LLMRequest) (*interfaces.LLMResponse, error) {
	if c.modelName == "" {
		return nil, fmt.Errorf("no model loaded")
	}

	ollamaReq := OllamaGenerateRequest{
		Model:  c.modelName,
		Prompt: request.Prompt,
		Stream: false,
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp OllamaGenerateResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if ollamaResp.Error != "" {
		return nil, fmt.Errorf("Ollama error: %s", ollamaResp.Error)
	}

	return &interfaces.LLMResponse{
		Text:   ollamaResp.Response,
		Tokens: len(strings.Fields(ollamaResp.Response)), // Rough token count
	}, nil
}

// Embed generates embeddings for the given text
func (c *Client) Embed(text string) ([]float64, error) {
	if c.modelName == "" {
		return nil, fmt.Errorf("no model loaded")
	}

	ollamaReq := OllamaEmbedRequest{
		Model: c.modelName,
		Input: text,
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embed", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp OllamaEmbedResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if ollamaResp.Error != "" {
		return nil, fmt.Errorf("Ollama error: %s", ollamaResp.Error)
	}

	if len(ollamaResp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return ollamaResp.Embeddings[0], nil
}

// LoadModel loads a specific model for inference
func (c *Client) LoadModel(modelName string) error {
	// Check if model exists locally
	if !c.isModelAvailable(modelName) {
		return fmt.Errorf("model %s is not available locally", modelName)
	}

	c.modelName = modelName
	return nil
}

// GetModelInfo returns information about the currently loaded model
func (c *Client) GetModelInfo() (*interfaces.ModelInfo, error) {
	if c.modelName == "" {
		return nil, fmt.Errorf("no model loaded")
	}

	models, err := c.listModels()
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	for _, model := range models {
		if model.Name == c.modelName {
			return &interfaces.ModelInfo{
				Name:    model.Name,
				Version: model.Digest[:12], // Use first 12 chars of digest as version
				Size:    formatBytes(model.Size),
				Status:  "loaded",
			}, nil
		}
	}

	return nil, fmt.Errorf("model %s not found", c.modelName)
}

// isModelAvailable checks if a model is available locally
func (c *Client) isModelAvailable(modelName string) bool {
	models, err := c.listModels()
	if err != nil {
		return false
	}

	for _, model := range models {
		if model.Name == modelName {
			return true
		}
	}
	return false
}

// listModels returns a list of available models
func (c *Client) listModels() ([]OllamaModelInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var listResp OllamaListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return listResp.Models, nil
}

// formatBytes formats bytes into human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// DownloadNeovimModel downloads the recommended neovim model
func (c *Client) DownloadNeovimModel() error {
	return c.modelManager.DownloadNeovimModel()
}

// DownloadModelWithRetry downloads a specific model with retry logic
func (c *Client) DownloadModelWithRetry(modelName string, maxRetries int) error {
	return c.modelManager.DownloadModelWithRetry(modelName, maxRetries)
}

// VerifyModel verifies model integrity and loading capability
func (c *Client) VerifyModel(modelName string) error {
	return c.modelManager.VerifyModel(modelName)
}

// GetRecommendedModel returns the recommended model for neovim keybinding search
func (c *Client) GetRecommendedModel() string {
	return c.modelManager.GetRecommendedModel()
}

// ListAvailableModels returns a list of locally available models
func (c *Client) ListAvailableModels() ([]string, error) {
	return c.modelManager.ListAvailableModels()
}

// Generate KeybindingResponse generates a response specifically for keybinding queries
func (c *Client) GenerateKeybindingResponse(query string) (*ParsedResponse, error) {
	if c.modelName == "" {
		return nil, fmt.Errorf("no model loaded")
	}

	// Create a specialized prompt for keybinding search
	prompt := c.buildKeybindingPrompt(query)

	// Generate response with timeout
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	response, err := c.generateWithContext(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// Parse the response
	parsed, err := c.responseParser.ParseKeybindingResponse(response, query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Validate the response
	if err := c.responseParser.ValidateResponse(parsed); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return parsed, nil
}

// buildKeybindingPrompt creates a specialized prompt for keybinding search
func (c *Client) buildKeybindingPrompt(query string) string {
	return fmt.Sprintf(`You are a Neovim expert assistant. The user is asking about keybindings and vim motions.

Query: "%s"

Please provide relevant vim/neovim keybindings that match this query. Format your response as follows:

For each relevant keybinding, provide:
- Keys: The actual key combination
- Command: The vim command name (if applicable)
- Description: What the keybinding does
- Mode: The vim mode (normal, insert, visual, command, terminal)
- Explanation: Why this keybinding is relevant to the query

Try to provide 3-5 most relevant results, ordered by relevance.

Example format:
dd: Delete current line
Command: delete
Mode: normal
Explanation: This deletes the entire line where the cursor is positioned.

Focus on practical, commonly used keybindings that directly address the user's query.`, query)
}

// generateWithContext generates text with context and timeout handling
func (c *Client) generateWithContext(ctx context.Context, prompt string) (string, error) {
	ollamaReq := OllamaGenerateRequest{
		Model:  c.modelName,
		Prompt: prompt,
		Stream: false,
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("request timeout exceeded")
		}
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp OllamaGenerateResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if ollamaResp.Error != "" {
		return "", fmt.Errorf("Ollama error: %s", ollamaResp.Error)
	}

	if strings.TrimSpace(ollamaResp.Response) == "" {
		return "", fmt.Errorf("empty response from model")
	}

	return ollamaResp.Response, nil
}

// SetRequestTimeout sets the timeout for HTTP requests
func (c *Client) SetRequestTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}
