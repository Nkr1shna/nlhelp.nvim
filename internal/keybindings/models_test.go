package keybindings

import (
	"testing"

	"nvim-smart-keybind-search/internal/interfaces"
)

func TestNewKeybindingParser(t *testing.T) {
	parser := NewKeybindingParser()
	if parser == nil {
		t.Fatal("NewKeybindingParser returned nil")
	}
	
	if parser.validModes == nil {
		t.Error("validModes not initialized")
	}
	
	if parser.keyPattern == nil {
		t.Error("keyPattern not initialized")
	}
}

func TestParseKeybinding(t *testing.T) {
	parser := NewKeybindingParser()
	
	tests := []struct {
		name    string
		input   map[string]interface{}
		want    *interfaces.Keybinding
		wantErr bool
	}{
		{
			name: "valid basic keybinding",
			input: map[string]interface{}{
				"keys":        "dd",
				"command":     "delete line",
				"description": "Delete current line",
				"mode":        "n",
			},
			want: &interfaces.Keybinding{
				Keys:        "dd",
				Command:     "delete line",
				Description: "Delete current line",
				Mode:        "n",
				Metadata:    map[string]string{},
			},
			wantErr: false,
		},
		{
			name: "keybinding with plugin and metadata",
			input: map[string]interface{}{
				"keys":        "<leader>ff",
				"command":     "find files",
				"description": "Find files with telescope",
				"mode":        "n",
				"plugin":      "telescope",
				"metadata": map[string]interface{}{
					"category": "search",
					"priority": "high",
				},
			},
			want: &interfaces.Keybinding{
				Keys:        "<leader>ff",
				Command:     "find files",
				Description: "Find files with telescope",
				Mode:        "n",
				Plugin:      "telescope",
				Metadata: map[string]string{
					"category": "search",
					"priority": "high",
				},
			},
			wantErr: false,
		},
		{
			name: "missing keys field",
			input: map[string]interface{}{
				"command": "delete line",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing command field",
			input: map[string]interface{}{
				"keys": "dd",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "default mode when not specified",
			input: map[string]interface{}{
				"keys":    "yy",
				"command": "yank line",
			},
			want: &interfaces.Keybinding{
				Keys:     "yy",
				Command:  "yank line",
				Mode:     "n", // should default to normal mode
				Metadata: map[string]string{},
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.ParseKeybinding(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKeybinding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr {
				return
			}
			
			// Check all fields except ID (which is generated)
			if got.Keys != tt.want.Keys {
				t.Errorf("Keys = %v, want %v", got.Keys, tt.want.Keys)
			}
			if got.Command != tt.want.Command {
				t.Errorf("Command = %v, want %v", got.Command, tt.want.Command)
			}
			if got.Description != tt.want.Description {
				t.Errorf("Description = %v, want %v", got.Description, tt.want.Description)
			}
			if got.Mode != tt.want.Mode {
				t.Errorf("Mode = %v, want %v", got.Mode, tt.want.Mode)
			}
			if got.Plugin != tt.want.Plugin {
				t.Errorf("Plugin = %v, want %v", got.Plugin, tt.want.Plugin)
			}
			
			// Check metadata
			if len(got.Metadata) != len(tt.want.Metadata) {
				t.Errorf("Metadata length = %v, want %v", len(got.Metadata), len(tt.want.Metadata))
			}
			for k, v := range tt.want.Metadata {
				if got.Metadata[k] != v {
					t.Errorf("Metadata[%s] = %v, want %v", k, got.Metadata[k], v)
				}
			}
			
			// Check that ID was generated
			if got.ID == "" {
				t.Error("ID should be generated when not provided")
			}
		})
	}
}

func TestValidateKeybinding(t *testing.T) {
	parser := NewKeybindingParser()
	
	tests := []struct {
		name    string
		kb      *interfaces.Keybinding
		wantErr bool
	}{
		{
			name: "valid keybinding",
			kb: &interfaces.Keybinding{
				ID:          "test1",
				Keys:        "dd",
				Command:     "delete line",
				Description: "Delete current line",
				Mode:        "n",
				Metadata:    map[string]string{},
			},
			wantErr: false,
		},
		{
			name:    "nil keybinding",
			kb:      nil,
			wantErr: true,
		},
		{
			name: "empty keys",
			kb: &interfaces.Keybinding{
				ID:      "test2",
				Keys:    "",
				Command: "delete line",
				Mode:    "n",
			},
			wantErr: true,
		},
		{
			name: "empty command",
			kb: &interfaces.Keybinding{
				ID:      "test3",
				Keys:    "dd",
				Command: "",
				Mode:    "n",
			},
			wantErr: true,
		},
		{
			name: "empty ID",
			kb: &interfaces.Keybinding{
				ID:      "",
				Keys:    "dd",
				Command: "delete line",
				Mode:    "n",
			},
			wantErr: true,
		},
		{
			name: "invalid mode",
			kb: &interfaces.Keybinding{
				ID:      "test4",
				Keys:    "dd",
				Command: "delete line",
				Mode:    "invalid",
			},
			wantErr: true,
		},
		{
			name: "malformed key sequence",
			kb: &interfaces.Keybinding{
				ID:      "test5",
				Keys:    "<<leader>>",
				Command: "test command",
				Mode:    "n",
			},
			wantErr: true,
		},
		{
			name: "complex valid key sequence",
			kb: &interfaces.Keybinding{
				ID:      "test6",
				Keys:    "<C-w>h",
				Command: "window left",
				Mode:    "n",
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidateKeybinding(tt.kb)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKeybinding() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	parser := NewKeybindingParser()
	
	kb1 := &interfaces.Keybinding{
		Keys:    "dd",
		Command: "delete line",
		Mode:    "n",
		Plugin:  "",
	}
	
	kb2 := &interfaces.Keybinding{
		Keys:    "dd",
		Command: "delete line",
		Mode:    "n",
		Plugin:  "",
	}
	
	kb3 := &interfaces.Keybinding{
		Keys:    "yy",
		Command: "yank line",
		Mode:    "n",
		Plugin:  "",
	}
	
	id1 := parser.GenerateID(kb1)
	id2 := parser.GenerateID(kb2)
	id3 := parser.GenerateID(kb3)
	
	// Same keybindings should generate same ID
	if id1 != id2 {
		t.Errorf("Same keybindings should generate same ID: %s != %s", id1, id2)
	}
	
	// Different keybindings should generate different IDs
	if id1 == id3 {
		t.Errorf("Different keybindings should generate different IDs: %s == %s", id1, id3)
	}
	
	// ID should start with "kb_"
	if len(id1) < 3 || id1[:3] != "kb_" {
		t.Errorf("ID should start with 'kb_': %s", id1)
	}
}

func TestGenerateHash(t *testing.T) {
	parser := NewKeybindingParser()
	
	kb1 := &interfaces.Keybinding{
		Keys:        "dd",
		Command:     "delete line",
		Description: "Delete current line",
		Mode:        "n",
		Plugin:      "",
		Metadata:    map[string]string{"category": "edit"},
	}
	
	kb2 := &interfaces.Keybinding{
		Keys:        "dd",
		Command:     "delete line",
		Description: "Delete current line",
		Mode:        "n",
		Plugin:      "",
		Metadata:    map[string]string{"category": "edit"},
	}
	
	kb3 := &interfaces.Keybinding{
		Keys:        "dd",
		Command:     "delete line",
		Description: "Delete current line modified",
		Mode:        "n",
		Plugin:      "",
		Metadata:    map[string]string{"category": "edit"},
	}
	
	hash1 := parser.GenerateHash(kb1)
	hash2 := parser.GenerateHash(kb2)
	hash3 := parser.GenerateHash(kb3)
	
	// Same keybindings should generate same hash
	if hash1 != hash2 {
		t.Errorf("Same keybindings should generate same hash: %s != %s", hash1, hash2)
	}
	
	// Different keybindings should generate different hashes
	if hash1 == hash3 {
		t.Errorf("Different keybindings should generate different hashes: %s == %s", hash1, hash3)
	}
}

func TestJSONSerialization(t *testing.T) {
	parser := NewKeybindingParser()
	
	original := &interfaces.Keybinding{
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
	
	// Test ToJSON
	jsonData, err := parser.ToJSON(original)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	
	// Test FromJSON
	restored, err := parser.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}
	
	// Compare all fields
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: %s != %s", restored.ID, original.ID)
	}
	if restored.Keys != original.Keys {
		t.Errorf("Keys mismatch: %s != %s", restored.Keys, original.Keys)
	}
	if restored.Command != original.Command {
		t.Errorf("Command mismatch: %s != %s", restored.Command, original.Command)
	}
	if restored.Description != original.Description {
		t.Errorf("Description mismatch: %s != %s", restored.Description, original.Description)
	}
	if restored.Mode != original.Mode {
		t.Errorf("Mode mismatch: %s != %s", restored.Mode, original.Mode)
	}
	if restored.Plugin != original.Plugin {
		t.Errorf("Plugin mismatch: %s != %s", restored.Plugin, original.Plugin)
	}
	
	// Compare metadata
	if len(restored.Metadata) != len(original.Metadata) {
		t.Errorf("Metadata length mismatch: %d != %d", len(restored.Metadata), len(original.Metadata))
	}
	for k, v := range original.Metadata {
		if restored.Metadata[k] != v {
			t.Errorf("Metadata[%s] mismatch: %s != %s", k, restored.Metadata[k], v)
		}
	}
}

func TestParseKeybindingsFromJSON(t *testing.T) {
	parser := NewKeybindingParser()
	
	jsonData := `[
		{
			"keys": "dd",
			"command": "delete line",
			"description": "Delete current line",
			"mode": "n"
		},
		{
			"keys": "yy",
			"command": "yank line",
			"description": "Copy current line",
			"mode": "n",
			"plugin": "builtin"
		}
	]`
	
	keybindings, err := parser.ParseKeybindingsFromJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseKeybindingsFromJSON failed: %v", err)
	}
	
	if len(keybindings) != 2 {
		t.Errorf("Expected 2 keybindings, got %d", len(keybindings))
	}
	
	// Check first keybinding
	if keybindings[0].Keys != "dd" {
		t.Errorf("First keybinding keys: %s != dd", keybindings[0].Keys)
	}
	if keybindings[0].Command != "delete line" {
		t.Errorf("First keybinding command: %s != delete line", keybindings[0].Command)
	}
	
	// Check second keybinding
	if keybindings[1].Keys != "yy" {
		t.Errorf("Second keybinding keys: %s != yy", keybindings[1].Keys)
	}
	if keybindings[1].Plugin != "builtin" {
		t.Errorf("Second keybinding plugin: %s != builtin", keybindings[1].Plugin)
	}
}