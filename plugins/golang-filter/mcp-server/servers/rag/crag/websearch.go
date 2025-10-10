package crag

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DuckDuckGoSearcher implements web search using DuckDuckGo
type DuckDuckGoSearcher struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
}

// NewDuckDuckGoSearcher creates a new DuckDuckGo web searcher
func NewDuckDuckGoSearcher() *DuckDuckGoSearcher {
	return &DuckDuckGoSearcher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:   "https://api.duckduckgo.com/",
		userAgent: "Mozilla/5.0 (compatible; RAG-Agent/1.0)",
	}
}

// Search performs web search and returns relevant documents
func (d *DuckDuckGoSearcher) Search(ctx context.Context, query string, maxResults int) ([]WebDocument, error) {
	return d.SearchWithFilters(ctx, query, &SearchFilters{
		MaxResults: maxResults,
	})
}

// SearchWithFilters performs web search with domain/content filters
func (d *DuckDuckGoSearcher) SearchWithFilters(ctx context.Context, query string, filters *SearchFilters) ([]WebDocument, error) {
	if filters == nil {
		filters = &SearchFilters{MaxResults: 5}
	}
	
	// Build search URL
	searchURL, err := d.buildSearchURL(query, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", d.userAgent)
	
	// Execute request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned status %d", resp.StatusCode)
	}
	
	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	return d.parseSearchResponse(body, filters.MaxResults)
}

// buildSearchURL constructs the search URL with parameters
func (d *DuckDuckGoSearcher) buildSearchURL(query string, filters *SearchFilters) (string, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	params.Set("skip_disambig", "1")
	
	// Apply filters
	if len(filters.Domains) > 0 {
		siteQuery := ""
		for _, domain := range filters.Domains {
			if siteQuery != "" {
				siteQuery += " OR "
			}
			siteQuery += "site:" + domain
		}
		params.Set("q", fmt.Sprintf("%s (%s)", query, siteQuery))
	}
	
	return d.baseURL + "?" + params.Encode(), nil
}

// parseSearchResponse parses DuckDuckGo API response
func (d *DuckDuckGoSearcher) parseSearchResponse(body []byte, maxResults int) ([]WebDocument, error) {
	var response struct {
		AbstractText   string `json:"AbstractText"`
		AbstractURL    string `json:"AbstractURL"`
		AbstractSource string `json:"AbstractSource"`
		Results        []struct {
			Text      string `json:"Text"`
			FirstURL  string `json:"FirstURL"`
		} `json:"Results"`
		RelatedTopics []struct {
			Text      string `json:"Text"`
			FirstURL  string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}
	
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	
	var documents []WebDocument
	now := time.Now()
	
	// Add abstract if available
	if response.AbstractText != "" && response.AbstractURL != "" {
		documents = append(documents, WebDocument{
			Title:       "Abstract - " + response.AbstractSource,
			Content:     response.AbstractText,
			URL:         response.AbstractURL,
			Score:       0.9, // High score for abstract
			Source:      "duckduckgo_abstract",
			Snippet:     d.truncateText(response.AbstractText, 200),
			RetrievedAt: now,
		})
	}
	
	// Add results
	for i, result := range response.Results {
		if len(documents) >= maxResults {
			break
		}
		
		if result.Text != "" && result.FirstURL != "" {
			documents = append(documents, WebDocument{
				Title:       d.extractTitle(result.Text),
				Content:     result.Text,
				URL:         result.FirstURL,
				Score:       0.8 - float64(i)*0.1, // Decreasing score
				Source:      "duckduckgo_results",
				Snippet:     d.truncateText(result.Text, 200),
				RetrievedAt: now,
			})
		}
	}
	
	// Add related topics if needed
	for i, topic := range response.RelatedTopics {
		if len(documents) >= maxResults {
			break
		}
		
		if topic.Text != "" && topic.FirstURL != "" {
			documents = append(documents, WebDocument{
				Title:       d.extractTitle(topic.Text),
				Content:     topic.Text,
				URL:         topic.FirstURL,
				Score:       0.6 - float64(i)*0.05, // Lower score for related topics
				Source:      "duckduckgo_related",
				Snippet:     d.truncateText(topic.Text, 200),
				RetrievedAt: now,
			})
		}
	}
	
	return documents, nil
}

// extractTitle extracts title from text (first part before dash or period)
func (d *DuckDuckGoSearcher) extractTitle(text string) string {
	// Try to extract title from the beginning of text
	if dashIndex := strings.Index(text, " - "); dashIndex != -1 && dashIndex < 100 {
		return strings.TrimSpace(text[:dashIndex])
	}
	
	if periodIndex := strings.Index(text, ". "); periodIndex != -1 && periodIndex < 100 {
		return strings.TrimSpace(text[:periodIndex])
	}
	
	// Fallback: use first 50 characters
	if len(text) > 50 {
		return strings.TrimSpace(text[:50]) + "..."
	}
	
	return text
}

// truncateText truncates text to specified length
func (d *DuckDuckGoSearcher) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	
	truncated := text[:maxLen]
	
	// Try to cut at word boundary
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLen-20 {
		truncated = truncated[:lastSpace]
	}
	
	return truncated + "..."
}

// MockWebSearcher implements WebSearcher for testing
type MockWebSearcher struct {
	searchResults map[string][]WebDocument
}

// NewMockWebSearcher creates a new mock web searcher
func NewMockWebSearcher() *MockWebSearcher {
	return &MockWebSearcher{
		searchResults: make(map[string][]WebDocument),
	}
}

// AddMockResult adds a mock search result
func (m *MockWebSearcher) AddMockResult(query string, docs []WebDocument) {
	m.searchResults[query] = docs
}

// Search performs mock web search
func (m *MockWebSearcher) Search(ctx context.Context, query string, maxResults int) ([]WebDocument, error) {
	return m.SearchWithFilters(ctx, query, &SearchFilters{MaxResults: maxResults})
}

// SearchWithFilters performs mock web search with filters
func (m *MockWebSearcher) SearchWithFilters(ctx context.Context, query string, filters *SearchFilters) ([]WebDocument, error) {
	if results, exists := m.searchResults[query]; exists {
		maxResults := filters.MaxResults
		if maxResults == 0 {
			maxResults = len(results)
		}
		
		if len(results) > maxResults {
			return results[:maxResults], nil
		}
		return results, nil
	}
	
	// Return empty results for unknown queries
	return []WebDocument{}, nil
}

// SimpleWebSearcher provides a basic web search implementation using HTTP requests
type SimpleWebSearcher struct {
	httpClient *http.Client
	searchURL  string
	apiKey     string
}

// NewSimpleWebSearcher creates a simple web searcher
func NewSimpleWebSearcher(searchURL, apiKey string) *SimpleWebSearcher {
	return &SimpleWebSearcher{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		searchURL:  searchURL,
		apiKey:     apiKey,
	}
}

// Search performs web search using configured search service
func (s *SimpleWebSearcher) Search(ctx context.Context, query string, maxResults int) ([]WebDocument, error) {
	// This is a placeholder implementation
	// In practice, you would integrate with actual search APIs like:
	// - Bing Search API
	// - Google Custom Search API
	// - SerpAPI
	// - etc.
	
	return []WebDocument{
		{
			Title:       "Sample Web Result 1",
			Content:     "This is a sample web search result for: " + query,
			URL:         "https://example.com/1",
			Score:       0.8,
			Source:      "simple_web_search",
			Snippet:     "Sample snippet for " + query,
			RetrievedAt: time.Now(),
		},
		{
			Title:       "Sample Web Result 2", 
			Content:     "Another sample web search result for: " + query,
			URL:         "https://example.com/2",
			Score:       0.7,
			Source:      "simple_web_search",
			Snippet:     "Another snippet for " + query,
			RetrievedAt: time.Now(),
		},
	}, nil
}

// SearchWithFilters performs web search with filters
func (s *SimpleWebSearcher) SearchWithFilters(ctx context.Context, query string, filters *SearchFilters) ([]WebDocument, error) {
	// Apply filters to the query or request
	results, err := s.Search(ctx, query, filters.MaxResults)
	if err != nil {
		return nil, err
	}
	
	// Filter results based on domain restrictions
	if len(filters.Domains) > 0 || len(filters.ExcludeDomains) > 0 {
		var filteredResults []WebDocument
		for _, result := range results {
			if s.shouldIncludeResult(result, filters) {
				filteredResults = append(filteredResults, result)
			}
		}
		results = filteredResults
	}
	
	return results, nil
}

// shouldIncludeResult checks if a result should be included based on filters
func (s *SimpleWebSearcher) shouldIncludeResult(result WebDocument, filters *SearchFilters) bool {
	parsedURL, err := url.Parse(result.URL)
	if err != nil {
		return false
	}
	
	domain := parsedURL.Hostname()
	
	// Check excluded domains
	for _, excludeDomain := range filters.ExcludeDomains {
		if strings.Contains(domain, excludeDomain) {
			return false
		}
	}
	
	// Check allowed domains (if specified)
	if len(filters.Domains) > 0 {
		allowed := false
		for _, allowedDomain := range filters.Domains {
			if strings.Contains(domain, allowedDomain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	
	return true
}