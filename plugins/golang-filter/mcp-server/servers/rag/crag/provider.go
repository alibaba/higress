package crag

import (
	"context"
	"fmt"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/llm"
)

// Provider represents a CRAG provider
type Provider struct {
	processor   CRAGProcessor
	evaluator   RetrievalEvaluator
	webSearcher WebSearcher
	refinement  KnowledgeRefinement
	config      *CRAGConfig
}

// LLMProviderAdapter adapts LLM provider to CRAG LLM interface
type LLMProviderAdapter struct {
	provider llm.Provider
}

// GenerateCompletion implements CRAG LLMProvider interface
func (a *LLMProviderAdapter) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	return a.provider.GenerateCompletion(ctx, prompt)
}

// NewProvider creates a new CRAG provider with the given configuration
func NewProvider(cfg *config.Config) (*Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	// Create CRAG configuration
	cragConfig := &CRAGConfig{
		HighConfidenceThreshold: 0.8,
		LowConfidenceThreshold:  0.5,
		WebSearchEnabled:        true,
		MaxWebResults:          5,
		WebSearchTimeout:       10000000000, // 10 seconds in nanoseconds
		RefinementEnabled:      true,
		RelevanceThreshold:     0.3,
		MaxDocuments:          10,
		EvaluationModel:       cfg.LLM.Model,
		EvaluationTimeout:     5000000000, // 5 seconds in nanoseconds
		SearchProvider:        "duckduckgo",
		SearchProviderConfig:  make(map[string]interface{}),
	}

	// Create components
	var evaluator RetrievalEvaluator
	var webSearcher WebSearcher
	var refinement KnowledgeRefinement

	// Create LLM-based evaluator if LLM is configured
	if cfg.LLM.Provider != "" && cfg.LLM.APIKey != "" {
		llmProvider, err := llm.NewLLMProvider(cfg.LLM)
		if err != nil {
			return nil, fmt.Errorf("failed to create LLM provider: %w", err)
		}

		llmAdapter := &LLMProviderAdapter{provider: llmProvider}
		evaluator = NewLLMBasedEvaluator(llmAdapter, DefaultEvaluationCriteria())
	} else {
		// Use simple evaluator if no LLM configured
		evaluator = &SimpleEvaluator{
			highThreshold: cragConfig.HighConfidenceThreshold,
			lowThreshold:  cragConfig.LowConfidenceThreshold,
		}
	}

	// Create web searcher
	switch cragConfig.SearchProvider {
	case "duckduckgo":
		webSearcher = NewDuckDuckGoSearcher()
	default:
		webSearcher = NewMockWebSearcher()
	}

	// Create knowledge refinement
	refinement = NewStandardKnowledgeRefinement(evaluator)

	// Create CRAG processor
	processor := NewStandardCRAGProcessor(evaluator, webSearcher, refinement, cragConfig)

	return &Provider{
		processor:   processor,
		evaluator:   evaluator,
		webSearcher: webSearcher,
		refinement:  refinement,
		config:      cragConfig,
	}, nil
}

// ProcessQuery implements the full CRAG workflow
func (p *Provider) ProcessQuery(ctx context.Context, query string, initialDocs []Document) (*CRAGResult, error) {
	return p.processor.ProcessQuery(ctx, query, initialDocs)
}

// EvaluateRetrieval assesses the quality of retrieved documents
func (p *Provider) EvaluateRetrieval(ctx context.Context, query string, documents []Document) (*EvaluationResult, error) {
	return p.evaluator.EvaluateRetrieval(ctx, query, documents)
}

// SearchWeb performs web search
func (p *Provider) SearchWeb(ctx context.Context, query string, maxResults int) ([]WebDocument, error) {
	return p.webSearcher.Search(ctx, query, maxResults)
}

// RefineKnowledge processes and improves retrieved information
func (p *Provider) RefineKnowledge(ctx context.Context, query string, documents []Document) ([]Document, error) {
	return p.refinement.RefineKnowledge(ctx, query, documents)
}

// SetThresholds configures confidence thresholds
func (p *Provider) SetThresholds(high, low float64) {
	p.config.HighConfidenceThreshold = high
	p.config.LowConfidenceThreshold = low
	p.evaluator.SetThresholds(high, low)
}

// GetConfig returns the current CRAG configuration
func (p *Provider) GetConfig() *CRAGConfig {
	return p.config
}

// UpdateConfig updates the CRAG configuration
func (p *Provider) UpdateConfig(config *CRAGConfig) {
	if config != nil {
		p.config = config
	}
}

// Close cleans up provider resources
func (p *Provider) Close() error {
	// Currently no cleanup needed
	return nil
}

// SimpleEvaluator provides a basic evaluation without LLM
type SimpleEvaluator struct {
	highThreshold float64
	lowThreshold  float64
}

// EvaluateRetrieval implements simple evaluation based on document scores
func (e *SimpleEvaluator) EvaluateRetrieval(ctx context.Context, query string, documents []Document) (*EvaluationResult, error) {
	if len(documents) == 0 {
		return &EvaluationResult{
			ConfidenceLevel: NoConfidence,
			OverallScore:    0.0,
			DocumentScores:  []DocumentScore{},
			Reasoning:       "No documents retrieved",
		}, nil
	}

	var totalScore float64
	var documentScores []DocumentScore

	for _, doc := range documents {
		score := doc.Score
		if score > 1.0 {
			score = 1.0
		}
		if score < 0.0 {
			score = 0.0
		}

		documentScores = append(documentScores, DocumentScore{
			DocumentID:     doc.ID,
			RelevanceScore: score,
			QualityScore:   score,
			OverallScore:   score,
		})

		totalScore += score
	}

	overallScore := totalScore / float64(len(documents))
	confidenceLevel := e.determineConfidenceLevel(overallScore)

	return &EvaluationResult{
		ConfidenceLevel: confidenceLevel,
		OverallScore:    overallScore,
		DocumentScores:  documentScores,
		Reasoning:       fmt.Sprintf("Simple evaluation based on document scores. Average score: %.2f", overallScore),
	}, nil
}

// SetThresholds configures confidence thresholds
func (e *SimpleEvaluator) SetThresholds(high, low float64) {
	e.highThreshold = high
	e.lowThreshold = low
}

// determineConfidenceLevel determines confidence level based on score
func (e *SimpleEvaluator) determineConfidenceLevel(score float64) ConfidenceLevel {
	if score >= e.highThreshold {
		return HighConfidence
	} else if score >= e.lowThreshold {
		return LowConfidence
	} else {
		return NoConfidence
	}
}