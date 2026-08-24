package graphql

import (
	"testing"

	"github.com/graph-gophers/graphql-go"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/resolver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/schema"
)

// TestSchemaBindsToResolver parses the full GraphQL schema against the root resolver
// exactly as the server does at startup (see graphql.go). graphql.MustParseSchema
// panics on any field/argument binding mismatch — for example a required argument
// (declared with "!") bound to a pointer resolver field, or a resolver method that
// does not match its schema field. Running the parse here fails the test fast on any
// schema/resolver drift instead of panicking at server boot.
func TestSchemaBindsToResolver(t *testing.T) {
	schemaStr, err := schema.String()
	if err != nil {
		t.Fatalf("failed to load embedded schema: %v", err)
	}

	// Mirrors the options used in newGraphqlHandler (graphql.go). UseFieldResolvers is
	// the option that governs how arguments/fields bind to the resolver, so it must be
	// present for this check to match production binding behavior.
	graphql.MustParseSchema(schemaStr, resolver.NewRootResolver(), graphql.UseFieldResolvers())
}
