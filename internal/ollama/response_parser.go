package ollama

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// KeybindingResult represents a parsed keybinding result from LLM response
type KeybindingResult struct {
	Keys        string  `json:"keys"`
	Command     string  `json:"command"`
	Description string  `json:"description"`
	Mode        string  `json:"mode"`
	Relevance   float64 `json:"relevance"`
	Explanation string  `json:"explanation"`
}

// ParsedResponse represents the structured response from LLM
type ParsedResponse struct {
	Results   []KeybindingResult `json:"results"`
	Reasoning string             `json:"reasoning"`
	Query     string             `json:"query"`
}

// ResponseParser handles parsing LLM responses for keybinding search
type ResponseParser struct {
	// JSON parsing patterns
	jsonPattern *regexp.Regexp
	// Keybinding extraction patterns
	keybindPattern *regexp.Regexp
	// Mode extraction patterns
	modePattern *regexp.Regexp
}

// NewResponseParser creates a new response parser
func NewResponseParser() *ResponseParser {
	return &ResponseParser{
		jsonPattern:    regexp.MustCompile(`\{[\s\S]*\}`),
		keybindPattern: regexp.MustCompile(`([a-zA-Z0-9<>-]+):\s*(.+)`),
		modePattern:    regexp.MustCompile(`\b(normal|insert|visual|command|terminal)\b`),
	}
}

// ParseKeybindingResponse parses LLM response for keybinding search results
func (p *ResponseParser) ParseKeybindingResponse(response string, query string) (*ParsedResponse, error) {
	// Try to parse as JSON first
	if parsed, err := p.parseJSONResponse(response, query); err == nil {
		return parsed, nil
	}

	// Fallback to text parsing
	return p.parseTextResponse(response, query)
}

// parseJSONResponse attempts to parse structured JSON response
func (p *ResponseParser) parseJSONResponse(response string, query string) (*ParsedResponse, error) {
	// Extract JSON from response
	jsonMatch := p.jsonPattern.FindString(response)
	if jsonMatch == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var parsed ParsedResponse
	if err := json.Unmarshal([]byte(jsonMatch), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	parsed.Query = query
	return &parsed, nil
}

// parseTextResponse parses unstructured text response
func (p *ResponseParser) parseTextResponse(response string, query string) (*ParsedResponse, error) {
	lines := strings.Split(response, "\n")
	var results []KeybindingResult
	var reasoning strings.Builder

	currentResult := KeybindingResult{}
	inKeybinding := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for keybinding patterns
		if matches := p.keybindPattern.FindStringSubmatch(line); len(matches) == 3 {
			// Save previous result if exists
			if inKeybinding && currentResult.Keys != "" {
				results = append(results, currentResult)
			}

			// Start new result
			currentResult = KeybindingResult{
				Keys:        matches[1],
				Description: matches[2],
				Mode:        p.extractMode(line),
				Relevance:   p.calculateRelevance(matches[1], matches[2], query),
			}
			inKeybinding = true
			continue
		}

		// Check for command descriptions
		if inKeybinding && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch strings.ToLower(key) {
				case "command":
					currentResult.Command = value
				case "description":
					currentResult.Description = value
				case "mode":
					currentResult.Mode = value
				case "explanation":
					currentResult.Explanation = value
				}
			}
			continue
		}

		// Collect reasoning text
		if !inKeybinding {
			reasoning.WriteString(line + " ")
		}
	}

	// Add final result
	if inKeybinding && currentResult.Keys != "" {
		results = append(results, currentResult)
	}

	// If no structured results found, try to extract simple keybindings
	if len(results) == 0 {
		results = p.extractSimpleKeybindings(response, query)
	}

	return &ParsedResponse{
		Results:   results,
		Reasoning: strings.TrimSpace(reasoning.String()),
		Query:     query,
	}, nil
}

// extractSimpleKeybindings extracts simple keybinding mentions from text
func (p *ResponseParser) extractSimpleKeybindings(text string, query string) []KeybindingResult {
	var results []KeybindingResult

	// Common vim keybinding patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b([a-zA-Z]{1,3})\b`),              // Simple keys like dd, yy, etc.
		regexp.MustCompile(`\b(Ctrl\+[a-zA-Z])\b`),             // Ctrl combinations
		regexp.MustCompile(`\b(<[^>]+>)\b`),                    // Angle bracket notation
		regexp.MustCompile(`\b([0-9]*[a-zA-Z][a-zA-Z0-9]*)\b`), // Commands with numbers
	}

	words := strings.Fields(text)
	for _, word := range words {
		for _, pattern := range patterns {
			if pattern.MatchString(word) && p.isLikelyKeybinding(word) {
				results = append(results, KeybindingResult{
					Keys:        word,
					Description: p.inferDescription(word),
					Mode:        "normal",
					Relevance:   p.calculateRelevance(word, "", query),
				})
			}
		}
	}

	return results
}

// isLikelyKeybinding checks if a string is likely a vim keybinding
func (p *ResponseParser) isLikelyKeybinding(s string) bool {
	// Common vim keybindings
	commonKeys := []string{
		"dd", "yy", "cc", "dw", "yw", "cw", "diw", "yiw", "ciw",
		"gg", "G", "0", "$", "^", "w", "b", "e", "f", "t", "F", "T",
		"h", "j", "k", "l", "x", "X", "r", "R", "s", "S", "o", "O",
		"a", "A", "i", "I", "u", "p", "P", "v", "V", "n", "N",
	}

	for _, key := range commonKeys {
		if s == key {
			return true
		}
	}

	// Check for patterns
	if len(s) >= 2 && len(s) <= 4 {
		return true
	}

	return false
}

// inferDescription infers a description for a keybinding
func (p *ResponseParser) inferDescription(keys string) string {
	descriptions := map[string]string{
		"dd":  "Delete current line",
		"yy":  "Yank (copy) current line",
		"cc":  "Change current line",
		"dw":  "Delete word",
		"yw":  "Yank word",
		"cw":  "Change word",
		"diw": "Delete inner word",
		"yiw": "Yank inner word",
		"ciw": "Change inner word",
		"gg":  "Go to first line",
		"G":   "Go to last line",
		"0":   "Go to beginning of line",
		"$":   "Go to end of line",
		"^":   "Go to first non-blank character",
		"w":   "Move to next word",
		"b":   "Move to previous word",
		"e":   "Move to end of word",
		"h":   "Move left",
		"j":   "Move down",
		"k":   "Move up",
		"l":   "Move right",
		"x":   "Delete character under cursor",
		"X":   "Delete character before cursor",
		"r":   "Replace character",
		"s":   "Substitute character",
		"o":   "Open new line below",
		"O":   "Open new line above",
		"a":   "Append after cursor",
		"A":   "Append at end of line",
		"i":   "Insert before cursor",
		"I":   "Insert at beginning of line",
		"u":   "Undo",
		"p":   "Paste after cursor",
		"P":   "Paste before cursor",
		"v":   "Visual mode",
		"V":   "Visual line mode",
		"n":   "Next search result",
		"N":   "Previous search result",
	}

	if desc, exists := descriptions[keys]; exists {
		return desc
	}

	return fmt.Sprintf("Vim command: %s", keys)
}

// extractMode extracts vim mode from text
func (p *ResponseParser) extractMode(text string) string {
	matches := p.modePattern.FindStringSubmatch(strings.ToLower(text))
	if len(matches) > 1 {
		return matches[1]
	}
	return "normal" // Default mode
}

// calculateRelevance calculates relevance score based on query similarity
func (p *ResponseParser) calculateRelevance(keys string, description string, query string) float64 {
	query = strings.ToLower(query)
	keys = strings.ToLower(keys)
	description = strings.ToLower(description)

	score := 0.0

	// Exact key match
	if strings.Contains(query, keys) {
		score += 0.8
	}

	// Description similarity
	queryWords := strings.Fields(query)
	descWords := strings.Fields(description)

	matchCount := 0
	for _, qWord := range queryWords {
		for _, dWord := range descWords {
			if strings.Contains(dWord, qWord) || strings.Contains(qWord, dWord) {
				matchCount++
				break
			}
		}
	}

	if len(queryWords) > 0 {
		score += 0.5 * float64(matchCount) / float64(len(queryWords))
	}

	// Boost common operations
	commonOperations := []string{"delete", "copy", "paste", "move", "jump", "search", "replace"}
	for _, op := range commonOperations {
		if strings.Contains(query, op) && strings.Contains(description, op) {
			score += 0.2
			break
		}
	}

	// Normalize score to 0-1 range
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// ValidateResponse validates that the parsed response contains useful results
func (p *ResponseParser) ValidateResponse(parsed *ParsedResponse) error {
	if parsed == nil {
		return fmt.Errorf("parsed response is nil")
	}

	if len(parsed.Results) == 0 {
		return fmt.Errorf("no keybinding results found")
	}

	// Check that at least one result has keys
	hasValidResult := false
	for _, result := range parsed.Results {
		if result.Keys != "" {
			hasValidResult = true
			break
		}
	}

	if !hasValidResult {
		return fmt.Errorf("no valid keybinding results found")
	}

	return nil
}
