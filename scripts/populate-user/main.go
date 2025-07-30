package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"nvim-smart-keybind-search/internal/chromadb"
	"nvim-smart-keybind-search/internal/interfaces"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// UserKeybinding represents a user-configured keybinding
type UserKeybinding struct {
	ID          string            `json:"id"`
	Keys        string            `json:"keys"`
	Command     string            `json:"command"`
	Description string            `json:"description"`
	Mode        string            `json:"mode"`
	Plugin      string            `json:"plugin,omitempty"`
	Buffer      int               `json:"buffer,omitempty"`
	Silent      bool              `json:"silent,omitempty"`
	Noremap     bool              `json:"noremap,omitempty"`
	Expr        bool              `json:"expr,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run populate_user_keybindings.go <keybindings.json>")
		fmt.Println("This script is typically called by the Lua plugin with extracted keybindings")
		os.Exit(1)
	}

	jsonFile := os.Args[1]
	fmt.Printf("Loading user keybindings from: %s\n", jsonFile)

	// Read the JSON file
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		log.Fatalf("Failed to read JSON file: %v", err)
	}

	// Parse the user keybindings
	var userKeybindings []UserKeybinding
	if err := json.Unmarshal(data, &userKeybindings); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	fmt.Printf("Loaded %d user keybindings\n", len(userKeybindings))

	// Initialize ChromaDB client
	config := chromadb.DefaultConfig()
	client, err := chromadb.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create ChromaDB client: %v", err)
	}

	// Create collection manager
	cm := chromadb.NewCollectionManager(client)

	// Initialize collections
	if err := cm.Initialize(); err != nil {
		log.Fatalf("Failed to initialize collections: %v", err)
	}

	// Convert user keybindings to documents
	documents := convertUserKeybindingsToDocuments(userKeybindings)

	// Store documents in ChromaDB
	fmt.Printf("Storing %d user keybindings in ChromaDB...\n", len(documents))

	err = cm.StoreUserKeybindings(documents)
	if err != nil {
		log.Fatalf("Failed to store user keybindings: %v", err)
	}

	// Get collection count
	count, err := cm.GetUserCollectionCount()
	if err != nil {
		log.Printf("Warning: failed to get collection count: %v", err)
	} else {
		fmt.Printf("Total documents in user keybindings collection: %d\n", count)
	}

	fmt.Println("Successfully populated user keybindings collection!")
}

// convertUserKeybindingsToDocuments converts user keybindings to ChromaDB documents
func convertUserKeybindingsToDocuments(keybindings []UserKeybinding) []interfaces.Document {
	var documents []interfaces.Document

	for i, kb := range keybindings {
		// Create content for vectorization
		content := fmt.Sprintf("%s %s %s %s", kb.Keys, kb.Command, kb.Description, kb.Mode)

		// Add plugin information if available
		if kb.Plugin != "" {
			content += " plugin:" + kb.Plugin
		}

		// Create metadata map
		metadataMap := map[string]interface{}{
			"keys":        kb.Keys,
			"command":     kb.Command,
			"description": kb.Description,
			"mode":        kb.Mode,
			"source":      "user_config",
			"type":        "user_keybinding",
			"silent":      kb.Silent,
			"noremap":     kb.Noremap,
			"expr":        kb.Expr,
		}

		// Add plugin information
		if kb.Plugin != "" {
			metadataMap["plugin"] = kb.Plugin
		}

		// Add buffer information
		if kb.Buffer != 0 {
			metadataMap["buffer"] = fmt.Sprintf("%d", kb.Buffer)
		}

		// Add custom metadata
		for k, v := range kb.Metadata {
			metadataMap[k] = v
		}

		// Convert to chroma.DocumentMetadata
		metadata, err := chroma.NewDocumentMetadataFromMap(metadataMap)
		if err != nil {
			log.Printf("Warning: failed to create metadata for document %d: %v", i, err)
			metadata = chroma.NewDocumentMetadata()
		}

		document := interfaces.Document{
			ID:       chroma.DocumentID(fmt.Sprintf("user_%s_%d", kb.Mode, i)),
			Content:  content,
			Metadata: metadata,
		}

		documents = append(documents, document)
	}

	return documents
}

// Example function to demonstrate the expected JSON format
func generateExampleKeybindings() []UserKeybinding {
	return []UserKeybinding{
		{
			ID:          "user_leader_ff",
			Keys:        "<leader>ff",
			Command:     "Telescope find_files",
			Description: "Find files using Telescope",
			Mode:        "n",
			Plugin:      "telescope.nvim",
			Silent:      true,
			Noremap:     true,
			Metadata: map[string]string{
				"category": "file_navigation",
				"priority": "high",
			},
		},
		{
			ID:          "user_leader_lg",
			Keys:        "<leader>lg",
			Command:     "Telescope live_grep",
			Description: "Live grep using Telescope",
			Mode:        "n",
			Plugin:      "telescope.nvim",
			Silent:      true,
			Noremap:     true,
			Metadata: map[string]string{
				"category": "search",
				"priority": "high",
			},
		},
		{
			ID:          "user_jk_escape",
			Keys:        "jk",
			Command:     "<Esc>",
			Description: "Exit insert mode with jk",
			Mode:        "i",
			Silent:      true,
			Noremap:     true,
			Metadata: map[string]string{
				"category": "mode_switching",
				"priority": "medium",
			},
		},
	}
}

// saveExampleKeybindings saves example keybindings to a JSON file for reference
func saveExampleKeybindings(filename string) error {
	examples := generateExampleKeybindings()

	data, err := json.MarshalIndent(examples, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keybindings: %v", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write keybindings file: %v", err)
	}

	fmt.Printf("Example keybindings saved to %s\n", filename)
	return nil
}
