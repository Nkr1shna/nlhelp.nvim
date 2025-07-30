package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"nvim-smart-keybind-search/internal/chromadb"
	"nvim-smart-keybind-search/internal/interfaces"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// HuggingFaceDataset represents the structure of the aegis-nvim/neovim-help dataset
type HuggingFaceDataset struct {
	Data []HuggingFaceEntry `json:"data"`
}

// HuggingFaceEntry represents a single entry from the HuggingFace dataset
type HuggingFaceEntry struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags,omitempty"`
	Category    string   `json:"category,omitempty"`
	Section     string   `json:"section,omitempty"`
	Subsection  string   `json:"subsection,omitempty"`
	HelpFile    string   `json:"help_file,omitempty"`
	URL         string   `json:"url,omitempty"`
	LastUpdated string   `json:"last_updated,omitempty"`
}

// ScrapedKnowledge represents a piece of knowledge from your scraping (legacy support)
type ScrapedKnowledge struct {
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	Content    string            `json:"content"`
	Category   string            `json:"category"`
	Tags       []string          `json:"tags,omitempty"`
	Source     string            `json:"source,omitempty"`
	Difficulty string            `json:"difficulty,omitempty"`
	Examples   []string          `json:"examples,omitempty"`
	Related    []string          `json:"related,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

func main() {
	fmt.Println("Populating general knowledge from HuggingFace dataset (aegis-nvim/neovim-help)...")

	// Try to fetch from HuggingFace first, fallback to local file
	var knowledgeData []ScrapedKnowledge
	var err error

	// Check if we have a local cached file first
	cacheFile := "scripts/huggingface_cache.json"
	if _, err := os.Stat(cacheFile); err == nil {
		fmt.Println("Using cached HuggingFace data...")
		knowledgeData, err = loadFromCache(cacheFile)
		if err != nil {
			log.Printf("Failed to load from cache: %v", err)
		}
	}

	// If no cache or cache failed, try to fetch from HuggingFace
	if len(knowledgeData) == 0 {
		fmt.Println("Fetching data from HuggingFace dataset...")
		hfData, err := fetchHuggingFaceDataset()
		if err != nil {
			log.Printf("Failed to fetch from HuggingFace: %v", err)
			// Fallback to example data
			knowledgeData = getExampleKnowledgeData()
		} else {
			knowledgeData = convertHuggingFaceToScraped(hfData)
			// Cache the data for future use
			if err := saveToCache(knowledgeData, cacheFile); err != nil {
				log.Printf("Warning: failed to cache data: %v", err)
			}
		}
	}

	// If we still have no data, use fallback
	if len(knowledgeData) == 0 {
		fmt.Println("Using fallback example data...")
		knowledgeData = getExampleKnowledgeData()
	}

	fmt.Printf("Loaded %d knowledge items\n", len(knowledgeData))

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

	// Convert scraped data to documents
	documents := convertScrapedDataToDocuments(knowledgeData)

	// Store documents in ChromaDB
	fmt.Printf("Storing %d documents in general knowledge collection...\n", len(documents))

	err = cm.StoreGeneralKnowledge(documents)
	if err != nil {
		log.Fatalf("Failed to store general knowledge: %v", err)
	}

	// Get collection count
	count, err := cm.GetGeneralKnowledgeCount()
	if err != nil {
		log.Printf("Warning: failed to get collection count: %v", err)
	} else {
		fmt.Printf("Total documents in general knowledge collection: %d\n", count)
	}

	fmt.Println("Successfully populated general knowledge collection!")
}

// fetchHuggingFaceDataset fetches the dataset from HuggingFace
func fetchHuggingFaceDataset() ([]HuggingFaceEntry, error) {
	// HuggingFace dataset API URL for aegis-nvim/neovim-help
	url := "https://datasets-server.huggingface.co/rows?dataset=aegis-nvim%2Fneovim-help&config=default&split=train"

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("User-Agent", "nvim-smart-keybind-search/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dataset: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse the HuggingFace response format
	var hfResponse struct {
		Rows []struct {
			Row HuggingFaceEntry `json:"row"`
		} `json:"rows"`
	}

	if err := json.Unmarshal(body, &hfResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	// Extract the actual data
	var entries []HuggingFaceEntry
	for _, row := range hfResponse.Rows {
		entries = append(entries, row.Row)
	}

	return entries, nil
}

// convertHuggingFaceToScraped converts HuggingFace entries to ScrapedKnowledge format
func convertHuggingFaceToScraped(hfEntries []HuggingFaceEntry) []ScrapedKnowledge {
	var scraped []ScrapedKnowledge

	for _, entry := range hfEntries {
		// Determine difficulty based on content complexity
		difficulty := "beginner"
		if strings.Contains(strings.ToLower(entry.Content), "advanced") ||
			strings.Contains(strings.ToLower(entry.Content), "complex") {
			difficulty = "advanced"
		} else if strings.Contains(strings.ToLower(entry.Content), "intermediate") {
			difficulty = "intermediate"
		}

		// Extract examples from content (simple heuristic)
		var examples []string
		if strings.Contains(entry.Content, "example:") || strings.Contains(entry.Content, "Example:") {
			// This is a simple extraction - could be improved
			examples = []string{"See content for examples"}
		}

		scraped = append(scraped, ScrapedKnowledge{
			ID:         entry.ID,
			Title:      entry.Title,
			Content:    entry.Content,
			Category:   entry.Category,
			Tags:       entry.Tags,
			Source:     "huggingface:aegis-nvim/neovim-help",
			Difficulty: difficulty,
			Examples:   examples,
			Metadata: map[string]string{
				"help_file":    entry.HelpFile,
				"section":      entry.Section,
				"subsection":   entry.Subsection,
				"url":          entry.URL,
				"last_updated": entry.LastUpdated,
			},
		})
	}

	return scraped
}

// loadFromCache loads cached data from a JSON file
func loadFromCache(filename string) ([]ScrapedKnowledge, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cached []ScrapedKnowledge
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	return cached, nil
}

// saveToCache saves data to a cache file
func saveToCache(data []ScrapedKnowledge, filename string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, jsonData, 0644)
}

// getExampleKnowledgeData provides fallback example data
func getExampleKnowledgeData() []ScrapedKnowledge {
	return []ScrapedKnowledge{
		{
			ID:         "vim_modes_concept",
			Title:      "Vim Modes",
			Content:    "Vim operates in different modes: Normal mode for navigation and commands, Insert mode for typing text, Visual mode for selecting text, and Command-line mode for executing commands. Understanding modes is fundamental to using Vim effectively.",
			Category:   "concepts",
			Tags:       []string{"modes", "basics", "fundamentals"},
			Source:     "fallback_data",
			Difficulty: "beginner",
			Examples:   []string{"Press 'i' to enter Insert mode", "Press 'v' to enter Visual mode", "Press ':' to enter Command-line mode"},
		},
		{
			ID:         "vim_text_objects",
			Title:      "Text Objects",
			Content:    "Text objects in Vim allow you to operate on structured text like words, sentences, paragraphs, and code blocks. They can be used with operators like delete (d), change (c), and yank (y) for precise editing.",
			Category:   "editing",
			Tags:       []string{"text_objects", "operators", "precision"},
			Source:     "fallback_data",
			Difficulty: "intermediate",
			Examples:   []string{"diw - delete inner word", "ci\" - change inside quotes", "yap - yank a paragraph"},
		},
		{
			ID:         "vim_registers",
			Title:      "Vim Registers",
			Content:    "Registers in Vim are storage locations for text. The unnamed register (\"\") stores the last deleted or yanked text. Named registers (a-z) can store text explicitly. Special registers include the system clipboard (+) and selection (*).",
			Category:   "advanced",
			Tags:       []string{"registers", "clipboard", "storage"},
			Source:     "fallback_data",
			Difficulty: "advanced",
			Examples:   []string{"\"ayy - yank line to register 'a'", "\"ap - paste from register 'a'", "\"+y - yank to system clipboard"},
		},
		{
			ID:         "vim_search_replace",
			Title:      "Search and Replace",
			Content:    "Vim provides powerful search and replace functionality using regular expressions. The substitute command (:s) can replace text with various options for scope and behavior.",
			Category:   "search",
			Tags:       []string{"search", "replace", "regex"},
			Source:     "fallback_data",
			Difficulty: "intermediate",
			Examples:   []string{":%s/old/new/g - replace all occurrences", ":s/old/new/gc - replace with confirmation", "/pattern - search forward"},
		},
		{
			ID:         "vim_macros",
			Title:      "Vim Macros",
			Content:    "Macros in Vim allow you to record and replay sequences of commands. They are powerful for automating repetitive tasks and can be stored in registers for reuse.",
			Category:   "automation",
			Tags:       []string{"macros", "automation", "repetition"},
			Source:     "fallback_data",
			Difficulty: "advanced",
			Examples:   []string{"qa - start recording macro 'a'", "q - stop recording", "@a - replay macro 'a'", "10@a - replay macro 10 times"},
		},
	}
}

// convertScrapedDataToDocuments converts scraped data to ChromaDB documents
func convertScrapedDataToDocuments(scrapedData []ScrapedKnowledge) []interfaces.Document {
	var documents []interfaces.Document

	for i, knowledge := range scrapedData {
		// Create content for vectorization
		content := fmt.Sprintf("%s %s %s", knowledge.Title, knowledge.Content, knowledge.Category)

		// Add tags if available
		if len(knowledge.Tags) > 0 {
			content += " " + fmt.Sprintf("%v", knowledge.Tags)
		}

		// Add examples if available
		if len(knowledge.Examples) > 0 {
			content += " examples: " + fmt.Sprintf("%v", knowledge.Examples)
		}

		// Add related topics if available
		if len(knowledge.Related) > 0 {
			content += " related: " + fmt.Sprintf("%v", knowledge.Related)
		}

		// Create metadata map
		metadataMap := map[string]interface{}{
			"id":         knowledge.ID,
			"title":      knowledge.Title,
			"content":    knowledge.Content,
			"category":   knowledge.Category,
			"source":     knowledge.Source,
			"difficulty": knowledge.Difficulty,
			"type":       "general_knowledge",
		}

		// Add tags to metadata
		if len(knowledge.Tags) > 0 {
			metadataMap["tags"] = fmt.Sprintf("%v", knowledge.Tags)
		}

		// Add examples to metadata
		if len(knowledge.Examples) > 0 {
			metadataMap["examples"] = fmt.Sprintf("%v", knowledge.Examples)
		}

		// Add related topics to metadata
		if len(knowledge.Related) > 0 {
			metadataMap["related"] = fmt.Sprintf("%v", knowledge.Related)
		}

		// Add custom metadata
		for k, v := range knowledge.Metadata {
			metadataMap[k] = v
		}

		// Convert to chroma.DocumentMetadata
		metadata, err := chroma.NewDocumentMetadataFromMap(metadataMap)
		if err != nil {
			log.Printf("Warning: failed to create metadata for document %d: %v", i, err)
			metadata = chroma.NewDocumentMetadata()
		}

		document := interfaces.Document{
			ID:       chroma.DocumentID(fmt.Sprintf("scraped_%d", i)),
			Content:  content,
			Metadata: metadata,
		}

		documents = append(documents, document)
	}

	return documents
}
