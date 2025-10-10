package crag

import (
	"context"
	"fmt"
	"time"
)

// StandardCRAGProcessor implements the complete CRAG mechanism
type StandardCRAGProcessor struct {
	evaluator       RetrievalEvaluator
	webSearcher     WebSearcher
	refinement      KnowledgeRefinement
	config          *CRAGConfig
}

// NewStandardCRAGProcessor creates a new CRAG processor
func NewStandardCRAGProcessor(
	evaluator RetrievalEvaluator,
	webSearcher WebSearcher,
	refinement KnowledgeRefinement,
	config *CRAGConfig,
) *StandardCRAGProcessor {
	if config == nil {
		config = DefaultCRAGConfig()
	}
	
	return &StandardCRAGProcessor{
		evaluator:   evaluator,
		webSearcher: webSearcher,
		refinement:  refinement,
		config:      config,
	}
}

// ProcessQuery implements the full CRAG workflow
func (p *StandardCRAGProcessor) ProcessQuery(ctx context.Context, query string, initialDocs []Document) (*CRAGResult, error) {
	startTime := time.Now()
	
	// Step 1: Evaluate initial retrieval quality
	routingDecision, err := p.EvaluateAndRoute(ctx, query, initialDocs)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate and route: %w", err)
	}
	
	var finalDocuments []Document
	webSearchUsed := false
	
	// Step 2: Execute routing decision
	switch routingDecision.Action {
	case UseRetrieved:
		// Use retrieved documents directly with optional refinement
		if p.config.RefinementEnabled {
			finalDocuments, err = p.refinement.RefineKnowledge(ctx, query, initialDocs)
			if err != nil {
				return nil, fmt.Errorf("failed to refine knowledge: %w", err)
			}
		} else {
			finalDocuments = initialDocs
		}
		
	case EnrichWithWeb:
		// Combine retrieved documents with web search results
		webDocs, err := p.performWebSearch(ctx, query)
		if err != nil {
			// Fall back to retrieved documents if web search fails
			finalDocuments = initialDocs
		} else {
			webSearchUsed = true
			combinedDocs := append(initialDocs, p.convertWebDocsToDocuments(webDocs)...)
			
			if p.config.RefinementEnabled {
				finalDocuments, err = p.refinement.RefineKnowledge(ctx, query, combinedDocs)
				if err != nil {
					return nil, fmt.Errorf("failed to refine combined knowledge: %w", err)
				}
			} else {
				finalDocuments = combinedDocs
			}
		}
		
	case ReplaceWithWeb:
		// Replace with web search results
		webDocs, err := p.performWebSearch(ctx, query)
		if err != nil {
			// Fall back to retrieved documents if web search fails
			finalDocuments = initialDocs
		} else {
			webSearchUsed = true
			webDocuments := p.convertWebDocsToDocuments(webDocs)
			
			if p.config.RefinementEnabled {
				finalDocuments, err = p.refinement.RefineKnowledge(ctx, query, webDocuments)
				if err != nil {
					return nil, fmt.Errorf("failed to refine web knowledge: %w", err)
				}
			} else {
				finalDocuments = webDocuments
			}
		}
		
	default:
		return nil, fmt.Errorf("unknown routing action: %v", routingDecision.Action)
	}
	
	// Limit final documents
	if len(finalDocuments) > p.config.MaxDocuments {
		finalDocuments = finalDocuments[:p.config.MaxDocuments]
	}
	
	processingTime := time.Since(startTime)
	
	return &CRAGResult{
		Query:           query,
		FinalDocuments:  finalDocuments,
		RoutingDecision: *routingDecision,
		WebSearchUsed:   webSearchUsed,
		ProcessingTime:  processingTime,
		ProcessedAt:     time.Now(),
	}, nil
}

// EvaluateAndRoute evaluates retrieval quality and determines routing decision
func (p *StandardCRAGProcessor) EvaluateAndRoute(ctx context.Context, query string, docs []Document) (*RoutingDecision, error) {
	// Evaluate retrieval quality
	evaluation, err := p.evaluator.EvaluateRetrieval(ctx, query, docs)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate retrieval: %w", err)
	}
	
	// Determine action based on confidence level
	var action CRAGAction
	var reasoning string
	
	switch evaluation.ConfidenceLevel {
	case HighConfidence:
		action = UseRetrieved
		reasoning = fmt.Sprintf("High confidence (%.2f) in retrieved documents. Using retrieved content directly.", 
			evaluation.OverallScore)
		
	case LowConfidence:
		if p.config.WebSearchEnabled {
			action = EnrichWithWeb
			reasoning = fmt.Sprintf("Moderate confidence (%.2f) in retrieved documents. Enriching with web search.", 
				evaluation.OverallScore)
		} else {
			action = UseRetrieved
			reasoning = fmt.Sprintf("Moderate confidence (%.2f) in retrieved documents. Web search disabled, using retrieved content.", 
				evaluation.OverallScore)
		}
		
	case NoConfidence:
		if p.config.WebSearchEnabled {
			action = ReplaceWithWeb
			reasoning = fmt.Sprintf("Low confidence (%.2f) in retrieved documents. Replacing with web search results.", 
				evaluation.OverallScore)
		} else {
			action = UseRetrieved
			reasoning = fmt.Sprintf("Low confidence (%.2f) in retrieved documents. Web search disabled, using retrieved content with refinement.", 
				evaluation.OverallScore)
		}
		
	default:
		action = UseRetrieved
		reasoning = "Unknown confidence level. Defaulting to retrieved documents."
	}
	
	return &RoutingDecision{
		Action:          action,
		ConfidenceLevel: evaluation.ConfidenceLevel,
		Reasoning:       reasoning,
		Documents:       docs,
		DecidedAt:       time.Now(),
	}, nil
}

// performWebSearch performs web search with configuration
func (p *StandardCRAGProcessor) performWebSearch(ctx context.Context, query string) ([]WebDocument, error) {
	if !p.config.WebSearchEnabled {
		return nil, fmt.Errorf("web search is disabled")
	}
	
	// Create context with timeout
	searchCtx, cancel := context.WithTimeout(ctx, p.config.WebSearchTimeout)
	defer cancel()
	
	// Perform web search
	return p.webSearcher.Search(searchCtx, query, p.config.MaxWebResults)
}

// convertWebDocsToDocuments converts web documents to standard documents
func (p *StandardCRAGProcessor) convertWebDocsToDocuments(webDocs []WebDocument) []Document {
	var documents []Document
	
	for _, webDoc := range webDocs {
		doc := Document{
			ID:          generateDocumentID(webDoc.URL),
			Content:     webDoc.Content,
			Title:       webDoc.Title,
			URL:         webDoc.URL,
			Score:       webDoc.Score,
			Source:      "web_search",
			Metadata:    webDoc.Metadata,
			RetrievedAt: webDoc.RetrievedAt,
		}
		
		if doc.Metadata == nil {
			doc.Metadata = make(map[string]interface{})
		}
		doc.Metadata["original_source"] = webDoc.Source
		doc.Metadata["snippet"] = webDoc.Snippet
		
		documents = append(documents, doc)
	}
	
	return documents
}

// generateDocumentID generates a unique document ID from URL
func generateDocumentID(url string) string {
	// Simple ID generation based on URL
	// In practice, you might want to use a proper hash function
	if url == "" {
		return fmt.Sprintf("doc_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("web_%x", []byte(url))
}

// SimpleCRAGProcessor provides a lightweight CRAG implementation
type SimpleCRAGProcessor struct {
	webSearcher       WebSearcher
	highThreshold     float64
	lowThreshold      float64
	maxWebResults     int
	webSearchEnabled  bool
}

// NewSimpleCRAGProcessor creates a simplified CRAG processor
func NewSimpleCRAGProcessor(webSearcher WebSearcher) *SimpleCRAGProcessor {
	return &SimpleCRAGProcessor{
		webSearcher:      webSearcher,
		highThreshold:    0.8,
		lowThreshold:     0.5,
		maxWebResults:    5,
		webSearchEnabled: true,
	}
}

// ProcessQuery implements a simplified CRAG workflow
func (p *SimpleCRAGProcessor) ProcessQuery(ctx context.Context, query string, initialDocs []Document) (*CRAGResult, error) {
	startTime := time.Now()
	
	// Simple evaluation based on document scores
	avgScore := p.calculateAverageScore(initialDocs)
	confidenceLevel := p.determineConfidenceLevel(avgScore)
	
	var finalDocuments []Document
	var action CRAGAction
	var webSearchUsed bool
	
	switch confidenceLevel {
	case HighConfidence:
		action = UseRetrieved
		finalDocuments = initialDocs
		
	case LowConfidence:
		if p.webSearchEnabled {
			action = EnrichWithWeb
			webDocs, err := p.webSearcher.Search(ctx, query, p.maxWebResults)
			if err == nil {
				webSearchUsed = true
				finalDocuments = append(initialDocs, p.convertWebDocsToDocuments(webDocs)...)
			} else {
				finalDocuments = initialDocs
			}
		} else {
			action = UseRetrieved
			finalDocuments = initialDocs
		}
		
	case NoConfidence:
		if p.webSearchEnabled {
			action = ReplaceWithWeb
			webDocs, err := p.webSearcher.Search(ctx, query, p.maxWebResults)
			if err == nil {
				webSearchUsed = true
				finalDocuments = p.convertWebDocsToDocuments(webDocs)
			} else {
				finalDocuments = initialDocs
			}
		} else {
			action = UseRetrieved
			finalDocuments = initialDocs
		}
	}
	
	routingDecision := RoutingDecision{
		Action:          action,
		ConfidenceLevel: confidenceLevel,
		Reasoning:       fmt.Sprintf("Simple evaluation with average score %.2f", avgScore),
		Documents:       initialDocs,
		DecidedAt:       time.Now(),
	}
	
	return &CRAGResult{
		Query:           query,
		FinalDocuments:  finalDocuments,
		RoutingDecision: routingDecision,
		WebSearchUsed:   webSearchUsed,
		ProcessingTime:  time.Since(startTime),
		ProcessedAt:     time.Now(),
	}, nil
}

// calculateAverageScore calculates average score of documents
func (p *SimpleCRAGProcessor) calculateAverageScore(docs []Document) float64 {
	if len(docs) == 0 {
		return 0
	}
	
	var totalScore float64
	for _, doc := range docs {
		totalScore += doc.Score
	}
	
	return totalScore / float64(len(docs))
}

// determineConfidenceLevel determines confidence level based on score
func (p *SimpleCRAGProcessor) determineConfidenceLevel(score float64) ConfidenceLevel {
	if score >= p.highThreshold {
		return HighConfidence
	} else if score >= p.lowThreshold {
		return LowConfidence
	} else {
		return NoConfidence
	}
}

// convertWebDocsToDocuments converts web documents (simplified version)
func (p *SimpleCRAGProcessor) convertWebDocsToDocuments(webDocs []WebDocument) []Document {
	var documents []Document
	
	for _, webDoc := range webDocs {
		doc := Document{
			ID:          generateDocumentID(webDoc.URL),
			Content:     webDoc.Content,
			Title:       webDoc.Title,
			URL:         webDoc.URL,
			Score:       webDoc.Score,
			Source:      "web_search",
			RetrievedAt: webDoc.RetrievedAt,
		}
		
		documents = append(documents, doc)
	}
	
	return documents
}