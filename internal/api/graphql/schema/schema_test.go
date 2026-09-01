package schema_test

import (
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/resolver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/schema"
)

func TestString(t *testing.T) {
	s, err := schema.String()

	require.NoError(t, err)
	require.NotEmpty(t, s)
}

// TestSchemaBindsToResolver builds the full schema against the real RootResolver. graphql-go
// resolves fields by reflection, so a type mismatch between the SDL and a resolver/input struct
// compiles fine in Go but panics at server startup - this catches it at test time instead.
func TestSchemaBindsToResolver(t *testing.T) {
	s, err := schema.String()
	require.NoError(t, err)

	require.NotPanics(t, func() {
		graphql.MustParseSchema(s, resolver.NewRootResolver(), graphql.UseFieldResolvers())
	})
}
