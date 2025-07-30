package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"nvim-smart-keybind-search/internal/chromadb"
	"nvim-smart-keybind-search/internal/interfaces"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// BuiltinKeybinding represents a built-in Neovim keybinding from quick reference
type BuiltinKeybinding struct {
	Keys        string            `json:"keys"`
	Command     string            `json:"command"`
	Description string            `json:"description"`
	Mode        string            `json:"mode"`
	Category    string            `json:"category"`
	Section     string            `json:"section"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func main() {
	fmt.Println("Populating built-in knowledge from Neovim quick reference...")

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

	// Get built-in keybindings
	keybindings := getBuiltinKeybindings()

	// Convert to documents
	documents := convertBuiltinToDocuments(keybindings)

	// Store in ChromaDB
	fmt.Printf("Storing %d built-in keybindings in ChromaDB...\n", len(documents))
	err = cm.StoreBuiltinKnowledge(documents)
	if err != nil {
		log.Fatalf("Failed to store built-in knowledge: %v", err)
	}

	// Get collection count
	count, err := cm.GetBuiltinCollectionCount()
	if err != nil {
		log.Printf("Warning: failed to get collection count: %v", err)
	} else {
		fmt.Printf("Total documents in built-in collection: %d\n", count)
	}

	fmt.Println("Successfully populated built-in knowledge!")
}

// getBuiltinKeybindings returns comprehensive built-in Neovim keybindings
// Based on Neovim quick reference: https://neovim.io/doc/user/quickref.html
func getBuiltinKeybindings() []BuiltinKeybinding {
	return []BuiltinKeybinding{
		// Movement commands
		{
			Keys:        "h",
			Command:     "cursor left",
			Description: "Move cursor one character to the left",
			Mode:        "n",
			Category:    "movement",
			Section:     "basic_movement",
		},
		{
			Keys:        "j",
			Command:     "cursor down",
			Description: "Move cursor one line down",
			Mode:        "n",
			Category:    "movement",
			Section:     "basic_movement",
		},
		{
			Keys:        "k",
			Command:     "cursor up",
			Description: "Move cursor one line up",
			Mode:        "n",
			Category:    "movement",
			Section:     "basic_movement",
		},
		{
			Keys:        "l",
			Command:     "cursor right",
			Description: "Move cursor one character to the right",
			Mode:        "n",
			Category:    "movement",
			Section:     "basic_movement",
		},
		{
			Keys:        "w",
			Command:     "word forward",
			Description: "Move cursor to the beginning of the next word",
			Mode:        "n",
			Category:    "movement",
			Section:     "word_movement",
		},
		{
			Keys:        "W",
			Command:     "WORD forward",
			Description: "Move cursor to the beginning of the next WORD (whitespace separated)",
			Mode:        "n",
			Category:    "movement",
			Section:     "word_movement",
		},
		{
			Keys:        "b",
			Command:     "word backward",
			Description: "Move cursor to the beginning of the previous word",
			Mode:        "n",
			Category:    "movement",
			Section:     "word_movement",
		},
		{
			Keys:        "B",
			Command:     "WORD backward",
			Description: "Move cursor to the beginning of the previous WORD (whitespace separated)",
			Mode:        "n",
			Category:    "movement",
			Section:     "word_movement",
		},
		{
			Keys:        "e",
			Command:     "end of word",
			Description: "Move cursor to the end of the current word",
			Mode:        "n",
			Category:    "movement",
			Section:     "word_movement",
		},
		{
			Keys:        "E",
			Command:     "end of WORD",
			Description: "Move cursor to the end of the current WORD (whitespace separated)",
			Mode:        "n",
			Category:    "movement",
			Section:     "word_movement",
		},
		{
			Keys:        "0",
			Command:     "beginning of line",
			Description: "Move cursor to the beginning of the current line",
			Mode:        "n",
			Category:    "movement",
			Section:     "line_movement",
		},
		{
			Keys:        "^",
			Command:     "first non-blank",
			Description: "Move cursor to the first non-blank character of the line",
			Mode:        "n",
			Category:    "movement",
			Section:     "line_movement",
		},
		{
			Keys:        "$",
			Command:     "end of line",
			Description: "Move cursor to the end of the current line",
			Mode:        "n",
			Category:    "movement",
			Section:     "line_movement",
		},
		{
			Keys:        "g_",
			Command:     "last non-blank",
			Description: "Move cursor to the last non-blank character of the line",
			Mode:        "n",
			Category:    "movement",
			Section:     "line_movement",
		},
		{
			Keys:        "gg",
			Command:     "first line",
			Description: "Move cursor to the first line of the file",
			Mode:        "n",
			Category:    "movement",
			Section:     "file_movement",
		},
		{
			Keys:        "G",
			Command:     "last line",
			Description: "Move cursor to the last line of the file",
			Mode:        "n",
			Category:    "movement",
			Section:     "file_movement",
		},
		{
			Keys:        "{",
			Command:     "paragraph backward",
			Description: "Move cursor to the beginning of the previous paragraph",
			Mode:        "n",
			Category:    "movement",
			Section:     "paragraph_movement",
		},
		{
			Keys:        "}",
			Command:     "paragraph forward",
			Description: "Move cursor to the beginning of the next paragraph",
			Mode:        "n",
			Category:    "movement",
			Section:     "paragraph_movement",
		},
		{
			Keys:        "(",
			Command:     "sentence backward",
			Description: "Move cursor to the beginning of the previous sentence",
			Mode:        "n",
			Category:    "movement",
			Section:     "sentence_movement",
		},
		{
			Keys:        ")",
			Command:     "sentence forward",
			Description: "Move cursor to the beginning of the next sentence",
			Mode:        "n",
			Category:    "movement",
			Section:     "sentence_movement",
		},

		// Character search
		{
			Keys:        "f{char}",
			Command:     "find character",
			Description: "Find the next occurrence of {char} on the current line",
			Mode:        "n",
			Category:    "movement",
			Section:     "character_search",
		},
		{
			Keys:        "F{char}",
			Command:     "find character backward",
			Description: "Find the previous occurrence of {char} on the current line",
			Mode:        "n",
			Category:    "movement",
			Section:     "character_search",
		},
		{
			Keys:        "t{char}",
			Command:     "till character",
			Description: "Move to the character before the next occurrence of {char}",
			Mode:        "n",
			Category:    "movement",
			Section:     "character_search",
		},
		{
			Keys:        "T{char}",
			Command:     "till character backward",
			Description: "Move to the character after the previous occurrence of {char}",
			Mode:        "n",
			Category:    "movement",
			Section:     "character_search",
		},
		{
			Keys:        ";",
			Command:     "repeat find",
			Description: "Repeat the last f, F, t, or T command",
			Mode:        "n",
			Category:    "movement",
			Section:     "character_search",
		},
		{
			Keys:        ",",
			Command:     "repeat find reverse",
			Description: "Repeat the last f, F, t, or T command in the opposite direction",
			Mode:        "n",
			Category:    "movement",
			Section:     "character_search",
		},

		// Editing commands
		{
			Keys:        "i",
			Command:     "insert",
			Description: "Enter insert mode before the cursor",
			Mode:        "n",
			Category:    "editing",
			Section:     "insert_mode",
		},
		{
			Keys:        "I",
			Command:     "insert at beginning",
			Description: "Enter insert mode at the beginning of the line",
			Mode:        "n",
			Category:    "editing",
			Section:     "insert_mode",
		},
		{
			Keys:        "a",
			Command:     "append",
			Description: "Enter insert mode after the cursor",
			Mode:        "n",
			Category:    "editing",
			Section:     "insert_mode",
		},
		{
			Keys:        "A",
			Command:     "append at end",
			Description: "Enter insert mode at the end of the line",
			Mode:        "n",
			Category:    "editing",
			Section:     "insert_mode",
		},
		{
			Keys:        "o",
			Command:     "open line below",
			Description: "Open a new line below the current line and enter insert mode",
			Mode:        "n",
			Category:    "editing",
			Section:     "insert_mode",
		},
		{
			Keys:        "O",
			Command:     "open line above",
			Description: "Open a new line above the current line and enter insert mode",
			Mode:        "n",
			Category:    "editing",
			Section:     "insert_mode",
		},

		// Delete commands
		{
			Keys:        "x",
			Command:     "delete character",
			Description: "Delete the character under the cursor",
			Mode:        "n",
			Category:    "editing",
			Section:     "delete",
		},
		{
			Keys:        "X",
			Command:     "delete character before",
			Description: "Delete the character before the cursor",
			Mode:        "n",
			Category:    "editing",
			Section:     "delete",
		},
		{
			Keys:        "dd",
			Command:     "delete line",
			Description: "Delete the current line",
			Mode:        "n",
			Category:    "editing",
			Section:     "delete",
		},
		{
			Keys:        "dw",
			Command:     "delete word",
			Description: "Delete from cursor to the beginning of the next word",
			Mode:        "n",
			Category:    "editing",
			Section:     "delete",
		},
		{
			Keys:        "dW",
			Command:     "delete WORD",
			Description: "Delete from cursor to the beginning of the next WORD",
			Mode:        "n",
			Category:    "editing",
			Section:     "delete",
		},
		{
			Keys:        "d$",
			Command:     "delete to end of line",
			Description: "Delete from cursor to the end of the line",
			Mode:        "n",
			Category:    "editing",
			Section:     "delete",
		},
		{
			Keys:        "d0",
			Command:     "delete to beginning of line",
			Description: "Delete from cursor to the beginning of the line",
			Mode:        "n",
			Category:    "editing",
			Section:     "delete",
		},

		// Copy (yank) commands
		{
			Keys:        "yy",
			Command:     "yank line",
			Description: "Copy the current line",
			Mode:        "n",
			Category:    "editing",
			Section:     "yank",
		},
		{
			Keys:        "yw",
			Command:     "yank word",
			Description: "Copy from cursor to the beginning of the next word",
			Mode:        "n",
			Category:    "editing",
			Section:     "yank",
		},
		{
			Keys:        "y$",
			Command:     "yank to end of line",
			Description: "Copy from cursor to the end of the line",
			Mode:        "n",
			Category:    "editing",
			Section:     "yank",
		},
		{
			Keys:        "y0",
			Command:     "yank to beginning of line",
			Description: "Copy from cursor to the beginning of the line",
			Mode:        "n",
			Category:    "editing",
			Section:     "yank",
		},

		// Paste commands
		{
			Keys:        "p",
			Command:     "paste after",
			Description: "Paste the contents of the register after the cursor",
			Mode:        "n",
			Category:    "editing",
			Section:     "paste",
		},
		{
			Keys:        "P",
			Command:     "paste before",
			Description: "Paste the contents of the register before the cursor",
			Mode:        "n",
			Category:    "editing",
			Section:     "paste",
		},

		// Change commands
		{
			Keys:        "cc",
			Command:     "change line",
			Description: "Delete the current line and enter insert mode",
			Mode:        "n",
			Category:    "editing",
			Section:     "change",
		},
		{
			Keys:        "cw",
			Command:     "change word",
			Description: "Delete from cursor to the beginning of the next word and enter insert mode",
			Mode:        "n",
			Category:    "editing",
			Section:     "change",
		},
		{
			Keys:        "c$",
			Command:     "change to end of line",
			Description: "Delete from cursor to the end of the line and enter insert mode",
			Mode:        "n",
			Category:    "editing",
			Section:     "change",
		},
		{
			Keys:        "C",
			Command:     "change to end of line",
			Description: "Delete from cursor to the end of the line and enter insert mode (same as c$)",
			Mode:        "n",
			Category:    "editing",
			Section:     "change",
		},

		// Text objects
		{
			Keys:        "iw",
			Command:     "inner word",
			Description: "Select the word under the cursor (excluding surrounding whitespace)",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "word_objects",
		},
		{
			Keys:        "aw",
			Command:     "a word",
			Description: "Select the word under the cursor (including surrounding whitespace)",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "word_objects",
		},
		{
			Keys:        "iW",
			Command:     "inner WORD",
			Description: "Select the WORD under the cursor (excluding surrounding whitespace)",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "word_objects",
		},
		{
			Keys:        "aW",
			Command:     "a WORD",
			Description: "Select the WORD under the cursor (including surrounding whitespace)",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "word_objects",
		},
		{
			Keys:        "is",
			Command:     "inner sentence",
			Description: "Select the sentence under the cursor",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "sentence_objects",
		},
		{
			Keys:        "as",
			Command:     "a sentence",
			Description: "Select the sentence under the cursor (including surrounding whitespace)",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "sentence_objects",
		},
		{
			Keys:        "ip",
			Command:     "inner paragraph",
			Description: "Select the paragraph under the cursor",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "paragraph_objects",
		},
		{
			Keys:        "ap",
			Command:     "a paragraph",
			Description: "Select the paragraph under the cursor (including surrounding blank lines)",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "paragraph_objects",
		},

		// Quote and bracket text objects
		{
			Keys:        "i\"",
			Command:     "inner double quotes",
			Description: "Select the text inside double quotes",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "quote_objects",
		},
		{
			Keys:        "a\"",
			Command:     "a double quotes",
			Description: "Select the text inside double quotes including the quotes",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "quote_objects",
		},
		{
			Keys:        "i'",
			Command:     "inner single quotes",
			Description: "Select the text inside single quotes",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "quote_objects",
		},
		{
			Keys:        "a'",
			Command:     "a single quotes",
			Description: "Select the text inside single quotes including the quotes",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "quote_objects",
		},
		{
			Keys:        "i`",
			Command:     "inner backticks",
			Description: "Select the text inside backticks",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "quote_objects",
		},
		{
			Keys:        "a`",
			Command:     "a backticks",
			Description: "Select the text inside backticks including the backticks",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "quote_objects",
		},
		{
			Keys:        "i(",
			Command:     "inner parentheses",
			Description: "Select the text inside parentheses",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "bracket_objects",
		},
		{
			Keys:        "a(",
			Command:     "a parentheses",
			Description: "Select the text inside parentheses including the parentheses",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "bracket_objects",
		},
		{
			Keys:        "i[",
			Command:     "inner square brackets",
			Description: "Select the text inside square brackets",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "bracket_objects",
		},
		{
			Keys:        "a[",
			Command:     "a square brackets",
			Description: "Select the text inside square brackets including the brackets",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "bracket_objects",
		},
		{
			Keys:        "i{",
			Command:     "inner curly braces",
			Description: "Select the text inside curly braces",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "bracket_objects",
		},
		{
			Keys:        "a{",
			Command:     "a curly braces",
			Description: "Select the text inside curly braces including the braces",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "bracket_objects",
		},
		{
			Keys:        "i<",
			Command:     "inner angle brackets",
			Description: "Select the text inside angle brackets",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "bracket_objects",
		},
		{
			Keys:        "a<",
			Command:     "a angle brackets",
			Description: "Select the text inside angle brackets including the brackets",
			Mode:        "v",
			Category:    "text_objects",
			Section:     "bracket_objects",
		},

		// Visual mode
		{
			Keys:        "v",
			Command:     "visual mode",
			Description: "Enter visual mode to select text character by character",
			Mode:        "n",
			Category:    "visual",
			Section:     "visual_modes",
		},
		{
			Keys:        "V",
			Command:     "visual line mode",
			Description: "Enter visual line mode to select entire lines",
			Mode:        "n",
			Category:    "visual",
			Section:     "visual_modes",
		},
		{
			Keys:        "<C-v>",
			Command:     "visual block mode",
			Description: "Enter visual block mode to select rectangular blocks of text",
			Mode:        "n",
			Category:    "visual",
			Section:     "visual_modes",
		},
		{
			Keys:        "gv",
			Command:     "reselect visual",
			Description: "Reselect the last visual selection",
			Mode:        "n",
			Category:    "visual",
			Section:     "visual_modes",
		},

		// Search and replace
		{
			Keys:        "/",
			Command:     "search forward",
			Description: "Search for a pattern forward from the cursor",
			Mode:        "n",
			Category:    "search",
			Section:     "search_commands",
		},
		{
			Keys:        "?",
			Command:     "search backward",
			Description: "Search for a pattern backward from the cursor",
			Mode:        "n",
			Category:    "search",
			Section:     "search_commands",
		},
		{
			Keys:        "n",
			Command:     "next match",
			Description: "Go to the next search match",
			Mode:        "n",
			Category:    "search",
			Section:     "search_commands",
		},
		{
			Keys:        "N",
			Command:     "previous match",
			Description: "Go to the previous search match",
			Mode:        "n",
			Category:    "search",
			Section:     "search_commands",
		},
		{
			Keys:        "*",
			Command:     "search word under cursor",
			Description: "Search for the word under the cursor forward",
			Mode:        "n",
			Category:    "search",
			Section:     "search_commands",
		},
		{
			Keys:        "#",
			Command:     "search word under cursor backward",
			Description: "Search for the word under the cursor backward",
			Mode:        "n",
			Category:    "search",
			Section:     "search_commands",
		},

		// Undo and redo
		{
			Keys:        "u",
			Command:     "undo",
			Description: "Undo the last change",
			Mode:        "n",
			Category:    "editing",
			Section:     "undo_redo",
		},
		{
			Keys:        "<C-r>",
			Command:     "redo",
			Description: "Redo the last undone change",
			Mode:        "n",
			Category:    "editing",
			Section:     "undo_redo",
		},
		{
			Keys:        "U",
			Command:     "undo line",
			Description: "Undo all changes on the current line",
			Mode:        "n",
			Category:    "editing",
			Section:     "undo_redo",
		},

		// Window management
		{
			Keys:        "<C-w>h",
			Command:     "window left",
			Description: "Move to the window to the left",
			Mode:        "n",
			Category:    "windows",
			Section:     "window_navigation",
		},
		{
			Keys:        "<C-w>j",
			Command:     "window down",
			Description: "Move to the window below",
			Mode:        "n",
			Category:    "windows",
			Section:     "window_navigation",
		},
		{
			Keys:        "<C-w>k",
			Command:     "window up",
			Description: "Move to the window above",
			Mode:        "n",
			Category:    "windows",
			Section:     "window_navigation",
		},
		{
			Keys:        "<C-w>l",
			Command:     "window right",
			Description: "Move to the window to the right",
			Mode:        "n",
			Category:    "windows",
			Section:     "window_navigation",
		},
		{
			Keys:        "<C-w>s",
			Command:     "split horizontal",
			Description: "Split the current window horizontally",
			Mode:        "n",
			Category:    "windows",
			Section:     "window_splitting",
		},
		{
			Keys:        "<C-w>v",
			Command:     "split vertical",
			Description: "Split the current window vertically",
			Mode:        "n",
			Category:    "windows",
			Section:     "window_splitting",
		},
		{
			Keys:        "<C-w>c",
			Command:     "close window",
			Description: "Close the current window",
			Mode:        "n",
			Category:    "windows",
			Section:     "window_management",
		},
		{
			Keys:        "<C-w>o",
			Command:     "only window",
			Description: "Close all windows except the current one",
			Mode:        "n",
			Category:    "windows",
			Section:     "window_management",
		},

		// Marks and jumps
		{
			Keys:        "m{a-zA-Z}",
			Command:     "set mark",
			Description: "Set a mark at the current position",
			Mode:        "n",
			Category:    "marks",
			Section:     "mark_commands",
		},
		{
			Keys:        "'{a-zA-Z}",
			Command:     "jump to mark line",
			Description: "Jump to the line containing the mark",
			Mode:        "n",
			Category:    "marks",
			Section:     "mark_commands",
		},
		{
			Keys:        "`{a-zA-Z}",
			Command:     "jump to mark position",
			Description: "Jump to the exact position of the mark",
			Mode:        "n",
			Category:    "marks",
			Section:     "mark_commands",
		},
		{
			Keys:        "''",
			Command:     "jump to previous position",
			Description: "Jump to the line of the previous position",
			Mode:        "n",
			Category:    "marks",
			Section:     "mark_commands",
		},
		{
			Keys:        "``",
			Command:     "jump to previous position exact",
			Description: "Jump to the exact previous position",
			Mode:        "n",
			Category:    "marks",
			Section:     "mark_commands",
		},

		// Macros
		{
			Keys:        "q{a-zA-Z}",
			Command:     "record macro",
			Description: "Start recording a macro into register {a-zA-Z}",
			Mode:        "n",
			Category:    "macros",
			Section:     "macro_commands",
		},
		{
			Keys:        "q",
			Command:     "stop recording",
			Description: "Stop recording the current macro",
			Mode:        "n",
			Category:    "macros",
			Section:     "macro_commands",
		},
		{
			Keys:        "@{a-zA-Z}",
			Command:     "play macro",
			Description: "Play the macro stored in register {a-zA-Z}",
			Mode:        "n",
			Category:    "macros",
			Section:     "macro_commands",
		},
		{
			Keys:        "@@",
			Command:     "repeat macro",
			Description: "Repeat the last played macro",
			Mode:        "n",
			Category:    "macros",
			Section:     "macro_commands",
		},

		// Folding
		{
			Keys:        "zf",
			Command:     "create fold",
			Description: "Create a fold for the selected text",
			Mode:        "v",
			Category:    "folding",
			Section:     "fold_commands",
		},
		{
			Keys:        "zo",
			Command:     "open fold",
			Description: "Open the fold under the cursor",
			Mode:        "n",
			Category:    "folding",
			Section:     "fold_commands",
		},
		{
			Keys:        "zc",
			Command:     "close fold",
			Description: "Close the fold under the cursor",
			Mode:        "n",
			Category:    "folding",
			Section:     "fold_commands",
		},
		{
			Keys:        "za",
			Command:     "toggle fold",
			Description: "Toggle the fold under the cursor",
			Mode:        "n",
			Category:    "folding",
			Section:     "fold_commands",
		},
		{
			Keys:        "zR",
			Command:     "open all folds",
			Description: "Open all folds in the buffer",
			Mode:        "n",
			Category:    "folding",
			Section:     "fold_commands",
		},
		{
			Keys:        "zM",
			Command:     "close all folds",
			Description: "Close all folds in the buffer",
			Mode:        "n",
			Category:    "folding",
			Section:     "fold_commands",
		},

		// Miscellaneous
		{
			Keys:        ".",
			Command:     "repeat command",
			Description: "Repeat the last change command",
			Mode:        "n",
			Category:    "editing",
			Section:     "repeat",
		},
		{
			Keys:        "~",
			Command:     "toggle case",
			Description: "Toggle the case of the character under the cursor",
			Mode:        "n",
			Category:    "editing",
			Section:     "case_change",
		},
		{
			Keys:        "gu",
			Command:     "lowercase",
			Description: "Convert text to lowercase (use with motion)",
			Mode:        "n",
			Category:    "editing",
			Section:     "case_change",
		},
		{
			Keys:        "gU",
			Command:     "uppercase",
			Description: "Convert text to uppercase (use with motion)",
			Mode:        "n",
			Category:    "editing",
			Section:     "case_change",
		},
		{
			Keys:        ">>",
			Command:     "indent line",
			Description: "Indent the current line",
			Mode:        "n",
			Category:    "editing",
			Section:     "indentation",
		},
		{
			Keys:        "<<",
			Command:     "unindent line",
			Description: "Unindent the current line",
			Mode:        "n",
			Category:    "editing",
			Section:     "indentation",
		},
		{
			Keys:        "==",
			Command:     "auto-indent line",
			Description: "Auto-indent the current line",
			Mode:        "n",
			Category:    "editing",
			Section:     "indentation",
		},
		{
			Keys:        "J",
			Command:     "join lines",
			Description: "Join the current line with the next line",
			Mode:        "n",
			Category:    "editing",
			Section:     "line_manipulation",
		},
		{
			Keys:        "gJ",
			Command:     "join lines without space",
			Description: "Join the current line with the next line without inserting a space",
			Mode:        "n",
			Category:    "editing",
			Section:     "line_manipulation",
		},
		{
			Keys:        "%",
			Command:     "match bracket",
			Description: "Jump to the matching bracket, parenthesis, or brace",
			Mode:        "n",
			Category:    "movement",
			Section:     "bracket_matching",
		},
	}
}

// convertBuiltinToDocuments converts built-in keybindings to ChromaDB documents
func convertBuiltinToDocuments(keybindings []BuiltinKeybinding) []interfaces.Document {
	var documents []interfaces.Document

	for i, kb := range keybindings {
		// Create content for vectorization
		content := fmt.Sprintf("%s %s %s %s %s %s",
			kb.Keys, kb.Command, kb.Description, kb.Mode, kb.Category, kb.Section)

		// Create metadata map
		metadataMap := map[string]interface{}{
			"keys":        kb.Keys,
			"command":     kb.Command,
			"description": kb.Description,
			"mode":        kb.Mode,
			"category":    kb.Category,
			"section":     kb.Section,
			"source":      "neovim_quickref",
			"type":        "builtin_keybinding",
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

// fetchNeovimQuickRef fetches the Neovim quick reference from the official documentation
// This is a placeholder for future enhancement to parse the actual quick reference
func fetchNeovimQuickRef() (string, error) {
	url := "https://neovim.io/doc/user/quickref.html"

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch quick reference: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	return string(body), nil
}

// parseQuickRef parses the Neovim quick reference HTML and extracts keybindings
// This is a placeholder for future enhancement
func parseQuickRef(html string) []BuiltinKeybinding {
	// This would parse the HTML and extract keybindings
	// For now, we return the hardcoded list
	return getBuiltinKeybindings()
}

// saveBuiltinKeybindings saves the built-in keybindings to a JSON file for reference
func saveBuiltinKeybindings(keybindings []BuiltinKeybinding, filename string) error {
	data, err := json.MarshalIndent(keybindings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keybindings: %v", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write keybindings file: %v", err)
	}

	fmt.Printf("Built-in keybindings saved to %s\n", filename)
	return nil
}
