package postprocessing

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// StandardCompressor implements context compression and summarization
type StandardCompressor struct {
	config *PostProcessingConfig
}

// NewStandardCompressor creates a new standard context compressor
func NewStandardCompressor(config *PostProcessingConfig) *StandardCompressor {
	return &StandardCompressor{
		config: config,
	}
}

// Compress reduces context while preserving important information
func (c *StandardCompressor) Compress(ctx context.Context, query string, results []SearchResult, options *CompressionOptions) (*CompressedContext, error) {
	if options == nil {
		options = DefaultCompressionOptions()
	}

	if len(results) == 0 {
		return &CompressedContext{
			Summary:          "No relevant information found.",
			KeyPoints:        []string{},
			RelevantSections: []ContextSection{},
			OriginalLength:   0,
			CompressedLength: 0,
			CompressionRatio: 0.0,
			QualityScore:     0.0,
		}, nil
	}

	// Calculate original length
	originalLength := c.calculateTotalLength(results)

	// Apply compression method
	switch options.Method {
	case ExtractiveSummary:
		return c.extractiveSummary(ctx, query, results, options, originalLength)
	case AbstractiveSummary:
		return c.abstractiveSummary(ctx, query, results, options, originalLength)
	case KeywordExtraction:
		return c.keywordExtraction(ctx, query, results, options, originalLength)
	case TemplateBased:
		return c.templateBased(ctx, query, results, options, originalLength)
	case LLMCompression:
		return c.llmCompression(ctx, query, results, options, originalLength)
	default:
		return c.extractiveSummary(ctx, query, results, options, originalLength)
	}
}

// Summarize creates a summary of the content
func (c *StandardCompressor) Summarize(ctx context.Context, content string, maxLength int) (string, error) {
	if len(content) <= maxLength {
		return content, nil
	}

	// Simple extractive summarization
	sentences := c.splitIntoSentences(content)
	if len(sentences) == 0 {
		return content[:maxLength], nil
	}

	// Score sentences based on position and content
	scored := c.scoreSentences(sentences)

	// Select best sentences within length limit
	selected := c.selectSentences(scored, maxLength)

	return strings.Join(selected, " "), nil
}

// ExtractKeyPoints extracts key points from content
func (c *StandardCompressor) ExtractKeyPoints(ctx context.Context, content string, maxPoints int) ([]string, error) {
	sentences := c.splitIntoSentences(content)
	if len(sentences) == 0 {
		return []string{}, nil
	}

	// Score and rank sentences
	scored := c.scoreSentences(sentences)

	// Extract top sentences as key points
	var keyPoints []string
	count := maxPoints
	if count > len(scored) {
		count = len(scored)
	}

	for i := 0; i < count; i++ {
		keyPoints = append(keyPoints, scored[i].content)
	}

	return keyPoints, nil
}

// extractiveSummary creates summary by extracting key sentences
func (c *StandardCompressor) extractiveSummary(ctx context.Context, query string, results []SearchResult, options *CompressionOptions, originalLength int) (*CompressedContext, error) {
	// Combine all content
	allContent := c.combineContent(results)

	// Create summary
	summary, err := c.Summarize(ctx, allContent, options.MaxSummaryLength)
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}

	// Extract key points
	keyPoints, err := c.ExtractKeyPoints(ctx, allContent, 5)
	if err != nil {
		keyPoints = []string{}
	}

	// Create relevant sections
	sections := c.createRelevantSections(query, results, options)

	compressedLength := len(summary)
	for _, section := range sections {
		compressedLength += section.Length
	}

	compressionRatio := 0.0
	if originalLength > 0 {
		compressionRatio = 1.0 - (float64(compressedLength) / float64(originalLength))
	}

	qualityScore := c.calculateQualityScore(summary, keyPoints, sections)

	return &CompressedContext{
		Summary:          summary,
		KeyPoints:        keyPoints,
		RelevantSections: sections,
		OriginalLength:   originalLength,
		CompressedLength: compressedLength,
		CompressionRatio: compressionRatio,
		QualityScore:     qualityScore,
	}, nil
}

// abstractiveSummary creates summary by generating new content
func (c *StandardCompressor) abstractiveSummary(ctx context.Context, query string, results []SearchResult, options *CompressionOptions, originalLength int) (*CompressedContext, error) {
	// For now, fall back to extractive summary
	// In a real implementation, this would use an LLM for abstractive summarization
	return c.extractiveSummary(ctx, query, results, options, originalLength)
}

// keywordExtraction creates summary by extracting keywords
func (c *StandardCompressor) keywordExtraction(ctx context.Context, query string, results []SearchResult, options *CompressionOptions, originalLength int) (*CompressedContext, error) {
	allContent := c.combineContent(results)

	// Extract keywords
	keywords := c.extractKeywords(allContent, 20)

	// Create summary from keywords
	summary := strings.Join(keywords, ", ")
	if len(summary) > options.MaxSummaryLength {
		summary = summary[:options.MaxSummaryLength]
	}

	// Use keywords as key points
	keyPoints := keywords
	if len(keyPoints) > 5 {
		keyPoints = keyPoints[:5]
	}

	sections := c.createRelevantSections(query, results, options)

	compressedLength := len(summary)
	compressionRatio := 1.0 - (float64(compressedLength) / float64(originalLength))

	return &CompressedContext{
		Summary:          summary,
		KeyPoints:        keyPoints,
		RelevantSections: sections,
		OriginalLength:   originalLength,
		CompressedLength: compressedLength,
		CompressionRatio: compressionRatio,
		QualityScore:     0.7, // Default quality for keyword extraction
	}, nil
}

// templateBased creates summary using templates
func (c *StandardCompressor) templateBased(ctx context.Context, query string, results []SearchResult, options *CompressionOptions, originalLength int) (*CompressedContext, error) {
	// Create template-based summary
	summary := c.createTemplateSummary(query, results)

	// Extract key points
	keyPoints := c.extractTemplateKeyPoints(results)

	sections := c.createRelevantSections(query, results, options)

	compressedLength := len(summary)
	compressionRatio := 1.0 - (float64(compressedLength) / float64(originalLength))

	return &CompressedContext{
		Summary:          summary,
		KeyPoints:        keyPoints,
		RelevantSections: sections,
		OriginalLength:   originalLength,
		CompressedLength: compressedLength,
		CompressionRatio: compressionRatio,
		QualityScore:     0.8, // Higher quality for structured templates
	}, nil
}

// llmCompression creates summary using LLM
func (c *StandardCompressor) llmCompression(ctx context.Context, query string, results []SearchResult, options *CompressionOptions, originalLength int) (*CompressedContext, error) {
	// For now, fall back to extractive summary
	// In a real implementation, this would use LLM for intelligent compression
	return c.extractiveSummary(ctx, query, results, options, originalLength)
}

// Helper methods

func (c *StandardCompressor) calculateTotalLength(results []SearchResult) int {
	total := 0
	for _, result := range results {
		total += len(result.Content) + len(result.Title)
	}
	return total
}

func (c *StandardCompressor) combineContent(results []SearchResult) string {
	var combined []string
	for _, result := range results {
		if result.Title != "" {
			combined = append(combined, result.Title)
		}
		if result.Content != "" {
			combined = append(combined, result.Content)
		}
	}
	return strings.Join(combined, "\n\n")
}

func (c *StandardCompressor) splitIntoSentences(content string) []string {
	// Simple sentence splitting
	sentences := strings.Split(content, ".")
	var cleaned []string

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 10 { // Filter very short sentences
			cleaned = append(cleaned, sentence)
		}
	}

	return cleaned
}

type scoredSentence struct {
	content string
	score   float64
	length  int
}

func (c *StandardCompressor) scoreSentences(sentences []string) []scoredSentence {
	var scored []scoredSentence

	for i, sentence := range sentences {
		score := c.calculateSentenceScore(sentence, i, len(sentences))
		scored = append(scored, scoredSentence{
			content: sentence,
			score:   score,
			length:  len(sentence),
		})
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	return scored
}

func (c *StandardCompressor) calculateSentenceScore(sentence string, position, total int) float64 {
	score := 0.0

	// Position-based scoring (beginning and end are more important)
	positionScore := 0.0
	if position < total/3 {
		positionScore = 0.3 // Beginning
	} else if position > 2*total/3 {
		positionScore = 0.2 // End
	} else {
		positionScore = 0.1 // Middle
	}

	// Length-based scoring (moderate length preferred)
	lengthScore := 0.0
	length := len(sentence)
	if length >= 50 && length <= 200 {
		lengthScore = 0.3
	} else if length >= 20 && length <= 300 {
		lengthScore = 0.2
	} else {
		lengthScore = 0.1
	}

	// Content-based scoring (presence of important words)
	contentScore := c.calculateContentScore(sentence)

	score = positionScore + lengthScore + contentScore

	return score
}

func (c *StandardCompressor) calculateContentScore(sentence string) float64 {
	// Simple content scoring based on keyword presence
	importantWords := []string{
		"important", "significant", "key", "main", "primary", "essential",
		"result", "conclusion", "finding", "evidence", "data", "analysis",
		"solution", "method", "approach", "strategy", "recommend",
	}

	lower := strings.ToLower(sentence)
	score := 0.0

	for _, word := range importantWords {
		if strings.Contains(lower, word) {
			score += 0.1
		}
	}

	return score
}

func (c *StandardCompressor) selectSentences(scored []scoredSentence, maxLength int) []string {
	var selected []string
	totalLength := 0

	for _, sentence := range scored {
		if totalLength+sentence.length <= maxLength {
			selected = append(selected, sentence.content)
			totalLength += sentence.length
		}
	}

	return selected
}

func (c *StandardCompressor) extractKeywords(content string, maxKeywords int) []string {
	words := strings.Fields(strings.ToLower(content))
	
	// Count word frequencies
	freq := make(map[string]int)
	for _, word := range words {
		// Clean word
		cleaned := strings.Trim(word, ".,!?;:\"'()[]{}/-")
		if len(cleaned) > 3 { // Filter short words
			freq[cleaned]++
		}
	}

	// Convert to slice and sort by frequency
	type wordFreq struct {
		word  string
		count int
	}

	var freqs []wordFreq
	for word, count := range freq {
		freqs = append(freqs, wordFreq{word, count})
	}

	sort.Slice(freqs, func(i, j int) bool {
		return freqs[i].count > freqs[j].count
	})

	// Extract top keywords
	var keywords []string
	count := maxKeywords
	if count > len(freqs) {
		count = len(freqs)
	}

	for i := 0; i < count; i++ {
		keywords = append(keywords, freqs[i].word)
	}

	return keywords
}

func (c *StandardCompressor) createRelevantSections(query string, results []SearchResult, options *CompressionOptions) []ContextSection {
	var sections []ContextSection

	for i, result := range results {
		if len(sections) >= 5 { // Limit number of sections
			break
		}

		relevance := c.calculateRelevanceToQuery(query, result.Content)
		
		section := ContextSection{
			Content:   c.truncateContent(result.Content, 200),
			Source:    result.Source,
			Relevance: relevance,
			Position:  i,
			Length:    len(result.Content),
		}

		if relevance >= 0.3 { // Only include reasonably relevant sections
			sections = append(sections, section)
		}
	}

	// Sort by relevance
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Relevance > sections[j].Relevance
	})

	return sections
}

func (c *StandardCompressor) calculateRelevanceToQuery(query, content string) float64 {
	if query == "" || content == "" {
		return 0.0
	}

	queryWords := strings.Fields(strings.ToLower(query))
	contentLower := strings.ToLower(content)

	matches := 0
	for _, word := range queryWords {
		if strings.Contains(contentLower, word) {
			matches++
		}
	}

	if len(queryWords) == 0 {
		return 0.0
	}

	return float64(matches) / float64(len(queryWords))
}

func (c *StandardCompressor) truncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	// Try to break at word boundary
	truncated := content[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLength/2 {
		return truncated[:lastSpace] + "..."
	}

	return truncated + "..."
}

func (c *StandardCompressor) createTemplateSummary(query string, results []SearchResult) string {
	if len(results) == 0 {
		return "No relevant information found."
	}

	template := fmt.Sprintf("Based on %d sources, here are the key findings for \"%s\":\n\n", len(results), query)

	// Add top results
	count := 3
	if count > len(results) {
		count = len(results)
	}

	for i := 0; i < count; i++ {
		result := results[i]
		snippet := c.truncateContent(result.Content, 100)
		template += fmt.Sprintf("• %s\n", snippet)
	}

	return template
}

func (c *StandardCompressor) extractTemplateKeyPoints(results []SearchResult) []string {
	var keyPoints []string

	for i, result := range results {
		if i >= 5 { // Limit to top 5 results
			break
		}

		if result.Title != "" {
			keyPoints = append(keyPoints, result.Title)
		} else {
			// Extract first sentence as key point
			sentences := c.splitIntoSentences(result.Content)
			if len(sentences) > 0 {
				keyPoints = append(keyPoints, sentences[0])
			}
		}
	}

	return keyPoints
}

func (c *StandardCompressor) calculateQualityScore(summary string, keyPoints []string, sections []ContextSection) float64 {
	score := 0.0

	// Summary quality
	if len(summary) > 50 && len(summary) < 500 {
		score += 0.3
	} else if len(summary) > 20 {
		score += 0.1
	}

	// Key points quality
	if len(keyPoints) >= 3 && len(keyPoints) <= 7 {
		score += 0.3
	} else if len(keyPoints) > 0 {
		score += 0.1
	}

	// Sections quality
	if len(sections) >= 2 && len(sections) <= 5 {
		score += 0.2
	} else if len(sections) > 0 {
		score += 0.1
	}

	// Average relevance of sections
	if len(sections) > 0 {
		totalRelevance := 0.0
		for _, section := range sections {
			totalRelevance += section.Relevance
		}
		avgRelevance := totalRelevance / float64(len(sections))
		score += avgRelevance * 0.2
	}

	return score
}

// LLMCompressor implements LLM-based compression
type LLMCompressor struct {
	config *PostProcessingConfig
}

// NewLLMCompressor creates a new LLM-based compressor
func NewLLMCompressor(config *PostProcessingConfig) *LLMCompressor {
	return &LLMCompressor{
		config: config,
	}
}

// Compress implements LLM-based context compression
func (l *LLMCompressor) Compress(ctx context.Context, query string, results []SearchResult, options *CompressionOptions) (*CompressedContext, error) {
	// LLM compression implementation would go here
	// For now, fall back to standard compression
	standardCompressor := NewStandardCompressor(l.config)
	return standardCompressor.Compress(ctx, query, results, options)
}

func (l *LLMCompressor) Summarize(ctx context.Context, content string, maxLength int) (string, error) {
	// LLM summarization would be implemented here
	standardCompressor := NewStandardCompressor(l.config)
	return standardCompressor.Summarize(ctx, content, maxLength)
}

func (l *LLMCompressor) ExtractKeyPoints(ctx context.Context, content string, maxPoints int) ([]string, error) {
	// LLM key point extraction would be implemented here
	standardCompressor := NewStandardCompressor(l.config)
	return standardCompressor.ExtractKeyPoints(ctx, content, maxPoints)
}