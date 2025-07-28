package chromadb

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// DatabaseInitializer handles pre-built database initialization
type DatabaseInitializer struct {
	config           *Config
	prebuiltPath     string
	userDatabasePath string
}

// NewDatabaseInitializer creates a new database initializer
func NewDatabaseInitializer(config *Config) *DatabaseInitializer {
	if config == nil {
		config = DefaultConfig()
	}

	// Determine pre-built database path (typically bundled with the binary)
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	prebuiltPath := filepath.Join(execDir, "data", "chromadb")

	return &DatabaseInitializer{
		config:           config,
		prebuiltPath:     prebuiltPath,
		userDatabasePath: config.DatabasePath,
	}
}

// InitializeDatabase initializes the user database from pre-built data
func (di *DatabaseInitializer) InitializeDatabase() error {
	log.Printf("Initializing ChromaDB database at: %s", di.userDatabasePath)

	// Check if user database already exists and is valid
	if di.isDatabaseValid() {
		log.Println("Valid database already exists, skipping initialization")
		return nil
	}

	// Check if pre-built database exists
	if !di.prebuiltDatabaseExists() {
		log.Println("Pre-built database not found, will create empty database")
		return di.createEmptyDatabase()
	}

	// Copy pre-built database to user directory
	if err := di.copyPrebuiltDatabase(); err != nil {
		log.Printf("Failed to copy pre-built database: %v", err)
		log.Println("Falling back to empty database creation")
		return di.createEmptyDatabase()
	}

	// Validate the copied database
	if !di.isDatabaseValid() {
		log.Println("Copied database is invalid, falling back to empty database")
		return di.createEmptyDatabase()
	}

	log.Println("Successfully initialized database from pre-built data")
	return nil
}

// prebuiltDatabaseExists checks if the pre-built database exists
func (di *DatabaseInitializer) prebuiltDatabaseExists() bool {
	if _, err := os.Stat(di.prebuiltPath); os.IsNotExist(err) {
		return false
	}

	// Check if it contains expected ChromaDB files
	expectedFiles := []string{"chroma.sqlite3", "index"}
	for _, file := range expectedFiles {
		if _, err := os.Stat(filepath.Join(di.prebuiltPath, file)); os.IsNotExist(err) {
			log.Printf("Pre-built database missing expected file: %s", file)
			return false
		}
	}

	return true
}

// isDatabaseValid checks if the user database exists and is valid
func (di *DatabaseInitializer) isDatabaseValid() bool {
	// Check if database directory exists
	if _, err := os.Stat(di.userDatabasePath); os.IsNotExist(err) {
		return false
	}

	// Check for essential ChromaDB files
	essentialFiles := []string{"chroma.sqlite3"}
	for _, file := range essentialFiles {
		filePath := filepath.Join(di.userDatabasePath, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("Database missing essential file: %s", file)
			return false
		}

		// Check if file is not empty
		if info, err := os.Stat(filePath); err == nil && info.Size() == 0 {
			log.Printf("Database file is empty: %s", file)
			return false
		}
	}

	// Try to validate database integrity by checking checksum if available
	return di.validateDatabaseIntegrity()
}

// validateDatabaseIntegrity validates the database using checksums
func (di *DatabaseInitializer) validateDatabaseIntegrity() bool {
	checksumFile := filepath.Join(di.userDatabasePath, ".checksum")
	
	// If no checksum file exists, assume database is valid
	if _, err := os.Stat(checksumFile); os.IsNotExist(err) {
		return true
	}

	// Read expected checksum
	expectedChecksum, err := os.ReadFile(checksumFile)
	if err != nil {
		log.Printf("Failed to read checksum file: %v", err)
		return true // Assume valid if we can't read checksum
	}

	// Calculate current checksum
	currentChecksum, err := di.calculateDatabaseChecksum()
	if err != nil {
		log.Printf("Failed to calculate database checksum: %v", err)
		return true // Assume valid if we can't calculate checksum
	}

	isValid := strings.TrimSpace(string(expectedChecksum)) == currentChecksum
	if !isValid {
		log.Println("Database checksum validation failed")
	}

	return isValid
}

// calculateDatabaseChecksum calculates a checksum for the database files
func (di *DatabaseInitializer) calculateDatabaseChecksum() (string, error) {
	hash := sha256.New()

	// Hash the main database file
	dbFile := filepath.Join(di.userDatabasePath, "chroma.sqlite3")
	file, err := os.Open(dbFile)
	if err != nil {
		return "", fmt.Errorf("failed to open database file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to hash database file: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// copyPrebuiltDatabase copies the pre-built database to user directory
func (di *DatabaseInitializer) copyPrebuiltDatabase() error {
	log.Printf("Copying pre-built database from %s to %s", di.prebuiltPath, di.userDatabasePath)

	// Remove existing user database if it exists
	if err := os.RemoveAll(di.userDatabasePath); err != nil {
		return fmt.Errorf("failed to remove existing database: %w", err)
	}

	// Create user database directory
	if err := os.MkdirAll(di.userDatabasePath, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Copy all files from pre-built to user directory
	err := filepath.WalkDir(di.prebuiltPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(di.prebuiltPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		destPath := filepath.Join(di.userDatabasePath, relPath)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(destPath, d.Type())
		}

		// Copy file
		return di.copyFile(path, destPath)
	})

	if err != nil {
		return fmt.Errorf("failed to copy database files: %w", err)
	}

	// Generate checksum for the copied database
	if err := di.generateDatabaseChecksum(); err != nil {
		log.Printf("Warning: failed to generate database checksum: %v", err)
	}

	return nil
}

// copyFile copies a single file from src to dst
func (di *DatabaseInitializer) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	// Copy file contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Copy file permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// generateDatabaseChecksum generates and saves a checksum for the database
func (di *DatabaseInitializer) generateDatabaseChecksum() error {
	checksum, err := di.calculateDatabaseChecksum()
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	checksumFile := filepath.Join(di.userDatabasePath, ".checksum")
	if err := os.WriteFile(checksumFile, []byte(checksum), 0644); err != nil {
		return fmt.Errorf("failed to write checksum file: %w", err)
	}

	return nil
}

// createEmptyDatabase creates an empty database structure
func (di *DatabaseInitializer) createEmptyDatabase() error {
	log.Println("Creating empty database structure")

	// Create database directory
	if err := os.MkdirAll(di.userDatabasePath, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Create a marker file to indicate this is an empty database
	markerFile := filepath.Join(di.userDatabasePath, ".empty_db")
	if err := os.WriteFile(markerFile, []byte("empty"), 0644); err != nil {
		return fmt.Errorf("failed to create empty database marker: %w", err)
	}

	log.Println("Empty database structure created")
	return nil
}

// IsEmptyDatabase checks if the database was created as empty
func (di *DatabaseInitializer) IsEmptyDatabase() bool {
	markerFile := filepath.Join(di.userDatabasePath, ".empty_db")
	_, err := os.Stat(markerFile)
	return err == nil
}

// GetDatabaseInfo returns information about the current database
func (di *DatabaseInitializer) GetDatabaseInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["user_database_path"] = di.userDatabasePath
	info["prebuilt_database_path"] = di.prebuiltPath
	info["database_exists"] = di.isDatabaseValid()
	info["prebuilt_exists"] = di.prebuiltDatabaseExists()
	info["is_empty_database"] = di.IsEmptyDatabase()

	// Get database size if it exists
	if di.isDatabaseValid() {
		if size, err := di.getDatabaseSize(); err == nil {
			info["database_size_bytes"] = size
		}
	}

	return info
}

// getDatabaseSize calculates the total size of the database
func (di *DatabaseInitializer) getDatabaseSize() (int64, error) {
	var totalSize int64

	err := filepath.WalkDir(di.userDatabasePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			totalSize += info.Size()
		}

		return nil
	})

	return totalSize, err
}

// RepairDatabase attempts to repair a corrupted database
func (di *DatabaseInitializer) RepairDatabase() error {
	log.Println("Attempting to repair database")

	// First, try to backup the current database
	backupPath := di.userDatabasePath + ".backup"
	if err := di.backupDatabase(backupPath); err != nil {
		log.Printf("Warning: failed to backup database: %v", err)
	}

	// Try to reinitialize from pre-built database
	if err := di.InitializeDatabase(); err != nil {
		// If repair fails, try to restore backup
		if err := di.restoreDatabase(backupPath); err != nil {
			log.Printf("Warning: failed to restore backup: %v", err)
		}
		return fmt.Errorf("database repair failed: %w", err)
	}

	// Remove backup if repair was successful
	os.RemoveAll(backupPath)
	log.Println("Database repair completed successfully")
	return nil
}

// backupDatabase creates a backup of the current database
func (di *DatabaseInitializer) backupDatabase(backupPath string) error {
	if !di.isDatabaseValid() {
		return fmt.Errorf("no valid database to backup")
	}

	// Remove existing backup
	os.RemoveAll(backupPath)

	// Copy current database to backup location
	return filepath.WalkDir(di.userDatabasePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(di.userDatabasePath, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(backupPath, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, d.Type())
		}

		return di.copyFile(path, destPath)
	})
}

// restoreDatabase restores a database from backup
func (di *DatabaseInitializer) restoreDatabase(backupPath string) error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup does not exist")
	}

	// Remove current database
	os.RemoveAll(di.userDatabasePath)

	// Copy backup to current location
	return filepath.WalkDir(backupPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(backupPath, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(di.userDatabasePath, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, d.Type())
		}

		return di.copyFile(path, destPath)
	})
}