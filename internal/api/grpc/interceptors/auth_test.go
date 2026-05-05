package interceptors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthenticationUnary(t *testing.T) {
	tests := []struct {
		name         string
		meta         metadata.MD
		authErr      error
		caller       auth.Caller
		expectedCode codes.Code
		expectCaller bool
	}{
		{
			name:         "no metadata returns invalid argument",
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "valid token sets caller",
			meta:         metadata.Pairs("authorization", "valid-token"),
			caller:       auth.NewMockCaller(t),
			expectCaller: true,
		},
		{
			name:    "unauthorized error is ignored and caller is nil",
			meta:    metadata.Pairs("authorization", "bad-token"),
			authErr: errors.New("unauthorized", errors.WithErrorCode(errors.EUnauthorized)),
		},
		{
			name:         "non-unauthorized error returns error",
			meta:         metadata.Pairs("authorization", "bad-token"),
			authErr:      errors.New("internal error", errors.WithErrorCode(errors.EInternal)),
			expectedCode: codes.Internal,
		},
		{
			name: "no authorization header authenticates with empty token",
			meta: metadata.Pairs("other-header", "value"),
		},
		{
			name: "empty authorization header authenticates with empty token",
			meta: metadata.Pairs("authorization", ""),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockAuth := auth.NewMockAuthenticator(t)
			if test.meta != nil {
				mockAuth.On("Authenticate", mock.Anything, mock.Anything, false).
					Return(test.caller, test.authErr)
			}

			interceptor := AuthenticationUnary(mockAuth)

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

			if test.expectedCode != 0 {
				require.Error(t, err)
				st, _ := status.FromError(err)
				assert.Equal(t, test.expectedCode, st.Code())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "ok", resp)

			if test.expectCaller {
				assert.NotNil(t, auth.GetCaller(handlerCtx))
			} else {
				assert.Nil(t, auth.GetCaller(handlerCtx))
			}
		})
	}
}
