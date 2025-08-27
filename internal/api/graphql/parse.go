package graphql

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

var acceptedContentTypes = []string{
	"application/json",
	"application/graphql",
}

func parse(r *http.Request) (request, error) {
	// We always need to read and close the request body.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return request{}, errors.New("unable to read request body")
	}
	_ = r.Body.Close()

	switch r.Method {
	case "POST":
		if err = validateContentType(r); err != nil {
			return request{}, err
		}
		return parsePost(body), nil
	case "GET":
		return parseGet(r.URL.Query()), nil
	default:
		return request{}, errors.New("only POST and GET requests are supported")
	}
}

func validateContentType(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return nil
	}

	// Parse the media type, ignoring parameters like charset
	mediaType := strings.ToLower(strings.Split(contentType, ";")[0])
	mediaType = strings.TrimSpace(mediaType)

	if slices.Contains(acceptedContentTypes, mediaType) {
		return nil
	}

	return fmt.Errorf("header Content-Type must be one of %s for POST requests", strings.Join(acceptedContentTypes, ", "))
}

func parseGet(v url.Values) request {
	var (
		queries   = v["query"]
		names     = v["operationName"]
		variables = v["variables"]
		qLen      = len(queries)
		nLen      = len(names)
		vLen      = len(variables)
	)

	if qLen == 0 {
		return request{}
	}

	var requests = make([]query, 0, qLen)
	var isBatch bool

	// This loop assumes there will be a corresponding element at each index
	// for query, operation name, and variable fields.
	for i, q := range queries {
		var n string
		if i < nLen {
			n = names[i]
		}

		var m = map[string]interface{}{}
		if i < vLen {
			str := variables[i]
			if err := json.Unmarshal([]byte(str), &m); err != nil {
				m = nil
			}
		}

		requests = append(requests, query{Query: q, OpName: n, Variables: m})
	}

	if qLen > 1 {
		isBatch = true
	}

	return request{queries: requests, isBatch: isBatch}
}

func parsePost(b []byte) request {
	if len(b) == 0 {
		return request{}
	}

	var queries []query
	var isBatch bool

	// Inspect the first character to inform how the body is parsed.
	switch b[0] {
	case '{':
		q := query{}
		err := json.Unmarshal(b, &q)
		if err == nil {
			queries = append(queries, q)
		}
	case '[':
		isBatch = true
		_ = json.Unmarshal(b, &queries)
	}

	return request{queries: queries, isBatch: isBatch}
}
