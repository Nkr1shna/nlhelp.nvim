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
}

// NewCollectionManager creates a new collection manager
func NewCollectionManager(client *Client) *CollectionManager {
	return &CollectionManager{
		client:          client,
		builtinCollName: "vim_knowledge",
		userCollName:    "user_keybindings",
	}
}

// Initialize sets up both built-in and user collections
func (cm *CollectionManager) Initialize() error {
	// Initialize built-in vim knowledge collection
	if _, err := cm.client.getOrCreateCollection(cm.builtinCollName); err != nil {
		return fmt.Errorf("failed to initialize built-in collection: %w", err)
	}

	// Initialize user keybindings collection
	if _, err := cm.client.getOrCreateCollection(cm.userCollName); err != nil {
		return fmt.Errorf("failed to initialize user collection: %w", err)
	}

	log.Printf("Collection manager initialized with collections: %s, %s",
		cm.builtinCollName, cm.userCollName)
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

// SearchBoth searches both collections and merges results with user keybindings prioritized
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

// searchCollection searches a specific collection
func (cm *CollectionManager) searchCollection(collectionName string, query string, limit int) ([]interfaces.VectorSearchResult, error) {
	return cm.client.SearchInCollection(query, limit, collectionName)
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

// GetUserCollectionCount returns the number of documents in the user collection
func (cm *CollectionManager) GetUserCollectionCount() (int, error) {
	return cm.client.GetCollectionCountByName(cm.userCollName)
}

// GetBuiltinCollectionCount returns the number of documents in the built-in collection
func (cm *CollectionManager) GetBuiltinCollectionCount() (int, error) {
	return cm.client.GetCollectionCountByName(cm.builtinCollName)
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
