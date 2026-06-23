package interceptors

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"google.golang.org/grpc"
)

// DrainStream terminates active streams with EServiceUnavailable once the drain
// context is cancelled, signaling clients to reconnect to another instance.
func DrainStream(drainCtx context.Context) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, cancel := context.WithCancel(ss.Context())
		defer cancel()

		// Cancel this stream's context when the server starts draining so the handler ends promptly.
		stop := context.AfterFunc(drainCtx, cancel)

		err := handler(srv, newWrappedStream(ctx, ss))

		// stop() returns false only if drain already fired its cancel, meaning drain — not a
		// natural completion racing with it — is what ended this stream.
		if drainWasCause := !stop(); drainWasCause && drainCtx.Err() != nil {
			return errors.New("server is shutting down, please reconnect", errors.WithErrorCode(errors.EServiceUnavailable))
		}

		return err
	}
}
