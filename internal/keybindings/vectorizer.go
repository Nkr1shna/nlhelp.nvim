package keybindings

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"nvim-smart-keybind-search/internal/interfaces"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// KeybindingVectorizer handles vectorization of keybindings with change detection
type KeybindingVectorizer struct {
	parser    *KeybindingParser
	vectorDB  interfaces.VectorDB
	llmClient interfaces.LLMClient // Added for text embeddings
	hashStore map[string]string    // ID -> hash mapping for change detection
	mu        sync.RWMutex
	config    *VectorizerConfig
}

// VectorizerConfig holds configuration for the vectorizer
type VectorizerConfig struct {
	BatchSize             int
	IncludeDescription    bool
	IncludeMode           bool
	IncludePlugin         bool
	ContentTemplate       string
	EnableChangeDetection bool
}

// DefaultVectorizerConfig returns default configuration
func DefaultVectorizerConfig() *VectorizerConfig {
	return &VectorizerConfig{
		BatchSize:             50,
		IncludeDescription:    true,
		IncludeMode:           true,
		IncludePlugin:         true,
		ContentTemplate:       "{keys} {command} {description} {mode} {plugin}",
		EnableChangeDetection: true,
	}
}

// NewKeybindingVectorizer creates a new keybinding vectorizer
func NewKeybindingVectorizer(vectorDB interfaces.VectorDB, llmClient interfaces.LLMClient, config *VectorizerConfig) *KeybindingVectorizer {
	if config == nil {
		config = DefaultVectorizerConfig()
	}

	return &KeybindingVectorizer{
		parser:    NewKeybindingParser(),
		vectorDB:  vectorDB,
		llmClient: llmClient,
		hashStore: make(map[string]string),
		config:    config,
	}
}

// VectorizeKeybinding converts a single keybinding to a document for vector storage
func (v *KeybindingVectorizer) VectorizeKeybinding(kb *interfaces.Keybinding) (*interfaces.Document, error) {
	if err := v.parser.ValidateKeybinding(kb); err != nil {
		return nil, fmt.Errorf("invalid keybinding: %w", err)
	}

	// Generate content for vectorization
	content := v.generateContent(kb)

	// Generate text embedding using LLM client
	vector, err := v.generateEmbedding(content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding for keybinding %s: %w", kb.ID, err)
	}

	// Create metadata
	metadata := v.generateMetadata(kb)

	// Create document with vector
	doc := &interfaces.Document{
		ID:       chroma.DocumentID(kb.ID),
		Content:  content,
		Metadata: metadata,
		Vector:   vector,
	}

	return doc, nil
}

// VectorizeKeybindings converts multiple keybindings to documents
func (v *KeybindingVectorizer) VectorizeKeybindings(keybindings []interfaces.Keybinding) ([]interfaces.Document, error) {
	if len(keybindings) == 0 {
		return nil, nil
	}

	documents := make([]interfaces.Document, 0, len(keybindings))

	for i, kb := range keybindings {
		doc, err := v.VectorizeKeybinding(&kb)
		if err != nil {
			return nil, fmt.Errorf("failed to vectorize keybinding at index %d: %w", i, err)
		}
		documents = append(documents, *doc)
	}

	return documents, nil
}

// BatchVectorizeAndStore vectorizes and stores keybindings in batches for efficiency
func (v *KeybindingVectorizer) BatchVectorizeAndStore(keybindings []interfaces.Keybinding) error {
	if len(keybindings) == 0 {
		return nil
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	log.Printf("Starting batch vectorization of %d keybindings", len(keybindings))
	start := time.Now()

	// Process in batches
	batchSize := v.config.BatchSize
	for i := 0; i < len(keybindings); i += batchSize {
		end := i + batchSize
		if end > len(keybindings) {
			end = len(keybindings)
		}

		batch := keybindings[i:end]

		// Vectorize batch
		documents, err := v.VectorizeKeybindings(batch)
		if err != nil {
			return fmt.Errorf("failed to vectorize batch %d-%d: %w", i, end, err)
		}

		// Store batch
		if err := v.vectorDB.Store(documents); err != nil {
			return fmt.Errorf("failed to store batch %d-%d: %w", i, end, err)
		}

		// Update hash store for change detection
		if v.config.EnableChangeDetection {
			for _, kb := range batch {
				v.hashStore[kb.ID] = v.parser.GenerateHash(&kb)
			}
		}

		log.Printf("Processed batch %d-%d (%d keybindings)", i, end, len(batch))
	}

	duration := time.Since(start)
	log.Printf("Batch vectorization completed in %v", duration)

	return nil
}

// DetectChanges identifies which keybindings have changed since last vectorization
// Note: This method assumes the caller already holds the appropriate lock
func (v *KeybindingVectorizer) DetectChanges(keybindings []interfaces.Keybinding) ([]interfaces.Keybinding, []string, error) {
	if !v.config.EnableChangeDetection {
		// If change detection is disabled, treat all as new
		return keybindings, nil, nil
	}

	var changed []interfaces.Keybinding
	var deleted []string

	// Track which IDs we've seen in the current set
	currentIDs := make(map[string]bool)

	// Check for new or changed keybindings
	for _, kb := range keybindings {
		currentIDs[kb.ID] = true

		currentHash := v.parser.GenerateHash(&kb)
		storedHash, exists := v.hashStore[kb.ID]

		if !exists || storedHash != currentHash {
			// New or changed keybinding
			changed = append(changed, kb)
		}
	}

	// Check for deleted keybindings
	for id := range v.hashStore {
		if !currentIDs[id] {
			deleted = append(deleted, id)
		}
	}

	log.Printf("Change detection: %d changed, %d deleted", len(changed), len(deleted))
	return changed, deleted, nil
}

// detectChangesInternal is the internal implementation without locking
func (v *KeybindingVectorizer) detectChangesInternal(keybindings []interfaces.Keybinding) ([]interfaces.Keybinding, []string, error) {
	if !v.config.EnableChangeDetection {
		// If change detection is disabled, treat all as new
		return keybindings, nil, nil
	}

	var changed []interfaces.Keybinding
	var deleted []string

	// Track which IDs we've seen in the current set
	currentIDs := make(map[string]bool)

	// Check for new or changed keybindings
	for _, kb := range keybindings {
		currentIDs[kb.ID] = true

		currentHash := v.parser.GenerateHash(&kb)
		storedHash, exists := v.hashStore[kb.ID]

		if !exists || storedHash != currentHash {
			// New or changed keybinding
			changed = append(changed, kb)
		}
	}

	// Check for deleted keybindings
	for id := range v.hashStore {
		if !currentIDs[id] {
			deleted = append(deleted, id)
		}
	}

	log.Printf("Change detection: %d changed, %d deleted", len(changed), len(deleted))
	return changed, deleted, nil
}

// UpdateVectorDatabase performs incremental updates to the vector database
func (v *KeybindingVectorizer) UpdateVectorDatabase(keybindings []interfaces.Keybinding) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Detect changes (call internal method without locking)
	changed, deleted, err := v.detectChangesInternal(keybindings)
	if err != nil {
		return fmt.Errorf("failed to detect changes: %w", err)
	}

	// Delete removed keybindings
	if len(deleted) > 0 {
		if err := v.vectorDB.Delete(deleted); err != nil {
			return fmt.Errorf("failed to delete keybindings: %w", err)
		}

		// Remove from hash store
		for _, id := range deleted {
			delete(v.hashStore, id)
		}

		log.Printf("Deleted %d keybindings from vector database", len(deleted))
	}

	// Add/update changed keybindings
	if len(changed) > 0 {
		documents, err := v.VectorizeKeybindings(changed)
		if err != nil {
			return fmt.Errorf("failed to vectorize changed keybindings: %w", err)
		}

		if err := v.vectorDB.Store(documents); err != nil {
			return fmt.Errorf("failed to store changed keybindings: %w", err)
		}

		// Update hash store
		for _, kb := range changed {
			v.hashStore[kb.ID] = v.parser.GenerateHash(&kb)
		}

		log.Printf("Updated %d keybindings in vector database", len(changed))
	}

	return nil
}

// generateContent creates the text content for vectorization
func (v *KeybindingVectorizer) generateContent(kb *interfaces.Keybinding) string {
	var parts []string

	// Always include keys and command
	parts = append(parts, kb.Keys)
	parts = append(parts, kb.Command)

	// Include description if available and enabled
	if v.config.IncludeDescription && kb.Description != "" {
		parts = append(parts, kb.Description)
	}

	// Include mode if enabled
	if v.config.IncludeMode && kb.Mode != "" {
		parts = append(parts, fmt.Sprintf("mode:%s", kb.Mode))
	}

	// Include plugin if available and enabled
	if v.config.IncludePlugin && kb.Plugin != "" {
		parts = append(parts, fmt.Sprintf("plugin:%s", kb.Plugin))
	}

	// Add metadata as searchable content
	for key, value := range kb.Metadata {
		if value != "" {
			parts = append(parts, fmt.Sprintf("%s:%s", key, value))
		}
	}

	// Join all parts with spaces
	content := strings.Join(parts, " ")

	// Clean up the content
	content = strings.TrimSpace(content)
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\t", " ")

	// Remove multiple spaces
	words := strings.Fields(content)
	content = strings.Join(words, " ")

	return content
}

// generateMetadata creates metadata for the document
func (v *KeybindingVectorizer) generateMetadata(kb *interfaces.Keybinding) chroma.DocumentMetadata {
	metadataMap := make(map[string]interface{})

	// Copy original metadata
	for k, v := range kb.Metadata {
		metadataMap[k] = v
	}

	// Add keybinding-specific metadata
	metadataMap["keybinding_id"] = kb.ID
	metadataMap["keys"] = kb.Keys
	metadataMap["command"] = kb.Command
	metadataMap["mode"] = kb.Mode
	metadataMap["vectorized_at"] = time.Now().Format(time.RFC3339)

	if kb.Description != "" {
		metadataMap["description"] = kb.Description
	}

	if kb.Plugin != "" {
		metadataMap["plugin"] = kb.Plugin
	}

	// Add content length for potential filtering
	content := v.generateContent(kb)
	metadataMap["content_length"] = fmt.Sprintf("%d", len(content))

	// Add searchable flags
	metadataMap["has_description"] = fmt.Sprintf("%t", kb.Description != "")
	metadataMap["has_plugin"] = fmt.Sprintf("%t", kb.Plugin != "")

	// Convert to DocumentMetadata
	metadata, err := chroma.NewDocumentMetadataFromMap(metadataMap)
	if err != nil {
		// Fallback to empty metadata if conversion fails
		return chroma.NewDocumentMetadata()
	}

	return metadata
}

// LoadHashStore loads the hash store from existing vector database
func (v *KeybindingVectorizer) LoadHashStore() error {
	if !v.config.EnableChangeDetection {
		return nil
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	// Search for all documents to rebuild hash store
	// This is a simplified approach - in production you might want to store hashes separately
	results, err := v.vectorDB.Search("", 10000) // Large limit to get all documents
	if err != nil {
		log.Printf("Warning: failed to load hash store: %v", err)
		return nil // Don't fail initialization, just disable change detection for this session
	}

	// Rebuild hash store from existing documents
	for _, result := range results {
		if result.Document.Metadata != nil {
			if kbID, ok := result.Document.Metadata.GetString("keybinding_id"); ok {
				// We can't recreate the exact hash without the original keybinding,
				// so we'll use a placeholder that will force re-vectorization on first update
				v.hashStore[kbID] = "placeholder"
			}
		}
	}

	log.Printf("Loaded hash store with %d entries", len(v.hashStore))
	return nil
}

// ClearHashStore clears the hash store (useful for testing or reset)
func (v *KeybindingVectorizer) ClearHashStore() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.hashStore = make(map[string]string)
	log.Println("Hash store cleared")
}

// IncrementalUpdate performs an optimized incremental update with change detection
func (v *KeybindingVectorizer) IncrementalUpdate(keybindings []interfaces.Keybinding) (*UpdateResult, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	start := time.Now()
	log.Printf("Starting incremental update for %d keybindings", len(keybindings))

	// Detect changes
	changed, deleted, err := v.detectChangesInternal(keybindings)
	if err != nil {
		return nil, fmt.Errorf("failed to detect changes: %w", err)
	}

	result := &UpdateResult{
		TotalProcessed: len(keybindings),
		ChangedCount:   len(changed),
		DeletedCount:   len(deleted),
		StartTime:      start,
	}

	// Delete removed keybindings
	if len(deleted) > 0 {
		deleteStart := time.Now()
		if err := v.vectorDB.Delete(deleted); err != nil {
			return result, fmt.Errorf("failed to delete keybindings: %w", err)
		}

		// Remove from hash store
		for _, id := range deleted {
			delete(v.hashStore, id)
		}

		result.DeleteDuration = time.Since(deleteStart)
		log.Printf("Deleted %d keybindings in %v", len(deleted), result.DeleteDuration)
	}

	// Process changed keybindings
	if len(changed) > 0 {
		updateStart := time.Now()

		// Generate content for changed keybindings
		contents := make([]string, len(changed))
		for i, kb := range changed {
			contents[i] = v.generateContent(&kb)
		}

		// Generate embeddings for changed keybindings
		vectors, err := v.BatchGenerateEmbeddings(contents)
		if err != nil {
			return result, fmt.Errorf("failed to generate embeddings for changed keybindings: %w", err)
		}

		// Create documents
		documents := make([]interfaces.Document, len(changed))
		for i, kb := range changed {
			metadata := v.generateMetadata(&kb)
			documents[i] = interfaces.Document{
				ID:       chroma.DocumentID(kb.ID),
				Content:  contents[i],
				Metadata: metadata,
				Vector:   vectors[i],
			}
		}

		// Store updated documents
		if err := v.vectorDB.Store(documents); err != nil {
			return result, fmt.Errorf("failed to store changed keybindings: %w", err)
		}

		// Update hash store
		for _, kb := range changed {
			v.hashStore[kb.ID] = v.parser.GenerateHash(&kb)
		}

		result.UpdateDuration = time.Since(updateStart)
		log.Printf("Updated %d keybindings in %v", len(changed), result.UpdateDuration)
	}

	result.TotalDuration = time.Since(start)
	log.Printf("Incremental update completed in %v (changed: %d, deleted: %d)",
		result.TotalDuration, result.ChangedCount, result.DeletedCount)

	return result, nil
}

// UpdateResult contains statistics about an incremental update operation
type UpdateResult struct {
	TotalProcessed int
	ChangedCount   int
	DeletedCount   int
	StartTime      time.Time
	TotalDuration  time.Duration
	UpdateDuration time.Duration
	DeleteDuration time.Duration
}

// GetChangeDetectionStats returns detailed statistics about change detection
func (v *KeybindingVectorizer) GetChangeDetectionStats(keybindings []interfaces.Keybinding) (*ChangeDetectionStats, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.config.EnableChangeDetection {
		return &ChangeDetectionStats{
			Enabled: false,
		}, nil
	}

	stats := &ChangeDetectionStats{
		Enabled:          true,
		HashStoreSize:    len(v.hashStore),
		TotalKeybindings: len(keybindings),
	}

	// Analyze changes without modifying state
	currentIDs := make(map[string]bool)
	for _, kb := range keybindings {
		currentIDs[kb.ID] = true

		currentHash := v.parser.GenerateHash(&kb)
		storedHash, exists := v.hashStore[kb.ID]

		if !exists {
			stats.NewCount++
		} else if storedHash != currentHash {
			stats.ModifiedCount++
		} else {
			stats.UnchangedCount++
		}
	}

	// Count deleted keybindings
	for id := range v.hashStore {
		if !currentIDs[id] {
			stats.DeletedCount++
		}
	}

	return stats, nil
}

// ChangeDetectionStats contains statistics about change detection analysis
type ChangeDetectionStats struct {
	Enabled          bool
	HashStoreSize    int
	TotalKeybindings int
	NewCount         int
	ModifiedCount    int
	UnchangedCount   int
	DeletedCount     int
}

// RebuildHashStore rebuilds the hash store from current keybindings
func (v *KeybindingVectorizer) RebuildHashStore(keybindings []interfaces.Keybinding) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	log.Printf("Rebuilding hash store for %d keybindings", len(keybindings))
	start := time.Now()

	// Clear existing hash store
	v.hashStore = make(map[string]string)

	// Rebuild from current keybindings
	for _, kb := range keybindings {
		v.hashStore[kb.ID] = v.parser.GenerateHash(&kb)
	}

	duration := time.Since(start)
	log.Printf("Hash store rebuilt with %d entries in %v", len(v.hashStore), duration)

	return nil
}

// GetStats returns statistics about the vectorizer
func (v *KeybindingVectorizer) GetStats() map[string]interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["hash_store_size"] = len(v.hashStore)
	stats["change_detection_enabled"] = v.config.EnableChangeDetection
	stats["batch_size"] = v.config.BatchSize
	stats["include_description"] = v.config.IncludeDescription
	stats["include_mode"] = v.config.IncludeMode
	stats["include_plugin"] = v.config.IncludePlugin

	return stats
}

// generateEmbedding generates a text embedding for the given content
func (v *KeybindingVectorizer) generateEmbedding(content string) ([]float64, error) {
	if v.llmClient == nil {
		return nil, fmt.Errorf("LLM client is not initialized")
	}

	// Clean and prepare content for embedding
	cleanContent := v.prepareContentForEmbedding(content)
	if cleanContent == "" {
		return nil, fmt.Errorf("content is empty after cleaning")
	}

	// Generate embedding using LLM client
	vector, err := v.llmClient.Embed(cleanContent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(vector) == 0 {
		return nil, fmt.Errorf("received empty embedding vector")
	}

	return vector, nil
}

// prepareContentForEmbedding cleans and optimizes content for embedding generation
func (v *KeybindingVectorizer) prepareContentForEmbedding(content string) string {
	// Remove excessive whitespace
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	// Normalize whitespace
	words := strings.Fields(content)
	content = strings.Join(words, " ")

	// Limit content length to avoid token limits (typical models have ~512-2048 token limits)
	maxLength := 1000 // Conservative limit for embedding models
	if len(content) > maxLength {
		// Truncate at word boundary
		truncated := content[:maxLength]
		lastSpace := strings.LastIndex(truncated, " ")
		if lastSpace > maxLength/2 { // Only truncate at word boundary if it's not too short
			content = truncated[:lastSpace]
		} else {
			content = truncated
		}
	}

	return content
}

// BatchGenerateEmbeddings generates embeddings for multiple content strings efficiently
func (v *KeybindingVectorizer) BatchGenerateEmbeddings(contents []string) ([][]float64, error) {
	if len(contents) == 0 {
		return nil, nil
	}

	vectors := make([][]float64, len(contents))
	errors := make([]error, len(contents))

	// Process embeddings with controlled concurrency to avoid overwhelming the LLM service
	maxConcurrency := 5 // Limit concurrent embedding requests
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, content := range contents {
		wg.Add(1)
		go func(index int, text string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			vector, err := v.generateEmbedding(text)
			vectors[index] = vector
			errors[index] = err
		}(i, content)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for content %d: %w", i, err)
		}
	}

	return vectors, nil
}

// OptimizedBatchVectorizeAndStore vectorizes and stores keybindings with optimized batch embedding
func (v *KeybindingVectorizer) OptimizedBatchVectorizeAndStore(keybindings []interfaces.Keybinding) error {
	if len(keybindings) == 0 {
		return nil
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	log.Printf("Starting optimized batch vectorization of %d keybindings", len(keybindings))
	start := time.Now()

	// Prepare all content for embedding generation
	contents := make([]string, len(keybindings))
	for i, kb := range keybindings {
		contents[i] = v.generateContent(&kb)
	}

	// Generate all embeddings in optimized batches
	log.Printf("Generating embeddings for %d keybindings", len(contents))
	embedStart := time.Now()
	vectors, err := v.BatchGenerateEmbeddings(contents)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}
	log.Printf("Embedding generation completed in %v", time.Since(embedStart))

	// Create documents with pre-generated embeddings
	documents := make([]interfaces.Document, len(keybindings))
	for i, kb := range keybindings {
		metadata := v.generateMetadata(&kb)
		documents[i] = interfaces.Document{
			ID:       chroma.DocumentID(kb.ID),
			Content:  contents[i],
			Metadata: metadata,
			Vector:   vectors[i],
		}
	}

	// Store documents in batches
	batchSize := v.config.BatchSize
	for i := 0; i < len(documents); i += batchSize {
		end := i + batchSize
		if end > len(documents) {
			end = len(documents)
		}

		batch := documents[i:end]
		if err := v.vectorDB.Store(batch); err != nil {
			return fmt.Errorf("failed to store batch %d-%d: %w", i, end, err)
		}

		// Update hash store for change detection
		if v.config.EnableChangeDetection {
			for j := i; j < end; j++ {
				v.hashStore[keybindings[j].ID] = v.parser.GenerateHash(&keybindings[j])
			}
		}

		log.Printf("Stored batch %d-%d (%d documents)", i, end, len(batch))
	}

	duration := time.Since(start)
	log.Printf("Optimized batch vectorization completed in %v", duration)

	return nil
}

// ValidateConfiguration validates the vectorizer configuration
func (v *KeybindingVectorizer) ValidateConfiguration() error {
	if v.config.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}

	if v.config.ContentTemplate == "" {
		return fmt.Errorf("content template cannot be empty")
	}

	if v.vectorDB == nil {
		return fmt.Errorf("vector database is required")
	}

	if v.llmClient == nil {
		return fmt.Errorf("LLM client is required")
	}

	return nil
}
