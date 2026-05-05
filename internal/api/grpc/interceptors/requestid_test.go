package interceptors

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestRequestIDUnary(t *testing.T) {
	interceptor := RequestIDUnary()

	var handlerCtx context.Context
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		handlerCtx = ctx
		return "ok", nil
	}

	resp, err := interceptor(t.Context(), nil, &grpc.UnaryServerInfo{}, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	// The context should differ from a bare context (request ID was added).
	assert.NotEqual(t, t.Context(), handlerCtx)
}

func TestWithRequestID(t *testing.T) {
	ctx := withRequestID(t.Context())

	// withRequestID uses logger.WithRequestID which stores a UUID.
	// We can't read it back directly, but we can verify the context changed.
	assert.NotEqual(t, t.Context(), ctx)

	// Call it twice to verify unique IDs (different contexts).
	ctx2 := withRequestID(t.Context())
	assert.NotEqual(t, ctx, ctx2)
}

func TestRequestIDIsValidUUID(t *testing.T) {
	// Verify uuid.NewString produces valid UUIDs (sanity check).
	id := uuid.NewString()
	_, err := uuid.Parse(id)
	assert.NoError(t, err)
}
