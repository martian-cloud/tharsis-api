package tools

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDocumentRepository struct {
	searchIndexFunc func(ctx context.Context) ([]docSearchDocument, error)
	pageContentFunc func(ctx context.Context, urlPath string) (string, error)
}

func (m *mockDocumentRepository) getSearchIndex(ctx context.Context) ([]docSearchDocument, error) {
	return m.searchIndexFunc(ctx)
}

func (m *mockDocumentRepository) getPageContent(ctx context.Context, urlPath string) (string, error) {
	return m.pageContentFunc(ctx, urlPath)
}

type failingTransport struct{}

func (f *failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network error")
}

func TestDocumentSearchService(t *testing.T) {
	t.Run("initialization", func(t *testing.T) {
		service, err := NewDocumentSearchService(nil)
		require.NoError(t, err)
		require.NotNil(t, service)
		assert.Contains(t, service.indexPath, indexDirPrefix)
		os.RemoveAll(service.indexPath)
	})

	t.Run("buildIndex", func(t *testing.T) {
		service, _ := NewDocumentSearchService(nil)
		defer os.RemoveAll(service.indexPath)

		docs := []docSearchDocument{
			{Title: "Getting Started", URL: "/docs/intro", Breadcrumbs: []string{"Home", "Docs"}},
			{Title: "CLI Commands", URL: "/cli/commands", Breadcrumbs: []string{"CLI"}},
		}

		err := service.buildIndex(docs)
		require.NoError(t, err)
		assert.NotNil(t, service.index)
		assert.Len(t, service.documents, 2)
	})

	t.Run("ensureIndexReady", func(t *testing.T) {
		type testCase struct {
			name      string
			setupFunc func() *DocumentSearchService
			wantErr   bool
		}
		tests := []testCase{
			{
				name: "builds index on first call",
				setupFunc: func() *DocumentSearchService {
					s := &DocumentSearchService{
						repo: &mockDocumentRepository{
							searchIndexFunc: func(_ context.Context) ([]docSearchDocument, error) {
								return []docSearchDocument{{Title: "Test", URL: "/test", Breadcrumbs: []string{"Test"}}}, nil
							},
						},
						documents: make(map[string]*docSearchDocument),
					}
					s.indexPath, _ = os.MkdirTemp("", indexDirPrefix)
					return s
				},
			},
			{
				name: "reuses existing index",
				setupFunc: func() *DocumentSearchService {
					s, _ := NewDocumentSearchService(nil)
					s.buildIndex([]docSearchDocument{{Title: "Test", URL: "/test", Breadcrumbs: []string{"Test"}}})
					return s
				},
			},
			{
				name: "returns error when fetch fails",
				setupFunc: func() *DocumentSearchService {
					s := &DocumentSearchService{
						repo: &mockDocumentRepository{
							searchIndexFunc: func(_ context.Context) ([]docSearchDocument, error) {
								return nil, errors.New("fetch failed")
							},
						},
						documents: make(map[string]*docSearchDocument),
					}
					s.indexPath, _ = os.MkdirTemp("", indexDirPrefix)
					return s
				},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				service := tt.setupFunc()
				defer os.RemoveAll(service.indexPath)

				err := service.ensureIndexReady(context.Background())
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, service.index)
				}
			})
		}
	})

	t.Run("search", func(t *testing.T) {
		service, _ := NewDocumentSearchService(nil)
		defer os.RemoveAll(service.indexPath)

		docs := []docSearchDocument{
			{Title: "Getting Started", URL: "/docs/intro", Breadcrumbs: []string{"Home", "Docs"}},
			{Title: "CLI Commands", URL: "/cli/commands", Breadcrumbs: []string{"CLI"}, Section: "Overview"},
			{Title: "API Reference", URL: "/api/reference", Breadcrumbs: []string{"API"}},
		}
		service.buildIndex(docs)

		type testCase struct {
			name      string
			query     string
			limit     int
			wantEmpty bool
		}
		tests := []testCase{
			{name: "exact title match", query: "CLI Commands", limit: 10},
			{name: "partial match", query: "CLI", limit: 10},
			{name: "respects limit", query: "docs", limit: 1},
			{name: "no matches", query: "nonexistent", limit: 10, wantEmpty: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				results, err := service.search(tt.query, tt.limit)
				require.NoError(t, err)
				if tt.wantEmpty {
					assert.Empty(t, results)
				} else {
					assert.NotEmpty(t, results)
					if tt.limit > 0 {
						assert.LessOrEqual(t, len(results), tt.limit)
					}
				}
			})
		}
	})

	t.Run("getDocument", func(t *testing.T) {
		service, _ := NewDocumentSearchService(nil)
		defer os.RemoveAll(service.indexPath)

		service.buildIndex([]docSearchDocument{
			{Title: "Test Doc", URL: "/test", Breadcrumbs: []string{"Test"}},
		})

		doc := service.getDocument("/test")
		assert.NotNil(t, doc)
		assert.Equal(t, "Test Doc", doc.Title)

		doc = service.getDocument("/nonexistent")
		assert.Nil(t, doc)
	})

	t.Run("edge cases", func(t *testing.T) {
		service, _ := NewDocumentSearchService(nil)
		defer os.RemoveAll(service.indexPath)

		// Empty documents list
		err := service.buildIndex([]docSearchDocument{})
		assert.NoError(t, err)

		// Search with special characters
		service.buildIndex([]docSearchDocument{
			{Title: "Test & Special <chars>", URL: "/test", Breadcrumbs: []string{"Test"}},
		})
		_, err = service.search("&", 10)
		assert.NoError(t, err)
	})
}
func TestHTTPDocumentRepository(t *testing.T) {
	t.Run("getSearchIndex caching", func(t *testing.T) {
		repo := newHTTPDocumentRepository(nil)
		repo.cachedDocs = []docSearchDocument{
			{Title: "Test", URL: "/test", Breadcrumbs: []string{"Home"}},
		}

		docs, err := repo.getSearchIndex(context.Background())
		require.NoError(t, err)
		assert.Len(t, docs, 1)

		docs2, err := repo.getSearchIndex(context.Background())
		require.NoError(t, err)
		assert.Equal(t, docs, docs2)
	})

	t.Run("getPageContent request error", func(t *testing.T) {
		client := &http.Client{
			Transport: &failingTransport{},
		}
		repo := newHTTPDocumentRepository(client)
		_, err := repo.getPageContent(context.Background(), "/test")
		assert.Error(t, err)
	})
}

func TestSearchDocumentationTool(t *testing.T) {
	service := &DocumentSearchService{
		repo: &mockDocumentRepository{
			searchIndexFunc: func(_ context.Context) ([]docSearchDocument, error) {
				return []docSearchDocument{{Title: "Test", URL: "/test", Breadcrumbs: []string{"Test"}}}, nil
			},
		},
		documents: make(map[string]*docSearchDocument),
	}
	service.indexPath, _ = os.MkdirTemp("", indexDirPrefix)
	defer os.RemoveAll(service.indexPath)

	tool, handler := SearchDocumentation(service)

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "search_documentation", tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.True(t, tool.Annotations.ReadOnlyHint)
	})

	service.buildIndex([]docSearchDocument{
		{Title: "Test Doc", URL: "/test", Breadcrumbs: []string{"Test"}},
	})

	type testCase struct {
		name    string
		input   searchDocumentationInput
		wantErr bool
	}
	tests := []testCase{
		{name: "successful search", input: searchDocumentationInput{Query: "Test", Limit: 10}},
		{name: "empty query", input: searchDocumentationInput{Query: "   "}, wantErr: true},
		{name: "default limit", input: searchDocumentationInput{Query: "Test", Limit: 0}},
		{name: "max limit", input: searchDocumentationInput{Query: "Test", Limit: 200}},
		{name: "negative limit", input: searchDocumentationInput{Query: "Test", Limit: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, output, err := handler(context.Background(), nil, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, output)
			}
		})
	}
}

func TestGetDocumentationPageTool(t *testing.T) {
	service := &DocumentSearchService{
		repo: &mockDocumentRepository{
			searchIndexFunc: func(_ context.Context) ([]docSearchDocument, error) {
				return []docSearchDocument{{Title: "Test", URL: "/test", Breadcrumbs: []string{"Test"}}}, nil
			},
			pageContentFunc: func(_ context.Context, _ string) (string, error) {
				return "Test content", nil
			},
		},
		documents: make(map[string]*docSearchDocument),
	}
	service.indexPath, _ = os.MkdirTemp("", indexDirPrefix)
	defer os.RemoveAll(service.indexPath)

	tool, handler := GetDocumentationPage(service)

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "get_documentation_page", tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.True(t, tool.Annotations.ReadOnlyHint)
	})

	service.buildIndex([]docSearchDocument{
		{Title: "Test Doc", URL: "/test", Breadcrumbs: []string{"Test"}},
	})

	type testCase struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}
	tests := []testCase{
		{name: "successful fetch", url: "https://tharsis.martian-cloud.io/test"},
		{name: "invalid URL", url: "not-a-url", wantErr: true, errMsg: "invalid URL format"},
		{name: "page not found", url: "https://tharsis.martian-cloud.io/nonexistent", wantErr: true, errMsg: "not found"},
		{name: "empty URL", url: "", wantErr: true},
		{name: "URL with query params", url: "https://tharsis.martian-cloud.io/test?foo=bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, output, err := handler(context.Background(), nil, getDocumentationPageInput{URL: tt.url})
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "Test Doc", output.Title)
				assert.Equal(t, "Test content", output.Content)
			}
		})
	}

	t.Run("content fetch error", func(t *testing.T) {
		service.repo = &mockDocumentRepository{
			searchIndexFunc: func(_ context.Context) ([]docSearchDocument, error) {
				return []docSearchDocument{{Title: "Test", URL: "/test", Breadcrumbs: []string{"Test"}}}, nil
			},
			pageContentFunc: func(_ context.Context, _ string) (string, error) {
				return "", errors.New("fetch failed")
			},
		}
		service.buildIndex([]docSearchDocument{{Title: "Test", URL: "/test", Breadcrumbs: []string{"Test"}}})

		_, _, err := handler(context.Background(), nil, getDocumentationPageInput{
			URL: "https://tharsis.martian-cloud.io/test",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch page content")
	})
}
