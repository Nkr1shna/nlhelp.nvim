package keybindings

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"nvim-smart-keybind-search/internal/interfaces"
)

// KeybindingParser handles parsing and validation of keybinding data
type KeybindingParser struct {
	validModes map[string]bool
	keyPattern *regexp.Regexp
}

// NewKeybindingParser creates a new keybinding parser with validation rules
func NewKeybindingParser() *KeybindingParser {
	validModes := map[string]bool{
		"n":  true, // normal mode
		"i":  true, // insert mode
		"v":  true, // visual mode
		"x":  true, // visual block mode
		"s":  true, // select mode
		"o":  true, // operator-pending mode
		"c":  true, // command-line mode
		"t":  true, // terminal mode
		"":   true, // all modes
		"!":  true, // insert and command-line modes
		"ic": true, // insert and command-line modes
	}

	// Pattern to validate key sequences (basic vim key notation)
	keyPattern := regexp.MustCompile(`^[<>a-zA-Z0-9\-_\s\[\]{}().,;:'"\\|/+=*&^%$#@!~` + "`" + `?]*$`)

	return &KeybindingParser{
		validModes: validModes,
		keyPattern: keyPattern,
	}
}

// ParseKeybinding parses a raw keybinding map into a structured Keybinding
func (p *KeybindingParser) ParseKeybinding(data map[string]interface{}) (*interfaces.Keybinding, error) {
	kb := &interfaces.Keybinding{
		Metadata: make(map[string]string),
	}

	// Parse required fields
	if keys, ok := data["keys"].(string); ok {
		kb.Keys = strings.TrimSpace(keys)
	} else {
		return nil, fmt.Errorf("missing or invalid 'keys' field")
	}

	if command, ok := data["command"].(string); ok {
		kb.Command = strings.TrimSpace(command)
	} else {
		return nil, fmt.Errorf("missing or invalid 'command' field")
	}

	// Parse optional fields
	if desc, ok := data["description"].(string); ok {
		kb.Description = strings.TrimSpace(desc)
	}

	if mode, ok := data["mode"].(string); ok {
		kb.Mode = strings.TrimSpace(mode)
	} else {
		kb.Mode = "n" // default to normal mode
	}

	if plugin, ok := data["plugin"].(string); ok {
		kb.Plugin = strings.TrimSpace(plugin)
	}

	// Parse metadata
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		for k, v := range metadata {
			if str, ok := v.(string); ok {
				kb.Metadata[k] = str
			}
		}
	}

	// Generate ID if not provided
	if id, ok := data["id"].(string); ok {
		kb.ID = strings.TrimSpace(id)
	} else {
		kb.ID = p.GenerateID(kb)
	}

	return kb, nil
}

// ParseKeybindingsFromJSON parses a JSON array of keybindings
func (p *KeybindingParser) ParseKeybindingsFromJSON(jsonData []byte) ([]interfaces.Keybinding, error) {
	var rawKeybindings []map[string]interface{}
	if err := json.Unmarshal(jsonData, &rawKeybindings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	keybindings := make([]interfaces.Keybinding, 0, len(rawKeybindings))
	for i, raw := range rawKeybindings {
		kb, err := p.ParseKeybinding(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse keybinding at index %d: %w", i, err)
		}
		keybindings = append(keybindings, *kb)
	}

	return keybindings, nil
}

// ValidateKeybinding validates a keybinding for integrity and correctness
func (p *KeybindingParser) ValidateKeybinding(kb *interfaces.Keybinding) error {
	if kb == nil {
		return fmt.Errorf("keybinding is nil")
	}

	// Validate required fields
	if kb.Keys == "" {
		return fmt.Errorf("keys field is required and cannot be empty")
	}

	if kb.Command == "" {
		return fmt.Errorf("command field is required and cannot be empty")
	}

	if kb.ID == "" {
		return fmt.Errorf("id field is required and cannot be empty")
	}

	// Validate key sequence format
	if !p.keyPattern.MatchString(kb.Keys) {
		return fmt.Errorf("invalid key sequence format: %s", kb.Keys)
	}

	// Validate mode
	if !p.validModes[kb.Mode] {
		return fmt.Errorf("invalid mode: %s", kb.Mode)
	}

	// Validate key sequence doesn't contain obvious errors
	if strings.Contains(kb.Keys, "<<") || strings.Contains(kb.Keys, ">>") {
		return fmt.Errorf("malformed key sequence with double brackets: %s", kb.Keys)
	}

	return nil
}

// GenerateID generates a unique ID for a keybinding based on its content
func (p *KeybindingParser) GenerateID(kb *interfaces.Keybinding) string {
	// Create a hash based on keys, command, mode, and plugin
	content := fmt.Sprintf("%s|%s|%s|%s", kb.Keys, kb.Command, kb.Mode, kb.Plugin)
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("kb_%x", hash[:8]) // Use first 8 bytes for shorter ID
}

// GenerateHash generates a content hash for change detection
func (p *KeybindingParser) GenerateHash(kb *interfaces.Keybinding) string {
	// Include all fields that matter for change detection
	content := fmt.Sprintf("%s|%s|%s|%s|%s", 
		kb.Keys, kb.Command, kb.Description, kb.Mode, kb.Plugin)
	
	// Add metadata in sorted order for consistent hashing
	if len(kb.Metadata) > 0 {
		var metaPairs []string
		for k, v := range kb.Metadata {
			metaPairs = append(metaPairs, fmt.Sprintf("%s=%s", k, v))
		}
		content += "|" + strings.Join(metaPairs, "&")
	}
	
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// ToJSON serializes a keybinding to JSON
func (p *KeybindingParser) ToJSON(kb *interfaces.Keybinding) ([]byte, error) {
	return json.Marshal(kb)
}

// ToJSONArray serializes multiple keybindings to JSON array
func (p *KeybindingParser) ToJSONArray(keybindings []interfaces.Keybinding) ([]byte, error) {
	return json.Marshal(keybindings)
}

// FromJSON deserializes a keybinding from JSON
func (p *KeybindingParser) FromJSON(jsonData []byte) (*interfaces.Keybinding, error) {
	var kb interfaces.Keybinding
	if err := json.Unmarshal(jsonData, &kb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal keybinding: %w", err)
	}
	
	// Validate the deserialized keybinding
	if err := p.ValidateKeybinding(&kb); err != nil {
		return nil, fmt.Errorf("invalid keybinding after deserialization: %w", err)
	}
	
	return &kb, nil
}