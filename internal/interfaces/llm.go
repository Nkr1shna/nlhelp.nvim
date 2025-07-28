package interfaces

// LLMRequest represents a request to the language model
type LLMRequest struct {
	Prompt      string            `json:"prompt"`
	Context     string            `json:"context,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// LLMResponse represents a response from the language model
type LLMResponse struct {
	Text     string            `json:"text"`
	Tokens   int               `json:"tokens"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Error    string            `json:"error,omitempty"`
}

// ModelInfo represents information about the loaded model
type ModelInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Size    string `json:"size"`
	Status  string `json:"status"`
}

// LLMClient defines the interface for language model operations
type LLMClient interface {
	// Generate generates text based on the given request
	Generate(request LLMRequest) (*LLMResponse, error)
	
	// Embed generates embeddings for the given text
	Embed(text string) ([]float64, error)
	
	// LoadModel loads a specific model for inference
	LoadModel(modelName string) error
	
	// GetModelInfo returns information about the currently loaded model
	GetModelInfo() (*ModelInfo, error)
	
	// Initialize sets up the LLM client and verifies connectivity
	Initialize() error
	
	// HealthCheck verifies the LLM service is available and responsive
	HealthCheck() error
	
	// Close closes the LLM client connection
	Close() error
}