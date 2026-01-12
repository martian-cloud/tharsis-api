package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/blevesearch/bleve/v2"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	docsBaseURL        = "https://tharsis.martian-cloud.io"
	searchIndexURL     = docsBaseURL + "/search-index.json"
	defaultSearchLimit = 10
	maxSearchLimit     = 100
	indexDirPrefix     = "tharsis-docs-index-"
)

// docSearchDocument represents a document from the Docusaurus search index.
type docSearchDocument struct {
	Title       string   `json:"t"`
	URL         string   `json:"u"`
	Breadcrumbs []string `json:"b"`
	Section     string   `json:"s,omitempty"`
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
	getSearchIndex(ctx context.Context) ([]docSearchDocument, error)
	getPageContent(ctx context.Context, urlPath string) (string, error)
}

// httpDocumentRepository fetches documentation via HTTP with caching.
type httpDocumentRepository struct {
	mu         sync.RWMutex
	client     *http.Client
	cachedDocs []docSearchDocument
	converter  *md.Converter
}

// newHTTPDocumentRepository creates a new HTTP-based repository.
func newHTTPDocumentRepository(client *http.Client) *httpDocumentRepository {
	return &httpDocumentRepository{
		client:    client,
		converter: md.NewConverter(docsBaseURL, true, nil),
	}
}

// getSearchIndex fetches the search index once and caches it.
func (r *httpDocumentRepository) getSearchIndex(ctx context.Context) ([]docSearchDocument, error) {
	r.mu.RLock()
	if r.cachedDocs != nil {
		defer r.mu.RUnlock()
		return r.cachedDocs, nil
	}
	r.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchIndexURL, nil)
	if err != nil {
		return nil, err
	}

	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var indexes []struct {
		Documents []docSearchDocument `json:"documents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&indexes); err != nil {
		return nil, err
	}

	if len(indexes) == 0 || len(indexes[0].Documents) == 0 {
		return nil, fmt.Errorf("empty search index")
	}

	r.mu.Lock()
	r.cachedDocs = indexes[0].Documents
	r.mu.Unlock()

	return indexes[0].Documents, nil
}

// getPageContent fetches and extracts formatted content from a documentation page.
func (r *httpDocumentRepository) getPageContent(ctx context.Context, urlPath string) (string, error) {
	fullURL, err := url.JoinPath(docsBaseURL, urlPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return "", err
	}

	r.mu.RLock()
	client := r.client
	converter := r.converter
	r.mu.RUnlock()

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	markdown, err := converter.ConvertResponse(resp)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(markdown), nil
}

// DocumentSearchService handles search operations using Bleve.
type DocumentSearchService struct {
	mu        sync.RWMutex
	repo      documentRepository
	index     bleve.Index
	documents map[string]*docSearchDocument
	indexPath string
}

// NewDocumentSearchService creates a new search service with an HTTP repository.
func NewDocumentSearchService(client *http.Client) (*DocumentSearchService, error) {
	// Create unique temp directory for this instance
	indexPath, err := os.MkdirTemp("", indexDirPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory for search index: %w", err)
	}

	return &DocumentSearchService{
		repo:      newHTTPDocumentRepository(client),
		documents: make(map[string]*docSearchDocument),
		indexPath: indexPath,
	}, nil
}

// EnsureIndexReady builds the search index if not already initialized.
func (s *DocumentSearchService) ensureIndexReady(ctx context.Context) error {
	s.mu.RLock()
	hasIndex := s.index != nil
	s.mu.RUnlock()

	if hasIndex {
		return nil
	}

	docs, err := s.repo.getSearchIndex(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch search index: %w", err)
	}

	return s.buildIndex(docs)
}

// buildIndex creates a new Bleve index from documentation documents.
func (s *DocumentSearchService) buildIndex(documents []docSearchDocument) error {
	index, err := bleve.New(s.indexPath, bleve.NewIndexMapping())
	if err != nil {
		return err
	}

	newDocuments := make(map[string]*docSearchDocument, len(documents))
	for i := range documents {
		doc := &documents[i]
		newDocuments[doc.URL] = doc

		if err := index.Index(doc.URL, map[string]interface{}{
			"title":       doc.Title,
			"breadcrumbs": strings.Join(doc.Breadcrumbs, " "),
			"section":     doc.Section,
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

// Search performs a full-text search with field boosting and fuzzy matching.
func (s *DocumentSearchService) search(queryStr string, limit int) ([]docResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	titleMatch := bleve.NewMatchQuery(queryStr)
	titleMatch.SetField("title")
	titleMatch.SetBoost(3.0)

	breadcrumbsMatch := bleve.NewMatchQuery(queryStr)
	breadcrumbsMatch.SetField("breadcrumbs")
	breadcrumbsMatch.SetBoost(1.5)

	sectionMatch := bleve.NewMatchQuery(queryStr)
	sectionMatch.SetField("section")

	fuzzyQuery := bleve.NewMatchQuery(queryStr)
	fuzzyQuery.SetFuzziness(1)

	query := bleve.NewDisjunctionQuery(titleMatch, breadcrumbsMatch, sectionMatch, fuzzyQuery)
	searchRequest := bleve.NewSearchRequest(query)
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
