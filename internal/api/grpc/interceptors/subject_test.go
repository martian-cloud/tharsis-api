package interceptors

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

func TestSubjectUnary(t *testing.T) {
	tests := []struct {
		name            string
		setupCaller     func(t *testing.T) auth.Caller
		peer            *peer.Peer
		expectedSubject *string
	}{
		{
			name: "sets subject from caller",
			setupCaller: func(t *testing.T) auth.Caller {
				c := auth.NewMockCaller(t)
				c.On("GetSubject").Return("sa-123")
				return c
			},
			expectedSubject: new("sa-123"),
		},
		{
			name: "sets anonymous subject from peer IP",
			peer: &peer.Peer{Addr: &net.TCPAddr{
				IP:   net.ParseIP("192.168.1.1"),
				Port: 5000,
			}},
			expectedSubject: new("anonymous-192.168.1.1"),
		},
		{
			name: "no caller and no peer returns unchanged context",
		},
		{
			name: "peer with invalid address format returns unchanged context",
			peer: &peer.Peer{Addr: &net.IPAddr{IP: net.ParseIP("10.0.0.1")}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			interceptor := SubjectUnary()

			ctx := t.Context()
			if test.setupCaller != nil {
				ctx = auth.WithCaller(ctx, test.setupCaller(t))
			}
			if test.peer != nil {
				ctx = peer.NewContext(ctx, test.peer)
			}

			var handlerCtx context.Context
			handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
				handlerCtx = ctx
				return "ok", nil
			}

			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
			require.NoError(t, err)
			assert.Equal(t, "ok", resp)

			subject := auth.GetSubject(handlerCtx)
			if test.expectedSubject != nil {
				require.NotNil(t, subject)
				assert.Equal(t, *test.expectedSubject, *subject)
			} else {
				assert.Nil(t, subject)
			}
		})
	}
}
