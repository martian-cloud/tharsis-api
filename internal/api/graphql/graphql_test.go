package graphql

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	complexity "gitlab.com/infor-cloud/martian-cloud/tharsis/graphql-query-complexity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/resolver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type panickingResolver struct{}

func (*panickingResolver) Boom() string {
	panic("raw panic detail that must not leak")
}

func TestHTTPHandler_ServeHTTP_PanicSanitizedAndLogged(t *testing.T) {
	log, observed := logger.NewForTest()

	schema := graphql.MustParseSchema(
		`schema { query: Query } type Query { boom: String! }`,
		&panickingResolver{},
		graphql.UseFieldResolvers(),
		graphql.Logger(&panicLogger{logger: log}),
		graphql.PanicHandler(&panicHandler{}),
	)

	complexityOpts := complexity.DefaultOptions()
	calculator, err := complexity.NewCalculator(schema.AST(), &complexityOpts)
	require.NoError(t, err)

	h := &httpHandler{
		schema: schema,
		logger: log,
		ctxGenerator: &contextGenerator{
			resolverState: &resolver.State{Logger: log},
			loaders:       loader.NewCollection(),
		},
		queryComplexityCalculator: calculator,
		maxGraphqlComplexity:      0, // disables rate limiting; the rate-limit store is never touched
		maxBodySize:               1 << 20,
	}

	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(`{"query":"{ boom }"}`))
	req = req.WithContext(auth.WithSubject(context.Background(), "test-subject"))
	rec := httptest.NewRecorder()

	require.NotPanics(t, func() { h.ServeHTTP(rec, req) })

	body := rec.Body.String()
	// Client gets the sanitized message, not the raw panic text.
	assert.Contains(t, body, errors.InternalErrorMessage)
	assert.NotContains(t, body, "raw panic detail that must not leak")
	assert.Contains(t, body, "INTERNAL_SERVER_ERROR")

	// The panic is logged at ERROR with the value and stack as structured fields.
	entries := observed.All()
	require.NotEmpty(t, entries)
	entry := entries[0]
	assert.Equal(t, "error", entry.Level.String())
	assert.Equal(t, "graphql panic recovered", entry.Message)
	fields := entry.ContextMap()
	assert.Equal(t, "raw panic detail that must not leak", fields["panic"])
	assert.Contains(t, fields["stack"], "goroutine")
}
