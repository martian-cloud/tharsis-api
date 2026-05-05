package interceptors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUserAgentUnary(t *testing.T) {
	tests := []struct {
		name           string
		meta           metadata.MD
		expectEnriched bool
	}{
		{
			name:           "extracts user agent from metadata",
			meta:           metadata.Pairs("user-agent", "tharsis-runner/1.0"),
			expectEnriched: true,
		},
		{
			name: "no metadata does not enrich context",
		},
		{
			name: "no user-agent header does not enrich context",
			meta: metadata.Pairs("other", "value"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			interceptor := UserAgentUnary()

			ctx := t.Context()
			if test.meta != nil {
				ctx = metadata.NewIncomingContext(ctx, test.meta)
			}

			var handlerCtx context.Context
			handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
				handlerCtx = ctx
				return "ok", nil
			}

			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
			require.NoError(t, err)
			assert.Equal(t, "ok", resp)

			if test.expectEnriched {
				// Context should differ from input (user agent was added).
				assert.NotEqual(t, ctx, handlerCtx)
			}
		})
	}
}

func TestWithUserAgent(t *testing.T) {
	// Without metadata, context is unchanged.
	ctx := t.Context()
	result := withUserAgent(ctx)
	assert.Equal(t, ctx, result)

	// With user-agent metadata, context is enriched.
	ctx = metadata.NewIncomingContext(t.Context(), metadata.Pairs("user-agent", "test/1.0"))
	result = withUserAgent(ctx)
	assert.NotEqual(t, ctx, result)
}
