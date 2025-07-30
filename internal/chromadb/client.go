package chromadb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"nvim-smart-keybind-search/internal/interfaces"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// Client implements the VectorDB interface for ChromaDB
type Client struct {
	client     chroma.Client
	collection *chroma.Collection
	config     *Config
	ctx        context.Context
}

// Config holds ChromaDB client configuration
type Config struct {
	Host           string
	Port           int
	DatabasePath   string
	CollectionName string
	Timeout        time.Duration
}

// DefaultConfig returns a default ChromaDB configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		Host:           "localhost",
		Port:           8000,
		DatabasePath:   filepath.Join(homeDir, ".config", "nvim-smart-keybind-search", "chromadb"),
		CollectionName: "keybindings",
		Timeout:        30 * time.Second,
	}
}

// NewClient creates a new ChromaDB client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create ChromaDB client with proper base path
	client, err := chroma.NewHTTPClient(chroma.WithBaseURL(fmt.Sprintf("http://%s:%d", config.Host, config.Port)))
	if err != nil {
		return nil, fmt.Errorf("failed to create ChromaDB client: %w", err)
	}

	return &Client{
		client: client,
		config: config,
		ctx:    context.Background(),
	}, nil
}

// Initialize sets up the vector database connection and collections
func (c *Client) Initialize() error {
	// Check if ChromaDB is installed
	if !c.isChromaInstalled() {
		if err := c.installChroma(); err != nil {
			return fmt.Errorf("failed to install ChromaDB: %w", err)
		}
	}

	// Check if ChromaDB service is running
	if !c.isServiceRunning() {
		if err := c.startService(); err != nil {
			return fmt.Errorf("failed to start ChromaDB service: %w", err)
		}
	}

	// Ensure database directory exists
	if err := os.MkdirAll(c.config.DatabasePath, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Test connection to ChromaDB
	if err := c.HealthCheck(); err != nil {
		return fmt.Errorf("ChromaDB connection failed: %w", err)
	}

	// Get or create collection
	if _, err := c.getOrCreateCollection(c.config.CollectionName); err != nil {
		return fmt.Errorf("failed to initialize collection: %w", err)
	}

	log.Printf("ChromaDB initialized with collection: %s", c.config.CollectionName)
	return nil
}

// getOrCreateCollection gets an existing collection or creates a new one
func (c *Client) getOrCreateCollection(name string) (chroma.Collection, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	collection, err := c.client.GetOrCreateCollection(ctx, name)
	if err == nil {
		log.Printf("Using existing ChromaDB collection: %s", name)
		return collection, nil
	}
	return nil, err
}

// getCollection gets an existing collection by name
func (c *Client) getCollection(name string) (*chroma.Collection, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()
	collection, err := c.client.GetOrCreateCollection(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection %s: %w", name, err)
	}

	return &collection, nil
}

// Store stores documents in the vector database
func (c *Client) Store(documents []interfaces.Document) error {
	return c.StoreInCollection(documents, c.config.CollectionName)
}

// StoreInCollection stores documents in a specific collection
func (c *Client) StoreInCollection(documents []interfaces.Document, collectionName string) error {
	if len(documents) == 0 {
		return nil
	}

	// Ensure collection exists
	collection, err := c.getOrCreateCollection(collectionName)
	if err != nil {
		return fmt.Errorf("failed to get or create collection: %w", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	// Prepare data for batch add
	var ids []chroma.DocumentID
	var texts []string
	var metadatas []chroma.DocumentMetadata

	for _, doc := range documents {
		ids = append(ids, doc.ID)
		texts = append(texts, doc.Content)
		metadatas = append(metadatas, doc.Metadata)
	}

	// Execute add operation
	err = collection.Add(ctx,
		chroma.WithIDs(ids...),
		chroma.WithTexts(texts...),
		chroma.WithMetadatas(metadatas...),
	)
	if err != nil {
		return fmt.Errorf("failed to add documents: %v", err)
	}

	log.Printf("Stored %d documents in ChromaDB collection: %s", len(documents), collectionName)
	return nil
}

// Search performs semantic search and returns similar documents
func (c *Client) Search(query string, limit int) ([]interfaces.VectorSearchResult, error) {
	return c.SearchInCollection(query, limit, c.config.CollectionName)
}

// SearchInCollection performs semantic search in a specific collection
func (c *Client) SearchInCollection(query string, limit int, collectionName string) ([]interfaces.VectorSearchResult, error) {
	// Get the collection
	collection, err := c.getCollection(collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection %s: %w", collectionName, err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	// Execute query operation
	results, err := (*collection).Query(ctx,
		chroma.WithQueryTexts(query),
		chroma.WithNResults(limit),
		chroma.WithIncludeQuery(chroma.IncludeDocuments, chroma.IncludeMetadatas),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	return c.convertQueryResults(&results), nil
}

// convertStringsToDocumentIDs converts []string to []DocumentID
func convertStringsToDocumentIDs(ids []string) []chroma.DocumentID {
	result := make([]chroma.DocumentID, len(ids))
	for i, id := range ids {
		result[i] = chroma.DocumentID(id)
	}
	return result
}

// convertQueryResults converts ChromaDB query results to our interface format
func (c *Client) convertQueryResults(r *chroma.QueryResult) []interfaces.VectorSearchResult {
	var searchResults []interfaces.VectorSearchResult
	results := *r
	if results == nil {
		return searchResults
	}

	// Get the result groups
	documentsGroups := results.GetDocumentsGroups()
	metadatasGroups := results.GetMetadatasGroups()
	distancesGroups := results.GetDistancesGroups()
	idsGroups := results.GetIDGroups()

	if len(documentsGroups) == 0 {
		return searchResults
	}

	// Process the first group (since we only query with one query)
	documents := documentsGroups[0]
	var metadatas []chroma.DocumentMetadata
	var distances []float32
	var ids []chroma.DocumentID

	if len(metadatasGroups) > 0 {
		metadatas = metadatasGroups[0]
	}
	if len(distancesGroups) > 0 {
		// Convert embeddings.Distances to []float32
		embeddingDistances := distancesGroups[0]
		distances = make([]float32, len(embeddingDistances))
		for i, d := range embeddingDistances {
			distances[i] = float32(d)
		}
	}
	if len(idsGroups) > 0 {
		ids = idsGroups[0]
	}

	// Process each document in the group
	for i := 0; i < len(documents); i++ {
		var metadata chroma.DocumentMetadata
		if i < len(metadatas) {
			metadata = metadatas[i]
		}

		var distance float32
		if i < len(distances) {
			distance = distances[i]
		}

		var id chroma.DocumentID
		if i < len(ids) {
			id = ids[i]
		}

		document := interfaces.Document{
			ID:       id,
			Content:  documents[i].ContentString(),
			Metadata: metadata,
		}

		// Convert distance to score (lower distance = higher score)
		score := 1.0 - float64(distance)

		searchResult := interfaces.VectorSearchResult{
			Document: document,
			Score:    score,
			Distance: float64(distance),
		}

		searchResults = append(searchResults, searchResult)
	}

	return searchResults
}

// Delete removes documents by their IDs
func (c *Client) Delete(ids []string) error {
	return c.DeleteFromCollection(ids, c.config.CollectionName)
}

// DeleteFromCollection removes documents by their IDs from a specific collection
func (c *Client) DeleteFromCollection(ids []string, collectionName string) error {
	if len(ids) == 0 {
		return nil
	}

	// Get the collection
	collection, err := c.getCollection(collectionName)
	if err != nil {
		return fmt.Errorf("failed to get collection %s: %w", collectionName, err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	// Convert string IDs to DocumentIDs
	documentIDs := convertStringsToDocumentIDs(ids)

	// Execute delete operation
	err = (*collection).Delete(ctx, chroma.WithIDsDelete(documentIDs...))
	if err != nil {
		return fmt.Errorf("failed to delete documents: %v", err)
	}

	log.Printf("Deleted %d documents from ChromaDB collection: %s", len(ids), collectionName)
	return nil
}

// HealthCheck checks if ChromaDB is healthy
func (c *Client) HealthCheck() error {
	// Use HTTP request to v2 API
	resp, err := http.Get(fmt.Sprintf("http://%s:%d/api/v2/heartbeat", c.config.Host, c.config.Port))
	if err != nil {
		return fmt.Errorf("ChromaDB health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("ChromaDB health check failed: status %d", resp.StatusCode)
	}

	return nil
}

// Close closes the database connection
func (c *Client) Close() error {
	// ChromaDB Go client doesn't require explicit closing
	log.Println("ChromaDB client closed")
	return nil
}

// GetCollectionCount returns the number of documents in the collection
func (c *Client) GetCollectionCount() (int, error) {
	return c.GetCollectionCountByName(c.config.CollectionName)
}

// GetCollectionCountByName returns the number of documents in a specific collection
func (c *Client) GetCollectionCountByName(collectionName string) (int, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	// Get collection count using HTTP request
	url := fmt.Sprintf("http://%s:%d/api/v2/collections/%s/count", c.config.Host, c.config.Port, collectionName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection count: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("failed to get collection count: status %d", resp.StatusCode)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if count, ok := result["count"].(float64); ok {
		return int(count), nil
	}

	return 0, fmt.Errorf("failed to extract count from response")
}

// ListCollections returns all available collections
func (c *Client) ListCollections() ([]string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	// List collections using HTTP request
	url := fmt.Sprintf("http://%s:%d/api/v2/collections", c.config.Host, c.config.Port)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to list collections: status %d", resp.StatusCode)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var names []string
	if collections, ok := result["collections"].([]interface{}); ok {
		for _, collection := range collections {
			if col, ok := collection.(map[string]interface{}); ok {
				if name, ok := col["name"].(string); ok {
					names = append(names, name)
				}
			}
		}
	}

	return names, nil
}

// installChroma automatically installs ChromaDB
func (c *Client) installChroma() error {
	fmt.Println("Installing ChromaDB...")

	// Check if uv is installed, install it if not
	if !c.isUvInstalled() {
		if err := c.installUv(); err != nil {
			return fmt.Errorf("failed to install uv: %w", err)
		}
	}

	// Use uv to create virtual environment and install chromadb
	return c.installChromaWithUv()
}

// installUv installs uv using curl or wget
func (c *Client) installUv() error {
	fmt.Println("Installing uv...")

	// Check if curl is available
	if c.isCurlInstalled() {
		return c.installUvWithCurl()
	}

	// Check if wget is available
	if c.isWgetInstalled() {
		return c.installUvWithWget()
	}

	return fmt.Errorf("neither curl nor wget is available to install uv")
}

// isCurlInstalled checks if curl is installed
func (c *Client) isCurlInstalled() bool {
	_, err := exec.LookPath("curl")
	return err == nil
}

// isWgetInstalled checks if wget is installed
func (c *Client) isWgetInstalled() bool {
	_, err := exec.LookPath("wget")
	return err == nil
}

// installUvWithCurl installs uv using curl
func (c *Client) installUvWithCurl() error {
	fmt.Println("Installing uv using curl...")

	cmd := exec.Command("curl", "-LsSf", "https://astral.sh/uv/install.sh")
	installScript, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to download uv installer with curl: %w", err)
	}

	// Execute the install script
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, "sh", "-c", string(installScript))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute uv installer: %w", err)
	}

	// Verify installation
	if !c.isUvInstalled() {
		return fmt.Errorf("uv installation verification failed")
	}

	fmt.Println("uv installed successfully")
	return nil
}

// installUvWithWget installs uv using wget
func (c *Client) installUvWithWget() error {
	fmt.Println("Installing uv using wget...")

	cmd := exec.Command("wget", "-qO-", "https://astral.sh/uv/install.sh")
	installScript, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to download uv installer with wget: %w", err)
	}

	// Execute the install script
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, "sh", "-c", string(installScript))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute uv installer: %w", err)
	}

	// Verify installation
	if !c.isUvInstalled() {
		return fmt.Errorf("uv installation verification failed")
	}

	fmt.Println("uv installed successfully")
	return nil
}

// installChromaWithUv installs ChromaDB using uv
func (c *Client) installChromaWithUv() error {
	fmt.Println("Installing ChromaDB using uv...")

	// Create a project directory in the database path
	projectPath := filepath.Join(c.config.DatabasePath, "chromadb-project")

	// Create project directory
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Change to project directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(projectPath); err != nil {
		return fmt.Errorf("failed to change to project directory: %w", err)
	}

	// Initialize a new Python project
	cmd := exec.Command("uv", "init", "--no-readme")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize project: %w", err)
	}

	// Add chromadb to the project using uv add
	cmd = exec.Command("uv", "add", "chromadb")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add chromadb to project: %w", err)
	}

	fmt.Println("ChromaDB installed successfully via uv")
	return nil
}

// isUvInstalled checks if uv is installed
func (c *Client) isUvInstalled() bool {
	_, err := exec.LookPath("uv")
	return err == nil
}

// isChromaInstalled checks if ChromaDB is installed
func (c *Client) isChromaInstalled() bool {
	// Check if uv project exists (highest priority)
	projectPath := filepath.Join(c.config.DatabasePath, "chromadb-project")
	if _, err := os.Stat(projectPath); err == nil {
		// Check if chromadb is in the project dependencies
		pyprojectPath := filepath.Join(projectPath, "pyproject.toml")
		if _, err := os.Stat(pyprojectPath); err == nil {
			return true
		}
	}

	// Check if chromadb Python module is available
	cmd := exec.Command("python3", "-c", "import chromadb")
	if err := cmd.Run(); err == nil {
		return true
	}

	// Check with pip3
	cmd = exec.Command("pip3", "show", "chromadb")
	if err := cmd.Run(); err == nil {
		return true
	}

	// Check with pip
	cmd = exec.Command("pip", "show", "chromadb")
	if err := cmd.Run(); err == nil {
		return true
	}

	// Check for system chroma binary (but exclude npm installations)
	if chromaPath, err := exec.LookPath("chroma"); err == nil {
		// Check if it's not from npm
		if !strings.Contains(chromaPath, ".nvm") && !strings.Contains(chromaPath, "node") {
			return true
		}
	}

	return false
}

// isServiceRunning checks if ChromaDB service is running
func (c *Client) isServiceRunning() bool {
	// Try to connect to the ChromaDB service
	resp, err := http.Get("http://localhost:8000/api/v2/heartbeat")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// startService starts the ChromaDB service
func (c *Client) startService() error {
	fmt.Println("Starting ChromaDB service...")

	// Ensure database directory exists before starting service
	if err := os.MkdirAll(c.config.DatabasePath, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	fmt.Printf("Database path: %s\n", c.config.DatabasePath)

	// Try different ways to start ChromaDB based on installation method
	var cmd *exec.Cmd

	// Check if uv project exists and use it
	projectPath := filepath.Join(c.config.DatabasePath, "chromadb-project")
	if _, err := os.Stat(projectPath); err == nil {
		fmt.Printf("Found uv project at: %s\n", projectPath)
		// Use uv project to run chroma
		cmd = exec.Command("uv", "run", "chroma", "run", "--path", c.config.DatabasePath)
		cmd.Dir = projectPath
	} else {
		fmt.Printf("No uv project found, checking for system chroma\n")
		// Try chroma binary first (but exclude npm installations)
		if chromaPath, err := exec.LookPath("chroma"); err == nil && !strings.Contains(chromaPath, ".nvm") && !strings.Contains(chromaPath, "node") {
			fmt.Printf("Found system chroma binary: %s\n", chromaPath)
			cmd = exec.Command("chroma", "run", "--path", c.config.DatabasePath)
		} else {
			fmt.Printf("No system chroma binary found, trying python module\n")
			// Try using python module
			cmd = exec.Command("python3", "-m", "chromadb", "run", "--path", c.config.DatabasePath)
		}
	}

	fmt.Printf("Starting ChromaDB with command: %s\n", cmd.String())

	// Start the service in background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ChromaDB service: %w", err)
	}

	fmt.Printf("ChromaDB process started with PID: %d\n", cmd.Process.Pid)

	// Wait for service to be ready
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		if c.isServiceRunning() {
			fmt.Println("ChromaDB service started successfully")
			return nil
		}
		fmt.Printf("Waiting for ChromaDB to be ready... (%d/%d)\n", i+1, maxRetries)
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("ChromaDB service failed to start within timeout")
}
