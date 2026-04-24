package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	docsBaseURL        = "https://tharsis.martian-cloud.io"
	searchIndexURL     = docsBaseURL + "/search-index.json"
	defaultSearchLimit = 10
	maxSearchLimit     = 100
	maxIndexSize       = 4 * 1024 * 1024 // 4 MiB
	maxPageSize        = 2 * 1024 * 1024 // 2 MiB
)

// docSearchDocument represents a document from the Docusaurus search index.
type docSearchDocument struct {
	Title       string   `json:"t"`
	URL         string   `json:"u"`
	Breadcrumbs []string `json:"b"`
	Section     string   `json:"s,omitempty"`
}

// searchIndexEntry represents one entry in the Docusaurus search index array.
type searchIndexEntry struct {
	Documents []docSearchDocument `json:"documents"`
}

// searchIndex holds the parsed search index data.
type searchIndex struct {
	documents    []docSearchDocument
	keywords     map[string]string
	descriptions map[string]string
}

// docResult represents a single search result.
type docResult struct {
	Title       string   `json:"title" jsonschema:"Page title"`
	URL         string   `json:"url" jsonschema:"Full URL to the documentation page"`
	Breadcrumbs []string `json:"breadcrumbs" jsonschema:"Navigation breadcrumb trail"`
	Section     string   `json:"section,omitempty" jsonschema:"Section title within the page"`
	Relevance   float64  `json:"relevance" jsonschema:"Search relevance score"`
}

// documentRepository handles fetching documentation data.
type documentRepository interface {
	getSearchIndex(ctx context.Context) (*searchIndex, error)
	getPageContent(ctx context.Context, urlPath string) (string, error)
}

// httpDocumentRepository fetches documentation via HTTP with caching.
type httpDocumentRepository struct {
	mu     sync.RWMutex
	client *http.Client
	cached *searchIndex
}

// newHTTPDocumentRepository creates a new HTTP-based repository.
func newHTTPDocumentRepository(client *http.Client) *httpDocumentRepository {
	return &httpDocumentRepository{client: client}
}

// getSearchIndex fetches the search index once and caches it.
func (r *httpDocumentRepository) getSearchIndex(ctx context.Context) (*searchIndex, error) {
	r.mu.RLock()
	if r.cached != nil {
		defer r.mu.RUnlock()
		return r.cached, nil
	}
	r.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchIndexURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var indexes []searchIndexEntry
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxIndexSize)).Decode(&indexes); err != nil {
		return nil, err
	}

	if len(indexes) == 0 || len(indexes[0].Documents) == 0 {
		return nil, fmt.Errorf("empty search index")
	}

	// Index 0 = titles, 1 = headings, 2 = descriptions, 3 = keywords, 4 = content.
	idx := &searchIndex{
		documents:    indexes[0].Documents,
		keywords:     extractMetadata(indexes, 3),
		descriptions: extractMetadata(indexes, 2),
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Re-check after acquiring write lock to avoid redundant fetches.
	if r.cached != nil {
		return r.cached, nil
	}

	r.cached = idx

	return idx, nil
}

// getPageContent fetches markdown content by appending .md to the URL.
func (r *httpDocumentRepository) getPageContent(ctx context.Context, urlPath string) (string, error) {
	fullURL, err := url.JoinPath(docsBaseURL, urlPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	if !strings.HasSuffix(fullURL, ".md") {
		fullURL += ".md"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/") && !strings.Contains(contentType, "markdown") {
		return "", fmt.Errorf("unexpected content type: %s, expected markdown", contentType)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageSize))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

// DocumentSearchService handles search operations using Bleve.
type DocumentSearchService struct {
	mu        sync.RWMutex
	repo      documentRepository
	index     bleve.Index
	documents map[string]*docSearchDocument
}

// NewDocumentSearchService creates a new search service with an HTTP repository.
func NewDocumentSearchService(client *http.Client) *DocumentSearchService {
	return &DocumentSearchService{
		repo:      newHTTPDocumentRepository(client),
		documents: make(map[string]*docSearchDocument),
	}
}

// ensureIndexReady builds the search index if not already initialized.
func (s *DocumentSearchService) ensureIndexReady(ctx context.Context) error {
	s.mu.RLock()
	hasIndex := s.index != nil
	s.mu.RUnlock()

	if hasIndex {
		return nil
	}

	idx, err := s.repo.getSearchIndex(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch search index: %w", err)
	}

	// Re-check under write path; another goroutine may have built it.
	s.mu.RLock()
	hasIndex = s.index != nil
	s.mu.RUnlock()

	if hasIndex {
		return nil
	}

	return s.buildIndex(idx)
}

// buildIndex creates a new in-memory Bleve index from the search index data.
func (s *DocumentSearchService) buildIndex(idx *searchIndex) error {
	index, err := bleve.NewMemOnly(bleve.NewIndexMapping())
	if err != nil {
		return err
	}

	newDocuments := make(map[string]*docSearchDocument, len(idx.documents))
	for i := range idx.documents {
		doc := &idx.documents[i]
		newDocuments[doc.URL] = doc

		if err := index.Index(doc.URL, map[string]interface{}{
			"title":       doc.Title,
			"breadcrumbs": strings.Join(doc.Breadcrumbs, " "),
			"section":     doc.Section,
			"keywords":    idx.keywords[doc.URL],
			"description": idx.descriptions[doc.URL],
		}); err != nil {
			index.Close()
			return err
		}
	}

	s.mu.Lock()
	if s.index != nil {
		s.index.Close()
	}
	s.index = index
	s.documents = newDocuments
	s.mu.Unlock()

	return nil
}

// search performs a full-text search with field boosting and fuzzy matching.
func (s *DocumentSearchService) search(queryStr string, limit int) ([]docResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fuzzyQuery := bleve.NewMatchQuery(queryStr)
	fuzzyQuery.SetFuzziness(1)

	q := bleve.NewDisjunctionQuery(
		boostedMatchQuery(queryStr, "title", 3.0),
		boostedMatchQuery(queryStr, "keywords", 2.0),
		boostedMatchQuery(queryStr, "breadcrumbs", 1.5),
		boostedMatchQuery(queryStr, "description", 1.5),
		boostedMatchQuery(queryStr, "section", 1.0),
		fuzzyQuery,
	)

	searchRequest := bleve.NewSearchRequest(q)
	searchRequest.Size = limit

	searchResults, err := s.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	results := make([]docResult, 0, len(searchResults.Hits))
	for _, hit := range searchResults.Hits {
		if doc := s.documents[hit.ID]; doc != nil {
			results = append(results, docResult{
				Title:       doc.Title,
				URL:         docsBaseURL + doc.URL,
				Breadcrumbs: doc.Breadcrumbs,
				Section:     doc.Section,
				Relevance:   hit.Score,
			})
		}
	}

	return results, nil
}

// getDocument retrieves a document by URL path.
func (s *DocumentSearchService) getDocument(urlPath string) *docSearchDocument {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.documents[urlPath]
}

// searchDocumentationInput is the input for the search_documentation tool.
type searchDocumentationInput struct {
	Query string `json:"query" jsonschema:"required,Search query for documentation"`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of results to return (default: 10)"`
}

// searchDocumentationOutput is the output for the search_documentation tool.
type searchDocumentationOutput struct {
	Results []docResult `json:"results" jsonschema:"List of matching documentation pages"`
}

// SearchDocumentation returns an MCP tool for searching documentation.
func SearchDocumentation(service *DocumentSearchService) (mcp.Tool, mcp.ToolHandlerFor[searchDocumentationInput, searchDocumentationOutput]) {
	tool := mcp.Tool{
		Name:        "search_documentation",
		Description: "Search Tharsis documentation by keywords. Returns matching pages with titles, URLs, and breadcrumbs.",
		Annotations: &mcp.ToolAnnotations{
			Title:          "Search Documentation",
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input searchDocumentationInput) (*mcp.CallToolResult, searchDocumentationOutput, error) {
		if err := service.ensureIndexReady(ctx); err != nil {
			return nil, searchDocumentationOutput{}, fmt.Errorf("failed to load documentation index: %w", err)
		}

		if strings.TrimSpace(input.Query) == "" {
			return nil, searchDocumentationOutput{}, fmt.Errorf("search query cannot be empty")
		}

		limit := input.Limit
		if limit <= 0 {
			limit = defaultSearchLimit
		} else if limit > maxSearchLimit {
			limit = maxSearchLimit
		}

		results, err := service.search(input.Query, limit)
		if err != nil {
			return nil, searchDocumentationOutput{}, fmt.Errorf("search failed: %w", err)
		}

		return nil, searchDocumentationOutput{Results: results}, nil
	}

	return tool, handler
}

// getDocumentationPageInput is the input for the get_documentation_page tool.
type getDocumentationPageInput struct {
	URL string `json:"url" jsonschema:"required,Full documentation page URL from search results (e.g. https://tharsis.martian-cloud.io/cli/tharsis/commands)"`
}

// getDocumentationPageOutput is the output for the get_documentation_page tool.
type getDocumentationPageOutput struct {
	Title       string   `json:"title" jsonschema:"Page title"`
	URL         string   `json:"url" jsonschema:"Full URL to the documentation page"`
	Breadcrumbs []string `json:"breadcrumbs" jsonschema:"Navigation breadcrumb trail"`
	Content     string   `json:"content" jsonschema:"Full text content of the page"`
}

// GetDocumentationPage returns an MCP tool for fetching documentation page content.
func GetDocumentationPage(service *DocumentSearchService) (mcp.Tool, mcp.ToolHandlerFor[getDocumentationPageInput, getDocumentationPageOutput]) {
	tool := mcp.Tool{
		Name:        "get_documentation_page",
		Description: "Fetch the full content of a Tharsis documentation page. Use after searching to get detailed information.",
		Annotations: &mcp.ToolAnnotations{
			Title:          "Get Documentation Page",
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getDocumentationPageInput) (*mcp.CallToolResult, getDocumentationPageOutput, error) {
		if err := service.ensureIndexReady(ctx); err != nil {
			return nil, getDocumentationPageOutput{}, fmt.Errorf("failed to load documentation index: %w", err)
		}

		parsedURL, err := url.ParseRequestURI(input.URL)
		if err != nil {
			return nil, getDocumentationPageOutput{}, fmt.Errorf("invalid URL format '%s': must be a full URL (e.g. https://tharsis.martian-cloud.io/path)", input.URL)
		}

		if parsedURL.Host != "" && "https://"+parsedURL.Host != docsBaseURL {
			return nil, getDocumentationPageOutput{}, fmt.Errorf("URL must be from %s", docsBaseURL)
		}

		cleanPath := path.Clean(parsedURL.Path)

		doc := service.getDocument(cleanPath)
		if doc == nil {
			return nil, getDocumentationPageOutput{}, fmt.Errorf("documentation page not found at path '%s'. Use search_documentation to find valid pages", cleanPath)
		}

		content, err := service.repo.getPageContent(ctx, cleanPath)
		if err != nil {
			return nil, getDocumentationPageOutput{}, fmt.Errorf("failed to fetch page content: %w", err)
		}

		return nil, getDocumentationPageOutput{
			Title:       doc.Title,
			URL:         docsBaseURL + doc.URL,
			Breadcrumbs: doc.Breadcrumbs,
			Content:     content,
		}, nil
	}

	return tool, handler
}

// boostedMatchQuery creates a match query on a specific field with a boost value.
func boostedMatchQuery(queryStr, field string, boost float64) *query.MatchQuery {
	q := bleve.NewMatchQuery(queryStr)
	q.SetField(field)
	q.SetBoost(boost)
	return q
}

// extractMetadata pulls URL -> title mappings from a specific index array.
func extractMetadata(indexes []searchIndexEntry, i int) map[string]string {
	m := map[string]string{}
	if i < len(indexes) {
		for _, doc := range indexes[i].Documents {
			m[doc.URL] = doc.Title
		}
	}
	return m
}
