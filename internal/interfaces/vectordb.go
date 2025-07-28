package interfaces

import (
	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// Document represents a document stored in the vector database
type Document struct {
	ID       chroma.DocumentID       `json:"id"`
	Content  string                  `json:"content"`
	Metadata chroma.DocumentMetadata `json:"metadata"`
	Vector   []float64               `json:"vector,omitempty"`
}

// VectorSearchResult represents a result from vector similarity search
type VectorSearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
	Distance float64  `json:"distance"`
}

// VectorDB defines the interface for vector database operations
type VectorDB interface {
	// Store stores documents in the vector database
	Store(documents []Document) error

	// Search performs semantic search and returns similar documents
	Search(query string, limit int) ([]VectorSearchResult, error)

	// Delete removes documents by their IDs
	Delete(ids []string) error

	// Initialize sets up the vector database connection and collections
	Initialize() error

	// HealthCheck verifies the database connection is working
	HealthCheck() error

	// Close closes the database connection
	Close() error
}
