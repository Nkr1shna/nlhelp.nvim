package keybindings

import (
	"strings"
	"testing"

	"nvim-smart-keybind-search/internal/interfaces"
)

// MockVectorDB is a mock implementation of the VectorDB interface for testing
type MockVectorDB struct {
	documents []interfaces.Document
	deleted   []string
}

func NewMockVectorDB() *MockVectorDB {
	return &MockVectorDB{
		documents: make([]interfaces.Document, 0),
		deleted:   make([]string, 0),
	}
}

func (m *MockVectorDB) Store(documents []interfaces.Document) error {
	// For each new document, replace existing one with same ID or append if new
	for _, newDoc := range documents {
		found := false
		for i, existingDoc := range m.documents {
			if string(existingDoc.ID) == string(newDoc.ID) {
				m.documents[i] = newDoc // Replace existing document
				found = true
				break
			}
		}
		if !found {
			m.documents = append(m.documents, newDoc) // Add new document
		}
	}
	return nil
}

func (m *MockVectorDB) Search(query string, limit int) ([]interfaces.VectorSearchResult, error) {
	results := make([]interfaces.VectorSearchResult, 0)
	for _, doc := range m.documents {
		results = append(results, interfaces.VectorSearchResult{
			Document: doc,
			Score:    0.8,
			Distance: 0.2,
		})
	}
	return results, nil
}

func (m *MockVectorDB) Delete(ids []string) error {
	m.deleted = append(m.deleted, ids...)
	// Remove from documents
	filtered := make([]interfaces.Document, 0)
	for _, doc := range m.documents {
		shouldDelete := false
		for _, id := range ids {
			if string(doc.ID) == id {
				shouldDelete = true
				break
			}
		}
		if !shouldDelete {
			filtered = append(filtered, doc)
		}
	}
	m.documents = filtered
	return nil
}

func (m *MockVectorDB) Initialize() error {
	return nil
}

func (m *MockVectorDB) HealthCheck() error {
	return nil
}

func (m *MockVectorDB) Close() error {
	return nil
}

// MockLLMClient is a mock implementation of the LLMClient interface for testing
type MockLLMClient struct {
	embeddings map[string][]float64
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		embeddings: make(map[string][]float64),
	}
}

func (m *MockLLMClient) Generate(request interfaces.LLMRequest) (*interfaces.LLMResponse, error) {
	return &interfaces.LLMResponse{
		Text:   "mock response",
		Tokens: 10,
	}, nil
}

func (m *MockLLMClient) Embed(text string) ([]float64, error) {
	// Return a mock embedding based on text hash for consistency
	if embedding, exists := m.embeddings[text]; exists {
		return embedding, nil
	}

	// Generate a simple mock embedding (384 dimensions is common)
	embedding := make([]float64, 384)
	for i := range embedding {
		embedding[i] = float64(len(text)+i) / 1000.0 // Simple deterministic embedding
	}

	m.embeddings[text] = embedding
	return embedding, nil
}

func (m *MockLLMClient) LoadModel(modelName string) error {
	return nil
}

func (m *MockLLMClient) GetModelInfo() (*interfaces.ModelInfo, error) {
	return &interfaces.ModelInfo{
		Name:    "mock-model",
		Version: "1.0",
		Size:    "100MB",
		Status:  "loaded",
	}, nil
}

func (m *MockLLMClient) Initialize() error {
	return nil
}

func (m *MockLLMClient) HealthCheck() error {
	return nil
}

func (m *MockLLMClient) Close() error {
	return nil
}

func TestNewKeybindingVectorizer(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	if vectorizer == nil {
		t.Fatal("NewKeybindingVectorizer returned nil")
	}

	if vectorizer.parser == nil {
		t.Error("parser not initialized")
	}

	if vectorizer.vectorDB == nil {
		t.Error("vectorDB not initialized")
	}

	if vectorizer.llmClient == nil {
		t.Error("llmClient not initialized")
	}

	if vectorizer.hashStore == nil {
		t.Error("hashStore not initialized")
	}

	if vectorizer.config == nil {
		t.Error("config not initialized")
	}
}

func TestVectorizeKeybinding(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	kb := &interfaces.Keybinding{
		ID:          "test1",
		Keys:        "dd",
		Command:     "delete line",
		Description: "Delete current line",
		Mode:        "n",
		Plugin:      "builtin",
		Metadata: map[string]string{
			"category": "edit",
		},
	}

	doc, err := vectorizer.VectorizeKeybinding(kb)
	if err != nil {
		t.Fatalf("VectorizeKeybinding failed: %v", err)
	}

	if string(doc.ID) != kb.ID {
		t.Errorf("Document ID mismatch: %s != %s", string(doc.ID), kb.ID)
	}

	// Check that content contains key elements
	if doc.Content == "" {
		t.Error("Document content is empty")
	}

	// Check that vector was generated
	if len(doc.Vector) == 0 {
		t.Error("Document vector is empty")
	}

	// Content should contain keys, command, description
	expectedParts := []string{"dd", "delete line", "Delete current line"}
	for _, part := range expectedParts {
		if !contains(doc.Content, part) {
			t.Errorf("Content missing expected part: %s", part)
		}
	}

	// Check metadata
	if keybindingID, _ := doc.Metadata.GetString("keybinding_id"); keybindingID != kb.ID {
		t.Errorf("Metadata keybinding_id mismatch: %s != %s", keybindingID, kb.ID)
	}

	if keys, _ := doc.Metadata.GetString("keys"); keys != kb.Keys {
		t.Errorf("Metadata keys mismatch: %s != %s", keys, kb.Keys)
	}

	if command, _ := doc.Metadata.GetString("command"); command != kb.Command {
		t.Errorf("Metadata command mismatch: %s != %s", command, kb.Command)
	}
}

func TestVectorizeKeybindings(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	keybindings := []interfaces.Keybinding{
		{
			ID:          "test1",
			Keys:        "dd",
			Command:     "delete line",
			Description: "Delete current line",
			Mode:        "n",
		},
		{
			ID:          "test2",
			Keys:        "yy",
			Command:     "yank line",
			Description: "Copy current line",
			Mode:        "n",
		},
	}

	documents, err := vectorizer.VectorizeKeybindings(keybindings)
	if err != nil {
		t.Fatalf("VectorizeKeybindings failed: %v", err)
	}

	if len(documents) != len(keybindings) {
		t.Errorf("Document count mismatch: %d != %d", len(documents), len(keybindings))
	}

	// Check first document
	if string(documents[0].ID) != keybindings[0].ID {
		t.Errorf("First document ID mismatch: %s != %s", string(documents[0].ID), keybindings[0].ID)
	}

	// Check second document
	if string(documents[1].ID) != keybindings[1].ID {
		t.Errorf("Second document ID mismatch: %s != %s", string(documents[1].ID), keybindings[1].ID)
	}
}

func TestBatchVectorizeAndStore(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	config := DefaultVectorizerConfig()
	config.BatchSize = 2 // Small batch size for testing
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, config)

	keybindings := []interfaces.Keybinding{
		{
			ID:      "test1",
			Keys:    "dd",
			Command: "delete line",
			Mode:    "n",
		},
		{
			ID:      "test2",
			Keys:    "yy",
			Command: "yank line",
			Mode:    "n",
		},
		{
			ID:      "test3",
			Keys:    "p",
			Command: "paste",
			Mode:    "n",
		},
	}

	err := vectorizer.BatchVectorizeAndStore(keybindings)
	if err != nil {
		t.Fatalf("BatchVectorizeAndStore failed: %v", err)
	}

	// Check that all documents were stored
	if len(mockDB.documents) != len(keybindings) {
		t.Errorf("Stored document count mismatch: %d != %d", len(mockDB.documents), len(keybindings))
	}

	// Check that hash store was updated
	if len(vectorizer.hashStore) != len(keybindings) {
		t.Errorf("Hash store size mismatch: %d != %d", len(vectorizer.hashStore), len(keybindings))
	}
}

func TestDetectChanges(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	// Initial keybindings
	original := []interfaces.Keybinding{
		{
			ID:          "test1",
			Keys:        "dd",
			Command:     "delete line",
			Description: "Delete current line",
			Mode:        "n",
		},
		{
			ID:          "test2",
			Keys:        "yy",
			Command:     "yank line",
			Description: "Copy current line",
			Mode:        "n",
		},
	}

	// Store initial hashes
	for _, kb := range original {
		vectorizer.hashStore[kb.ID] = vectorizer.parser.GenerateHash(&kb)
	}

	// Modified keybindings
	modified := []interfaces.Keybinding{
		{
			ID:          "test1",
			Keys:        "dd",
			Command:     "delete line",
			Description: "Delete current line MODIFIED", // Changed description
			Mode:        "n",
		},
		{
			ID:          "test3", // New keybinding
			Keys:        "p",
			Command:     "paste",
			Description: "Paste text",
			Mode:        "n",
		},
		// test2 is deleted (not in modified list)
	}

	changed, deleted, err := vectorizer.DetectChanges(modified)
	if err != nil {
		t.Fatalf("DetectChanges failed: %v", err)
	}

	// Should detect 2 changed (test1 modified, test3 new)
	if len(changed) != 2 {
		t.Errorf("Changed count mismatch: %d != 2", len(changed))
	}

	// Should detect 1 deleted (test2)
	if len(deleted) != 1 {
		t.Errorf("Deleted count mismatch: %d != 1", len(deleted))
	}

	if deleted[0] != "test2" {
		t.Errorf("Deleted ID mismatch: %s != test2", deleted[0])
	}
}

func TestUpdateVectorDatabase(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	// Initial setup
	original := []interfaces.Keybinding{
		{
			ID:      "test1",
			Keys:    "dd",
			Command: "delete line",
			Mode:    "n",
		},
		{
			ID:      "test2",
			Keys:    "yy",
			Command: "yank line",
			Mode:    "n",
		},
	}

	// Store initial keybindings
	err := vectorizer.BatchVectorizeAndStore(original)
	if err != nil {
		t.Fatalf("Initial store failed: %v", err)
	}

	// Modified keybindings
	modified := []interfaces.Keybinding{
		{
			ID:          "test1",
			Keys:        "dd",
			Command:     "delete line",
			Description: "MODIFIED", // Changed
			Mode:        "n",
		},
		{
			ID:      "test3", // New
			Keys:    "p",
			Command: "paste",
			Mode:    "n",
		},
		// test2 deleted
	}

	// Update database
	err = vectorizer.UpdateVectorDatabase(modified)
	if err != nil {
		t.Fatalf("UpdateVectorDatabase failed: %v", err)
	}

	// Check that test2 was deleted
	if len(mockDB.deleted) != 1 || mockDB.deleted[0] != "test2" {
		t.Errorf("Expected test2 to be deleted, got: %v", mockDB.deleted)
	}

	// Check that documents were updated
	// Should have test1 (modified) and test3 (new)
	foundTest1 := false
	foundTest3 := false
	for _, doc := range mockDB.documents {
		if string(doc.ID) == "test1" {
			foundTest1 = true
			if !contains(doc.Content, "MODIFIED") {
				t.Errorf("test1 should contain MODIFIED in content, got: %s", doc.Content)
			}
		}
		if string(doc.ID) == "test3" {
			foundTest3 = true
		}
	}

	if !foundTest1 {
		t.Error("test1 not found in documents")
	}
	if !foundTest3 {
		t.Error("test3 not found in documents")
	}
}

func TestGenerateContent(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	config := DefaultVectorizerConfig()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, config)

	kb := &interfaces.Keybinding{
		ID:          "test1",
		Keys:        "dd",
		Command:     "delete line",
		Description: "Delete current line",
		Mode:        "n",
		Plugin:      "builtin",
		Metadata: map[string]string{
			"category": "edit",
			"priority": "high",
		},
	}

	content := vectorizer.generateContent(kb)

	// Check that all expected parts are included
	expectedParts := []string{
		"dd",
		"delete line",
		"Delete current line",
		"mode:n",
		"plugin:builtin",
		"category:edit",
		"priority:high",
	}

	for _, part := range expectedParts {
		if !contains(content, part) {
			t.Errorf("Content missing expected part: %s\nContent: %s", part, content)
		}
	}
}

func TestGenerateMetadata(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	kb := &interfaces.Keybinding{
		ID:          "test1",
		Keys:        "dd",
		Command:     "delete line",
		Description: "Delete current line",
		Mode:        "n",
		Plugin:      "builtin",
		Metadata: map[string]string{
			"category": "edit",
		},
	}

	metadata := vectorizer.generateMetadata(kb)

	// Helper function to safely get string values from metadata
	getValue := func(key string) string {
		if val, exists := metadata.GetString(key); exists {
			return val
		}
		return ""
	}

	// Check required metadata fields
	requiredFields := map[string]string{
		"keybinding_id": kb.ID,
		"keys":          kb.Keys,
		"command":       kb.Command,
		"mode":          kb.Mode,
		"description":   kb.Description,
		"plugin":        kb.Plugin,
		"category":      "edit", // from original metadata
	}

	for key, expectedValue := range requiredFields {
		actualValue := getValue(key)
		if actualValue != expectedValue {
			t.Errorf("Metadata field %s mismatch: %s != %s", key, actualValue, expectedValue)
		}
	}

	// Check boolean flags
	if getValue("has_description") != "true" {
		t.Errorf("has_description should be true")
	}

	if getValue("has_plugin") != "true" {
		t.Errorf("has_plugin should be true")
	}

	// Check that vectorized_at is set
	if getValue("vectorized_at") == "" {
		t.Error("vectorized_at should be set")
	}
}

func TestGetStats(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	// Add some entries to hash store
	vectorizer.hashStore["test1"] = "hash1"
	vectorizer.hashStore["test2"] = "hash2"

	stats := vectorizer.GetStats()

	if stats["hash_store_size"] != 2 {
		t.Errorf("hash_store_size mismatch: %v != 2", stats["hash_store_size"])
	}

	if stats["change_detection_enabled"] != true {
		t.Errorf("change_detection_enabled should be true")
	}

	if stats["batch_size"] != DefaultVectorizerConfig().BatchSize {
		t.Errorf("batch_size mismatch: %v != %d", stats["batch_size"], DefaultVectorizerConfig().BatchSize)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
func TestGenerateEmbedding(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	content := "dd delete line Delete current line"

	vector, err := vectorizer.generateEmbedding(content)
	if err != nil {
		t.Fatalf("generateEmbedding failed: %v", err)
	}

	if len(vector) == 0 {
		t.Error("Generated vector is empty")
	}

	// Test with empty content
	_, err = vectorizer.generateEmbedding("")
	if err == nil {
		t.Error("Expected error for empty content")
	}
}

func TestPrepareContentForEmbedding(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"  dd   delete line  ", "dd delete line"},
		{"", ""},
		{"word\n\ttab\nspaces", "word tab spaces"},
	}

	for _, test := range tests {
		result := vectorizer.prepareContentForEmbedding(test.input)
		if result != test.expected {
			t.Errorf("prepareContentForEmbedding(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}

	// Test long content truncation
	longContent := strings.Repeat("word ", 300) // Create content longer than 1000 chars
	result := vectorizer.prepareContentForEmbedding(longContent)
	if len(result) > 1000 {
		t.Errorf("Content not truncated properly: length %d", len(result))
	}
}

func TestBatchGenerateEmbeddings(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	contents := []string{
		"dd delete line",
		"yy yank line",
		"p paste",
	}

	vectors, err := vectorizer.BatchGenerateEmbeddings(contents)
	if err != nil {
		t.Fatalf("BatchGenerateEmbeddings failed: %v", err)
	}

	if len(vectors) != len(contents) {
		t.Errorf("Vector count mismatch: %d != %d", len(vectors), len(contents))
	}

	for i, vector := range vectors {
		if len(vector) == 0 {
			t.Errorf("Vector %d is empty", i)
		}
	}

	// Test with empty input
	emptyVectors, err := vectorizer.BatchGenerateEmbeddings([]string{})
	if err != nil {
		t.Errorf("BatchGenerateEmbeddings failed with empty input: %v", err)
	}
	if emptyVectors != nil {
		t.Error("Expected nil for empty input")
	}
}

func TestOptimizedBatchVectorizeAndStore(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	config := DefaultVectorizerConfig()
	config.BatchSize = 2
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, config)

	keybindings := []interfaces.Keybinding{
		{
			ID:      "test1",
			Keys:    "dd",
			Command: "delete line",
			Mode:    "n",
		},
		{
			ID:      "test2",
			Keys:    "yy",
			Command: "yank line",
			Mode:    "n",
		},
		{
			ID:      "test3",
			Keys:    "p",
			Command: "paste",
			Mode:    "n",
		},
	}

	err := vectorizer.OptimizedBatchVectorizeAndStore(keybindings)
	if err != nil {
		t.Fatalf("OptimizedBatchVectorizeAndStore failed: %v", err)
	}

	// Check that all documents were stored with vectors
	if len(mockDB.documents) != len(keybindings) {
		t.Errorf("Stored document count mismatch: %d != %d", len(mockDB.documents), len(keybindings))
	}

	for i, doc := range mockDB.documents {
		if len(doc.Vector) == 0 {
			t.Errorf("Document %d has empty vector", i)
		}
	}

	// Check that hash store was updated
	if len(vectorizer.hashStore) != len(keybindings) {
		t.Errorf("Hash store size mismatch: %d != %d", len(vectorizer.hashStore), len(keybindings))
	}
}

func TestIncrementalUpdate(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	// Initial keybindings
	original := []interfaces.Keybinding{
		{
			ID:      "test1",
			Keys:    "dd",
			Command: "delete line",
			Mode:    "n",
		},
		{
			ID:      "test2",
			Keys:    "yy",
			Command: "yank line",
			Mode:    "n",
		},
	}

	// Store initial keybindings
	err := vectorizer.OptimizedBatchVectorizeAndStore(original)
	if err != nil {
		t.Fatalf("Initial store failed: %v", err)
	}

	// Modified keybindings
	modified := []interfaces.Keybinding{
		{
			ID:          "test1",
			Keys:        "dd",
			Command:     "delete line",
			Description: "MODIFIED", // Changed
			Mode:        "n",
		},
		{
			ID:      "test3", // New
			Keys:    "p",
			Command: "paste",
			Mode:    "n",
		},
		// test2 deleted
	}

	// Perform incremental update
	result, err := vectorizer.IncrementalUpdate(modified)
	if err != nil {
		t.Fatalf("IncrementalUpdate failed: %v", err)
	}

	// Check results
	if result.TotalProcessed != len(modified) {
		t.Errorf("TotalProcessed mismatch: %d != %d", result.TotalProcessed, len(modified))
	}

	if result.ChangedCount != 2 { // test1 modified, test3 new
		t.Errorf("ChangedCount mismatch: %d != 2", result.ChangedCount)
	}

	if result.DeletedCount != 1 { // test2 deleted
		t.Errorf("DeletedCount mismatch: %d != 1", result.DeletedCount)
	}

	// Check that test2 was deleted
	if len(mockDB.deleted) != 1 || mockDB.deleted[0] != "test2" {
		t.Errorf("Expected test2 to be deleted, got: %v", mockDB.deleted)
	}
}

func TestGetChangeDetectionStats(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	// Setup initial state
	original := []interfaces.Keybinding{
		{
			ID:      "test1",
			Keys:    "dd",
			Command: "delete line",
			Mode:    "n",
		},
		{
			ID:      "test2",
			Keys:    "yy",
			Command: "yank line",
			Mode:    "n",
		},
	}

	// Store hashes
	for _, kb := range original {
		vectorizer.hashStore[kb.ID] = vectorizer.parser.GenerateHash(&kb)
	}

	// Current keybindings with changes
	current := []interfaces.Keybinding{
		{
			ID:          "test1",
			Keys:        "dd",
			Command:     "delete line",
			Description: "MODIFIED", // Changed
			Mode:        "n",
		},
		{
			ID:      "test3", // New
			Keys:    "p",
			Command: "paste",
			Mode:    "n",
		},
		// test2 deleted
	}

	stats, err := vectorizer.GetChangeDetectionStats(current)
	if err != nil {
		t.Fatalf("GetChangeDetectionStats failed: %v", err)
	}

	if !stats.Enabled {
		t.Error("Change detection should be enabled")
	}

	if stats.TotalKeybindings != len(current) {
		t.Errorf("TotalKeybindings mismatch: %d != %d", stats.TotalKeybindings, len(current))
	}

	if stats.NewCount != 1 { // test3
		t.Errorf("NewCount mismatch: %d != 1", stats.NewCount)
	}

	if stats.ModifiedCount != 1 { // test1
		t.Errorf("ModifiedCount mismatch: %d != 1", stats.ModifiedCount)
	}

	if stats.DeletedCount != 1 { // test2
		t.Errorf("DeletedCount mismatch: %d != 1", stats.DeletedCount)
	}
}

func TestRebuildHashStore(t *testing.T) {
	mockDB := NewMockVectorDB()
	mockLLM := NewMockLLMClient()
	vectorizer := NewKeybindingVectorizer(mockDB, mockLLM, nil)

	keybindings := []interfaces.Keybinding{
		{
			ID:      "test1",
			Keys:    "dd",
			Command: "delete line",
			Mode:    "n",
		},
		{
			ID:      "test2",
			Keys:    "yy",
			Command: "yank line",
			Mode:    "n",
		},
	}

	err := vectorizer.RebuildHashStore(keybindings)
	if err != nil {
		t.Fatalf("RebuildHashStore failed: %v", err)
	}

	if len(vectorizer.hashStore) != len(keybindings) {
		t.Errorf("Hash store size mismatch: %d != %d", len(vectorizer.hashStore), len(keybindings))
	}

	// Verify hashes are correct
	for _, kb := range keybindings {
		expectedHash := vectorizer.parser.GenerateHash(&kb)
		if storedHash, exists := vectorizer.hashStore[kb.ID]; !exists {
			t.Errorf("Hash not found for keybinding %s", kb.ID)
		} else if storedHash != expectedHash {
			t.Errorf("Hash mismatch for keybinding %s: %s != %s", kb.ID, storedHash, expectedHash)
		}
	}
}
