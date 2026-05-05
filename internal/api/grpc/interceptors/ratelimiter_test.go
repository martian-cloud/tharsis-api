package interceptors

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"google.golang.org/grpc"
)

type mockRateLimitStore struct {
	ok  bool
	err error
}

func (m *mockRateLimitStore) TakeMany(_ context.Context, _ string, _ uint64) (uint64, uint64, uint64, bool, error) {
	return 100, 99, 0, m.ok, m.err
}

func TestRateLimiterUnary(t *testing.T) {
	tests := []struct {
		name            string
		subject         *string
		storeOK         bool
		storeErr        error
		expectErrorCode errors.CodeType
	}{
		{
			name:    "passes when under limit",
			subject: new("user-1"),
			storeOK: true,
		},
		{
			name:            "no subject returns invalid",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "store error returns internal",
			subject:         new("user-1"),
			storeErr:        fmt.Errorf("store failure"),
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "rate limited returns too many requests",
			subject:         new("user-1"),
			storeOK:         false,
			expectErrorCode: errors.ETooManyRequests,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &mockRateLimitStore{ok: test.storeOK, err: test.storeErr}
			interceptor := RateLimiterUnary(store)

			ctx := t.Context()
			if test.subject != nil {
				ctx = auth.WithSubject(ctx, *test.subject)
			}

			handler := func(_ context.Context, _ interface{}) (interface{}, error) {
				return "ok", nil
			}

			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

			if test.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "ok", resp)
		})
	}
}
