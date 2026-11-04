package graphql

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
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
			expectError: false,
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
			expectError: false,
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

// Modes for FuzzParse.
const (
	// parseModeRaw sends the fuzzed method, Content-Type, body and (for GET) raw query string.
	parseModeRaw = iota
	// parseModePostSingle POSTs a JSON object encoded from the fuzzed query fields.
	parseModePostSingle
	// parseModePostBatch POSTs a JSON array of two objects encoded from the fuzzed query fields.
	parseModePostBatch
	// parseModeGet sends the fuzzed query fields as GET query parameters.
	parseModeGet
	parseModeCount
)

// parseModeTruncate, when set on the mode byte in raw mode, caps the body one byte short of its
// length the way the handler's MaxBytesReader does.
const parseModeTruncate = 0x10

func FuzzParse(f *testing.F) {
	// mode, method, contentType, body, query, operationName, variables
	f.Add(uint8(parseModeRaw), "POST", "application/json", []byte(`{"query":"{ me { __typename } }"}`), "", "", "")
	f.Add(uint8(parseModeRaw), "POST", "", []byte(`[{"query":"{ a }"},{"query":"{ b }","operationName":"B"}]`), "", "", "")
	f.Add(uint8(parseModeRaw), "POST", "application/graphql", []byte(`{ __typename }`), "", "", "")
	f.Add(uint8(parseModeRaw), "POST", "text/plain", []byte(`{"query":"{ a }"}`), "", "", "")
	f.Add(uint8(parseModeRaw), "POST", "Application/JSON ; charset=utf-8", []byte(`{"query":"{ a }"`), "", "", "")
	f.Add(uint8(parseModeRaw), "POST", "application/json", []byte(`[1, {"query":"x"}]`), "", "", "")
	f.Add(uint8(parseModeRaw), "POST", "application/json", []byte(`{"query": 5}`), "", "", "")
	f.Add(uint8(parseModeRaw), "POST", "application/json", []byte(`  {"query":"{ a }"}`), "", "", "")
	f.Add(uint8(parseModeRaw), "PUT", "application/json", []byte(`{"query":"{ a }"}`), "", "", "")
	f.Add(uint8(parseModeRaw), "GET", "", []byte(nil), "query=%7B+a+%7D&variables=%7B%7D", "", "")
	f.Add(uint8(parseModeRaw), "GET", "", []byte(nil), "query=a;b&query=%zz", "", "")
	f.Add(uint8(parseModeRaw), "", "", []byte(nil), "0", "", "")
	f.Add(uint8(parseModeRaw|parseModeTruncate), "POST", "application/json", []byte(`{"query":"{ a }"}`), "", "", "")
	f.Add(uint8(parseModePostSingle), "", "application/json", []byte(nil), "query Q($id: ID!) { node(id: $id) { id } }", "Q", `{"id":"abc","n":1.5,"l":[null,true]}`)
	f.Add(uint8(parseModePostSingle), "", "", []byte(nil), "", "", "")
	f.Add(uint8(parseModePostSingle), "", "application/xml", []byte(nil), "{ a }", "", `null`)
	f.Add(uint8(parseModePostBatch), "", "application/graphql; charset=utf-8", []byte(nil), "{ a }", "op", `{"a":{"b":[1,2]}}`)
	f.Add(uint8(parseModeGet), "", "", []byte(nil), "{ me { __typename } }", "Me", `{"x":"y"}`)
	f.Add(uint8(parseModeGet), "", "", []byte(nil), "{ a }", "", "")
	f.Add(uint8(parseModeGet), "", "", []byte(nil), "{ a }", "", `not json`)
	f.Add(uint8(parseModeGet), "", "", []byte(nil), "{ a }", "", `[1,2]`)
	f.Add(uint8(parseModeGet), "", "", []byte(nil), "a&query=b;c=%", "\x00\xff", `null`)

	f.Fuzz(func(t *testing.T, mode uint8, method, contentType string, body []byte, q, opName, varsJSON string) {
		switch int(mode&0x0f) % parseModeCount {
		case parseModeRaw:
			fuzzParseRaw(t, mode&parseModeTruncate != 0, method, contentType, body, q)
		case parseModePostSingle, parseModePostBatch:
			fuzzParsePost(t, int(mode&0x0f)%parseModeCount == parseModePostBatch, contentType, q, opName, varsJSON)
		case parseModeGet:
			fuzzParseGet(t, q, opName, varsJSON)
		}
	})
}

// fuzzParseRaw checks that arbitrary requests never panic and are only ever rejected as a client
// error.
func fuzzParseRaw(t *testing.T, truncate bool, method, contentType string, body []byte, rawQuery string) {
	req, err := http.NewRequest(method, "http://example.com/graphql", bytes.NewReader(body))
	if err != nil {
		t.Skip() // not a valid HTTP method token
	}
	req.URL.RawQuery = rawQuery
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	truncated := truncate && len(body) > 0
	if truncated {
		req.Body = http.MaxBytesReader(nil, req.Body, int64(len(body)-1))
	}

	parsed, err := parse(req)
	if err != nil {
		assert.Empty(t, parsed.queries)
		if truncated {
			assert.Equal(t, errors.ETooLarge, errors.ErrorCode(err), err.Error())
		} else {
			assert.Equal(t, errors.EInvalid, errors.ErrorCode(err), err.Error())
		}
		return
	}

	require.False(t, truncated, "a body over the size limit must be rejected")
	// http.NewRequest treats an empty method as GET.
	require.Contains(t, []string{http.MethodPost, http.MethodGet}, req.Method)
	if req.Method == http.MethodGet {
		// Every query parameter is one query, and more than one makes a batch.
		queries := req.URL.Query()["query"]
		assert.Len(t, parsed.queries, len(queries))
		assert.Equal(t, len(queries) > 1, parsed.isBatch)
		for i, q := range parsed.queries {
			assert.Equal(t, queries[i], q.Query)
		}
		return
	}

	assertContentTypeAccepted(t, contentType)
	// A POST body is parsed as one query object or as a batch, decided by its first byte.
	switch {
	case len(body) > 0 && body[0] == '{':
		assert.False(t, parsed.isBatch)
		assert.LessOrEqual(t, len(parsed.queries), 1)
		if len(parsed.queries) == 1 {
			assert.True(t, json.Valid(body), "only a valid JSON object is parsed as a query")
		}
	case len(body) > 0 && body[0] == '[':
		assert.True(t, parsed.isBatch)
		if len(parsed.queries) > 0 {
			assert.True(t, json.Valid(body), "only a valid JSON array is parsed as a batch")
		}
	default:
		assert.Empty(t, parsed.queries)
		assert.False(t, parsed.isBatch)
	}
}

// fuzzParsePost checks that a JSON body built from the query fields round-trips through parse.
func fuzzParsePost(t *testing.T, batch bool, contentType, q, opName, varsJSON string) {
	if !utf8.ValidString(q) || !utf8.ValidString(opName) {
		t.Skip() // json.Marshal coerces invalid UTF-8, so the body would not carry these values
	}

	vars := decodeVariables(varsJSON)
	entry := map[string]any{"query": q, "operationName": opName}
	if vars != nil {
		entry["variables"] = vars
	}
	// The second batch entry swaps the strings so a mixed up index would be noticed.
	second := map[string]any{"query": opName, "operationName": q}

	want := []query{{Query: q, OpName: opName, Variables: vars}}
	var payload any = entry
	if batch {
		payload = []any{entry, second}
		want = append(want, query{Query: opName, OpName: q})
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "http://example.com/graphql", bytes.NewReader(body))
	require.NoError(t, err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	parsed, err := parse(req)
	if err != nil {
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err), err.Error())
		assert.NotEqual(t, "", contentType, "a POST without a Content-Type must be accepted")
		return
	}

	assertContentTypeAccepted(t, contentType)
	assert.Equal(t, batch, parsed.isBatch)
	assert.Equal(t, want, parsed.queries)
}

// fuzzParseGet checks that GET query parameters round-trip through parse.
func fuzzParseGet(t *testing.T, q, opName, varsJSON string) {
	values := url.Values{"query": {q}, "operationName": {opName}}
	// Without a variables parameter the query is given an empty variable map.
	wantVars := map[string]any{}
	if varsJSON != "" {
		values.Set("variables", varsJSON)
		wantVars = decodeVariables(varsJSON)
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com/graphql?"+values.Encode(), http.NoBody)
	require.NoError(t, err)

	parsed, err := parse(req)
	require.NoError(t, err)
	assert.False(t, parsed.isBatch)
	assert.Equal(t, []query{{Query: q, OpName: opName, Variables: wantVars}}, parsed.queries)
}

// decodeVariables decodes a variables document independently of parse, returning nil for anything
// that is not a JSON object.
func decodeVariables(varsJSON string) map[string]any {
	var decoded any
	if err := json.Unmarshal([]byte(varsJSON), &decoded); err != nil {
		return nil
	}
	m, _ := decoded.(map[string]any)
	return m
}

// assertContentTypeAccepted asserts that an accepted Content-Type is empty or names one of the
// accepted media types, optionally followed by parameters.
func assertContentTypeAccepted(t *testing.T, contentType string) {
	if contentType == "" {
		return
	}
	normalized := strings.ToLower(strings.TrimSpace(contentType))
	for _, accepted := range acceptedContentTypes {
		if rest, ok := strings.CutPrefix(normalized, accepted); ok {
			if rest = strings.TrimSpace(rest); rest == "" || strings.HasPrefix(rest, ";") {
				return
			}
		}
	}
	assert.Failf(t, "unexpected Content-Type accepted", "%q", contentType)
}
