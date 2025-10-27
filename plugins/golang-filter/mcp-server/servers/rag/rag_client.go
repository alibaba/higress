package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/bm25"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/crag"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/embedding"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/fusion"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/llm"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/postprocessing"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/queryenhancement"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/textsplitter"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/vectordb"
	"github.com/distribution/distribution/v3/uuid"
)

const (
	MAX_LIST_KNOWLEDGE_ROW_COUNT = 1000
	MAX_LIST_DOCUMENT_ROW_COUNT  = 1000
)

// RAGClient represents the RAG (Retrieval-Augmented Generation) client
type RAGClient struct {
	config            *config.Config
	vectordbProvider  vectordb.VectorStoreProvider
	embeddingProvider embedding.Provider
	textSplitter      textsplitter.TextSplitter
	llmProvider       llm.Provider
}

// EnhancedRAGClient represents the enhanced RAG client with advanced features
type EnhancedRAGClient struct {
	config               *config.Config
	vectordbProvider     vectordb.VectorStoreProvider
	embeddingProvider    embedding.Provider
	textSplitter         textsplitter.TextSplitter
	llmProvider          llm.Provider
	bm25Engine           bm25.BM25Engine
	hybridSearchProvider *fusion.HybridSearchProvider
	cragProcessor        *crag.CRAGProcessor
	queryEnhancer        *queryenhancement.QueryEnhancementProvider
	postProcessor        postprocessing.PostProcessor
}

// NewRAGClient creates a new RAG client instance
func NewRAGClient(config *config.Config) (*RAGClient, error) {
	ragclient := &RAGClient{
		config: config,
	}
	textSplitter, err := textsplitter.NewTextSplitter(&config.RAG.Splitter)
	if err != nil {
		return nil, fmt.Errorf("create text splitter failed, err: %w", err)
	}
	ragclient.textSplitter = textSplitter

	embeddingProvider, err := embedding.NewEmbeddingProvider(ragclient.config.Embedding)
	if err != nil {
		return nil, fmt.Errorf("create embedding provider failed, err: %w", err)
	}
	ragclient.embeddingProvider = embeddingProvider

	if ragclient.config.LLM.Provider == "" {
		ragclient.llmProvider = nil
	} else {
		llmProvider, err := llm.NewLLMProvider(ragclient.config.LLM)
		if err != nil {
			return nil, fmt.Errorf("create llm provider failed, err: %w", err)
		}
		ragclient.llmProvider = llmProvider
	}

	dim := ragclient.config.Embedding.Dimensions
	provider, err := vectordb.NewVectorDBProvider(&ragclient.config.VectorDB, dim)
	if err != nil {
		return nil, fmt.Errorf("create vector store provider failed, err: %w", err)
	}
	ragclient.vectordbProvider = provider
	return ragclient, nil
}

// NewEnhancedRAGClient creates a new enhanced RAG client with advanced features
func NewEnhancedRAGClient(config *config.Config) (*EnhancedRAGClient, error) {
	// Create basic components first
	textSplitter, err := textsplitter.NewTextSplitter(&config.RAG.Splitter)
	if err != nil {
		return nil, fmt.Errorf("create text splitter failed, err: %w", err)
	}

	embeddingProvider, err := embedding.NewEmbeddingProvider(config.Embedding)
	if err != nil {
		return nil, fmt.Errorf("create embedding provider failed, err: %w", err)
	}

	var llmProvider llm.Provider
	if config.LLM.Provider != "" {
		llmProvider, err = llm.NewLLMProvider(config.LLM)
		if err != nil {
			return nil, fmt.Errorf("create llm provider failed, err: %w", err)
		}
	}

	dim := config.Embedding.Dimensions
	vectordbProvider, err := vectordb.NewVectorDBProvider(&config.VectorDB, dim)
	if err != nil {
		return nil, fmt.Errorf("create vector store provider failed, err: %w", err)
	}

	enhancedClient := &EnhancedRAGClient{
		config:            config,
		vectordbProvider:  vectordbProvider,
		embeddingProvider: embeddingProvider,
		textSplitter:      textSplitter,
		llmProvider:       llmProvider,
	}

	// Initialize BM25 engine
	bm25Engine, err := bm25.NewBM25Engine(&bm25.BM25Config{
		K1:         1.2,
		B:          0.75,
		MaxResults: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("create BM25 engine failed, err: %w", err)
	}
	enhancedClient.bm25Engine = bm25Engine

	// Initialize hybrid search provider
	hybridConfig := fusion.DefaultFusionConfig()
	hybridSearchProvider := fusion.NewHybridSearchProvider(
		vectordbProvider,
		embeddingProvider,
		bm25Engine,
		hybridConfig,
	)
	enhancedClient.hybridSearchProvider = hybridSearchProvider

	// Initialize CRAG processor if LLM is available
	if llmProvider != nil {
		cragConfig := crag.DefaultCRAGConfig()
		cragProcessor, err := crag.NewCRAGProcessor(llmProvider, embeddingProvider, cragConfig)
		if err != nil {
			return nil, fmt.Errorf("create CRAG processor failed, err: %w", err)
		}
		enhancedClient.cragProcessor = cragProcessor
	}

	// Initialize query enhancement provider
	queryConfig := queryenhancement.DefaultQueryEnhancementConfig()
	queryEnhancer := queryenhancement.NewQueryEnhancementProvider(
		llmProvider,
		embeddingProvider,
		nil, // Vector database for query enhancement can be added later
		queryConfig,
	)
	enhancedClient.queryEnhancer = queryEnhancer

	// Initialize post-processing
	postConfig := postprocessing.DefaultPostProcessingConfig()
	if llmProvider != nil {
		postConfig.LLMConfig = postprocessing.LLMConfig{
			Provider:    config.LLM.Provider,
			APIKey:      config.LLM.ApiKey,
			BaseURL:     config.LLM.BaseUrl,
			Model:       config.LLM.Model,
			Temperature: config.LLM.Temperature,
			MaxTokens:   config.LLM.MaxTokens,
		}
	}
	if embeddingProvider != nil {
		postConfig.EmbeddingConfig = postprocessing.EmbeddingConfig{
			Provider:   config.Embedding.Provider,
			APIKey:     config.Embedding.ApiKey,
			BaseURL:    config.Embedding.BaseUrl,
			Model:      config.Embedding.Model,
			Dimensions: config.Embedding.Dimensions,
		}
	}
	
	postProcessor := postprocessing.NewStandardPostProcessor(postConfig)
	enhancedClient.postProcessor = postProcessor

	return enhancedClient, nil
}

// ListChunks lists document chunks by knowledge ID, returns in ascending order of DocumentIndex
func (r *RAGClient) ListChunks() ([]schema.Document, error) {
	docs, err := r.vectordbProvider.ListDocs(context.Background(), MAX_LIST_DOCUMENT_ROW_COUNT)
	if err != nil {
		return nil, fmt.Errorf("list chunks failed, err: %w", err)
	}
	return docs, nil
}

// DeleteChunk deletes a specific document chunk
func (r *RAGClient) DeleteChunk(id string) error {
	if err := r.vectordbProvider.DeleteDocs(context.Background(), []string{id}); err != nil {
		return fmt.Errorf("delete chunk failed, err: %w", err)
	}
	return nil
}

func (r *RAGClient) CreateChunkFromText(text string, title string) ([]schema.Document, error) {

	docs, err := textsplitter.CreateDocuments(r.textSplitter, []string{text}, make([]map[string]any, 0))
	if err != nil {
		return nil, fmt.Errorf("create documents failed, err: %w", err)
	}

	results := make([]schema.Document, 0, len(docs))

	for chunkIndex, doc := range docs {
		doc.ID = uuid.Generate().String()
		doc.Metadata["chunk_index"] = chunkIndex
		doc.Metadata["chunk_title"] = title
		doc.Metadata["chunk_size"] = len(doc.Content)
		// Generate embedding for the document
		embedding, err := r.embeddingProvider.GetEmbedding(context.Background(), doc.Content)
		if err != nil {
			return nil, fmt.Errorf("create embedding failed, err: %w", err)
		}
		doc.Vector = embedding
		doc.CreatedAt = time.Now()
		results = append(results, doc)
	}

	if err := r.vectordbProvider.AddDoc(context.Background(), results); err != nil {
		return nil, fmt.Errorf("add documents failed, err: %w", err)
	}

	return results, nil
}

// SearchChunks searches for document chunks
func (r *RAGClient) SearchChunks(query string, topK int, threshold float64) ([]schema.SearchResult, error) {

	vector, err := r.embeddingProvider.GetEmbedding(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("create embedding failed, err: %w", err)
	}
	options := &schema.SearchOptions{
		TopK:      topK,
		Threshold: threshold,
	}
	docs, err := r.vectordbProvider.SearchDocs(context.Background(), vector, options)
	if err != nil {
		return nil, fmt.Errorf("search chunks failed, err: %w", err)
	}
	return docs, nil
}

// Chat generates a response using LLM
func (r *RAGClient) Chat(query string) (string, error) {
	if r.llmProvider == nil {
		return "", fmt.Errorf("llm provider not initialized")
	}

	docs, err := r.SearchChunks(query, r.config.RAG.TopK, r.config.RAG.Threshold)
	if err != nil {
		return "", fmt.Errorf("search chunks failed, err: %w", err)
	}

	contexts := make([]string, 0, len(docs))
	for _, doc := range docs {
		contexts = append(contexts, strings.ReplaceAll(doc.Document.Content, "\n", " "))
	}

	prompt := llm.BuildPrompt(query, contexts, "\n\n")
	resp, err := r.llmProvider.GenerateCompletion(context.Background(), prompt)
	if err != nil {
		return "", fmt.Errorf("generate completion failed, err: %w", err)
	}
	return resp, nil
}

// Enhanced RAG Client Methods

// HybridSearch performs hybrid search using both vector and BM25 retrieval
func (r *EnhancedRAGClient) HybridSearch(ctx context.Context, query string, options *fusion.HybridSearchOptions) ([]*fusion.SearchResult, error) {
	if r.hybridSearchProvider == nil {
		return nil, fmt.Errorf("hybrid search provider not initialized")
	}

	return r.hybridSearchProvider.Search(ctx, query, options)
}

// EnhancedSearch performs search with query enhancement and post-processing
func (r *EnhancedRAGClient) EnhancedSearch(ctx context.Context, query string, topK int, threshold float64) (*EnhancedSearchResult, error) {
	startTime := time.Now()

	// Step 1: Enhance the query
	enhanceOptions := &queryenhancement.EnhancementOptions{
		EnableRewrite:             true,
		EnableExpansion:           true,
		EnableDecomposition:       false, // Keep simple for now
		EnableIntentClassification: true,
		MaxRewriteCount:           3,
		MaxExpansionTerms:         10,
	}

	enhancedQuery, err := r.queryEnhancer.EnhanceQuery(ctx, query, enhanceOptions)
	if err != nil {
		// Continue with original query if enhancement fails
		enhancedQuery = &queryenhancement.EnhancedQuery{
			OriginalQuery: query,
			ProcessedAt:   time.Now(),
		}
	}

	// Step 2: Perform hybrid search
	hybridOptions := &fusion.HybridSearchOptions{
		FusionMethod: fusion.RRFFusion,
		VectorTopK:   topK * 2,
		BM25TopK:     topK * 2,
		FinalTopK:    topK * 3, // Get more for post-processing
		VectorWeight: 0.6,
		BM25Weight:   0.4,
		MinScore:     threshold,
		EnableVector: true,
		EnableBM25:   true,
		FusionOptions: fusion.DefaultFusionOptions(),
	}

	// Use enhanced query or original query
	searchQuery := query
	if len(enhancedQuery.RewrittenQueries) > 0 {
		searchQuery = enhancedQuery.RewrittenQueries[0]
	} else if len(enhancedQuery.ExpandedTerms) > 0 {
		// Add expanded terms to query
		expandedQuery := query
		for i, term := range enhancedQuery.ExpandedTerms {
			if i >= 3 { // Limit expansion
				break
			}
			expandedQuery += " " + term
		}
		searchQuery = expandedQuery
	}

	fusionResults, err := r.hybridSearchProvider.Search(ctx, searchQuery, hybridOptions)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}

	// Step 3: Convert fusion results to post-processing format
	ppResults := r.convertFusionToPostProcessingResults(fusionResults)

	// Step 4: Apply post-processing
	ppOptions := &postprocessing.ProcessingOptions{
		EnableReranking:     true,
		EnableFiltering:     true,
		EnableDeduplication: true,
		EnableCompression:   false,
		MaxResults:          topK,
		MinRelevanceScore:   threshold,
		DiversityWeight:     0.1,
		RerankingOptions:    postprocessing.DefaultRerankingOptions(),
		FilteringOptions:    postprocessing.DefaultFilteringOptions(),
		DeduplicationOptions: postprocessing.DefaultDeduplicationOptions(),
	}

	processedResults, err := r.postProcessor.ProcessResults(ctx, searchQuery, ppResults, ppOptions)
	if err != nil {
		return nil, fmt.Errorf("post-processing failed: %w", err)
	}

	// Step 5: Convert back to schema format
	finalResults := r.convertPostProcessingToSchemaResults(processedResults.FinalResults)

	return &EnhancedSearchResult{
		Query:             query,
		EnhancedQuery:     enhancedQuery,
		Results:           finalResults,
		ProcessedResults:  processedResults,
		ProcessingTime:    time.Since(startTime),
		ProcessedAt:       time.Now(),
	}, nil
}

// CRAGSearch performs Corrective RAG search with web augmentation
func (r *EnhancedRAGClient) CRAGSearch(ctx context.Context, query string, topK int) (*CRAGSearchResult, error) {
	if r.cragProcessor == nil {
		return nil, fmt.Errorf("CRAG processor not initialized")
	}

	startTime := time.Now()

	// Perform initial retrieval
	docs, err := r.SearchChunks(query, topK*2, r.config.RAG.Threshold)
	if err != nil {
		return nil, fmt.Errorf("initial search failed: %w", err)
	}

	// Convert to CRAG documents
	cragDocs := r.convertSchemaToCAGDocuments(docs)

	// Process with CRAG
	cragOptions := &crag.CRAGOptions{
		EnableWebSearch:     true,
		EnableRefinement:    true,
		ConfidenceThreshold: 0.7,
		MaxWebResults:       5,
		MaxRefinements:      3,
	}

	cragResult, err := r.cragProcessor.ProcessQuery(ctx, query, cragDocs, cragOptions)
	if err != nil {
		return nil, fmt.Errorf("CRAG processing failed: %w", err)
	}

	return &CRAGSearchResult{
		Query:          query,
		Result:         cragResult,
		ProcessingTime: time.Since(startTime),
		ProcessedAt:    time.Now(),
	}, nil
}

// EnhancedChat generates responses using enhanced RAG with all features
func (r *EnhancedRAGClient) EnhancedChat(ctx context.Context, query string) (*EnhancedChatResult, error) {
	if r.llmProvider == nil {
		return nil, fmt.Errorf("llm provider not initialized")
	}

	startTime := time.Now()

	// Step 1: Perform enhanced search
	searchResult, err := r.EnhancedSearch(ctx, query, r.config.RAG.TopK, r.config.RAG.Threshold)
	if err != nil {
		return nil, fmt.Errorf("enhanced search failed: %w", err)
	}

	// Step 2: Optional CRAG processing for critical queries
	var cragResult *CRAGSearchResult
	if r.cragProcessor != nil && r.shouldUseCRAG(query, searchResult) {
		cragResult, err = r.CRAGSearch(ctx, query, r.config.RAG.TopK)
		if err != nil {
			// Continue without CRAG if it fails
			cragResult = nil
		}
	}

	// Step 3: Build context from results
	var contexts []string
	if cragResult != nil && cragResult.Result.Action == crag.UseWebAction {
		// Use web-augmented content
		for _, item := range cragResult.Result.WebSearchResults {
			contexts = append(contexts, strings.ReplaceAll(item.Content, "\n", " "))
		}
	} else {
		// Use retrieval results
		for _, doc := range searchResult.Results {
			contexts = append(contexts, strings.ReplaceAll(doc.Document.Content, "\n", " "))
		}
	}

	// Step 4: Generate response
	prompt := llm.BuildPrompt(query, contexts, "\n\n")
	response, err := r.llmProvider.GenerateCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("generate completion failed: %w", err)
	}

	return &EnhancedChatResult{
		Query:           query,
		Response:        response,
		SearchResult:    searchResult,
		CRAGResult:      cragResult,
		ContextSources:  len(contexts),
		ProcessingTime:  time.Since(startTime),
		ProcessedAt:     time.Now(),
	}, nil
}

// Helper methods for enhanced client

func (r *EnhancedRAGClient) convertFusionToPostProcessingResults(fusionResults []*fusion.SearchResult) []postprocessing.SearchResult {
	var results []postprocessing.SearchResult
	for _, fr := range fusionResults {
		result := postprocessing.SearchResult{
			ID:          fr.DocumentID,
			Content:     fr.Content,
			Title:       fr.Title,
			URL:         fr.URL,
			Score:       fr.Score,
			Rank:        fr.Rank,
			Source:      fr.Source,
			Method:      fr.Method,
			Metadata:    fr.Metadata,
			RetrievedAt: fr.RetrievedAt,
		}
		results = append(results, result)
	}
	return results
}

func (r *EnhancedRAGClient) convertPostProcessingToSchemaResults(ppResults []postprocessing.SearchResult) []schema.SearchResult {
	var results []schema.SearchResult
	for _, ppr := range ppResults {
		doc := schema.Document{
			ID:        ppr.ID,
			Content:   ppr.Content,
			Metadata:  ppr.Metadata,
			CreatedAt: ppr.RetrievedAt,
		}
		if ppr.Title != "" {
			doc.Metadata["chunk_title"] = ppr.Title
		}
		if ppr.URL != "" {
			doc.Metadata["url"] = ppr.URL
		}

		result := schema.SearchResult{
			Document: doc,
			Score:    ppr.Score,
		}
		results = append(results, result)
	}
	return results
}

func (r *EnhancedRAGClient) convertSchemaToCAGDocuments(docs []schema.SearchResult) []crag.Document {
	var cragDocs []crag.Document
	for _, doc := range docs {
		cragDoc := crag.Document{
			ID:       doc.Document.ID,
			Content:  doc.Document.Content,
			Title:    r.getStringFromMetadata(doc.Document.Metadata, "chunk_title"),
			URL:      r.getStringFromMetadata(doc.Document.Metadata, "url"),
			Score:    doc.Score,
			Metadata: doc.Document.Metadata,
		}
		cragDocs = append(cragDocs, cragDoc)
	}
	return cragDocs
}

func (r *EnhancedRAGClient) getStringFromMetadata(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	if val, ok := metadata[key].(string); ok {
		return val
	}
	return ""
}

func (r *EnhancedRAGClient) shouldUseCRAG(query string, searchResult *EnhancedSearchResult) bool {
	// Use CRAG for queries that might benefit from web augmentation
	queryLower := strings.ToLower(query)
	
	// Check for time-sensitive queries
	if strings.Contains(queryLower, "recent") || strings.Contains(queryLower, "latest") ||
		strings.Contains(queryLower, "current") || strings.Contains(queryLower, "new") {
		return true
	}
	
	// Check for factual questions that might need verification
	if strings.Contains(queryLower, "when") || strings.Contains(queryLower, "who") ||
		strings.Contains(queryLower, "what") || strings.Contains(queryLower, "where") {
		return true
	}
	
	// Check search result quality
	if len(searchResult.Results) == 0 {
		return true // No results found, try web search
	}
	
	// Check average score
	if len(searchResult.Results) > 0 {
		totalScore := 0.0
		for _, result := range searchResult.Results {
			totalScore += result.Score
		}
		avgScore := totalScore / float64(len(searchResult.Results))
		if avgScore < 0.6 { // Low confidence in results
			return true
		}
	}
	
	return false
}

// Result types for enhanced client

type EnhancedSearchResult struct {
	Query            string                               `json:"query"`
	EnhancedQuery    *queryenhancement.EnhancedQuery     `json:"enhanced_query"`
	Results          []schema.SearchResult               `json:"results"`
	ProcessedResults *postprocessing.ProcessedResults    `json:"processed_results"`
	ProcessingTime   time.Duration                       `json:"processing_time"`
	ProcessedAt      time.Time                           `json:"processed_at"`
}

type CRAGSearchResult struct {
	Query          string                 `json:"query"`
	Result         *crag.CRAGResult       `json:"result"`
	ProcessingTime time.Duration          `json:"processing_time"`
	ProcessedAt    time.Time              `json:"processed_at"`
}

type EnhancedChatResult struct {
	Query          string                `json:"query"`
	Response       string                `json:"response"`
	SearchResult   *EnhancedSearchResult `json:"search_result"`
	CRAGResult     *CRAGSearchResult     `json:"crag_result,omitempty"`
	ContextSources int                   `json:"context_sources"`
	ProcessingTime time.Duration         `json:"processing_time"`
	ProcessedAt    time.Time             `json:"processed_at"`
}
