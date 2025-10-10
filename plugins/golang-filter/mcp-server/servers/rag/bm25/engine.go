package bm25

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryBM25Engine implements BM25 search engine with in-memory storage
type MemoryBM25Engine struct {
	mu sync.RWMutex
	
	// Core data structures
	documents map[string]*BM25Document
	index     *InvertedIndex
	
	// Configuration
	config     *BM25Config
	tokenizer  Tokenizer
	parameters BM25Parameters
	
	// Statistics
	stats      *BM25Stats
	lastUpdate time.Time
}

// NewMemoryBM25Engine creates a new in-memory BM25 engine
func NewMemoryBM25Engine(config *BM25Config) (*MemoryBM25Engine, error) {
	if config == nil {
		config = DefaultBM25Config()
	}
	
	engine := &MemoryBM25Engine{
		documents:  make(map[string]*BM25Document),
		index:     NewInvertedIndex(),
		config:    config,
		tokenizer: NewSimpleTokenizer(&config.Tokenizer),
		parameters: config.Parameters,
		stats:     &BM25Stats{},
		lastUpdate: time.Now(),
	}
	
	return engine, nil
}

// DefaultBM25Config returns default BM25 configuration
func DefaultBM25Config() *BM25Config {
	return &BM25Config{
		Parameters: BM25Parameters{
			K1: 1.2,
			B:  0.75,
		},
		Tokenizer: *DefaultTokenizerConfig(),
		Storage: BM25StorageConfig{
			Type:     "memory",
			Settings: make(map[string]interface{}),
		},
	}
}

// NewInvertedIndex creates a new inverted index
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		TermDocFreq:   make(map[string]map[string]int),
		DocLengths:    make(map[string]int),
		TermDocCount:  make(map[string]int),
		TotalDocs:     0,
		AvgDocLength:  0,
	}
}

// AddDocument adds a document to the BM25 index
func (e *MemoryBM25Engine) AddDocument(ctx context.Context, doc *BM25Document) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Tokenize document content
	tokens := e.tokenizer.Tokenize(doc.Content)
	termFreqs := CalculateTermFrequency(tokens)
	
	// Update document
	doc.Terms = tokens
	doc.TermFreqs = termFreqs
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	
	// Remove existing document if it exists
	if _, exists := e.documents[doc.ID]; exists {
		e.removeDocumentFromIndex(doc.ID)
	}
	
	// Add document to collection
	e.documents[doc.ID] = doc
	
	// Update inverted index
	e.addDocumentToIndex(doc)
	
	// Update statistics
	e.updateStats()
	e.lastUpdate = time.Now()
	
	return nil
}

// AddDocuments adds multiple documents to the BM25 index
func (e *MemoryBM25Engine) AddDocuments(ctx context.Context, docs []*BM25Document) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	for _, doc := range docs {
		// Tokenize document content
		tokens := e.tokenizer.Tokenize(doc.Content)
		termFreqs := CalculateTermFrequency(tokens)
		
		// Update document
		doc.Terms = tokens
		doc.TermFreqs = termFreqs
		if doc.CreatedAt.IsZero() {
			doc.CreatedAt = time.Now()
		}
		
		// Remove existing document if it exists
		if _, exists := e.documents[doc.ID]; exists {
			e.removeDocumentFromIndex(doc.ID)
		}
		
		// Add document to collection
		e.documents[doc.ID] = doc
		
		// Update inverted index
		e.addDocumentToIndex(doc)
	}
	
	// Update statistics
	e.updateStats()
	e.lastUpdate = time.Now()
	
	return nil
}

// DeleteDocument removes a document from the BM25 index
func (e *MemoryBM25Engine) DeleteDocument(ctx context.Context, docID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.documents[docID]; !exists {
		return fmt.Errorf("document %s not found", docID)
	}
	
	// Remove from index
	e.removeDocumentFromIndex(docID)
	
	// Remove from documents
	delete(e.documents, docID)
	
	// Update statistics
	e.updateStats()
	e.lastUpdate = time.Now()
	
	return nil
}

// Search performs BM25 search with the given query
func (e *MemoryBM25Engine) Search(ctx context.Context, query string, options *BM25SearchOptions) ([]*BM25Result, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if options == nil {
		options = &BM25SearchOptions{
			TopK:     10,
			MinScore: 0.0,
			Highlight: false,
		}
	}
	
	// Tokenize query
	queryTerms := e.tokenizer.Tokenize(query)
	if len(queryTerms) == 0 {
		return []*BM25Result{}, nil
	}
	
	// Calculate BM25 scores for all documents
	scores := e.calculateBM25Scores(queryTerms, options.BoostTerms)
	
	// Filter by minimum score
	var results []*BM25Result
	for docID, score := range scores {
		if score >= options.MinScore {
			doc := e.documents[docID]
			result := &BM25Result{
				DocumentID: docID,
				Score:      score,
				Content:    doc.Content,
				Metadata:   doc.Metadata,
			}
			
			// Add highlighting if requested
			if options.Highlight {
				result.Highlight = e.generateHighlights(doc.Content, queryTerms)
			}
			
			results = append(results, result)
		}
	}
	
	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	// Limit results
	if options.TopK > 0 && len(results) > options.TopK {
		results = results[:options.TopK]
	}
	
	return results, nil
}

// UpdateDocument updates an existing document in the index
func (e *MemoryBM25Engine) UpdateDocument(ctx context.Context, doc *BM25Document) error {
	return e.AddDocument(ctx, doc)
}

// GetDocumentCount returns the total number of documents in the index
func (e *MemoryBM25Engine) GetDocumentCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.documents)
}

// GetTermCount returns the number of unique terms in the index
func (e *MemoryBM25Engine) GetTermCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.index.TermDocFreq)
}

// BuildIndex rebuilds the entire inverted index
func (e *MemoryBM25Engine) BuildIndex(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Clear existing index
	e.index = NewInvertedIndex()
	
	// Rebuild index from all documents
	for _, doc := range e.documents {
		e.addDocumentToIndex(doc)
	}
	
	// Update statistics
	e.updateStats()
	e.lastUpdate = time.Now()
	
	return nil
}

// GetIndex returns the current inverted index
func (e *MemoryBM25Engine) GetIndex() *InvertedIndex {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	// Return a copy to prevent external modification
	indexCopy := &InvertedIndex{
		TermDocFreq:   make(map[string]map[string]int),
		DocLengths:    make(map[string]int),
		TermDocCount:  make(map[string]int),
		TotalDocs:     e.index.TotalDocs,
		AvgDocLength:  e.index.AvgDocLength,
	}
	
	for term, docFreq := range e.index.TermDocFreq {
		indexCopy.TermDocFreq[term] = make(map[string]int)
		for docID, freq := range docFreq {
			indexCopy.TermDocFreq[term][docID] = freq
		}
	}
	
	for docID, length := range e.index.DocLengths {
		indexCopy.DocLengths[docID] = length
	}
	
	for term, count := range e.index.TermDocCount {
		indexCopy.TermDocCount[term] = count
	}
	
	return indexCopy
}

// Clear removes all documents from the index
func (e *MemoryBM25Engine) Clear(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.documents = make(map[string]*BM25Document)
	e.index = NewInvertedIndex()
	e.updateStats()
	e.lastUpdate = time.Now()
	
	return nil
}

// addDocumentToIndex adds a document to the inverted index
func (e *MemoryBM25Engine) addDocumentToIndex(doc *BM25Document) {
	docLength := len(doc.Terms)
	e.index.DocLengths[doc.ID] = docLength
	
	// Update term frequencies and document counts
	for term, freq := range doc.TermFreqs {
		if e.index.TermDocFreq[term] == nil {
			e.index.TermDocFreq[term] = make(map[string]int)
		}
		e.index.TermDocFreq[term][doc.ID] = freq
		
		// Update document count for this term
		if _, exists := e.index.TermDocFreq[term][doc.ID]; !exists {
			e.index.TermDocCount[term]++
		}
	}
	
	// Update total document count and average length
	e.index.TotalDocs = len(e.documents)
	e.updateAvgDocLength()
}

// removeDocumentFromIndex removes a document from the inverted index
func (e *MemoryBM25Engine) removeDocumentFromIndex(docID string) {
	doc, exists := e.documents[docID]
	if !exists {
		return
	}
	
	// Remove document length
	delete(e.index.DocLengths, docID)
	
	// Remove from term-document frequency map and update document counts
	for term := range doc.TermFreqs {
		if termDocs := e.index.TermDocFreq[term]; termDocs != nil {
			delete(termDocs, docID)
			e.index.TermDocCount[term]--
			
			// Remove term if no documents contain it
			if len(termDocs) == 0 {
				delete(e.index.TermDocFreq, term)
				delete(e.index.TermDocCount, term)
			}
		}
	}
	
	// Update total document count and average length
	e.index.TotalDocs = len(e.documents) - 1
	e.updateAvgDocLength()
}

// updateAvgDocLength updates the average document length
func (e *MemoryBM25Engine) updateAvgDocLength() {
	if e.index.TotalDocs == 0 {
		e.index.AvgDocLength = 0
		return
	}
	
	var totalLength int
	for _, length := range e.index.DocLengths {
		totalLength += length
	}
	e.index.AvgDocLength = float64(totalLength) / float64(e.index.TotalDocs)
}

// calculateBM25Scores calculates BM25 scores for all documents given query terms
func (e *MemoryBM25Engine) calculateBM25Scores(queryTerms []string, boostTerms map[string]float64) map[string]float64 {
	scores := make(map[string]float64)
	
	for _, term := range queryTerms {
		termDocs := e.index.TermDocFreq[term]
		if termDocs == nil {
			continue
		}
		
		// Calculate IDF for this term
		df := float64(e.index.TermDocCount[term])
		idf := math.Log((float64(e.index.TotalDocs) - df + 0.5) / (df + 0.5))
		
		// Apply term boost if specified
		boost := 1.0
		if boostTerms != nil {
			if b, exists := boostTerms[term]; exists {
				boost = b
			}
		}
		
		// Calculate score for each document containing this term
		for docID, tf := range termDocs {
			docLength := float64(e.index.DocLengths[docID])
			
			// BM25 formula
			numerator := float64(tf) * (e.parameters.K1 + 1)
			denominator := float64(tf) + e.parameters.K1 * (1 - e.parameters.B + e.parameters.B * (docLength / e.index.AvgDocLength))
			
			termScore := idf * (numerator / denominator) * boost
			scores[docID] += termScore
		}
	}
	
	return scores
}

// generateHighlights generates highlighted snippets for search results
func (e *MemoryBM25Engine) generateHighlights(content string, queryTerms []string) []string {
	var highlights []string
	lowerContent := strings.ToLower(content)
	
	for _, term := range queryTerms {
		lowerTerm := strings.ToLower(term)
		if index := strings.Index(lowerContent, lowerTerm); index != -1 {
			start := max(0, index-50)
			end := min(len(content), index+len(term)+50)
			
			snippet := content[start:end]
			// Highlight the term
			highlighted := strings.ReplaceAll(snippet, term, fmt.Sprintf("<mark>%s</mark>", term))
			highlights = append(highlights, highlighted)
		}
	}
	
	return highlights
}

// updateStats updates engine statistics
func (e *MemoryBM25Engine) updateStats() {
	e.stats.TotalDocuments = len(e.documents)
	e.stats.TotalTerms = len(e.index.TermDocFreq)
	e.stats.AverageDocLength = e.index.AvgDocLength
	e.stats.LastUpdated = time.Now()
	
	// Calculate approximate index size (in bytes)
	indexSize := int64(0)
	for term, docFreqs := range e.index.TermDocFreq {
		indexSize += int64(len(term)) // term string
		indexSize += int64(len(docFreqs) * 20) // approximate size per doc entry
	}
	e.stats.IndexSize = indexSize
}

// GetStats returns current engine statistics
func (e *MemoryBM25Engine) GetStats() *BM25Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	// Return a copy
	return &BM25Stats{
		TotalDocuments:   e.stats.TotalDocuments,
		TotalTerms:       e.stats.TotalTerms,
		AverageDocLength: e.stats.AverageDocLength,
		IndexSize:        e.stats.IndexSize,
		LastUpdated:      e.stats.LastUpdated,
	}
}

// GetDocument retrieves a document by ID
func (e *MemoryBM25Engine) GetDocument(docID string) (*BM25Document, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	doc, exists := e.documents[docID]
	return doc, exists
}

// ListDocuments returns all document IDs
func (e *MemoryBM25Engine) ListDocuments() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	ids := make([]string, 0, len(e.documents))
	for id := range e.documents {
		ids = append(ids, id)
	}
	return ids
}

// GetConfig returns the current engine configuration
func (e *MemoryBM25Engine) GetConfig() *BM25Config {
	return e.config
}

// SetParameters updates BM25 parameters
func (e *MemoryBM25Engine) SetParameters(params BM25Parameters) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.parameters = params
}

// GetParameters returns current BM25 parameters
func (e *MemoryBM25Engine) GetParameters() BM25Parameters {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.parameters
}

// MultiSearch performs multiple searches in batch
func (e *MemoryBM25Engine) MultiSearch(ctx context.Context, queries []string, options *BM25SearchOptions) (map[string][]*BM25Result, error) {
	results := make(map[string][]*BM25Result)
	
	for _, query := range queries {
		result, err := e.Search(ctx, query, options)
		if err != nil {
			return nil, fmt.Errorf("search failed for query '%s': %w", query, err)
		}
		results[query] = result
	}
	
	return results, nil
}

// SuggestTerms returns term suggestions based on partial input
func (e *MemoryBM25Engine) SuggestTerms(prefix string, limit int) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	var suggestions []string
	lowerPrefix := strings.ToLower(prefix)
	
	for term := range e.index.TermDocFreq {
		if strings.HasPrefix(strings.ToLower(term), lowerPrefix) {
			suggestions = append(suggestions, term)
			if limit > 0 && len(suggestions) >= limit {
				break
			}
		}
	}
	
	// Sort by term frequency
	sort.Slice(suggestions, func(i, j int) bool {
		return e.index.TermDocCount[suggestions[i]] > e.index.TermDocCount[suggestions[j]]
	})
	
	return suggestions
}

// ExplainScore returns detailed scoring information for a query-document pair
func (e *MemoryBM25Engine) ExplainScore(query, docID string) (*BM25ScoreExplanation, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	_, exists := e.documents[docID]
	if !exists {
		return nil, fmt.Errorf("document %s not found", docID)
	}
	
	queryTerms := e.tokenizer.Tokenize(query)
	explanation := &BM25ScoreExplanation{
		DocumentID: docID,
		Query:      query,
		QueryTerms: queryTerms,
		TermScores: make(map[string]*TermScore),
		TotalScore: 0,
	}
	
	for _, term := range queryTerms {
		termDocs := e.index.TermDocFreq[term]
		if termDocs == nil {
			continue
		}
		
		tf, exists := termDocs[docID]
		if !exists {
			continue
		}
		
		// Calculate components
		df := float64(e.index.TermDocCount[term])
		idf := math.Log((float64(e.index.TotalDocs) - df + 0.5) / (df + 0.5))
		docLength := float64(e.index.DocLengths[docID])
		
		numerator := float64(tf) * (e.parameters.K1 + 1)
		denominator := float64(tf) + e.parameters.K1*(1-e.parameters.B+e.parameters.B*(docLength/e.index.AvgDocLength))
		termScore := idf * (numerator / denominator)
		
		explanation.TermScores[term] = &TermScore{
			Term:           term,
			TermFrequency:  tf,
			DocumentFreq:   int(df),
			IDF:            idf,
			Normalization:  numerator / denominator,
			Score:          termScore,
		}
		
		explanation.TotalScore += termScore
	}
	
	return explanation, nil
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}