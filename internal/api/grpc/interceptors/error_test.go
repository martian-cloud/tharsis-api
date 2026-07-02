package interceptors

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBuildStatusError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
		expectedMsg  string
	}{
		{
			name:         "deadline exceeded",
			err:          context.DeadlineExceeded,
			expectedCode: codes.DeadlineExceeded,
			expectedMsg:  "context deadline exceeded",
		},
		{
			name:         "wrapped deadline exceeded",
			err:          fmt.Errorf("something: %w", context.DeadlineExceeded),
			expectedCode: codes.DeadlineExceeded,
			expectedMsg:  "something: context deadline exceeded",
		},
		{
			name:         "context canceled",
			err:          context.Canceled,
			expectedCode: codes.Canceled,
			expectedMsg:  "context canceled",
		},
		{
			name:         "wrapped context canceled",
			err:          fmt.Errorf("something: %w", context.Canceled),
			expectedCode: codes.Canceled,
			expectedMsg:  "something: context canceled",
		},
		{
			name:         "internal error",
			err:          errors.New("db failure", errors.WithErrorCode(errors.EInternal)),
			expectedCode: codes.Internal,
			expectedMsg:  errors.InternalErrorMessage,
		},
		{
			name:         "not found",
			err:          errors.New("resource not found", errors.WithErrorCode(errors.ENotFound)),
			expectedCode: codes.NotFound,
			expectedMsg:  "resource not found",
		},
		{
			name:         "invalid",
			err:          errors.New("bad input", errors.WithErrorCode(errors.EInvalid)),
			expectedCode: codes.InvalidArgument,
			expectedMsg:  "bad input",
		},
		{
			// ETooLarge maps to InvalidArgument like EInvalid; the runner distinguishes the log-size
			// cap from a generic invalid-argument rejection via the preserved message (the marker is
			// owned by internal/logstream, not this layer).
			name:         "too large",
			err:          errors.New("log size limit reached (1024 bytes)", errors.WithErrorCode(errors.ETooLarge)),
			expectedCode: codes.InvalidArgument,
			expectedMsg:  "log size limit reached (1024 bytes)",
		},
		{
			name:         "forbidden",
			err:          errors.New("access denied", errors.WithErrorCode(errors.EForbidden)),
			expectedCode: codes.PermissionDenied,
			expectedMsg:  "access denied",
		},
		{
			name:         "unauthorized",
			err:          errors.New("not authenticated", errors.WithErrorCode(errors.EUnauthorized)),
			expectedCode: codes.Unauthenticated,
			expectedMsg:  "not authenticated",
		},
		{
			name:         "conflict",
			err:          errors.New("already exists", errors.WithErrorCode(errors.EConflict)),
			expectedCode: codes.AlreadyExists,
			expectedMsg:  "already exists",
		},
		{
			name:         "too many requests",
			err:          errors.New("rate limited", errors.WithErrorCode(errors.ETooManyRequests)),
			expectedCode: codes.ResourceExhausted,
			expectedMsg:  "rate limited",
		},
		{
			name:         "unknown error defaults to internal",
			err:          fmt.Errorf("random error"),
			expectedCode: codes.Internal,
			expectedMsg:  errors.InternalErrorMessage,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := buildStatusError(test.err)

			st, ok := status.FromError(result)
			assert.True(t, ok)
			assert.Equal(t, test.expectedCode, st.Code())
			assert.Equal(t, test.expectedMsg, st.Message())
		})
	}
}

func TestHandleError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
		expectLog    bool
	}{
		{
			name:         "internal error is logged",
			err:          errors.New("db failure", errors.WithErrorCode(errors.EInternal)),
			expectedCode: codes.Internal,
			expectLog:    true,
		},
		{
			name:         "deadline exceeded is not logged",
			err:          context.DeadlineExceeded,
			expectedCode: codes.DeadlineExceeded,
			expectLog:    false,
		},
		{
			name:         "context canceled is not logged",
			err:          context.Canceled,
			expectedCode: codes.Canceled,
			expectLog:    false,
		},
		{
			name:         "not found is not logged",
			err:          errors.New("missing", errors.WithErrorCode(errors.ENotFound)),
			expectedCode: codes.NotFound,
			expectLog:    false,
		},
		{
			name:         "unknown error is logged",
			err:          fmt.Errorf("random error"),
			expectedCode: codes.Internal,
			expectLog:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testLogger, logs := logger.NewForTest()

			result := handleError(t.Context(), test.err, testLogger)

			st, ok := status.FromError(result)
			assert.True(t, ok)
			assert.Equal(t, test.expectedCode, st.Code())

			if test.expectLog {
				assert.Equal(t, 1, logs.Len())
			} else {
				assert.Equal(t, 0, logs.Len())
			}
		})
	}
}

func TestErrorHandlerUnary(t *testing.T) {
	testLogger, logs := logger.NewForTest()
	interceptor := ErrorHandlerUnary(testLogger)

	t.Run("nil error passes through", func(t *testing.T) {
		handler := func(_ context.Context, _ interface{}) (interface{}, error) {
			return "ok", nil
		}

		resp, err := interceptor(t.Context(), nil, &grpc.UnaryServerInfo{}, handler)
		require.NoError(t, err)
		assert.Equal(t, "ok", resp)
		assert.Equal(t, 0, logs.Len())
	})

	t.Run("error is converted to grpc status", func(t *testing.T) {
		handler := func(_ context.Context, _ interface{}) (interface{}, error) {
			return nil, errors.New("not found", errors.WithErrorCode(errors.ENotFound))
		}

		resp, err := interceptor(t.Context(), nil, &grpc.UnaryServerInfo{}, handler)
		assert.Nil(t, resp)
		require.Error(t, err)

		st, _ := status.FromError(err)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}
