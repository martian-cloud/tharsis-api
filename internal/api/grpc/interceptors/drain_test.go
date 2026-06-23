package interceptors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"google.golang.org/grpc"
)

type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeServerStream) Context() context.Context {
	return f.ctx
}

func TestDrainStream(t *testing.T) {
	handlerErr := errors.New("not found", errors.WithErrorCode(errors.ENotFound))

	tests := []struct {
		name           string
		drain          bool
		streamCanceled bool
		handler        grpc.StreamHandler
		expectErrCode  errors.CodeType
		expectErr      error
	}{
		{
			name: "no drain passes nil through",
			handler: func(_ any, _ grpc.ServerStream) error {
				return nil
			},
		},
		{
			name: "no drain passes handler error through",
			handler: func(_ any, _ grpc.ServerStream) error {
				return handlerErr
			},
			expectErr: handlerErr,
		},
		{
			name:  "drain cancels stream and returns unavailable",
			drain: true,
			handler: func(_ any, stream grpc.ServerStream) error {
				<-stream.Context().Done()
				return nil
			},
			expectErrCode: errors.EServiceUnavailable,
		},
		{
			name:           "client cancel without drain passes handler error through",
			streamCanceled: true,
			handler: func(_ any, stream grpc.ServerStream) error {
				<-stream.Context().Done()
				return handlerErr
			},
			expectErr: handlerErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			drainCtx, drainCancel := context.WithCancel(context.Background())
			defer drainCancel()
			if test.drain {
				drainCancel()
			}

			streamCtx, streamCancel := context.WithCancel(context.Background())
			defer streamCancel()
			if test.streamCanceled {
				streamCancel()
			}

			interceptor := DrainStream(drainCtx)
			ss := &fakeServerStream{ctx: streamCtx}

			err := interceptor(nil, ss, &grpc.StreamServerInfo{}, test.handler)

			switch {
			case test.expectErrCode != "":
				require.Error(t, err)
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
			case test.expectErr != nil:
				assert.Equal(t, test.expectErr, err)
			default:
				assert.NoError(t, err)
			}
		})
	}
}
