package crag

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// StandardKnowledgeRefinement implements knowledge refinement using standard algorithms
type StandardKnowledgeRefinement struct {
	evaluator RetrievalEvaluator
}

// NewStandardKnowledgeRefinement creates a new standard knowledge refinement processor
func NewStandardKnowledgeRefinement(evaluator RetrievalEvaluator) *StandardKnowledgeRefinement {
	return &StandardKnowledgeRefinement{
		evaluator: evaluator,
	}
}

// RefineKnowledge processes and improves the quality of retrieved information
func (r *StandardKnowledgeRefinement) RefineKnowledge(ctx context.Context, query string, documents []Document) ([]Document, error) {
	if len(documents) == 0 {
		return documents, nil
	}
	
	// Step 1: Filter relevant documents
	relevant, err := r.FilterRelevant(ctx, query, documents, 0.3)
	if err != nil {
		return nil, fmt.Errorf("failed to filter relevant documents: %w", err)
	}
	
	// Step 2: Remove duplicates
	deduplicated := r.removeDuplicates(relevant)
	
	// Step 3: Rerank documents
	reranked, err := r.RerankDocuments(ctx, query, deduplicated)
	if err != nil {
		return nil, fmt.Errorf("failed to rerank documents: %w", err)
	}
	
	// Step 4: Limit to reasonable number
	if len(reranked) > 10 {
		reranked = reranked[:10]
	}
	
	return reranked, nil
}

// FilterRelevant filters documents based on relevance to the query
func (r *StandardKnowledgeRefinement) FilterRelevant(ctx context.Context, query string, documents []Document, threshold float64) ([]Document, error) {
	var relevant []Document
	
	for _, doc := range documents {
		relevanceScore := r.calculateRelevanceScore(query, doc)
		
		if relevanceScore >= threshold {
			// Update document score
			doc.Score = relevanceScore
			relevant = append(relevant, doc)
		}
	}
	
	return relevant, nil
}

// RerankDocuments reorders documents based on relevance and quality
func (r *StandardKnowledgeRefinement) RerankDocuments(ctx context.Context, query string, documents []Document) ([]Document, error) {
	if len(documents) <= 1 {
		return documents, nil
	}
	
	// Calculate comprehensive scores for each document
	scoredDocs := make([]scoredDocument, 0, len(documents))
	
	for _, doc := range documents {
		score := r.calculateComprehensiveScore(query, doc)
		scoredDocs = append(scoredDocs, scoredDocument{
			Document: doc,
			Score:    score,
		})
	}
	
	// Sort by score (descending)
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].Score > scoredDocs[j].Score
	})
	
	// Extract documents in new order
	reranked := make([]Document, 0, len(documents))
	for _, scored := range scoredDocs {
		scored.Document.Score = scored.Score
		reranked = append(reranked, scored.Document)
	}
	
	return reranked, nil
}

// calculateRelevanceScore calculates relevance score using multiple factors
func (r *StandardKnowledgeRefinement) calculateRelevanceScore(query string, doc Document) float64 {
	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(doc.Content)
	titleLower := strings.ToLower(doc.Title)
	
	queryTerms := strings.Fields(queryLower)
	
	// Term frequency scoring
	tfScore := r.calculateTFScore(queryTerms, contentLower)
	
	// Title relevance scoring
	titleScore := r.calculateTitleScore(queryTerms, titleLower)
	
	// Position scoring (terms appearing earlier are more important)
	positionScore := r.calculatePositionScore(queryTerms, contentLower)
	
	// Phrase matching scoring
	phraseScore := r.calculatePhraseScore(queryLower, contentLower)
	
	// Combine scores
	relevanceScore := tfScore*0.4 + titleScore*0.3 + positionScore*0.2 + phraseScore*0.1
	
	// Normalize to [0, 1]
	if relevanceScore > 1.0 {
		relevanceScore = 1.0
	}
	
	return relevanceScore
}

// calculateComprehensiveScore calculates a comprehensive score including quality factors
func (r *StandardKnowledgeRefinement) calculateComprehensiveScore(query string, doc Document) float64 {
	// Base relevance score
	relevanceScore := r.calculateRelevanceScore(query, doc)
	
	// Quality factors
	qualityScore := r.calculateQualityScore(doc)
	
	// Source authority score
	authorityScore := r.calculateAuthorityScore(doc)
	
	// Freshness score
	freshnessScore := r.calculateFreshnessScore(doc)
	
	// Length appropriateness score
	lengthScore := r.calculateLengthScore(doc)
	
	// Combine all scores
	comprehensiveScore := relevanceScore*0.5 + 
		qualityScore*0.2 + 
		authorityScore*0.1 + 
		freshnessScore*0.1 + 
		lengthScore*0.1
	
	return comprehensiveScore
}

// calculateTFScore calculates term frequency score
func (r *StandardKnowledgeRefinement) calculateTFScore(queryTerms []string, content string) float64 {
	if len(queryTerms) == 0 {
		return 0
	}
	
	var totalScore float64
	for _, term := range queryTerms {
		termCount := strings.Count(content, term)
		totalScore += float64(termCount)
	}
	
	// Normalize by document length and query length
	contentWords := len(strings.Fields(content))
	if contentWords == 0 {
		return 0
	}
	
	return math.Min(totalScore/(float64(contentWords)*0.1), 1.0)
}

// calculateTitleScore calculates title relevance score
func (r *StandardKnowledgeRefinement) calculateTitleScore(queryTerms []string, title string) float64 {
	if title == "" || len(queryTerms) == 0 {
		return 0
	}
	
	matchCount := 0
	for _, term := range queryTerms {
		if strings.Contains(title, term) {
			matchCount++
		}
	}
	
	return float64(matchCount) / float64(len(queryTerms))
}

// calculatePositionScore gives higher scores to terms appearing earlier
func (r *StandardKnowledgeRefinement) calculatePositionScore(queryTerms []string, content string) float64 {
	if len(queryTerms) == 0 || content == "" {
		return 0
	}
	
	contentLen := len(content)
	var totalScore float64
	
	for _, term := range queryTerms {
		if index := strings.Index(content, term); index != -1 {
			// Earlier positions get higher scores
			positionScore := 1.0 - (float64(index) / float64(contentLen))
			totalScore += positionScore
		}
	}
	
	return totalScore / float64(len(queryTerms))
}

// calculatePhraseScore checks for phrase matching
func (r *StandardKnowledgeRefinement) calculatePhraseScore(query, content string) float64 {
	// Check if the entire query appears as a phrase
	if strings.Contains(content, query) {
		return 1.0
	}
	
	// Check for partial phrase matches
	queryWords := strings.Fields(query)
	if len(queryWords) < 2 {
		return 0
	}
	
	var maxPhraseLength float64
	for i := 0; i < len(queryWords)-1; i++ {
		for j := i + 2; j <= len(queryWords); j++ {
			phrase := strings.Join(queryWords[i:j], " ")
			if strings.Contains(content, phrase) {
				phraseLength := float64(j - i)
				if phraseLength > maxPhraseLength {
					maxPhraseLength = phraseLength
				}
			}
		}
	}
	
	return maxPhraseLength / float64(len(queryWords))
}

// calculateQualityScore assesses document quality
func (r *StandardKnowledgeRefinement) calculateQualityScore(doc Document) float64 {
	var score float64
	
	// Content length factor (not too short, not too long)
	contentLen := len(doc.Content)
	if contentLen >= 100 && contentLen <= 5000 {
		score += 0.4
	} else if contentLen >= 50 {
		score += 0.2
	}
	
	// Title presence
	if doc.Title != "" {
		score += 0.2
	}
	
	// URL quality (https, well-formed)
	if doc.URL != "" {
		if strings.HasPrefix(doc.URL, "https://") {
			score += 0.2
		} else if strings.HasPrefix(doc.URL, "http://") {
			score += 0.1
		}
	}
	
	// Content structure (presence of punctuation, capitalization)
	if r.hasGoodStructure(doc.Content) {
		score += 0.2
	}
	
	return math.Min(score, 1.0)
}

// calculateAuthorityScore assesses source authority
func (r *StandardKnowledgeRefinement) calculateAuthorityScore(doc Document) float64 {
	if doc.URL == "" {
		return 0.5 // Neutral score for documents without URL
	}
	
	authoritative_domains := []string{
		"wikipedia.org", "gov", "edu", "org",
		"stackoverflow.com", "github.com",
	}
	
	for _, domain := range authoritative_domains {
		if strings.Contains(doc.URL, domain) {
			return 0.9
		}
	}
	
	// Check for HTTPS
	if strings.HasPrefix(doc.URL, "https://") {
		return 0.6
	}
	
	return 0.3
}

// calculateFreshnessScore assesses document freshness
func (r *StandardKnowledgeRefinement) calculateFreshnessScore(doc Document) float64 {
	if doc.RetrievedAt.IsZero() {
		return 0.5 // Neutral score for unknown dates
	}
	
	age := time.Since(doc.RetrievedAt)
	
	// Recent documents get higher scores
	if age < 24*time.Hour {
		return 1.0
	} else if age < 7*24*time.Hour {
		return 0.8
	} else if age < 30*24*time.Hour {
		return 0.6
	} else if age < 365*24*time.Hour {
		return 0.4
	}
	
	return 0.2
}

// calculateLengthScore assesses content length appropriateness
func (r *StandardKnowledgeRefinement) calculateLengthScore(doc Document) float64 {
	contentLen := len(doc.Content)
	
	// Optimal length range
	if contentLen >= 200 && contentLen <= 2000 {
		return 1.0
	} else if contentLen >= 100 && contentLen <= 5000 {
		return 0.8
	} else if contentLen >= 50 {
		return 0.6
	} else if contentLen >= 20 {
		return 0.4
	}
	
	return 0.2
}

// hasGoodStructure checks if content has good structure
func (r *StandardKnowledgeRefinement) hasGoodStructure(content string) bool {
	// Check for punctuation
	hasPunctuation := strings.ContainsAny(content, ".!?")
	
	// Check for capitalization variety
	hasUpperCase := strings.ToLower(content) != content
	hasLowerCase := strings.ToUpper(content) != content
	
	// Check for reasonable sentence structure
	sentences := strings.FieldsFunc(content, func(c rune) bool {
		return c == '.' || c == '!' || c == '?'
	})
	
	hasReasonableSentences := len(sentences) > 0
	
	return hasPunctuation && hasUpperCase && hasLowerCase && hasReasonableSentences
}

// removeDuplicates removes duplicate documents based on content similarity
func (r *StandardKnowledgeRefinement) removeDuplicates(documents []Document) []Document {
	if len(documents) <= 1 {
		return documents
	}
	
	var unique []Document
	
	for _, doc := range documents {
		isDuplicate := false
		
		for _, existing := range unique {
			if r.areSimilar(doc, existing) {
				isDuplicate = true
				break
			}
		}
		
		if !isDuplicate {
			unique = append(unique, doc)
		}
	}
	
	return unique
}

// areSimilar checks if two documents are similar enough to be considered duplicates
func (r *StandardKnowledgeRefinement) areSimilar(doc1, doc2 Document) bool {
	// Check URL similarity
	if doc1.URL != "" && doc2.URL != "" && doc1.URL == doc2.URL {
		return true
	}
	
	// Check title similarity
	if doc1.Title != "" && doc2.Title != "" && 
		strings.EqualFold(doc1.Title, doc2.Title) {
		return true
	}
	
	// Check content similarity using simple Jaccard similarity
	similarity := r.calculateJaccardSimilarity(doc1.Content, doc2.Content)
	return similarity > 0.8
}

// calculateJaccardSimilarity calculates Jaccard similarity between two texts
func (r *StandardKnowledgeRefinement) calculateJaccardSimilarity(text1, text2 string) float64 {
	words1 := r.extractUniqueWords(text1)
	words2 := r.extractUniqueWords(text2)
	
	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	
	intersection := 0
	for word := range words1 {
		if words2[word] {
			intersection++
		}
	}
	
	union := len(words1) + len(words2) - intersection
	if union == 0 {
		return 0
	}
	
	return float64(intersection) / float64(union)
}

// extractUniqueWords extracts unique words from text
func (r *StandardKnowledgeRefinement) extractUniqueWords(text string) map[string]bool {
	words := make(map[string]bool)
	
	// Simple word extraction
	fields := strings.Fields(strings.ToLower(text))
	for _, field := range fields {
		// Remove basic punctuation
		cleaned := strings.Trim(field, ".,!?;:\"'()")
		if len(cleaned) > 2 { // Ignore very short words
			words[cleaned] = true
		}
	}
	
	return words
}

// scoredDocument is a helper struct for reranking
type scoredDocument struct {
	Document Document
	Score    float64
}