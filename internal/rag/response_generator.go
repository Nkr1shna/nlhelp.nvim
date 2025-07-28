package rag

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"nvim-smart-keybind-search/internal/interfaces"
)

// ResponseGenerator handles LLM response parsing and result ranking
type ResponseGenerator struct {
	llmClient interfaces.LLMClient
	config    *ResponseGeneratorConfig
}

// ResponseGeneratorConfig holds configuration for response generation
type ResponseGeneratorConfig struct {
	MaxResponseTokens  int
	Temperature        float64
	ResponseTimeout    time.Duration
	EnableExplanations bool
	EnableRanking      bool
	UserBoostFactor    float64
	RelevanceThreshold float64
	MaxFinalResults    int
}

// DefaultResponseGeneratorConfig returns default configuration
func DefaultResponseGeneratorConfig() *ResponseGeneratorConfig {
	return &ResponseGeneratorConfig{
		MaxResponseTokens:  500,
		Temperature:        0.1,
		ResponseTimeout:    30 * time.Second,
		EnableExplanations: true,
		EnableRanking:      true,
		UserBoostFactor:    0.2,
		RelevanceThreshold: 0.3,
		MaxFinalResults:    5,
	}
}

// ParsedKeybindingResult represents a keybinding result parsed from LLM response
type ParsedKeybindingResult struct {
	Keys        string
	Command     string
	Description string
	Mode        string
	Explanation string
	LLMScore    float64
	Rank        int
	Confidence  float64
}

// RankedResult represents a final ranked result combining vector and LLM scores
type RankedResult struct {
	Keybinding     interfaces.Keybinding
	VectorScore    float64
	LLMScore       float64
	FinalRelevance float64
	Explanation    string
	RankingFactors map[string]float64
	MatchedTerms   []string
}

// ResponseAnalysis contains analysis of the LLM response
type ResponseAnalysis struct {
	ParsedResults          []ParsedKeybindingResult
	Reasoning              string
	Confidence             float64
	AlternativeSuggestions []string
	ParseErrors            []string
	ProcessingTime         time.Duration
}

// NewResponseGenerator creates a new response generator
func NewResponseGenerator(llmClient interfaces.LLMClient, config *ResponseGeneratorConfig) *ResponseGenerator {
	if config == nil {
		config = DefaultResponseGeneratorConfig()
	}

	return &ResponseGenerator{
		llmClient: llmClient,
		config:    config,
	}
}

// GenerateAndParseResponse generates LLM response and parses it into structured results
func (rg *ResponseGenerator) GenerateAndParseResponse(query, context string) (*ResponseAnalysis, error) {
	start := time.Now()

	log.Printf("Generating LLM response for query: %s", query)

	// Generate LLM response
	llmResponse, err := rg.generateResponse(query, context)
	if err != nil {
		return nil, fmt.Errorf("failed to generate LLM response: %w", err)
	}

	// Parse the response
	analysis, err := rg.parseResponse(llmResponse.Text, query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	analysis.ProcessingTime = time.Since(start)
	log.Printf("Response generation and parsing completed in %v", analysis.ProcessingTime)

	return analysis, nil
}

// generateResponse generates LLM response using specialized prompt
func (rg *ResponseGenerator) generateResponse(query, context string) (*interfaces.LLMResponse, error) {
	prompt := rg.buildResponsePrompt(query, context)

	llmRequest := interfaces.LLMRequest{
		Prompt:      prompt,
		Context:     context,
		MaxTokens:   rg.config.MaxResponseTokens,
		Temperature: rg.config.Temperature,
	}

	response, err := rg.llmClient.Generate(llmRequest)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	if response.Error != "" {
		return nil, fmt.Errorf("LLM returned error: %s", response.Error)
	}

	if strings.TrimSpace(response.Text) == "" {
		return nil, fmt.Errorf("LLM returned empty response")
	}

	return response, nil
}

// buildResponsePrompt creates a specialized prompt for keybinding response generation
func (rg *ResponseGenerator) buildResponsePrompt(query, context string) string {
	return fmt.Sprintf(`You are a vim/neovim expert assistant. Based on the user's query and the relevant keybindings found, provide a structured analysis.

%s

Analyze the keybindings and provide a response in this exact format:

ANALYSIS:
[Brief explanation of what the user is trying to accomplish]

RECOMMENDATIONS:
1. Keys: [keybinding] | Command: [command] | Description: [description] | Mode: [mode] | Score: [0.0-1.0] | Explanation: [why this is relevant]
2. Keys: [keybinding] | Command: [command] | Description: [description] | Mode: [mode] | Score: [0.0-1.0] | Explanation: [why this is relevant]
3. Keys: [keybinding] | Command: [command] | Description: [description] | Mode: [mode] | Score: [0.0-1.0] | Explanation: [why this is relevant]

REASONING:
[Explain why these keybindings were selected and how they address the user's query]

ALTERNATIVES:
[Mention any alternative approaches or related keybindings]

Instructions:
- Rank recommendations by relevance to the query (most relevant first)
- Provide scores from 0.0 to 1.0 based on how well each keybinding matches the query
- Focus on practical, actionable keybindings
- Prioritize user-configured keybindings when available
- Include mode information (normal, insert, visual, command, terminal)
- Provide clear explanations for why each keybinding is relevant

Response:`, context)
}

// parseResponse parses the structured LLM response into analysis
func (rg *ResponseGenerator) parseResponse(response, query string) (*ResponseAnalysis, error) {
	analysis := &ResponseAnalysis{
		ParsedResults: []ParsedKeybindingResult{},
		ParseErrors:   []string{},
	}

	// Split response into sections
	sections := rg.splitResponseIntoSections(response)

	// Parse each section
	if analysisText, exists := sections["ANALYSIS"]; exists {
		// Extract overall confidence from analysis
		analysis.Confidence = rg.extractConfidenceFromText(analysisText)
	}

	if recommendationsText, exists := sections["RECOMMENDATIONS"]; exists {
		parsedResults, parseErrors := rg.parseRecommendations(recommendationsText)
		analysis.ParsedResults = parsedResults
		analysis.ParseErrors = append(analysis.ParseErrors, parseErrors...)
	}

	if reasoningText, exists := sections["REASONING"]; exists {
		analysis.Reasoning = strings.TrimSpace(reasoningText)
	}

	if alternativesText, exists := sections["ALTERNATIVES"]; exists {
		analysis.AlternativeSuggestions = rg.parseAlternatives(alternativesText)
	}

	// Validate parsed results
	if len(analysis.ParsedResults) == 0 {
		// Fallback parsing if structured format failed
		fallbackResults, err := rg.fallbackParsing(response, query)
		if err != nil {
			return nil, fmt.Errorf("both structured and fallback parsing failed: %w", err)
		}
		analysis.ParsedResults = fallbackResults
		analysis.ParseErrors = append(analysis.ParseErrors, "Used fallback parsing due to format issues")
	}

	return analysis, nil
}

// splitResponseIntoSections splits the response into labeled sections
func (rg *ResponseGenerator) splitResponseIntoSections(response string) map[string]string {
	sections := make(map[string]string)

	// Define section headers
	headers := []string{"ANALYSIS:", "RECOMMENDATIONS:", "REASONING:", "ALTERNATIVES:"}

	currentSection := ""
	var currentContent strings.Builder

	lines := strings.Split(response, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check if this line is a section header
		isHeader := false
		for _, header := range headers {
			if strings.HasPrefix(strings.ToUpper(line), header) {
				// Save previous section if exists
				if currentSection != "" {
					sections[currentSection] = strings.TrimSpace(currentContent.String())
				}

				// Start new section
				currentSection = strings.TrimSuffix(header, ":")
				currentContent.Reset()
				isHeader = true
				break
			}
		}

		// Add line to current section if not a header
		if !isHeader && currentSection != "" {
			currentContent.WriteString(line + "\n")
		}
	}

	// Save the last section
	if currentSection != "" {
		sections[currentSection] = strings.TrimSpace(currentContent.String())
	}

	return sections
}

// parseRecommendations parses the recommendations section
func (rg *ResponseGenerator) parseRecommendations(text string) ([]ParsedKeybindingResult, []string) {
	var results []ParsedKeybindingResult
	var errors []string

	lines := strings.Split(text, "\n")

	// Regex pattern to match the structured format
	pattern := regexp.MustCompile(`(?i)(\d+)\.\s*Keys:\s*([^|]+)\s*\|\s*Command:\s*([^|]*)\s*\|\s*Description:\s*([^|]*)\s*\|\s*Mode:\s*([^|]*)\s*\|\s*Score:\s*([0-9.]+)\s*\|\s*Explanation:\s*(.*)`)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := pattern.FindStringSubmatch(line)
		if len(matches) >= 8 {
			// Parse the structured format
			result := ParsedKeybindingResult{
				Keys:        strings.TrimSpace(matches[2]),
				Command:     strings.TrimSpace(matches[3]),
				Description: strings.TrimSpace(matches[4]),
				Mode:        strings.TrimSpace(matches[5]),
				Explanation: strings.TrimSpace(matches[7]),
			}

			// Parse rank
			if rank, err := strconv.Atoi(matches[1]); err == nil {
				result.Rank = rank
			} else {
				result.Rank = i + 1
			}

			// Parse score
			if score, err := strconv.ParseFloat(matches[6], 64); err == nil {
				result.LLMScore = score
				result.Confidence = score // Use score as confidence for now
			} else {
				errors = append(errors, fmt.Sprintf("Failed to parse score for line %d: %s", i+1, matches[6]))
				result.LLMScore = 0.5 // Default score
				result.Confidence = 0.5
			}

			results = append(results, result)
		} else {
			// Try simpler parsing patterns
			simpleResult := rg.parseSimpleRecommendationLine(line, i+1)
			if simpleResult != nil {
				results = append(results, *simpleResult)
			} else {
				errors = append(errors, fmt.Sprintf("Failed to parse recommendation line %d: %s", i+1, line))
			}
		}
	}

	return results, errors
}

// parseSimpleRecommendationLine parses a recommendation line with simpler format
func (rg *ResponseGenerator) parseSimpleRecommendationLine(line string, rank int) *ParsedKeybindingResult {
	// Try to extract keybinding from various formats
	// Format: "dd: Delete current line"
	// Format: "Keys: dd, Description: Delete line"
	// Format: "1. dd - Delete current line"

	result := &ParsedKeybindingResult{
		Rank:       rank,
		LLMScore:   0.7, // Default score for simple parsing
		Confidence: 0.6,
	}

	// Remove numbering if present
	line = regexp.MustCompile(`^\d+\.\s*`).ReplaceAllString(line, "")

	// Try colon format: "dd: Delete current line"
	if colonMatch := regexp.MustCompile(`^([^:]+):\s*(.+)$`).FindStringSubmatch(line); len(colonMatch) == 3 {
		result.Keys = strings.TrimSpace(colonMatch[1])
		result.Description = strings.TrimSpace(colonMatch[2])
		result.Explanation = result.Description
		return result
	}

	// Try dash format: "dd - Delete current line"
	if dashMatch := regexp.MustCompile(`^([^-]+)-\s*(.+)$`).FindStringSubmatch(line); len(dashMatch) == 3 {
		result.Keys = strings.TrimSpace(dashMatch[1])
		result.Description = strings.TrimSpace(dashMatch[2])
		result.Explanation = result.Description
		return result
	}

	// Try to extract keys from the beginning of the line
	words := strings.Fields(line)
	if len(words) > 0 {
		// Assume first word/phrase is the keybinding
		result.Keys = words[0]
		if len(words) > 1 {
			result.Description = strings.Join(words[1:], " ")
			result.Explanation = result.Description
		}
		return result
	}

	return nil
}

// extractConfidenceFromText extracts confidence level from analysis text
func (rg *ResponseGenerator) extractConfidenceFromText(text string) float64 {
	// Look for confidence indicators in the text
	text = strings.ToLower(text)

	if strings.Contains(text, "highly confident") || strings.Contains(text, "very confident") {
		return 0.9
	} else if strings.Contains(text, "confident") {
		return 0.8
	} else if strings.Contains(text, "likely") || strings.Contains(text, "probably") {
		return 0.7
	} else if strings.Contains(text, "possibly") || strings.Contains(text, "might") {
		return 0.6
	} else if strings.Contains(text, "uncertain") || strings.Contains(text, "unclear") {
		return 0.4
	}

	return 0.7 // Default confidence
}

// parseAlternatives parses alternative suggestions from the response
func (rg *ResponseGenerator) parseAlternatives(text string) []string {
	var alternatives []string

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Remove bullet points or numbering
			line = regexp.MustCompile(`^[-*•]\s*`).ReplaceAllString(line, "")
			line = regexp.MustCompile(`^\d+\.\s*`).ReplaceAllString(line, "")

			if line != "" {
				alternatives = append(alternatives, line)
			}
		}
	}

	return alternatives
}

// fallbackParsing attempts to parse response when structured format fails
func (rg *ResponseGenerator) fallbackParsing(response, query string) ([]ParsedKeybindingResult, error) {
	var results []ParsedKeybindingResult

	// Look for keybinding patterns in the text
	keybindingPattern := regexp.MustCompile(`(?i)(?:keys?:?\s*)?([a-zA-Z0-9<>]+(?:[+-][a-zA-Z0-9<>]+)*)\s*(?:[:,-]|\s+)\s*([^.\n]+)`)

	matches := keybindingPattern.FindAllStringSubmatch(response, -1)

	for i, match := range matches {
		if len(match) >= 3 {
			result := ParsedKeybindingResult{
				Keys:        strings.TrimSpace(match[1]),
				Description: strings.TrimSpace(match[2]),
				Explanation: strings.TrimSpace(match[2]),
				Rank:        i + 1,
				LLMScore:    0.6, // Lower score for fallback parsing
				Confidence:  0.5,
			}

			// Clean up description
			result.Description = strings.TrimSuffix(result.Description, ".")
			result.Explanation = result.Description

			results = append(results, result)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no keybindings found in response")
	}

	return results, nil
}

// RankAndCombineResults combines vector search results with LLM analysis to create final ranked results
func (rg *ResponseGenerator) RankAndCombineResults(
	query string,
	vectorResults []interfaces.VectorSearchResult,
	llmAnalysis *ResponseAnalysis,
	processedQuery *ProcessedQuery,
) ([]interfaces.SearchResult, string, error) {

	log.Printf("Ranking and combining %d vector results with %d LLM results",
		len(vectorResults), len(llmAnalysis.ParsedResults))

	// Create lookup map for vector results
	vectorMap := make(map[string]interfaces.VectorSearchResult)
	for _, result := range vectorResults {
		kbID := getMetadataStringFromResult(result, "keybinding_id")
		if kbID != "" {
			vectorMap[kbID] = result
		}
		// Also index by keys for matching
		keys := getMetadataStringFromResult(result, "keys")
		if keys != "" {
			vectorMap[keys] = result
		}
	}

	// Create ranked results by combining vector and LLM scores
	var rankedResults []RankedResult

	// Process LLM recommendations first (higher priority)
	for _, llmResult := range llmAnalysis.ParsedResults {
		rankedResult := rg.createRankedResult(llmResult, vectorMap, processedQuery)
		if rankedResult != nil {
			rankedResults = append(rankedResults, *rankedResult)
		}
	}

	// Add high-scoring vector results that weren't mentioned by LLM
	for _, vectorResult := range vectorResults {
		if vectorResult.Score >= rg.config.RelevanceThreshold {
			// Check if already included
			alreadyIncluded := false
			keys := getMetadataStringFromResult(vectorResult, "keys")

			for _, existing := range rankedResults {
				if existing.Keybinding.Keys == keys {
					alreadyIncluded = true
					break
				}
			}

			if !alreadyIncluded {
				rankedResult := rg.createRankedResultFromVector(vectorResult, query, processedQuery)
				rankedResults = append(rankedResults, rankedResult)
			}
		}
	}

	// Apply final ranking algorithm
	rg.applyFinalRanking(rankedResults, processedQuery)

	// Sort by final relevance score
	sort.Slice(rankedResults, func(i, j int) bool {
		return rankedResults[i].FinalRelevance > rankedResults[j].FinalRelevance
	})

	// Limit results
	if len(rankedResults) > rg.config.MaxFinalResults {
		rankedResults = rankedResults[:rg.config.MaxFinalResults]
	}

	// Convert to final search results
	finalResults := make([]interfaces.SearchResult, len(rankedResults))
	for i, ranked := range rankedResults {
		finalResults[i] = interfaces.SearchResult{
			Keybinding:  ranked.Keybinding,
			Relevance:   ranked.FinalRelevance,
			Explanation: ranked.Explanation,
		}
	}

	// Generate reasoning
	reasoning := rg.generateFinalReasoning(query, finalResults, llmAnalysis)

	log.Printf("Final ranking produced %d results", len(finalResults))
	return finalResults, reasoning, nil
}

// createRankedResult creates a ranked result from LLM recommendation
func (rg *ResponseGenerator) createRankedResult(
	llmResult ParsedKeybindingResult,
	vectorMap map[string]interfaces.VectorSearchResult,
	processedQuery *ProcessedQuery,
) *RankedResult {

	// Try to find matching vector result
	var vectorResult *interfaces.VectorSearchResult
	var vectorScore float64

	// First try exact key match
	if vr, exists := vectorMap[llmResult.Keys]; exists {
		vectorResult = &vr
		vectorScore = vr.Score
	} else {
		// Try fuzzy matching for similar keys
		for keys, vr := range vectorMap {
			if rg.keysAreSimilar(llmResult.Keys, keys) {
				vectorResult = &vr
				vectorScore = vr.Score * 0.9 // Slight penalty for fuzzy match
				break
			}
		}
	}

	// Create keybinding from LLM result
	keybinding := interfaces.Keybinding{
		Keys:        llmResult.Keys,
		Command:     llmResult.Command,
		Description: llmResult.Description,
		Mode:        llmResult.Mode,
		Metadata:    make(map[string]string),
	}

	// If we found a vector result, use its metadata
	if vectorResult != nil {
		keybinding.ID = getMetadataStringFromResult(*vectorResult, "keybinding_id")
		keybinding.Plugin = getMetadataStringFromResult(*vectorResult, "plugin")

		// Note: We can't easily iterate over all metadata with the v2 API
		// For now, we'll only copy the known fields
		keybinding.Metadata["source"] = getMetadataStringFromResult(*vectorResult, "source")
		keybinding.Metadata["mode"] = getMetadataStringFromResult(*vectorResult, "mode")
		keybinding.Metadata["description"] = getMetadataStringFromResult(*vectorResult, "description")
		keybinding.Metadata["command"] = getMetadataStringFromResult(*vectorResult, "command")
		keybinding.Metadata["keys"] = getMetadataStringFromResult(*vectorResult, "keys")
		keybinding.Metadata["plugin"] = getMetadataStringFromResult(*vectorResult, "plugin")
	} else {
		// Generate ID for LLM-only result
		keybinding.ID = fmt.Sprintf("llm_%s", strings.ReplaceAll(llmResult.Keys, " ", "_"))
		keybinding.Metadata["source"] = "llm_generated"
	}

	// Calculate ranking factors
	rankingFactors := make(map[string]float64)
	rankingFactors["llm_score"] = llmResult.LLMScore
	rankingFactors["llm_confidence"] = llmResult.Confidence
	rankingFactors["vector_score"] = vectorScore
	rankingFactors["rank_position"] = 1.0 - (float64(llmResult.Rank-1) * 0.1) // Higher rank = higher score

	// Add intent-based factors
	if processedQuery != nil && processedQuery.Intent != nil {
		rankingFactors["intent_confidence"] = processedQuery.Intent.Confidence

		// Boost if keybinding matches intent keywords
		for _, keyword := range processedQuery.Intent.Keywords {
			if strings.Contains(strings.ToLower(llmResult.Description), strings.ToLower(keyword)) ||
				strings.Contains(strings.ToLower(llmResult.Keys), strings.ToLower(keyword)) {
				rankingFactors["intent_keyword_match"] = 0.1
				break
			}
		}
	}

	return &RankedResult{
		Keybinding:     keybinding,
		VectorScore:    vectorScore,
		LLMScore:       llmResult.LLMScore,
		Explanation:    llmResult.Explanation,
		RankingFactors: rankingFactors,
		MatchedTerms:   rg.extractMatchedTerms(llmResult, processedQuery),
	}
}

// createRankedResultFromVector creates a ranked result from vector search only
func (rg *ResponseGenerator) createRankedResultFromVector(
	vectorResult interfaces.VectorSearchResult,
	query string,
	processedQuery *ProcessedQuery,
) RankedResult {

	keybinding := interfaces.Keybinding{
		ID:          getMetadataStringFromResult(vectorResult, "keybinding_id"),
		Keys:        getMetadataStringFromResult(vectorResult, "keys"),
		Command:     getMetadataStringFromResult(vectorResult, "command"),
		Description: getMetadataStringFromResult(vectorResult, "description"),
		Mode:        getMetadataStringFromResult(vectorResult, "mode"),
		Plugin:      getMetadataStringFromResult(vectorResult, "plugin"),
		Metadata:    make(map[string]string),
	}

	// Copy additional metadata (we can't iterate over DocumentMetadata with v2 API)
	// For now, we'll copy the known common fields
	keybinding.Metadata["source"] = getMetadataStringFromResult(vectorResult, "source")
	keybinding.Metadata["updated_at"] = getMetadataStringFromResult(vectorResult, "updated_at")
	keybinding.Metadata["vectorized_at"] = getMetadataStringFromResult(vectorResult, "vectorized_at")

	// Generate basic explanation
	explanation := rg.generateBasicExplanation(query, keybinding)

	// Calculate ranking factors
	rankingFactors := make(map[string]float64)
	rankingFactors["vector_score"] = vectorResult.Score
	rankingFactors["vector_only"] = -0.1 // Small penalty for not being mentioned by LLM

	return RankedResult{
		Keybinding:     keybinding,
		VectorScore:    vectorResult.Score,
		LLMScore:       0.0,
		Explanation:    explanation,
		RankingFactors: rankingFactors,
		MatchedTerms:   rg.extractMatchedTermsFromVector(vectorResult, processedQuery),
	}
}

// applyFinalRanking applies the final ranking algorithm to all results
func (rg *ResponseGenerator) applyFinalRanking(results []RankedResult, processedQuery *ProcessedQuery) {
	for i := range results {
		result := &results[i]

		// Base score combination (weighted average)
		vectorWeight := 0.4
		llmWeight := 0.6

		if result.LLMScore > 0 {
			// Both vector and LLM scores available
			result.FinalRelevance = (result.VectorScore * vectorWeight) + (result.LLMScore * llmWeight)
		} else {
			// Only vector score available
			result.FinalRelevance = result.VectorScore * 0.8 // Penalty for no LLM confirmation
		}

		// Apply ranking factors
		for factor, value := range result.RankingFactors {
			switch factor {
			case "intent_confidence":
				result.FinalRelevance += value * 0.1
			case "intent_keyword_match":
				result.FinalRelevance += value
			case "rank_position":
				result.FinalRelevance += value * 0.05
			case "vector_only":
				result.FinalRelevance += value
			}
		}

		// Apply user keybinding boost
		if source, exists := result.Keybinding.Metadata["source"]; exists && source == "user" {
			result.FinalRelevance += rg.config.UserBoostFactor
			result.RankingFactors["user_boost"] = rg.config.UserBoostFactor
		}

		// Apply query term matching boost
		if len(result.MatchedTerms) > 0 {
			termBoost := float64(len(result.MatchedTerms)) * 0.05
			result.FinalRelevance += termBoost
			result.RankingFactors["term_match_boost"] = termBoost
		}

		// Ensure relevance doesn't exceed 1.0
		if result.FinalRelevance > 1.0 {
			result.FinalRelevance = 1.0
		}
	}
}

// keysAreSimilar checks if two keybinding strings are similar
func (rg *ResponseGenerator) keysAreSimilar(keys1, keys2 string) bool {
	// Normalize keys for comparison
	normalize := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "<", "")
		s = strings.ReplaceAll(s, ">", "")
		s = strings.ReplaceAll(s, "-", "")
		s = strings.ReplaceAll(s, "_", "")
		s = strings.ReplaceAll(s, " ", "")
		return s
	}

	norm1 := normalize(keys1)
	norm2 := normalize(keys2)

	// Exact match after normalization
	if norm1 == norm2 {
		return true
	}

	// Check if one contains the other
	if strings.Contains(norm1, norm2) || strings.Contains(norm2, norm1) {
		return true
	}

	// Check for common abbreviations
	abbreviations := map[string][]string{
		"ctrl": {"c", "control"},
		"alt":  {"a", "meta"},
		"cmd":  {"d", "command"},
		"esc":  {"escape"},
		"ret":  {"return", "enter"},
		"tab":  {"t"},
		"spc":  {"space"},
	}

	for abbrev, expansions := range abbreviations {
		for _, expansion := range expansions {
			if (strings.Contains(norm1, abbrev) && strings.Contains(norm2, expansion)) ||
				(strings.Contains(norm1, expansion) && strings.Contains(norm2, abbrev)) {
				return true
			}
		}
	}

	return false
}

// extractMatchedTerms extracts terms that matched between query and result
func (rg *ResponseGenerator) extractMatchedTerms(llmResult ParsedKeybindingResult, processedQuery *ProcessedQuery) []string {
	var matchedTerms []string

	if processedQuery == nil {
		return matchedTerms
	}

	// Check against search terms
	for _, term := range processedQuery.SearchTerms {
		termLower := strings.ToLower(term)

		if strings.Contains(strings.ToLower(llmResult.Keys), termLower) ||
			strings.Contains(strings.ToLower(llmResult.Description), termLower) ||
			strings.Contains(strings.ToLower(llmResult.Command), termLower) {
			matchedTerms = append(matchedTerms, term)
		}
	}

	// Check against synonyms
	for _, synonym := range processedQuery.Synonyms {
		synonymLower := strings.ToLower(synonym)

		if strings.Contains(strings.ToLower(llmResult.Description), synonymLower) {
			matchedTerms = append(matchedTerms, synonym)
		}
	}

	return rg.removeDuplicateStrings(matchedTerms)
}

// extractMatchedTermsFromVector extracts matched terms from vector result
func (rg *ResponseGenerator) extractMatchedTermsFromVector(vectorResult interfaces.VectorSearchResult, processedQuery *ProcessedQuery) []string {
	var matchedTerms []string

	if processedQuery == nil {
		return matchedTerms
	}

	content := strings.ToLower(vectorResult.Document.Content)

	// Check against search terms
	for _, term := range processedQuery.SearchTerms {
		if strings.Contains(content, strings.ToLower(term)) {
			matchedTerms = append(matchedTerms, term)
		}
	}

	return rg.removeDuplicateStrings(matchedTerms)
}

// generateBasicExplanation generates a basic explanation for vector-only results
func (rg *ResponseGenerator) generateBasicExplanation(query string, keybinding interfaces.Keybinding) string {
	explanation := fmt.Sprintf("Keybinding '%s' matches your query", keybinding.Keys)

	if keybinding.Description != "" {
		explanation += fmt.Sprintf(": %s", keybinding.Description)
	}

	if keybinding.Mode != "" && keybinding.Mode != "normal" {
		explanation += fmt.Sprintf(" (in %s mode)", keybinding.Mode)
	}

	if source, exists := keybinding.Metadata["source"]; exists && source == "user" {
		explanation += " [User configured]"
	}

	return explanation
}

// generateFinalReasoning generates the final reasoning explanation
func (rg *ResponseGenerator) generateFinalReasoning(query string, results []interfaces.SearchResult, llmAnalysis *ResponseAnalysis) string {
	if len(results) == 0 {
		return "No relevant keybindings found for the given query."
	}

	reasoning := fmt.Sprintf("Found %d relevant keybinding(s) for query '%s'. ", len(results), query)

	// Count user vs builtin results
	userCount := 0
	builtinCount := 0

	for _, result := range results {
		if source, exists := result.Keybinding.Metadata["source"]; exists && source == "user" {
			userCount++
		} else {
			builtinCount++
		}
	}

	if userCount > 0 {
		reasoning += fmt.Sprintf("Includes %d user-configured keybinding(s). ", userCount)
	}

	if builtinCount > 0 {
		reasoning += fmt.Sprintf("Includes %d built-in vim keybinding(s). ", builtinCount)
	}

	// Add LLM reasoning if available
	if llmAnalysis != nil && llmAnalysis.Reasoning != "" {
		reasoning += fmt.Sprintf("Analysis: %s ", llmAnalysis.Reasoning)
	}

	reasoning += "Results are ranked by combining semantic similarity with expert analysis."

	return reasoning
}

// removeDuplicateStrings removes duplicate strings from slice
func (rg *ResponseGenerator) removeDuplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// getMetadataStringFromResult safely gets a string value from VectorSearchResult metadata
func getMetadataStringFromResult(result interfaces.VectorSearchResult, key string) string {
	if result.Document.Metadata == nil {
		return ""
	}
	if value, exists := result.Document.Metadata.GetString(key); exists {
		return value
	}
	return ""
}
