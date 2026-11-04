package interceptors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoveryUnary(t *testing.T) {
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	t.Run("response and error pass through", func(t *testing.T) {
		testLogger, logs := logger.NewForTest()
		interceptor := RecoveryUnary(testLogger)

		resp, err := interceptor(t.Context(), nil, info, func(_ context.Context, _ any) (any, error) {
			return "ok", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "ok", resp)

		handlerErr := errors.New("not found", errors.WithErrorCode(errors.ENotFound))
		_, err = interceptor(t.Context(), nil, info, func(_ context.Context, _ any) (any, error) {
			return nil, handlerErr
		})
		assert.Equal(t, handlerErr, err)
		assert.Equal(t, 0, logs.Len())
	})

	t.Run("panic is recovered as an internal error", func(t *testing.T) {
		testLogger, logs := logger.NewForTest()
		interceptor := RecoveryUnary(testLogger)

		resp, err := interceptor(t.Context(), nil, info, func(_ context.Context, _ any) (any, error) {
			panic("boom")
		})
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Equal(t, errors.InternalErrorMessage, st.Message())
		require.Equal(t, 1, logs.Len())
		assert.Equal(t, "grpc panic recovered", logs.All()[0].Message)
	})
}

func TestRecoveryStream(t *testing.T) {
	testLogger, logs := logger.NewForTest()
	interceptor := RecoveryStream(testLogger)
	info := &grpc.StreamServerInfo{FullMethod: "/test.Service/Stream"}
	stream := &wrappedStream{ctx: t.Context()}

	require.NoError(t, interceptor(nil, stream, info, func(_ any, _ grpc.ServerStream) error { return nil }))

	err := interceptor(nil, stream, info, func(_ any, _ grpc.ServerStream) error {
		panic("boom")
	})
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, 1, logs.Len())
}
