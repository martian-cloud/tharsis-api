package graphql

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestValidateContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expectError bool
	}{
		{
			name:        "valid application/json",
			contentType: "application/json",
			expectError: false,
		},
		{
			name:        "valid application/json with charset",
			contentType: "application/json; charset=utf-8",
			expectError: false,
		},
		{
			name:        "valid application/graphql",
			contentType: "application/graphql",
			expectError: false,
		},
		{
			name:        "case insensitive",
			contentType: "Application/JSON",
			expectError: false,
		},
		{
			name:        "missing content type",
			contentType: "",
			expectError: true,
		},
		{
			name:        "invalid content type",
			contentType: "text/plain",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{Header: make(http.Header)}
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			err := validateContentType(req)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestParseGet(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		opName   string
		vars     string
		expected int // expected number of queries
		isBatch  bool
	}{
		{
			name:     "single query",
			query:    "{ __typename }",
			expected: 1,
			isBatch:  false,
		},
		{
			name:     "query with operation name",
			query:    "query GetUser { user { id } }",
			opName:   "GetUser",
			expected: 1,
			isBatch:  false,
		},
		{
			name:     "query with variables",
			query:    "query($id: ID!) { user(id: $id) { name } }",
			vars:     `{"id": "123"}`,
			expected: 1,
			isBatch:  false,
		},
		{
			name:     "multiple queries",
			query:    "{ __typename }",
			expected: 2,
			isBatch:  true,
		},
		{
			name:     "no query",
			expected: 0,
			isBatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{}

			if tt.name == "multiple queries" {
				values.Add("query", "{ __typename }")
				values.Add("query", "{ user { id } }")
			} else if tt.query != "" {
				values.Set("query", tt.query)
			}

			if tt.opName != "" {
				values.Set("operationName", tt.opName)
			}
			if tt.vars != "" {
				values.Set("variables", tt.vars)
			}

			result := parseGet(values)

			if len(result.queries) != tt.expected {
				t.Errorf("expected %d queries, got %d", tt.expected, len(result.queries))
			}
			if result.isBatch != tt.isBatch {
				t.Errorf("expected isBatch=%v, got %v", tt.isBatch, result.isBatch)
			}
		})
	}
}

func TestParsePost(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected int // expected number of queries
		isBatch  bool
	}{
		{
			name:     "single query object",
			body:     `{"query": "{ __typename }"}`,
			expected: 1,
			isBatch:  false,
		},
		{
			name:     "single query with variables",
			body:     `{"query": "query($id: ID!) { user(id: $id) { name } }", "variables": {"id": "123"}}`,
			expected: 1,
			isBatch:  false,
		},
		{
			name:     "batch queries",
			body:     `[{"query": "{ __typename }"}, {"query": "{ user { id } }"}]`,
			expected: 2,
			isBatch:  true,
		},
		{
			name:     "empty body",
			body:     "",
			expected: 0,
			isBatch:  false,
		},
		{
			name:     "invalid JSON",
			body:     `{"query": "{ __typename }"`,
			expected: 0,
			isBatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePost([]byte(tt.body))

			if len(result.queries) != tt.expected {
				t.Errorf("expected %d queries, got %d", tt.expected, len(result.queries))
			}
			if result.isBatch != tt.isBatch {
				t.Errorf("expected isBatch=%v, got %v", tt.isBatch, result.isBatch)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		expectError bool
	}{
		{
			name:        "POST with valid JSON",
			method:      "POST",
			contentType: "application/json",
			body:        `{"query": "{ __typename }"}`,
			expectError: false,
		},
		{
			name:        "POST with valid GraphQL",
			method:      "POST",
			contentType: "application/graphql",
			body:        `{ __typename }`,
			expectError: false,
		},
		{
			name:        "POST with missing content type",
			method:      "POST",
			body:        `{"query": "{ __typename }"}`,
			expectError: true,
		},
		{
			name:        "POST with invalid content type",
			method:      "POST",
			contentType: "text/plain",
			body:        `{"query": "{ __typename }"}`,
			expectError: true,
		},
		{
			name:        "GET request",
			method:      "GET",
			expectError: false,
		},
		{
			name:        "unsupported method",
			method:      "PUT",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.method == "GET" {
				u, _ := url.Parse("http://example.com/graphql?query=" + url.QueryEscape("{ __typename }"))
				req, err = http.NewRequest(tt.method, u.String(), strings.NewReader(""))
			} else {
				req, err = http.NewRequest(tt.method, "http://example.com/graphql", strings.NewReader(tt.body))
			}

			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			_, parseErr := parse(req)

			if tt.expectError && parseErr == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && parseErr != nil {
				t.Errorf("expected no error but got: %v", parseErr)
			}
		})
	}
}
