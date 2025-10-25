package postprocessing

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// StandardPostProcessor implements comprehensive post-processing pipeline
type StandardPostProcessor struct {
	reranker      Reranker
	filter        Filter
	deduplicator  Deduplicator
	compressor    ContextCompressor
	config        *PostProcessingConfig
	metrics       *PostProcessingMetrics
}

// NewStandardPostProcessor creates a new standard post-processor
func NewStandardPostProcessor(config *PostProcessingConfig) *StandardPostProcessor {
	if config == nil {
		config = DefaultPostProcessingConfig()
	}

	processor := &StandardPostProcessor{
		config:  config,
		metrics: &PostProcessingMetrics{},
	}

	// Initialize components
	processor.reranker = NewStandardReranker(config)
	processor.filter = NewStandardFilter(config)
	processor.deduplicator = NewStandardDeduplicator(config)
	processor.compressor = NewStandardCompressor(config)

	return processor
}

// ProcessResults applies comprehensive post-processing to search results
func (p *StandardPostProcessor) ProcessResults(ctx context.Context, query string, results []SearchResult, options *ProcessingOptions) (*ProcessedResults, error) {
	startTime := time.Now()
	
	if options == nil {
		options = p.config.DefaultOptions
	}

	originalCount := len(results)
	processedResults := make([]SearchResult, len(results))
	copy(processedResults, results)

	summary := ProcessingSummary{
		TechniquesApplied: []string{},
		OriginalCount:     originalCount,
	}

	var err error

	// Step 1: Initial filtering
	if options.EnableFiltering {
		processedResults, err = p.FilterResults(ctx, query, processedResults, options.FilteringOptions)
		if err != nil {
			return nil, fmt.Errorf("filtering failed: %w", err)
		}
		summary.TechniquesApplied = append(summary.TechniquesApplied, "filtering")
		summary.FilteredCount = len(processedResults)
	}

	// Step 2: Deduplication
	if options.EnableDeduplication {
		processedResults, err = p.DeduplicateResults(ctx, processedResults, options.DeduplicationOptions)
		if err != nil {
			return nil, fmt.Errorf("deduplication failed: %w", err)
		}
		summary.TechniquesApplied = append(summary.TechniquesApplied, "deduplication")
		summary.DeduplicatedCount = len(processedResults)
	}

	// Step 3: Reranking
	if options.EnableReranking {
		processedResults, err = p.RerankResults(ctx, query, processedResults, options.RerankingOptions)
		if err != nil {
			return nil, fmt.Errorf("reranking failed: %w", err)
		}
		summary.TechniquesApplied = append(summary.TechniquesApplied, "reranking")
		summary.RerankedCount = len(processedResults)
	}

	// Step 4: Final limiting
	if options.MaxResults > 0 && len(processedResults) > options.MaxResults {
		processedResults = processedResults[:options.MaxResults]
	}

	summary.FinalCount = len(processedResults)
	summary.ProcessingTime = time.Since(startTime)

	// Calculate quality improvement
	if originalCount > 0 {
		summary.QualityImprovement = p.calculateQualityImprovement(results, processedResults)
	}

	result := &ProcessedResults{
		Query:             query,
		OriginalCount:     originalCount,
		FinalResults:      processedResults,
		ProcessingSummary: summary,
		ProcessedAt:       time.Now(),
	}

	// Step 5: Context compression (optional)
	if options.EnableCompression {
		compressed, err := p.CompressContext(ctx, query, processedResults, options.CompressionOptions)
		if err != nil {
			return nil, fmt.Errorf("compression failed: %w", err)
		}
		result.CompressedContext = compressed
		summary.TechniquesApplied = append(summary.TechniquesApplied, "compression")
	}

	// Update metrics
	p.updateMetrics(summary)

	return result, nil
}

// RerankResults reorders results based on relevance and quality
func (p *StandardPostProcessor) RerankResults(ctx context.Context, query string, results []SearchResult, options *RerankingOptions) ([]SearchResult, error) {
	if p.reranker == nil {
		return results, nil
	}

	if options == nil {
		options = DefaultRerankingOptions()
	}

	return p.reranker.Rerank(ctx, query, results, options)
}

// FilterResults filters results based on criteria
func (p *StandardPostProcessor) FilterResults(ctx context.Context, query string, results []SearchResult, options *FilteringOptions) ([]SearchResult, error) {
	if p.filter == nil {
		return results, nil
	}

	if options == nil {
		options = DefaultFilteringOptions()
	}

	return p.filter.Filter(ctx, query, results, options)
}

// DeduplicateResults removes duplicate or very similar results
func (p *StandardPostProcessor) DeduplicateResults(ctx context.Context, results []SearchResult, options *DeduplicationOptions) ([]SearchResult, error) {
	if p.deduplicator == nil {
		return results, nil
	}

	if options == nil {
		options = DefaultDeduplicationOptions()
	}

	return p.deduplicator.Deduplicate(ctx, results, options)
}

// CompressContext reduces the amount of context while preserving relevance
func (p *StandardPostProcessor) CompressContext(ctx context.Context, query string, results []SearchResult, options *CompressionOptions) (*CompressedContext, error) {
	if p.compressor == nil {
		return nil, fmt.Errorf("context compressor not available")
	}

	if options == nil {
		options = DefaultCompressionOptions()
	}

	return p.compressor.Compress(ctx, query, results, options)
}

// calculateQualityImprovement estimates quality improvement from processing
func (p *StandardPostProcessor) calculateQualityImprovement(original, processed []SearchResult) float64 {
	if len(original) == 0 || len(processed) == 0 {
		return 0.0
	}

	// Calculate average scores
	originalAvg := p.calculateAverageScore(original)
	processedAvg := p.calculateAverageScore(processed)

	// Calculate score improvement ratio
	if originalAvg > 0 {
		return (processedAvg - originalAvg) / originalAvg
	}

	return 0.0
}

// calculateAverageScore calculates average score of results
func (p *StandardPostProcessor) calculateAverageScore(results []SearchResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	total := 0.0
	for _, result := range results {
		total += result.Score
	}

	return total / float64(len(results))
}

// updateMetrics updates processing metrics
func (p *StandardPostProcessor) updateMetrics(summary ProcessingSummary) {
	p.metrics.ProcessingTime = summary.ProcessingTime
	p.metrics.InputResultCount = summary.OriginalCount
	p.metrics.OutputResultCount = summary.FinalCount
	p.metrics.FilteredCount = summary.FilteredCount
	p.metrics.DeduplicatedCount = summary.DeduplicatedCount
	p.metrics.QualityImprovement = summary.QualityImprovement
}

// GetMetrics returns current processing metrics
func (p *StandardPostProcessor) GetMetrics() *PostProcessingMetrics {
	return p.metrics
}

// PipelinePostProcessor implements a configurable processing pipeline
type PipelinePostProcessor struct {
	pipeline []ProcessingStep
	config   *PostProcessingConfig
}

// ProcessingStep represents a step in the processing pipeline
type ProcessingStep interface {
	Name() string
	Process(ctx context.Context, query string, results []SearchResult, options *ProcessingOptions) ([]SearchResult, error)
	ShouldApply(options *ProcessingOptions) bool
}

// NewPipelinePostProcessor creates a configurable pipeline processor
func NewPipelinePostProcessor(config *PostProcessingConfig, steps ...ProcessingStep) *PipelinePostProcessor {
	if config == nil {
		config = DefaultPostProcessingConfig()
	}

	return &PipelinePostProcessor{
		pipeline: steps,
		config:   config,
	}
}

// ProcessResults applies the processing pipeline
func (p *PipelinePostProcessor) ProcessResults(ctx context.Context, query string, results []SearchResult, options *ProcessingOptions) (*ProcessedResults, error) {
	startTime := time.Now()
	
	if options == nil {
		options = p.config.DefaultOptions
	}

	originalCount := len(results)
	processedResults := make([]SearchResult, len(results))
	copy(processedResults, results)

	summary := ProcessingSummary{
		TechniquesApplied: []string{},
		OriginalCount:     originalCount,
	}

	// Apply each step in the pipeline
	for _, step := range p.pipeline {
		if !step.ShouldApply(options) {
			continue
		}

		var err error
		processedResults, err = step.Process(ctx, query, processedResults, options)
		if err != nil {
			return nil, fmt.Errorf("step %s failed: %w", step.Name(), err)
		}

		summary.TechniquesApplied = append(summary.TechniquesApplied, step.Name())
	}

	summary.FinalCount = len(processedResults)
	summary.ProcessingTime = time.Since(startTime)

	return &ProcessedResults{
		Query:             query,
		OriginalCount:     originalCount,
		FinalResults:      processedResults,
		ProcessingSummary: summary,
		ProcessedAt:       time.Now(),
	}, nil
}

// Standard processing steps

// FilteringStep implements filtering as a pipeline step
type FilteringStep struct {
	filter Filter
}

func NewFilteringStep(config *PostProcessingConfig) *FilteringStep {
	return &FilteringStep{
		filter: NewStandardFilter(config),
	}
}

func (s *FilteringStep) Name() string {
	return "filtering"
}

func (s *FilteringStep) Process(ctx context.Context, query string, results []SearchResult, options *ProcessingOptions) ([]SearchResult, error) {
	return s.filter.Filter(ctx, query, results, options.FilteringOptions)
}

func (s *FilteringStep) ShouldApply(options *ProcessingOptions) bool {
	return options.EnableFiltering
}

// DeduplicationStep implements deduplication as a pipeline step
type DeduplicationStep struct {
	deduplicator Deduplicator
}

func NewDeduplicationStep(config *PostProcessingConfig) *DeduplicationStep {
	return &DeduplicationStep{
		deduplicator: NewStandardDeduplicator(config),
	}
}

func (s *DeduplicationStep) Name() string {
	return "deduplication"
}

func (s *DeduplicationStep) Process(ctx context.Context, query string, results []SearchResult, options *ProcessingOptions) ([]SearchResult, error) {
	return s.deduplicator.Deduplicate(ctx, results, options.DeduplicationOptions)
}

func (s *DeduplicationStep) ShouldApply(options *ProcessingOptions) bool {
	return options.EnableDeduplication
}

// RerankingStep implements reranking as a pipeline step
type RerankingStep struct {
	reranker Reranker
}

func NewRerankingStep(config *PostProcessingConfig) *RerankingStep {
	return &RerankingStep{
		reranker: NewStandardReranker(config),
	}
}

func (s *RerankingStep) Name() string {
	return "reranking"
}

func (s *RerankingStep) Process(ctx context.Context, query string, results []SearchResult, options *ProcessingOptions) ([]SearchResult, error) {
	return s.reranker.Rerank(ctx, query, results, options.RerankingOptions)
}

func (s *RerankingStep) ShouldApply(options *ProcessingOptions) bool {
	return options.EnableReranking
}

// LimitingStep implements result limiting as a pipeline step
type LimitingStep struct{}

func NewLimitingStep() *LimitingStep {
	return &LimitingStep{}
}

func (s *LimitingStep) Name() string {
	return "limiting"
}

func (s *LimitingStep) Process(ctx context.Context, query string, results []SearchResult, options *ProcessingOptions) ([]SearchResult, error) {
	if options.MaxResults > 0 && len(results) > options.MaxResults {
		// Sort by score before limiting to ensure best results are kept
		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})
		return results[:options.MaxResults], nil
	}
	return results, nil
}

func (s *LimitingStep) ShouldApply(options *ProcessingOptions) bool {
	return options.MaxResults > 0
}

// CreateDefaultPipeline creates a standard processing pipeline
func CreateDefaultPipeline(config *PostProcessingConfig) *PipelinePostProcessor {
	steps := []ProcessingStep{
		NewFilteringStep(config),
		NewDeduplicationStep(config),
		NewRerankingStep(config),
		NewLimitingStep(),
	}

	return NewPipelinePostProcessor(config, steps...)
}