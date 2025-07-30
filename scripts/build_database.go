package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"nvim-smart-keybind-search/internal/chromadb"
	"nvim-smart-keybind-search/internal/interfaces"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// VimKnowledgeBase represents the comprehensive vim knowledge base
type VimKnowledgeBase struct {
	Keybindings []VimKeybinding `json:"keybindings"`
	Categories  []Category      `json:"categories"`
}

// VimKeybinding represents a vim keybinding with comprehensive information
type VimKeybinding struct {
	Keys        string            `json:"keys"`
	Command     string            `json:"command"`
	Description string            `json:"description"`
	Mode        string            `json:"mode"`
	Category    string            `json:"category"`
	Difficulty  string            `json:"difficulty"` // beginner, intermediate, advanced
	Examples    []string          `json:"examples,omitempty"`
	Related     []string          `json:"related,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Category represents a category of vim commands
type Category struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parent      string `json:"parent,omitempty"`
}

func main() {
	fmt.Println("Building pre-populated ChromaDB database...")

	// Create knowledge base
	knowledgeBase := createVimKnowledgeBase()

	// Initialize ChromaDB client
	config := chromadb.DefaultConfig()
	client, err := chromadb.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create ChromaDB client: %v", err)
	}

	// Create collection manager
	cm := chromadb.NewCollectionManager(client)

	// Convert knowledge base to documents
	documents := convertToDocuments(knowledgeBase)

	// Store documents in ChromaDB
	fmt.Printf("Storing %d keybindings in ChromaDB...\n", len(documents))

	err = cm.StoreBuiltinKnowledge(documents)
	if err != nil {
		log.Fatalf("Failed to store knowledge base: %v", err)
	}

	fmt.Println("Successfully built pre-populated ChromaDB database!")
	fmt.Printf("Total keybindings stored: %d\n", len(documents))
}

// createVimKnowledgeBase creates a comprehensive vim knowledge base
func createVimKnowledgeBase() *VimKnowledgeBase {
	return &VimKnowledgeBase{
		Keybindings: []VimKeybinding{
			// Basic Movement
			{
				Keys:        "h",
				Command:     "move left",
				Description: "Move cursor one character to the left",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "beginner",
				Examples:    []string{"h", "5h"},
			},
			{
				Keys:        "j",
				Command:     "move down",
				Description: "Move cursor one line down",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "beginner",
				Examples:    []string{"j", "5j"},
			},
			{
				Keys:        "k",
				Command:     "move up",
				Description: "Move cursor one line up",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "beginner",
				Examples:    []string{"k", "5k"},
			},
			{
				Keys:        "l",
				Command:     "move right",
				Description: "Move cursor one character to the right",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "beginner",
				Examples:    []string{"l", "5l"},
			},
			{
				Keys:        "w",
				Command:     "move to next word",
				Description: "Move cursor to the beginning of the next word",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "beginner",
				Examples:    []string{"w", "5w"},
			},
			{
				Keys:        "b",
				Command:     "move to previous word",
				Description: "Move cursor to the beginning of the previous word",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "beginner",
				Examples:    []string{"b", "5b"},
			},
			{
				Keys:        "0",
				Command:     "move to beginning of line",
				Description: "Move cursor to the beginning of the current line",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "beginner",
			},
			{
				Keys:        "$",
				Command:     "move to end of line",
				Description: "Move cursor to the end of the current line",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "beginner",
			},
			{
				Keys:        "gg",
				Command:     "move to beginning of file",
				Description: "Move cursor to the beginning of the file",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "beginner",
			},
			{
				Keys:        "G",
				Command:     "move to end of file",
				Description: "Move cursor to the end of the file",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "beginner",
			},

			// Basic Editing
			{
				Keys:        "i",
				Command:     "insert mode",
				Description: "Enter insert mode before the cursor",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "beginner",
			},
			{
				Keys:        "a",
				Command:     "append",
				Description: "Enter insert mode after the cursor",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "beginner",
			},
			{
				Keys:        "A",
				Command:     "append at end of line",
				Description: "Enter insert mode at the end of the current line",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "beginner",
			},
			{
				Keys:        "o",
				Command:     "open line below",
				Description: "Open a new line below the current line and enter insert mode",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "beginner",
			},
			{
				Keys:        "O",
				Command:     "open line above",
				Description: "Open a new line above the current line and enter insert mode",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "beginner",
			},
			{
				Keys:        "x",
				Command:     "delete character",
				Description: "Delete the character under the cursor",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "beginner",
			},
			{
				Keys:        "dd",
				Command:     "delete line",
				Description: "Delete the current line",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "beginner",
			},
			{
				Keys:        "yy",
				Command:     "yank line",
				Description: "Copy the current line to the clipboard",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "beginner",
			},
			{
				Keys:        "p",
				Command:     "paste",
				Description: "Paste the contents of the clipboard after the cursor",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "beginner",
			},
			{
				Keys:        "P",
				Command:     "paste before",
				Description: "Paste the contents of the clipboard before the cursor",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "beginner",
			},

			// Visual Mode
			{
				Keys:        "v",
				Command:     "visual mode",
				Description: "Enter visual mode to select text",
				Mode:        "n",
				Category:    "visual",
				Difficulty:  "beginner",
			},
			{
				Keys:        "V",
				Command:     "visual line mode",
				Description: "Enter visual line mode to select entire lines",
				Mode:        "n",
				Category:    "visual",
				Difficulty:  "beginner",
			},
			{
				Keys:        "<C-v>",
				Command:     "visual block mode",
				Description: "Enter visual block mode to select rectangular blocks",
				Mode:        "n",
				Category:    "visual",
				Difficulty:  "intermediate",
			},

			// Search and Replace
			{
				Keys:        "/",
				Command:     "search forward",
				Description: "Search for a pattern forward from the cursor",
				Mode:        "n",
				Category:    "search",
				Difficulty:  "beginner",
			},
			{
				Keys:        "?",
				Command:     "search backward",
				Description: "Search for a pattern backward from the cursor",
				Mode:        "n",
				Category:    "search",
				Difficulty:  "beginner",
			},
			{
				Keys:        "n",
				Command:     "next match",
				Description: "Go to the next search match",
				Mode:        "n",
				Category:    "search",
				Difficulty:  "beginner",
			},
			{
				Keys:        "N",
				Command:     "previous match",
				Description: "Go to the previous search match",
				Mode:        "n",
				Category:    "search",
				Difficulty:  "beginner",
			},
			{
				Keys:        ":%s/old/new/g",
				Command:     "replace all",
				Description: "Replace all occurrences of 'old' with 'new' in the entire file",
				Mode:        "c",
				Category:    "search",
				Difficulty:  "intermediate",
			},

			// Advanced Movement
			{
				Keys:        "f",
				Command:     "find character",
				Description: "Find the next occurrence of a character on the current line",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "intermediate",
				Examples:    []string{"fa", "f;", "3fa"},
			},
			{
				Keys:        "F",
				Command:     "find character backward",
				Description: "Find the previous occurrence of a character on the current line",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "t",
				Command:     "till character",
				Description: "Move to the character before the next occurrence of a character",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "T",
				Command:     "till character backward",
				Description: "Move to the character after the previous occurrence of a character",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "%",
				Command:     "match bracket",
				Description: "Jump to the matching bracket",
				Mode:        "n",
				Category:    "movement",
				Difficulty:  "intermediate",
			},

			// Advanced Editing
			{
				Keys:        "ciw",
				Command:     "change inner word",
				Description: "Delete the word under the cursor and enter insert mode",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "caw",
				Command:     "change a word",
				Description: "Delete the word under the cursor (including trailing space) and enter insert mode",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "diw",
				Command:     "delete inner word",
				Description: "Delete the word under the cursor",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "daw",
				Command:     "delete a word",
				Description: "Delete the word under the cursor (including trailing space)",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "yiw",
				Command:     "yank inner word",
				Description: "Copy the word under the cursor",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "yaw",
				Command:     "yank a word",
				Description: "Copy the word under the cursor (including trailing space)",
				Mode:        "n",
				Category:    "editing",
				Difficulty:  "intermediate",
			},

			// Text Objects
			{
				Keys:        "iw",
				Command:     "inner word",
				Description: "Select the word under the cursor (excluding trailing space)",
				Mode:        "v",
				Category:    "text-objects",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "aw",
				Command:     "a word",
				Description: "Select the word under the cursor (including trailing space)",
				Mode:        "v",
				Category:    "text-objects",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "i\"",
				Command:     "inner quotes",
				Description: "Select the text inside double quotes",
				Mode:        "v",
				Category:    "text-objects",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "a\"",
				Command:     "a quotes",
				Description: "Select the text inside double quotes including the quotes",
				Mode:        "v",
				Category:    "text-objects",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "i(",
				Command:     "inner parentheses",
				Description: "Select the text inside parentheses",
				Mode:        "v",
				Category:    "text-objects",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "a(",
				Command:     "a parentheses",
				Description: "Select the text inside parentheses including the parentheses",
				Mode:        "v",
				Category:    "text-objects",
				Difficulty:  "intermediate",
			},

			// Window Management
			{
				Keys:        "<C-w>h",
				Command:     "window left",
				Description: "Move to the window to the left",
				Mode:        "n",
				Category:    "windows",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "<C-w>j",
				Command:     "window down",
				Description: "Move to the window below",
				Mode:        "n",
				Category:    "windows",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "<C-w>k",
				Command:     "window up",
				Description: "Move to the window above",
				Mode:        "n",
				Category:    "windows",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "<C-w>l",
				Command:     "window right",
				Description: "Move to the window to the right",
				Mode:        "n",
				Category:    "windows",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "<C-w>s",
				Command:     "split window",
				Description: "Split the current window horizontally",
				Mode:        "n",
				Category:    "windows",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "<C-w>v",
				Command:     "vsplit window",
				Description: "Split the current window vertically",
				Mode:        "n",
				Category:    "windows",
				Difficulty:  "intermediate",
			},
			{
				Keys:        "<C-w>c",
				Command:     "close window",
				Description: "Close the current window",
				Mode:        "n",
				Category:    "windows",
				Difficulty:  "intermediate",
			},

			// Buffer Management
			{
				Keys:        ":bn",
				Command:     "next buffer",
				Description: "Switch to the next buffer",
				Mode:        "c",
				Category:    "buffers",
				Difficulty:  "intermediate",
			},
			{
				Keys:        ":bp",
				Command:     "previous buffer",
				Description: "Switch to the previous buffer",
				Mode:        "c",
				Category:    "buffers",
				Difficulty:  "intermediate",
			},
			{
				Keys:        ":bd",
				Command:     "delete buffer",
				Description: "Delete the current buffer",
				Mode:        "c",
				Category:    "buffers",
				Difficulty:  "intermediate",
			},

			// Advanced Features
			{
				Keys:        "gq",
				Command:     "format text",
				Description: "Format the selected text",
				Mode:        "v",
				Category:    "formatting",
				Difficulty:  "advanced",
			},
			{
				Keys:        "gw",
				Command:     "format word",
				Description: "Format the word under the cursor",
				Mode:        "n",
				Category:    "formatting",
				Difficulty:  "advanced",
			},
			{
				Keys:        "gqq",
				Command:     "format line",
				Description: "Format the current line",
				Mode:        "n",
				Category:    "formatting",
				Difficulty:  "advanced",
			},
			{
				Keys:        "gqap",
				Command:     "format paragraph",
				Description: "Format the current paragraph",
				Mode:        "n",
				Category:    "formatting",
				Difficulty:  "advanced",
			},

			// Macros
			{
				Keys:        "q",
				Command:     "record macro",
				Description: "Start recording a macro",
				Mode:        "n",
				Category:    "macros",
				Difficulty:  "advanced",
				Examples:    []string{"qa", "qb"},
			},
			{
				Keys:        "@",
				Command:     "play macro",
				Description: "Play a recorded macro",
				Mode:        "n",
				Category:    "macros",
				Difficulty:  "advanced",
				Examples:    []string{"@a", "@b"},
			},
			{
				Keys:        "@@",
				Command:     "repeat macro",
				Description: "Repeat the last played macro",
				Mode:        "n",
				Category:    "macros",
				Difficulty:  "advanced",
			},

			// Marks
			{
				Keys:        "m",
				Command:     "set mark",
				Description: "Set a mark at the current position",
				Mode:        "n",
				Category:    "marks",
				Difficulty:  "advanced",
				Examples:    []string{"ma", "mb"},
			},
			{
				Keys:        "'",
				Command:     "jump to mark",
				Description: "Jump to a mark",
				Mode:        "n",
				Category:    "marks",
				Difficulty:  "advanced",
				Examples:    []string{"'a", "'b"},
			},
		},
		Categories: []Category{
			{
				Name:        "movement",
				Description: "Commands for moving the cursor around the file",
			},
			{
				Name:        "editing",
				Description: "Commands for editing text",
			},
			{
				Name:        "visual",
				Description: "Commands for selecting text in visual mode",
			},
			{
				Name:        "search",
				Description: "Commands for searching and replacing text",
			},
			{
				Name:        "text-objects",
				Description: "Commands for selecting text objects",
			},
			{
				Name:        "windows",
				Description: "Commands for managing multiple windows",
			},
			{
				Name:        "buffers",
				Description: "Commands for managing multiple buffers",
			},
			{
				Name:        "formatting",
				Description: "Commands for formatting text",
			},
			{
				Name:        "macros",
				Description: "Commands for recording and playing macros",
			},
			{
				Name:        "marks",
				Description: "Commands for setting and jumping to marks",
			},
		},
	}
}

// convertToDocuments converts the knowledge base to ChromaDB documents
func convertToDocuments(kb *VimKnowledgeBase) []interfaces.Document {
	var documents []interfaces.Document

	for i, kb := range kb.Keybindings {
		// Create content for vectorization
		content := fmt.Sprintf("%s %s %s %s %s",
			kb.Keys, kb.Command, kb.Description, kb.Mode, kb.Category)

		// Add examples if available
		if len(kb.Examples) > 0 {
			content += " examples: " + strings.Join(kb.Examples, " ")
		}

		// Add related commands if available
		if len(kb.Related) > 0 {
			content += " related: " + strings.Join(kb.Related, " ")
		}

		// Create metadata map
		metadataMap := map[string]interface{}{
			"keys":        kb.Keys,
			"command":     kb.Command,
			"description": kb.Description,
			"mode":        kb.Mode,
			"category":    kb.Category,
			"difficulty":  kb.Difficulty,
			"source":      "builtin",
			"type":        "keybinding",
		}

		// Add examples to metadata
		if len(kb.Examples) > 0 {
			metadataMap["examples"] = strings.Join(kb.Examples, ",")
		}

		// Add related commands to metadata
		if len(kb.Related) > 0 {
			metadataMap["related"] = strings.Join(kb.Related, ",")
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
			ID:       chroma.DocumentID(fmt.Sprintf("builtin_%d", i)),
			Content:  content,
			Metadata: metadata,
		}

		documents = append(documents, document)
	}

	return documents
}

// saveKnowledgeBase saves the knowledge base to a JSON file
func saveKnowledgeBase(kb *VimKnowledgeBase, filename string) error {
	data, err := json.MarshalIndent(kb, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge base: %v", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write knowledge base file: %v", err)
	}

	fmt.Printf("Knowledge base saved to %s\n", filename)
	return nil
}
