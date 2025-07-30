package chromadb

import (
	"fmt"
	"log"

	"nvim-smart-keybind-search/internal/interfaces"
)

// CollectionManager manages multiple ChromaDB collections
type CollectionManager struct {
	client          *Client
	builtinCollName string
	userCollName    string
	generalCollName string
}

// NewCollectionManager creates a new collection manager
func NewCollectionManager(client *Client) *CollectionManager {
	return &CollectionManager{
		client:          client,
		builtinCollName: "vim_knowledge",
		userCollName:    "user_keybindings",
		generalCollName: "general_knowledge",
	}
}

// Initialize sets up all collections
func (cm *CollectionManager) Initialize() error {
	// Initialize built-in vim knowledge collection
	if _, err := cm.client.getOrCreateCollection(cm.builtinCollName); err != nil {
		return fmt.Errorf("failed to initialize built-in collection: %w", err)
	}

	// Initialize user keybindings collection
	if _, err := cm.client.getOrCreateCollection(cm.userCollName); err != nil {
		return fmt.Errorf("failed to initialize user collection: %w", err)
	}

	// Initialize general knowledge collection
	if _, err := cm.client.getOrCreateCollection(cm.generalCollName); err != nil {
		return fmt.Errorf("failed to initialize general knowledge collection: %w", err)
	}

	log.Printf("Collection manager initialized with collections: %s, %s, %s",
		cm.builtinCollName, cm.userCollName, cm.generalCollName)
	return nil
}

// StoreBuiltinKnowledge stores documents in the built-in vim knowledge collection
func (cm *CollectionManager) StoreBuiltinKnowledge(documents []interfaces.Document) error {
	return cm.client.StoreInCollection(documents, cm.builtinCollName)
}

// StoreUserKeybindings stores documents in the user keybindings collection
func (cm *CollectionManager) StoreUserKeybindings(documents []interfaces.Document) error {
	return cm.client.StoreInCollection(documents, cm.userCollName)
}

// StoreGeneralKnowledge stores documents in the general knowledge collection
func (cm *CollectionManager) StoreGeneralKnowledge(documents []interfaces.Document) error {
	return cm.client.StoreInCollection(documents, cm.generalCollName)
}

// SearchAllCollections searches all collections and returns combined results
func (cm *CollectionManager) SearchAllCollections(query string, limit int) ([]interfaces.VectorSearchResult, error) {
	// Distribute limit across collections
	limitPerCollection := limit / 3
	if limitPerCollection < 1 {
		limitPerCollection = 1
	}

	// Search user keybindings (highest priority)
	userResults, err := cm.searchCollection(cm.userCollName, query, limitPerCollection)
	if err != nil {
		log.Printf("Warning: failed to search user collection: %v", err)
		userResults = []interfaces.VectorSearchResult{}
	}

	// Search built-in knowledge
	builtinResults, err := cm.searchCollection(cm.builtinCollName, query, limitPerCollection)
	if err != nil {
		log.Printf("Warning: failed to search built-in collection: %v", err)
		builtinResults = []interfaces.VectorSearchResult{}
	}

	// Search general knowledge
	generalResults, err := cm.searchCollection(cm.generalCollName, query, limitPerCollection)
	if err != nil {
		log.Printf("Warning: failed to search general knowledge collection: %v", err)
		generalResults = []interfaces.VectorSearchResult{}
	}

	// Merge all results with priority ordering
	return cm.mergeAllResults(userResults, builtinResults, generalResults, limit), nil
}

// SearchBoth searches both keybinding collections and merges results with user keybindings prioritized
func (cm *CollectionManager) SearchBoth(query string, limit int) ([]interfaces.VectorSearchResult, error) {
	// Search user keybindings first (higher priority)
	userResults, err := cm.searchCollection(cm.userCollName, query, limit/2)
	if err != nil {
		log.Printf("Warning: failed to search user collection: %v", err)
		userResults = []interfaces.VectorSearchResult{}
	}

	// Search built-in knowledge
	remainingLimit := limit - len(userResults)
	if remainingLimit <= 0 {
		remainingLimit = limit / 2 // Ensure we get some built-in results
	}

	builtinResults, err := cm.searchCollection(cm.builtinCollName, query, remainingLimit)
	if err != nil {
		log.Printf("Warning: failed to search built-in collection: %v", err)
		builtinResults = []interfaces.VectorSearchResult{}
	}

	// Merge results with user keybindings prioritized
	return cm.mergeResults(userResults, builtinResults, limit), nil
}

// SearchGeneralKnowledge searches only the general knowledge collection
func (cm *CollectionManager) SearchGeneralKnowledge(query string, limit int) ([]interfaces.VectorSearchResult, error) {
	return cm.searchCollection(cm.generalCollName, query, limit)
}

// searchCollection searches a specific collection
func (cm *CollectionManager) searchCollection(collectionName string, query string, limit int) ([]interfaces.VectorSearchResult, error) {
	return cm.client.SearchInCollection(query, limit, collectionName)
}

// mergeAllResults merges results from all collections with priority ordering
func (cm *CollectionManager) mergeAllResults(userResults, builtinResults, generalResults []interfaces.VectorSearchResult, limit int) []interfaces.VectorSearchResult {
	merged := make([]interfaces.VectorSearchResult, 0, len(userResults)+len(builtinResults)+len(generalResults))

	// Add user results first (highest priority) with score boost
	for _, result := range userResults {
		// Apply score boost for user results
		result.Score += 0.2
		merged = append(merged, result)
	}

	// Add built-in results with moderate boost
	for _, result := range builtinResults {
		result.Score += 0.1
		merged = append(merged, result)
	}

	// Add general knowledge results (no boost, but still included)
	merged = append(merged, generalResults...)

	// Remove duplicates based on document ID
	seen := make(map[string]bool)
	finalResults := make([]interfaces.VectorSearchResult, 0, len(merged))

	for _, result := range merged {
		if !seen[string(result.Document.ID)] {
			finalResults = append(finalResults, result)
			seen[string(result.Document.ID)] = true
		}
	}

	// Limit results
	if len(finalResults) > limit {
		finalResults = finalResults[:limit]
	}

	return finalResults
}

// mergeResults merges user and built-in results with user results prioritized
func (cm *CollectionManager) mergeResults(userResults, builtinResults []interfaces.VectorSearchResult, limit int) []interfaces.VectorSearchResult {
	merged := make([]interfaces.VectorSearchResult, 0, len(userResults)+len(builtinResults))

	// Add user results first (higher priority) with score boost
	for _, result := range userResults {
		// Apply score boost for user results
		result.Score += 0.1
		merged = append(merged, result)
	}

	// Add built-in results, avoiding duplicates
	seen := make(map[string]bool)
	for _, result := range userResults {
		seen[string(result.Document.ID)] = true
	}

	for _, result := range builtinResults {
		if !seen[string(result.Document.ID)] {
			merged = append(merged, result)
			seen[string(result.Document.ID)] = true
		}
	}

	// Limit results
	if len(merged) > limit {
		merged = merged[:limit]
	}

	return merged
}

// DeleteUserKeybindings deletes documents from the user keybindings collection
func (cm *CollectionManager) DeleteUserKeybindings(ids []string) error {
	return cm.client.DeleteFromCollection(ids, cm.userCollName)
}

// DeleteGeneralKnowledge deletes documents from the general knowledge collection
func (cm *CollectionManager) DeleteGeneralKnowledge(ids []string) error {
	return cm.client.DeleteFromCollection(ids, cm.generalCollName)
}

// GetUserCollectionCount returns the number of documents in the user collection
func (cm *CollectionManager) GetUserCollectionCount() (int, error) {
	return cm.client.GetCollectionCountByName(cm.userCollName)
}

// GetBuiltinCollectionCount returns the number of documents in the built-in collection
func (cm *CollectionManager) GetBuiltinCollectionCount() (int, error) {
	return cm.client.GetCollectionCountByName(cm.builtinCollName)
}

// GetGeneralKnowledgeCount returns the number of documents in the general knowledge collection
func (cm *CollectionManager) GetGeneralKnowledgeCount() (int, error) {
	return cm.client.GetCollectionCountByName(cm.generalCollName)
}

// ClearUserCollection deletes all documents from the user collection
func (cm *CollectionManager) ClearUserCollection() error {
	// Get all document IDs from the user collection
	// This is a simplified approach - in a real implementation, you might want to
	// get all IDs first, then delete them
	log.Printf("Clearing user collection: %s", cm.userCollName)

	// For now, we'll just log that we're clearing the collection
	// In a real implementation, you'd need to get all IDs first
	return fmt.Errorf("clear user collection not implemented - need to get all IDs first")
}

// ClearGeneralKnowledge deletes all documents from the general knowledge collection
func (cm *CollectionManager) ClearGeneralKnowledge() error {
	log.Printf("Clearing general knowledge collection: %s", cm.generalCollName)
	return fmt.Errorf("clear general knowledge collection not implemented - need to get all IDs first")
}
