package bm25

import (
	"regexp"
	"strings"
	"unicode"
)

// SimpleTokenizer implements basic tokenization for BM25
type SimpleTokenizer struct {
	config     *TokenizerConfig
	stopWords  map[string]bool
	stemmer    Stemmer
}

// NewSimpleTokenizer creates a new simple tokenizer
func NewSimpleTokenizer(config *TokenizerConfig) *SimpleTokenizer {
	if config == nil {
		config = DefaultTokenizerConfig()
	}
	
	tokenizer := &SimpleTokenizer{
		config:    config,
		stopWords: make(map[string]bool),
	}
	
	// Load stopwords
	if config.RemoveStops {
		tokenizer.loadStopWords()
	}
	
	// Initialize stemmer if needed
	if config.Stemming {
		tokenizer.stemmer = NewPorterStemmer()
	}
	
	return tokenizer
}

// DefaultTokenizerConfig returns default tokenizer configuration
func DefaultTokenizerConfig() *TokenizerConfig {
	return &TokenizerConfig{
		Language:      "english",
		RemoveStops:   true,
		MinTermLen:    2,
		MaxTermLen:    50,
		Stemming:      false,
		CaseSensitive: false,
		CustomStops:   []string{},
	}
}

// Tokenize splits text into terms/tokens
func (t *SimpleTokenizer) Tokenize(text string) []string {
	// Basic preprocessing
	if !t.config.CaseSensitive {
		text = strings.ToLower(text)
	}
	
	// Split into words using regex
	wordRegex := regexp.MustCompile(`\b\w+\b`)
	words := wordRegex.FindAllString(text, -1)
	
	var tokens []string
	for _, word := range words {
		// Apply length filters
		if len(word) < t.config.MinTermLen || len(word) > t.config.MaxTermLen {
			continue
		}
		
		// Remove stopwords
		if t.config.RemoveStops && t.stopWords[word] {
			continue
		}
		
		// Apply stemming
		if t.config.Stemming && t.stemmer != nil {
			word = t.stemmer.Stem(word)
		}
		
		tokens = append(tokens, word)
	}
	
	return tokens
}

// TokenizeWithPositions returns tokens with their positions
func (t *SimpleTokenizer) TokenizeWithPositions(text string) []TokenPosition {
	if !t.config.CaseSensitive {
		text = strings.ToLower(text)
	}
	
	wordRegex := regexp.MustCompile(`\b\w+\b`)
	matches := wordRegex.FindAllStringIndex(text, -1)
	
	var positions []TokenPosition
	position := 0
	
	for _, match := range matches {
		start, end := match[0], match[1]
		word := text[start:end]
		
		// Apply filters
		if len(word) < t.config.MinTermLen || len(word) > t.config.MaxTermLen {
			continue
		}
		
		if t.config.RemoveStops && t.stopWords[word] {
			continue
		}
		
		// Apply stemming
		if t.config.Stemming && t.stemmer != nil {
			word = t.stemmer.Stem(word)
		}
		
		tokenPos := TokenPosition{
			Term:     word,
			Start:    start,
			End:      end,
			Position: position,
		}
		
		positions = append(positions, tokenPos)
		position++
	}
	
	return positions
}

// NormalizeQuery normalizes a search query
func (t *SimpleTokenizer) NormalizeQuery(query string) string {
	if !t.config.CaseSensitive {
		query = strings.ToLower(query)
	}
	
	// Remove extra whitespace
	query = strings.TrimSpace(query)
	words := strings.Fields(query)
	
	var normalizedWords []string
	for _, word := range words {
		// Clean word
		cleanWord := t.cleanWord(word)
		if cleanWord != "" {
			normalizedWords = append(normalizedWords, cleanWord)
		}
	}
	
	return strings.Join(normalizedWords, " ")
}

// cleanWord removes punctuation and applies basic cleaning
func (t *SimpleTokenizer) cleanWord(word string) string {
	// Remove leading/trailing punctuation
	word = strings.TrimFunc(word, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	
	// Apply length filter
	if len(word) < t.config.MinTermLen || len(word) > t.config.MaxTermLen {
		return ""
	}
	
	return word
}

// loadStopWords loads stopwords for the configured language
func (t *SimpleTokenizer) loadStopWords() {
	// English stopwords
	englishStops := []string{
		"a", "an", "and", "are", "as", "at", "be", "by", "for", "from",
		"has", "he", "in", "is", "it", "its", "of", "on", "that", "the",
		"to", "was", "will", "with", "the", "this", "but", "they", "have",
		"had", "what", "said", "each", "which", "she", "do", "how", "their",
		"if", "up", "out", "many", "then", "them", "these", "so", "some",
		"her", "would", "make", "like", "into", "him", "time", "two", "more",
		"go", "no", "way", "could", "my", "than", "first", "been", "call",
		"who", "oil", "sit", "now", "find", "down", "day", "did", "get",
		"come", "made", "may", "part",
	}
	
	// Add standard stopwords
	for _, word := range englishStops {
		t.stopWords[word] = true
	}
	
	// Add custom stopwords
	for _, word := range t.config.CustomStops {
		if !t.config.CaseSensitive {
			word = strings.ToLower(word)
		}
		t.stopWords[word] = true
	}
}

// Stemmer defines interface for word stemming
type Stemmer interface {
	Stem(word string) string
}

// PorterStemmer implements Porter stemming algorithm (simplified)
type PorterStemmer struct{}

// NewPorterStemmer creates a new Porter stemmer
func NewPorterStemmer() *PorterStemmer {
	return &PorterStemmer{}
}

// Stem applies Porter stemming to a word (simplified implementation)
func (s *PorterStemmer) Stem(word string) string {
	if len(word) <= 2 {
		return word
	}
	
	// Step 1a
	if strings.HasSuffix(word, "sses") {
		word = word[:len(word)-2] // sses -> ss
	} else if strings.HasSuffix(word, "ies") {
		word = word[:len(word)-2] // ies -> i
	} else if strings.HasSuffix(word, "ss") {
		// ss -> ss (no change)
	} else if strings.HasSuffix(word, "s") && len(word) > 1 {
		word = word[:len(word)-1] // s -> (empty)
	}
	
	// Step 1b (simplified)
	if strings.HasSuffix(word, "ed") {
		if len(word) > 4 {
			word = word[:len(word)-2]
		}
	} else if strings.HasSuffix(word, "ing") {
		if len(word) > 5 {
			word = word[:len(word)-3]
		}
	}
	
	// Step 2 (simplified)
	if strings.HasSuffix(word, "ly") && len(word) > 4 {
		word = word[:len(word)-2]
	}
	
	return word
}

// AdvancedTokenizer provides more sophisticated tokenization
type AdvancedTokenizer struct {
	simple    *SimpleTokenizer
	config    *TokenizerConfig
	patterns  []*regexp.Regexp
}

// NewAdvancedTokenizer creates a new advanced tokenizer
func NewAdvancedTokenizer(config *TokenizerConfig) *AdvancedTokenizer {
	if config == nil {
		config = DefaultTokenizerConfig()
	}
	
	tokenizer := &AdvancedTokenizer{
		simple: NewSimpleTokenizer(config),
		config: config,
	}
	
	// Initialize regex patterns
	tokenizer.initPatterns()
	
	return tokenizer
}

// initPatterns initializes regex patterns for advanced tokenization
func (t *AdvancedTokenizer) initPatterns() {
	// Pattern for URLs
	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	
	// Pattern for email addresses
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	
	// Pattern for numbers with decimals
	numberPattern := regexp.MustCompile(`\d+\.?\d*`)
	
	// Pattern for hyphenated words
	hyphenPattern := regexp.MustCompile(`\b\w+-\w+\b`)
	
	t.patterns = []*regexp.Regexp{
		urlPattern,
		emailPattern,
		numberPattern,
		hyphenPattern,
	}
}

// Tokenize performs advanced tokenization
func (t *AdvancedTokenizer) Tokenize(text string) []string {
	// First, extract special patterns
	specialTokens := t.extractSpecialTokens(text)
	
	// Then apply simple tokenization to the rest
	simpleTokens := t.simple.Tokenize(text)
	
	// Combine results
	allTokens := append(specialTokens, simpleTokens...)
	
	// Remove duplicates while preserving order
	return t.removeDuplicates(allTokens)
}

// extractSpecialTokens extracts special tokens like URLs, emails, etc.
func (t *AdvancedTokenizer) extractSpecialTokens(text string) []string {
	var tokens []string
	
	for _, pattern := range t.patterns {
		matches := pattern.FindAllString(text, -1)
		for _, match := range matches {
			if len(match) >= t.config.MinTermLen && len(match) <= t.config.MaxTermLen {
				if !t.config.CaseSensitive {
					match = strings.ToLower(match)
				}
				tokens = append(tokens, match)
			}
		}
	}
	
	return tokens
}

// removeDuplicates removes duplicate tokens while preserving order
func (t *AdvancedTokenizer) removeDuplicates(tokens []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, token := range tokens {
		if !seen[token] {
			seen[token] = true
			result = append(result, token)
		}
	}
	
	return result
}

// TokenizeWithPositions returns tokens with positions (delegates to simple tokenizer)
func (t *AdvancedTokenizer) TokenizeWithPositions(text string) []TokenPosition {
	return t.simple.TokenizeWithPositions(text)
}

// NormalizeQuery normalizes query (delegates to simple tokenizer)
func (t *AdvancedTokenizer) NormalizeQuery(query string) string {
	return t.simple.NormalizeQuery(query)
}

// TokenAnalyzer provides token analysis utilities
type TokenAnalyzer struct {
	tokenizer Tokenizer
}

// NewTokenAnalyzer creates a new token analyzer
func NewTokenAnalyzer(tokenizer Tokenizer) *TokenAnalyzer {
	return &TokenAnalyzer{
		tokenizer: tokenizer,
	}
}

// AnalyzeText provides detailed analysis of text tokenization
func (a *TokenAnalyzer) AnalyzeText(text string) *TokenAnalysis {
	tokens := a.tokenizer.Tokenize(text)
	positions := a.tokenizer.TokenizeWithPositions(text)
	
	analysis := &TokenAnalysis{
		OriginalText: text,
		TokenCount:   len(tokens),
		UniqueTokens: len(a.getUniqueTokens(tokens)),
		Tokens:       tokens,
		Positions:    positions,
		TermFreqs:    CalculateTermFrequency(tokens),
	}
	
	return analysis
}

// getUniqueTokens returns unique tokens
func (a *TokenAnalyzer) getUniqueTokens(tokens []string) map[string]bool {
	unique := make(map[string]bool)
	for _, token := range tokens {
		unique[token] = true
	}
	return unique
}

// TokenAnalysis contains detailed token analysis results
type TokenAnalysis struct {
	OriginalText string              `json:"original_text"`
	TokenCount   int                 `json:"token_count"`
	UniqueTokens int                 `json:"unique_tokens"`
	Tokens       []string            `json:"tokens"`
	Positions    []TokenPosition     `json:"positions"`
	TermFreqs    map[string]int      `json:"term_frequencies"`
}

// CompareTokenizers compares different tokenization approaches
func CompareTokenizers(text string, tokenizers map[string]Tokenizer) map[string]*TokenAnalysis {
	results := make(map[string]*TokenAnalysis)
	
	for name, tokenizer := range tokenizers {
		analyzer := NewTokenAnalyzer(tokenizer)
		results[name] = analyzer.AnalyzeText(text)
	}
	
	return results
}