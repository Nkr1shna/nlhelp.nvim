package chromadb

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

// VectorService provides high-level vector operations with enhanced functionality
type VectorService struct {
	client            *Client
	collectionManager *CollectionManager
	initializer       *DatabaseInitializer
	config            *VectorServiceConfig
	mu                sync.RWMutex
}

// VectorServiceConfig holds configuration for the vector service
type VectorServiceConfig struct {
	BatchSize           int
	SimilarityThreshold float64
	EnableCaching       bool
}

// DefaultVectorServiceConfig returns default configuration
func DefaultVectorServiceConfig() *VectorServiceConfig {
	return &VectorServiceConfig{
		BatchSize:           100,
		SimilarityThreshold: 0.7,
		EnableCaching:       true,
	}
}

// SearchOptions provides options for search operations
type SearchOptions struct {
	Limit               int
	SimilarityThreshold float64
	IncludeMetadata     bool
	FilterBySource      string // "user", "builtin", or "" for both
	BoostUserResults    bool
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		Limit:               10,
		SimilarityThreshold: 0.0,
		IncludeMetadata:     true,
		FilterBySource:      "",
		BoostUserResults:    true,
	}
}

// BatchOperation represents a batch operation
type BatchOperation struct {
	Type      string // "store", "delete"
	Documents []interfaces.Document
	IDs       []string
}

// NewVectorService creates a new vector service
func NewVectorService(config *Config) (*VectorService, error) {
	if config == nil {
		config = DefaultConfig()
	}

	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChromaDB client: %w", err)
	}

	collectionManager := NewCollectionManager(client)
	initializer := NewDatabaseInitializer(config)

	return &VectorService{
		client:            client,
		collectionManager: collectionManager,
		initializer:       initializer,
		config:            DefaultVectorServiceConfig(),
	}, nil
}

// Initialize initializes the vector service
func (vs *VectorService) Initialize() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Initialize database from pre-built data
	if err := vs.initializer.InitializeDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize ChromaDB client
	if err := vs.client.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize ChromaDB client: %w", err)
	}

	// Initialize collection manager
	if err := vs.collectionManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize collection manager: %w", err)
	}

	log.Println("Vector service initialized successfully")
	return nil
}

// Store stores documents with metadata handling
func (vs *VectorService) Store(documents []interfaces.Document) error {
	if len(documents) == 0 {
		return nil
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Enhance documents with additional metadata
	enhancedDocs := vs.enhanceDocuments(documents)

	// Use collection manager to store in appropriate collection
	return vs.collectionManager.StoreUserKeybindings(enhancedDocs)
}

// StoreBuiltinKnowledge stores built-in vim knowledge
func (vs *VectorService) StoreBuiltinKnowledge(documents []interfaces.Document) error {
	if len(documents) == 0 {
		return nil
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Enhance documents with builtin metadata
	for i := range documents {
		metadataMap := map[string]interface{}{
			"source":     "builtin",
			"created_at": time.Now().Format(time.RFC3339),
		}

		metadata, err := chroma.NewDocumentMetadataFromMap(metadataMap)
		if err != nil {
			// Fallback to empty metadata if conversion fails
			metadata = chroma.NewDocumentMetadata()
		}
		documents[i].Metadata = metadata
	}

	return vs.collectionManager.StoreBuiltinKnowledge(documents)
}

// BatchStore performs batch storage operations efficiently
func (vs *VectorService) BatchStore(documents []interfaces.Document) error {
	if len(documents) == 0 {
		return nil
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Process documents in batches
	batchSize := vs.config.BatchSize
	for i := 0; i < len(documents); i += batchSize {
		end := i + batchSize
		if end > len(documents) {
			end = len(documents)
		}

		batch := documents[i:end]
		enhancedBatch := vs.enhanceDocuments(batch)

		if err := vs.collectionManager.StoreUserKeybindings(enhancedBatch); err != nil {
			return fmt.Errorf("failed to store batch %d-%d: %w", i, end, err)
		}

		log.Printf("Stored batch %d-%d (%d documents)", i, end, len(batch))
	}

	return nil
}

// Search performs semantic search with enhanced options
func (vs *VectorService) Search(query string, options *SearchOptions) ([]interfaces.VectorSearchResult, error) {
	if options == nil {
		options = DefaultSearchOptions()
	}

	vs.mu.RLock()
	defer vs.mu.RUnlock()

	var results []interfaces.VectorSearchResult
	var err error

	// Search based on filter options
	switch options.FilterBySource {
	case "user":
		results, err = vs.searchUserOnly(query, options.Limit)
	case "builtin":
		results, err = vs.searchBuiltinOnly(query, options.Limit)
	default:
		results, err = vs.collectionManager.SearchBoth(query, options.Limit)
	}

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Apply post-processing
	results = vs.postProcessResults(results, options)

	return results, nil
}

// searchUserOnly searches only user keybindings
func (vs *VectorService) searchUserOnly(query string, limit int) ([]interfaces.VectorSearchResult, error) {
	// Temporarily switch to user collection
	originalCollection := vs.client.collection
	vs.client.collection = vs.collectionManager.client.collection
	defer func() {
		vs.client.collection = originalCollection
	}()

	return vs.client.Search(query, limit)
}

// searchBuiltinOnly searches only built-in knowledge
func (vs *VectorService) searchBuiltinOnly(query string, limit int) ([]interfaces.VectorSearchResult, error) {
	// Get the built-in collection
	collection, err := vs.client.getCollection(vs.collectionManager.builtinCollName)
	if err != nil {
		return nil, fmt.Errorf("failed to get built-in collection: %w", err)
	}

	// Temporarily switch to builtin collection
	originalCollection := vs.client.collection
	vs.client.collection = collection
	defer func() {
		vs.client.collection = originalCollection
	}()

	return vs.client.Search(query, limit)
}

// enhanceDocuments adds metadata and preprocessing to documents
func (vs *VectorService) enhanceDocuments(documents []interfaces.Document) []interfaces.Document {
	enhanced := make([]interfaces.Document, len(documents))

	for i, doc := range documents {
		enhanced[i] = doc

		// Create metadata using v2 API
		metadataMap := make(map[string]interface{})

		// If existing metadata exists, preserve it
		if enhanced[i].Metadata != nil {
			// Note: We can't easily extract from DocumentMetadata interface
			// For now, we'll create new metadata
		}

		// Add timestamp
		metadataMap["created_at"] = time.Now().Format(time.RFC3339)
		metadataMap["source"] = "user"

		// Add content length for potential filtering
		metadataMap["content_length"] = fmt.Sprintf("%d", len(doc.Content))

		// Add content hash for deduplication
		metadataMap["content_hash"] = vs.calculateContentHash(doc.Content)

		// Convert to DocumentMetadata
		metadata, err := chroma.NewDocumentMetadataFromMap(metadataMap)
		if err != nil {
			// Fallback to empty metadata if conversion fails
			metadata = chroma.NewDocumentMetadata()
		}
		enhanced[i].Metadata = metadata

		// Normalize content for better search
		enhanced[i].Content = vs.normalizeContent(doc.Content)
	}

	return enhanced
}

// normalizeContent normalizes content for better search results
func (vs *VectorService) normalizeContent(content string) string {
	// Convert to lowercase for case-insensitive search
	normalized := strings.ToLower(content)

	// Remove extra whitespace
	normalized = strings.TrimSpace(normalized)

	// Replace multiple spaces with single space
	words := strings.Fields(normalized)
	normalized = strings.Join(words, " ")

	return normalized
}

// calculateContentHash calculates a simple hash for content deduplication
func (vs *VectorService) calculateContentHash(content string) string {
	// Simple hash based on content length and first/last characters
	if len(content) == 0 {
		return "empty"
	}

	hash := fmt.Sprintf("%d_%c_%c", len(content), content[0], content[len(content)-1])
	return hash
}

// postProcessResults applies post-processing to search results
func (vs *VectorService) postProcessResults(results []interfaces.VectorSearchResult, options *SearchOptions) []interfaces.VectorSearchResult {
	// Filter by similarity threshold
	if options.SimilarityThreshold > 0 {
		filtered := make([]interfaces.VectorSearchResult, 0, len(results))
		for _, result := range results {
			if result.Score >= options.SimilarityThreshold {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	// Boost user results if enabled
	if options.BoostUserResults {
		for i := range results {
			if results[i].Document.Metadata != nil {
				if source, ok := results[i].Document.Metadata.GetString("source"); ok && source == "user" {
					results[i].Score += 0.1 // Small boost for user keybindings
				}
			}
		}
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Remove metadata if not requested
	if !options.IncludeMetadata {
		for i := range results {
			results[i].Document.Metadata = nil
		}
	}

	// Apply limit
	if len(results) > options.Limit {
		results = results[:options.Limit]
	}

	return results
}

// Delete removes documents by their IDs
func (vs *VectorService) Delete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	return vs.collectionManager.DeleteUserKeybindings(ids)
}

// BatchDelete performs batch deletion operations
func (vs *VectorService) BatchDelete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Process deletions in batches
	batchSize := vs.config.BatchSize
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		batch := ids[i:end]
		if err := vs.collectionManager.DeleteUserKeybindings(batch); err != nil {
			return fmt.Errorf("failed to delete batch %d-%d: %w", i, end, err)
		}

		log.Printf("Deleted batch %d-%d (%d documents)", i, end, len(batch))
	}

	return nil
}

// GetStats returns statistics about the vector database
func (vs *VectorService) GetStats() (map[string]interface{}, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	stats := make(map[string]interface{})

	// Get collection counts
	userCount, err := vs.collectionManager.GetUserCollectionCount()
	if err != nil {
		log.Printf("Warning: failed to get user collection count: %v", err)
		userCount = -1
	}

	builtinCount, err := vs.collectionManager.GetBuiltinCollectionCount()
	if err != nil {
		log.Printf("Warning: failed to get builtin collection count: %v", err)
		builtinCount = -1
	}

	stats["user_documents"] = userCount
	stats["builtin_documents"] = builtinCount
	stats["total_documents"] = userCount + builtinCount

	// Get database info
	dbInfo := vs.initializer.GetDatabaseInfo()
	for k, v := range dbInfo {
		stats[k] = v
	}

	// Add service configuration
	stats["batch_size"] = vs.config.BatchSize
	stats["similarity_threshold"] = vs.config.SimilarityThreshold
	stats["cache_enabled"] = vs.config.EnableCaching

	return stats, nil
}

// HealthCheck verifies the service is working correctly
func (vs *VectorService) HealthCheck() error {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	// Check ChromaDB connection
	if err := vs.client.HealthCheck(); err != nil {
		return fmt.Errorf("ChromaDB health check failed: %w", err)
	}

	// Check collections exist by trying to get their counts
	if _, err := vs.collectionManager.GetUserCollectionCount(); err != nil {
		return fmt.Errorf("user collection not accessible: %w", err)
	}

	if _, err := vs.collectionManager.GetBuiltinCollectionCount(); err != nil {
		return fmt.Errorf("builtin collection not accessible: %w", err)
	}

	return nil
}

// Close closes the vector service
func (vs *VectorService) Close() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	return vs.client.Close()
}

// ClearUserData removes all user keybindings
func (vs *VectorService) ClearUserData() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	return vs.collectionManager.ClearUserCollection()
}

// ExecuteBatchOperations executes multiple operations in a batch
func (vs *VectorService) ExecuteBatchOperations(operations []BatchOperation) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	for i, op := range operations {
		switch op.Type {
		case "store":
			if err := vs.collectionManager.StoreUserKeybindings(op.Documents); err != nil {
				return fmt.Errorf("failed to execute store operation %d: %w", i, err)
			}
		case "delete":
			if err := vs.collectionManager.DeleteUserKeybindings(op.IDs); err != nil {
				return fmt.Errorf("failed to execute delete operation %d: %w", i, err)
			}
		default:
			return fmt.Errorf("unknown batch operation type: %s", op.Type)
		}
	}

	return nil
}
