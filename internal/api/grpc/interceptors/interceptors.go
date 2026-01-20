// Package interceptors contains the GRPC interceptors
package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

// wrappedStream wraps around the embedded grpc.ServerStream to return a custom context
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

func newWrappedStream(ctx context.Context, s grpc.ServerStream) grpc.ServerStream {
	return &wrappedStream{s, ctx}
}
