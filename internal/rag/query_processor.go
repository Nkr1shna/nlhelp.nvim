package rag

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"nvim-smart-keybind-search/internal/interfaces"
)

// QueryProcessor handles intelligent query understanding and processing
type QueryProcessor struct {
	llmClient       interfaces.LLMClient
	vectorDB        interfaces.VectorDB
	config          *QueryProcessorConfig
	synonymMap      map[string][]string
	intentPatterns  map[string]*regexp.Regexp
	contextBuilders map[string]ContextBuilder
}

// QueryProcessorConfig holds configuration for query processing
type QueryProcessorConfig struct {
	EnableIntentDetection bool
	EnableQueryExpansion  bool
	EnableContextBuilding bool
	MaxExpansionTerms     int
	ContextWindowSize     int
	SynonymBoostFactor    float64
	IntentBoostFactor     float64
}

// DefaultQueryProcessorConfig returns default configuration
func DefaultQueryProcessorConfig() *QueryProcessorConfig {
	return &QueryProcessorConfig{
		EnableIntentDetection: true,
		EnableQueryExpansion:  true,
		EnableContextBuilding: true,
		MaxExpansionTerms:     5,
		ContextWindowSize:     2000,
		SynonymBoostFactor:    0.1,
		IntentBoostFactor:     0.15,
	}
}

// QueryIntent represents the detected intent of a user query
type QueryIntent struct {
	Type        string  // "navigation", "editing", "visual", "search", "window", "buffer", etc.
	Confidence  float64 // 0.0 to 1.0
	Keywords    []string
	Context     map[string]string
	Suggestions []string // Alternative interpretations
}

// ProcessedQuery represents a query after intelligent processing
type ProcessedQuery struct {
	Original       string
	Expanded       string
	Intent         *QueryIntent
	Synonyms       []string
	Context        map[string]string
	SearchTerms    []string
	BoostFactors   map[string]float64
	ProcessingTime time.Duration
}

// ContextBuilder defines interface for building context from search results
type ContextBuilder interface {
	BuildContext(query string, results []interfaces.VectorSearchResult) string
	GetContextType() string
}

// NewQueryProcessor creates a new query processor
func NewQueryProcessor(llmClient interfaces.LLMClient, vectorDB interfaces.VectorDB, config *QueryProcessorConfig) *QueryProcessor {
	if config == nil {
		config = DefaultQueryProcessorConfig()
	}

	processor := &QueryProcessor{
		llmClient:       llmClient,
		vectorDB:        vectorDB,
		config:          config,
		synonymMap:      buildSynonymMap(),
		intentPatterns:  buildIntentPatterns(),
		contextBuilders: make(map[string]ContextBuilder),
	}

	// Initialize context builders
	processor.initializeContextBuilders()

	return processor
}

// ProcessQuery performs intelligent processing of a user query
func (qp *QueryProcessor) ProcessQuery(query string) (*ProcessedQuery, error) {
	start := time.Now()

	log.Printf("Processing query with intelligent understanding: %s", query)

	processed := &ProcessedQuery{
		Original:     query,
		Context:      make(map[string]string),
		BoostFactors: make(map[string]float64),
	}

	// Step 1: Detect intent
	if qp.config.EnableIntentDetection {
		intent, err := qp.detectIntent(query)
		if err != nil {
			log.Printf("Intent detection failed: %v", err)
		} else {
			processed.Intent = intent
			log.Printf("Detected intent: %s (confidence: %.2f)", intent.Type, intent.Confidence)
		}
	}

	// Step 2: Expand query with synonyms and variations
	if qp.config.EnableQueryExpansion {
		expanded, synonyms, err := qp.expandQuery(query, processed.Intent)
		if err != nil {
			log.Printf("Query expansion failed: %v", err)
			processed.Expanded = query
		} else {
			processed.Expanded = expanded
			processed.Synonyms = synonyms
			log.Printf("Expanded query: %s", expanded)
		}
	} else {
		processed.Expanded = query
	}

	// Step 3: Extract search terms
	processed.SearchTerms = qp.extractSearchTerms(processed.Expanded)

	// Step 4: Calculate boost factors based on intent and context
	processed.BoostFactors = qp.calculateBoostFactors(processed.Intent, processed.Synonyms)

	processed.ProcessingTime = time.Since(start)
	log.Printf("Query processing completed in %v", processed.ProcessingTime)

	return processed, nil
}

// detectIntent analyzes the query to determine user intent
func (qp *QueryProcessor) detectIntent(query string) (*QueryIntent, error) {
	// First, try pattern-based intent detection
	patternIntent := qp.detectIntentByPatterns(query)
	if patternIntent != nil && patternIntent.Confidence > 0.7 {
		return patternIntent, nil
	}

	// If pattern detection is not confident enough, use LLM
	llmIntent, err := qp.detectIntentByLLM(query)
	if err != nil {
		// Fallback to pattern-based result even if confidence is low
		if patternIntent != nil {
			return patternIntent, nil
		}
		return qp.createDefaultIntent(query), nil
	}

	// Combine pattern and LLM results for better accuracy
	if patternIntent != nil {
		return qp.combineIntentResults(patternIntent, llmIntent), nil
	}

	return llmIntent, nil
}

// detectIntentByPatterns uses regex patterns to detect intent
func (qp *QueryProcessor) detectIntentByPatterns(query string) *QueryIntent {
	queryLower := strings.ToLower(query)

	for intentType, pattern := range qp.intentPatterns {
		if pattern.MatchString(queryLower) {
			matches := pattern.FindStringSubmatch(queryLower)

			intent := &QueryIntent{
				Type:       intentType,
				Confidence: 0.8, // High confidence for pattern matches
				Keywords:   qp.extractKeywordsFromMatches(matches),
				Context:    make(map[string]string),
			}

			// Add context based on intent type
			qp.addIntentContext(intent, queryLower)

			return intent
		}
	}

	return nil
}

// detectIntentByLLM uses LLM to detect query intent
func (qp *QueryProcessor) detectIntentByLLM(query string) (*QueryIntent, error) {
	prompt := qp.buildIntentDetectionPrompt(query)

	llmRequest := interfaces.LLMRequest{
		Prompt:      prompt,
		MaxTokens:   200,
		Temperature: 0.1, // Low temperature for consistent intent detection
	}

	response, err := qp.llmClient.Generate(llmRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to detect intent via LLM: %w", err)
	}

	intent, err := qp.parseIntentFromLLMResponse(response.Text, query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intent from LLM response: %w", err)
	}

	return intent, nil
}

// buildIntentDetectionPrompt creates a prompt for intent detection
func (qp *QueryProcessor) buildIntentDetectionPrompt(query string) string {
	return fmt.Sprintf(`You are a vim/neovim expert. Analyze the following user query and determine their intent.

Query: "%s"

Classify the intent into one of these categories:
- navigation: Moving cursor, jumping to locations
- editing: Inserting, deleting, changing text
- visual: Visual mode operations, selections
- search: Finding text, patterns, files
- window: Window management, splits, tabs
- buffer: Buffer operations, switching files
- command: Ex commands, configuration
- motion: Text objects, movements
- macro: Recording, playing macros
- plugin: Plugin-specific functionality

Provide your analysis in this format:
Intent: [category]
Confidence: [0.0-1.0]
Keywords: [key terms from query]
Context: [relevant context or clarification]
Suggestions: [alternative interpretations if ambiguous]

Analysis:`, query)
}

// parseIntentFromLLMResponse parses intent information from LLM response
func (qp *QueryProcessor) parseIntentFromLLMResponse(response, query string) (*QueryIntent, error) {
	lines := strings.Split(response, "\n")

	intent := &QueryIntent{
		Context: make(map[string]string),
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "Intent:") {
			intent.Type = strings.TrimSpace(strings.TrimPrefix(line, "Intent:"))
		} else if strings.HasPrefix(line, "Confidence:") {
			confidenceStr := strings.TrimSpace(strings.TrimPrefix(line, "Confidence:"))
			if confidence, err := parseFloat(confidenceStr); err == nil {
				intent.Confidence = confidence
			}
		} else if strings.HasPrefix(line, "Keywords:") {
			keywordsStr := strings.TrimSpace(strings.TrimPrefix(line, "Keywords:"))
			intent.Keywords = strings.Split(keywordsStr, ",")
			for i := range intent.Keywords {
				intent.Keywords[i] = strings.TrimSpace(intent.Keywords[i])
			}
		} else if strings.HasPrefix(line, "Context:") {
			contextStr := strings.TrimSpace(strings.TrimPrefix(line, "Context:"))
			intent.Context["llm_context"] = contextStr
		} else if strings.HasPrefix(line, "Suggestions:") {
			suggestionsStr := strings.TrimSpace(strings.TrimPrefix(line, "Suggestions:"))
			intent.Suggestions = strings.Split(suggestionsStr, ",")
			for i := range intent.Suggestions {
				intent.Suggestions[i] = strings.TrimSpace(intent.Suggestions[i])
			}
		}
	}

	// Validate parsed intent
	if intent.Type == "" {
		return qp.createDefaultIntent(query), nil
	}

	// Set default confidence if not parsed
	if intent.Confidence == 0 {
		intent.Confidence = 0.6
	}

	return intent, nil
}

// expandQuery expands the query with synonyms and related terms
func (qp *QueryProcessor) expandQuery(query string, intent *QueryIntent) (string, []string, error) {
	var allSynonyms []string

	// Get synonyms from built-in map
	builtinSynonyms := qp.getSynonymsFromMap(query)
	allSynonyms = append(allSynonyms, builtinSynonyms...)

	// Get context-aware synonyms based on intent
	if intent != nil {
		intentSynonyms := qp.getIntentBasedSynonyms(query, intent)
		allSynonyms = append(allSynonyms, intentSynonyms...)
	}

	// Use LLM for additional expansion
	llmSynonyms, err := qp.getLLMBasedExpansion(query, intent)
	if err != nil {
		log.Printf("LLM-based expansion failed: %v", err)
	} else {
		allSynonyms = append(allSynonyms, llmSynonyms...)
	}

	// Remove duplicates and limit expansion
	uniqueSynonyms := qp.removeDuplicates(allSynonyms)
	if len(uniqueSynonyms) > qp.config.MaxExpansionTerms {
		uniqueSynonyms = uniqueSynonyms[:qp.config.MaxExpansionTerms]
	}

	// Build expanded query
	expanded := query
	if len(uniqueSynonyms) > 0 {
		expanded = fmt.Sprintf("%s %s", query, strings.Join(uniqueSynonyms, " "))
	}

	return expanded, uniqueSynonyms, nil
}

// getSynonymsFromMap gets synonyms from the built-in synonym map
func (qp *QueryProcessor) getSynonymsFromMap(query string) []string {
	var synonyms []string
	queryWords := strings.Fields(strings.ToLower(query))

	for _, word := range queryWords {
		if syns, exists := qp.synonymMap[word]; exists {
			synonyms = append(synonyms, syns...)
		}
	}

	return synonyms
}

// getIntentBasedSynonyms gets synonyms based on detected intent
func (qp *QueryProcessor) getIntentBasedSynonyms(query string, intent *QueryIntent) []string {
	var synonyms []string

	switch intent.Type {
	case "navigation":
		synonyms = append(synonyms, "move", "jump", "go", "cursor", "position")
	case "editing":
		synonyms = append(synonyms, "change", "modify", "insert", "delete", "edit", "text")
	case "visual":
		synonyms = append(synonyms, "select", "highlight", "mark", "block", "region")
	case "search":
		synonyms = append(synonyms, "find", "locate", "pattern", "match", "grep")
	case "window":
		synonyms = append(synonyms, "split", "pane", "tab", "layout", "view")
	case "buffer":
		synonyms = append(synonyms, "file", "document", "switch", "open", "close")
	}

	return synonyms
}

// getLLMBasedExpansion uses LLM to expand query terms
func (qp *QueryProcessor) getLLMBasedExpansion(query string, intent *QueryIntent) ([]string, error) {
	prompt := qp.buildExpansionPrompt(query, intent)

	llmRequest := interfaces.LLMRequest{
		Prompt:      prompt,
		MaxTokens:   100,
		Temperature: 0.3,
	}

	response, err := qp.llmClient.Generate(llmRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM expansion: %w", err)
	}

	return qp.parseExpansionTerms(response.Text), nil
}

// buildExpansionPrompt creates a prompt for query expansion
func (qp *QueryProcessor) buildExpansionPrompt(query string, intent *QueryIntent) string {
	intentContext := ""
	if intent != nil {
		intentContext = fmt.Sprintf(" The user's intent appears to be: %s.", intent.Type)
	}

	return fmt.Sprintf(`You are a vim/neovim expert. Expand the following query with related terms and synonyms that would help find relevant keybindings.%s

Original query: "%s"

Provide 3-5 related terms or synonyms that would improve search results. Focus on:
- Alternative vim terminology
- Related commands or motions
- Common variations of the request

Return only the expansion terms, separated by commas:`, intentContext, query)
}

// parseExpansionTerms parses expansion terms from LLM response
func (qp *QueryProcessor) parseExpansionTerms(response string) []string {
	// Clean up response
	response = strings.TrimSpace(response)

	// Remove common prefixes
	prefixes := []string{"Terms:", "Expansion:", "Related terms:"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(response, prefix) {
			response = strings.TrimSpace(strings.TrimPrefix(response, prefix))
		}
	}

	// Split by commas and clean up
	terms := strings.Split(response, ",")
	var cleanTerms []string

	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term != "" && len(term) > 1 {
			cleanTerms = append(cleanTerms, term)
		}
	}

	return cleanTerms
}

// BuildContextFromResults builds context for LLM from search results
func (qp *QueryProcessor) BuildContextFromResults(query string, results []interfaces.VectorSearchResult, intent *QueryIntent) string {
	if !qp.config.EnableContextBuilding {
		return qp.buildBasicContext(query, results)
	}

	// Use intent-specific context builder if available
	if intent != nil {
		if builder, exists := qp.contextBuilders[intent.Type]; exists {
			return builder.BuildContext(query, results)
		}
	}

	// Use enhanced context building
	return qp.buildEnhancedContext(query, results, intent)
}

// buildBasicContext builds basic context from search results
func (qp *QueryProcessor) buildBasicContext(query string, results []interfaces.VectorSearchResult) string {
	var contextParts []string

	contextParts = append(contextParts, fmt.Sprintf("User query: %s", query))
	contextParts = append(contextParts, "\nRelevant keybindings:")

	for i, result := range results {
		if i >= 8 { // Limit context size
			break
		}

		keys := getMetadataString(result.Document, "keys")
		command := getMetadataString(result.Document, "command")
		description := getMetadataString(result.Document, "description")

		contextPart := fmt.Sprintf("\n%d. %s", i+1, keys)
		if command != "" {
			contextPart += fmt.Sprintf(" (%s)", command)
		}
		if description != "" {
			contextPart += fmt.Sprintf(": %s", description)
		}

		contextParts = append(contextParts, contextPart)
	}

	return strings.Join(contextParts, "")
}

// buildEnhancedContext builds enhanced context with intent awareness
func (qp *QueryProcessor) buildEnhancedContext(query string, results []interfaces.VectorSearchResult, intent *QueryIntent) string {
	var contextParts []string

	// Add query and intent information
	contextParts = append(contextParts, fmt.Sprintf("User query: %s", query))

	if intent != nil {
		contextParts = append(contextParts, fmt.Sprintf("Detected intent: %s (confidence: %.2f)", intent.Type, intent.Confidence))
		if len(intent.Keywords) > 0 {
			contextParts = append(contextParts, fmt.Sprintf("Key terms: %s", strings.Join(intent.Keywords, ", ")))
		}
	}

	// Group results by type and source
	userKeybindings := []interfaces.VectorSearchResult{}
	builtinKeybindings := []interfaces.VectorSearchResult{}
	generalKnowledge := []interfaces.VectorSearchResult{}

	for _, result := range results {
		docType := getMetadataString(result.Document, "type")
		source := getMetadataString(result.Document, "source")

		if docType == "general_knowledge" {
			generalKnowledge = append(generalKnowledge, result)
		} else if source == "user" {
			userKeybindings = append(userKeybindings, result)
		} else {
			builtinKeybindings = append(builtinKeybindings, result)
		}
	}

	// Add user keybindings first (highest priority)
	if len(userKeybindings) > 0 {
		contextParts = append(contextParts, "\n\nUser-configured keybindings:")
		for i, result := range userKeybindings {
			if i >= 5 { // Limit user results
				break
			}
			contextParts = append(contextParts, qp.formatResultForContext(result, i+1))
		}
	}

	// Add built-in keybindings
	if len(builtinKeybindings) > 0 {
		contextParts = append(contextParts, "\n\nBuilt-in vim keybindings:")
		maxBuiltin := 5
		if len(userKeybindings) == 0 {
			maxBuiltin = 8 // More built-in results if no user results
		}

		for i, result := range builtinKeybindings {
			if i >= maxBuiltin {
				break
			}
			contextParts = append(contextParts, qp.formatResultForContext(result, i+1))
		}
	}

	// Add general knowledge
	if len(generalKnowledge) > 0 {
		contextParts = append(contextParts, "\n\nGeneral knowledge and concepts:")
		maxGeneral := 3 // Limit general knowledge to avoid overwhelming context

		for i, result := range generalKnowledge {
			if i >= maxGeneral {
				break
			}
			contextParts = append(contextParts, qp.formatGeneralKnowledgeForContext(result, i+1))
		}
	}

	context := strings.Join(contextParts, "")

	// Truncate if too long
	if len(context) > qp.config.ContextWindowSize {
		context = context[:qp.config.ContextWindowSize] + "..."
	}

	return context
}

// formatResultForContext formats a search result for context inclusion
func (qp *QueryProcessor) formatResultForContext(result interfaces.VectorSearchResult, index int) string {
	keys := getMetadataString(result.Document, "keys")
	command := getMetadataString(result.Document, "command")
	description := getMetadataString(result.Document, "description")
	mode := getMetadataString(result.Document, "mode")

	contextPart := fmt.Sprintf("\n%d. %s", index, keys)
	if command != "" {
		contextPart += fmt.Sprintf(" (%s)", command)
	}
	if description != "" {
		contextPart += fmt.Sprintf(": %s", description)
	}
	if mode != "" && mode != "n" {
		contextPart += fmt.Sprintf(" [%s mode]", mode)
	}

	contextPart += fmt.Sprintf(", Relevance: %.2f", result.Score)

	return contextPart
}

// formatGeneralKnowledgeForContext formats a general knowledge result for context inclusion
func (qp *QueryProcessor) formatGeneralKnowledgeForContext(result interfaces.VectorSearchResult, index int) string {
	title := getMetadataString(result.Document, "title")
	content := getMetadataString(result.Document, "content")
	category := getMetadataString(result.Document, "category")
	difficulty := getMetadataString(result.Document, "difficulty")

	contextPart := fmt.Sprintf("\n%d. %s", index, title)
	if content != "" {
		// Truncate content if too long
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		contextPart += fmt.Sprintf(": %s", content)
	}
	if category != "" {
		contextPart += fmt.Sprintf(" [%s]", category)
	}
	if difficulty != "" {
		contextPart += fmt.Sprintf(" (%s level)", difficulty)
	}

	contextPart += fmt.Sprintf(", Relevance: %.2f", result.Score)

	return contextPart
}

// extractSearchTerms extracts individual search terms from expanded query
func (qp *QueryProcessor) extractSearchTerms(query string) []string {
	// Simple word extraction - could be enhanced with NLP
	words := strings.Fields(strings.ToLower(query))

	// Filter out common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "how": true,
		"what": true, "where": true, "when": true, "why": true, "i": true,
		"want": true, "need": true, "can": true, "should": true,
	}

	var terms []string
	for _, word := range words {
		if !stopWords[word] && len(word) > 1 {
			terms = append(terms, word)
		}
	}

	return terms
}

// calculateBoostFactors calculates boost factors based on intent and synonyms
func (qp *QueryProcessor) calculateBoostFactors(intent *QueryIntent, synonyms []string) map[string]float64 {
	boostFactors := make(map[string]float64)

	// Intent-based boosts
	if intent != nil {
		boostFactors["intent_"+intent.Type] = qp.config.IntentBoostFactor * intent.Confidence

		// Boost for high-confidence intent detection
		if intent.Confidence > 0.8 {
			boostFactors["high_confidence_intent"] = 0.1
		}
	}

	// Synonym-based boosts
	if len(synonyms) > 0 {
		boostFactors["synonym_expansion"] = qp.config.SynonymBoostFactor
	}

	return boostFactors
}

// Helper functions

// createDefaultIntent creates a default intent when detection fails
func (qp *QueryProcessor) createDefaultIntent(query string) *QueryIntent {
	return &QueryIntent{
		Type:       "general",
		Confidence: 0.3,
		Keywords:   strings.Fields(strings.ToLower(query)),
		Context:    make(map[string]string),
	}
}

// combineIntentResults combines pattern-based and LLM-based intent results
func (qp *QueryProcessor) combineIntentResults(patternIntent, llmIntent *QueryIntent) *QueryIntent {
	// If both agree on intent type, increase confidence
	if patternIntent.Type == llmIntent.Type {
		combined := *llmIntent
		combined.Confidence = (patternIntent.Confidence + llmIntent.Confidence) / 2
		if combined.Confidence > 1.0 {
			combined.Confidence = 1.0
		}
		return &combined
	}

	// If they disagree, prefer the one with higher confidence
	if patternIntent.Confidence > llmIntent.Confidence {
		return patternIntent
	}
	return llmIntent
}

// extractKeywordsFromMatches extracts keywords from regex matches
func (qp *QueryProcessor) extractKeywordsFromMatches(matches []string) []string {
	var keywords []string
	for i, match := range matches {
		if i > 0 && match != "" { // Skip full match (index 0)
			keywords = append(keywords, match)
		}
	}
	return keywords
}

// addIntentContext adds context information based on intent type
func (qp *QueryProcessor) addIntentContext(intent *QueryIntent, query string) {
	switch intent.Type {
	case "navigation":
		intent.Context["category"] = "cursor_movement"
	case "editing":
		intent.Context["category"] = "text_manipulation"
	case "visual":
		intent.Context["category"] = "selection"
	case "search":
		intent.Context["category"] = "finding"
	case "window":
		intent.Context["category"] = "layout"
	case "buffer":
		intent.Context["category"] = "file_management"
	}
}

// removeDuplicates removes duplicate strings from slice
func (qp *QueryProcessor) removeDuplicates(slice []string) []string {
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

// parseFloat safely parses a float from string
func parseFloat(s string) (float64, error) {
	// Simple float parsing - could use strconv.ParseFloat for production
	if s == "1.0" || s == "1" {
		return 1.0, nil
	} else if s == "0.9" {
		return 0.9, nil
	} else if s == "0.8" {
		return 0.8, nil
	} else if s == "0.7" {
		return 0.7, nil
	} else if s == "0.6" {
		return 0.6, nil
	} else if s == "0.5" {
		return 0.5, nil
	}
	return 0.6, nil // Default confidence
}

// initializeContextBuilders initializes intent-specific context builders
func (qp *QueryProcessor) initializeContextBuilders() {
	qp.contextBuilders["navigation"] = &NavigationContextBuilder{}
	qp.contextBuilders["editing"] = &EditingContextBuilder{}
	qp.contextBuilders["visual"] = &VisualContextBuilder{}
	qp.contextBuilders["search"] = &SearchContextBuilder{}
	qp.contextBuilders["window"] = &WindowContextBuilder{}
	qp.contextBuilders["buffer"] = &BufferContextBuilder{}
}

// buildSynonymMap builds the built-in synonym map
func buildSynonymMap() map[string][]string {
	return map[string][]string{
		"delete":   {"remove", "cut", "erase", "clear"},
		"copy":     {"yank", "duplicate"},
		"paste":    {"put", "insert"},
		"move":     {"navigate", "jump", "go"},
		"search":   {"find", "locate", "grep"},
		"replace":  {"substitute", "change"},
		"word":     {"term", "token"},
		"line":     {"row"},
		"column":   {"col"},
		"start":    {"beginning", "begin"},
		"end":      {"finish", "last"},
		"next":     {"forward"},
		"previous": {"back", "backward", "prev"},
		"up":       {"above"},
		"down":     {"below"},
		"left":     {"backward"},
		"right":    {"forward"},
		"select":   {"highlight", "mark"},
		"split":    {"divide", "separate"},
		"close":    {"quit", "exit"},
		"open":     {"load", "edit"},
		"save":     {"write", "store"},
		"undo":     {"revert", "back"},
		"redo":     {"forward", "repeat"},
	}
}

// buildIntentPatterns builds regex patterns for intent detection
func buildIntentPatterns() map[string]*regexp.Regexp {
	patterns := map[string]*regexp.Regexp{
		"navigation": regexp.MustCompile(`(?i)(move|jump|go|navigate|cursor|position).*(to|at|beginning|end|start|line|word|character|top|bottom)`),
		"editing":    regexp.MustCompile(`(?i)(delete|remove|cut|insert|add|change|replace|edit|modify|substitute).*(word|line|character|text|block)`),
		"visual":     regexp.MustCompile(`(?i)(select|highlight|mark|visual|block|region).*(text|word|line|paragraph)`),
		"search":     regexp.MustCompile(`(?i)(search|find|locate|grep|pattern|match).*(text|word|string|file)`),
		"window":     regexp.MustCompile(`(?i)(window|split|pane|tab|layout|view).*(open|close|switch|move|resize)`),
		"buffer":     regexp.MustCompile(`(?i)(buffer|file|document).*(open|close|switch|save|load|edit)`),
		"command":    regexp.MustCompile(`(?i)(command|ex|colon).*(mode|execute|run)`),
		"motion":     regexp.MustCompile(`(?i)(motion|movement|text object).*(word|sentence|paragraph|block)`),
		"macro":      regexp.MustCompile(`(?i)(macro|record|replay|repeat).*(command|sequence)`),
	}

	return patterns
}

// Context builder implementations

// NavigationContextBuilder builds context for navigation queries
type NavigationContextBuilder struct{}

func (ncb *NavigationContextBuilder) BuildContext(query string, results []interfaces.VectorSearchResult) string {
	context := fmt.Sprintf("Navigation query: %s\n\nMovement keybindings:", query)

	for i, result := range results {
		if i >= 6 {
			break
		}

		keys := getMetadataString(result.Document, "keys")
		description := getMetadataString(result.Document, "description")
		mode := getMetadataString(result.Document, "mode")

		context += fmt.Sprintf("\n%d. %s", i+1, keys)
		if description != "" {
			context += fmt.Sprintf(": %s", description)
		}
		if mode != "" && mode != "normal" {
			context += fmt.Sprintf(" (%s mode)", mode)
		}
	}

	return context
}

func (ncb *NavigationContextBuilder) GetContextType() string {
	return "navigation"
}

// EditingContextBuilder builds context for editing queries
type EditingContextBuilder struct{}

func (ecb *EditingContextBuilder) BuildContext(query string, results []interfaces.VectorSearchResult) string {
	context := fmt.Sprintf("Text editing query: %s\n\nEditing keybindings:", query)

	for i, result := range results {
		if i >= 6 {
			break
		}

		keys := getMetadataString(result.Document, "keys")
		command := getMetadataString(result.Document, "command")
		description := getMetadataString(result.Document, "description")

		context += fmt.Sprintf("\n%d. %s", i+1, keys)
		if command != "" {
			context += fmt.Sprintf(" (%s)", command)
		}
		if description != "" {
			context += fmt.Sprintf(": %s", description)
		}
	}

	return context
}

func (ecb *EditingContextBuilder) GetContextType() string {
	return "editing"
}

// VisualContextBuilder builds context for visual mode queries
type VisualContextBuilder struct{}

func (vcb *VisualContextBuilder) BuildContext(query string, results []interfaces.VectorSearchResult) string {
	context := fmt.Sprintf("Visual selection query: %s\n\nVisual mode keybindings:", query)

	for i, result := range results {
		if i >= 6 {
			break
		}

		keys := getMetadataString(result.Document, "keys")
		description := getMetadataString(result.Document, "description")
		mode := getMetadataString(result.Document, "mode")

		context += fmt.Sprintf("\n%d. %s", i+1, keys)
		if description != "" {
			context += fmt.Sprintf(": %s", description)
		}
		if mode == "visual" {
			context += " [Visual mode]"
		}
	}

	return context
}

func (vcb *VisualContextBuilder) GetContextType() string {
	return "visual"
}

// SearchContextBuilder builds context for search queries
type SearchContextBuilder struct{}

func (scb *SearchContextBuilder) BuildContext(query string, results []interfaces.VectorSearchResult) string {
	context := fmt.Sprintf("Search/find query: %s\n\nSearch keybindings:", query)

	for i, result := range results {
		if i >= 6 {
			break
		}

		keys := getMetadataString(result.Document, "keys")
		command := getMetadataString(result.Document, "command")
		description := getMetadataString(result.Document, "description")

		context += fmt.Sprintf("\n%d. %s", i+1, keys)
		if command != "" {
			context += fmt.Sprintf(" (%s)", command)
		}
		if description != "" {
			context += fmt.Sprintf(": %s", description)
		}
	}

	return context
}

func (scb *SearchContextBuilder) GetContextType() string {
	return "search"
}

// WindowContextBuilder builds context for window management queries
type WindowContextBuilder struct{}

func (wcb *WindowContextBuilder) BuildContext(query string, results []interfaces.VectorSearchResult) string {
	context := fmt.Sprintf("Window management query: %s\n\nWindow keybindings:", query)

	for i, result := range results {
		if i >= 6 {
			break
		}

		keys := getMetadataString(result.Document, "keys")
		description := getMetadataString(result.Document, "description")

		context += fmt.Sprintf("\n%d. %s", i+1, keys)
		if description != "" {
			context += fmt.Sprintf(": %s", description)
		}
	}

	return context
}

func (wcb *WindowContextBuilder) GetContextType() string {
	return "window"
}

// BufferContextBuilder builds context for buffer management queries
type BufferContextBuilder struct{}

func (bcb *BufferContextBuilder) BuildContext(query string, results []interfaces.VectorSearchResult) string {
	context := fmt.Sprintf("Buffer management query: %s\n\nBuffer keybindings:", query)

	for i, result := range results {
		if i >= 6 {
			break
		}

		keys := getMetadataString(result.Document, "keys")
		command := getMetadataString(result.Document, "command")
		description := getMetadataString(result.Document, "description")

		context += fmt.Sprintf("\n%d. %s", i+1, keys)
		if command != "" {
			context += fmt.Sprintf(" (%s)", command)
		}
		if description != "" {
			context += fmt.Sprintf(": %s", description)
		}
	}

	return context
}

func (bcb *BufferContextBuilder) GetContextType() string {
	return "buffer"
}

// getMetadataString safely gets a string value from DocumentMetadata
func getMetadataString(metadata interfaces.Document, key string) string {
	if metadata.Metadata == nil {
		return ""
	}
	if value, exists := metadata.Metadata.GetString(key); exists {
		return value
	}
	return ""
}
