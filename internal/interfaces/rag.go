package interfaces

// QueryRequest represents a search query from the client
type QueryRequest struct {
	Query   string            `json:"query"`
	Context map[string]string `json:"context,omitempty"`
	Limit   int               `json:"limit,omitempty"`
}

// QueryResult represents the response to a search query
type QueryResult struct {
	Results   []SearchResult `json:"results"`
	Reasoning string         `json:"reasoning"`
	Error     string         `json:"error,omitempty"`
}

// SearchResult represents a single keybinding search result
type SearchResult struct {
	Keybinding  Keybinding `json:"keybinding"`
	Relevance   float64    `json:"relevance"`
	Explanation string     `json:"explanation"`
}

// Keybinding represents a vim keybinding
type Keybinding struct {
	ID          string            `json:"id"`
	Keys        string            `json:"keys"`
	Command     string            `json:"command"`
	Description string            `json:"description"`
	Mode        string            `json:"mode"`
	Plugin      string            `json:"plugin,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// RAGAgent defines the interface for the RAG-based search agent
type RAGAgent interface {
	// ProcessQuery processes a natural language query and returns relevant keybindings
	ProcessQuery(query string) (*QueryResult, error)
	
	// UpdateVectorDB updates the vector database with new keybindings
	UpdateVectorDB(keybindings []Keybinding) error
	
	// Initialize sets up the RAG agent with required dependencies
	Initialize() error
	
	// HealthCheck verifies the agent is functioning properly
	HealthCheck() error
}