package interceptors

import (
	"context"

	"github.com/google/uuid"
	log "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"google.golang.org/grpc"
)

// RequestIDUnary adds the request ID to unary requests
func RequestIDUnary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		return handler(withRequestID(ctx), req)
	}
}

// RequestIDStream adds the request ID to stream requests
func RequestIDStream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, newWrappedStream(withRequestID(ss.Context()), ss))
	}
}

func withRequestID(ctx context.Context) context.Context {
	return log.WithRequestID(ctx, uuid.NewString())
}
