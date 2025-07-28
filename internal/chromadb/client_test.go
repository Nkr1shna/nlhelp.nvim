package chromadb

import (
	"nvim-smart-keybind-search/internal/interfaces"
	"testing"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func TestNewClient(t *testing.T) {
	config := DefaultConfig()
	client, err := NewClient(config)

	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client is nil")
	}

	if client.config != config {
		t.Error("Config not set correctly")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", config.Host)
	}

	if config.Port != 8000 {
		t.Errorf("Expected port 8000, got %d", config.Port)
	}

	if config.CollectionName != "keybindings" {
		t.Errorf("Expected collection name 'keybindings', got '%s'", config.CollectionName)
	}
}

func TestEnhanceDocuments(t *testing.T) {
	config := DefaultConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create vector service to test document enhancement
	vs := &VectorService{
		client: client,
		config: DefaultVectorServiceConfig(),
	}

	metadata, _ := chroma.NewDocumentMetadataFromMap(map[string]interface{}{
		"type": "keybinding",
	})

	docs := []interfaces.Document{
		{
			ID:       chroma.DocumentID("test1"),
			Content:  "Test Content",
			Metadata: metadata,
		},
	}

	enhanced := vs.enhanceDocuments(docs)

	if len(enhanced) != 1 {
		t.Fatalf("Expected 1 enhanced document, got %d", len(enhanced))
	}

	doc := enhanced[0]
	if source, exists := doc.Metadata.GetString("source"); !exists || source != "user" {
		t.Error("Expected source to be 'user'")
	}

	if createdAt, exists := doc.Metadata.GetString("created_at"); !exists || createdAt == "" {
		t.Error("Expected created_at to be set")
	}

	if contentLength, exists := doc.Metadata.GetString("content_length"); !exists || contentLength != "12" {
		t.Errorf("Expected content_length to be '12', got '%s'", contentLength)
	}
}

func TestNormalizeContent(t *testing.T) {
	config := DefaultConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	vs := &VectorService{
		client: client,
		config: DefaultVectorServiceConfig(),
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"  Multiple   Spaces  ", "multiple spaces"},
		{"MixedCase Content", "mixedcase content"},
		{"", ""},
	}

	for _, test := range tests {
		result := vs.normalizeContent(test.input)
		if result != test.expected {
			t.Errorf("normalizeContent(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
