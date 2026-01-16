package interceptors

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	userAgentHeader = "user-agent"
)

// UserAgentUnary adds the user agent to unary requests
func UserAgentUnary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		return handler(withUserAgent(ctx), req)
	}
}

// UserAgentStream adds the user agent to stream requests
func UserAgentStream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, newWrappedStream(withUserAgent(ss.Context()), ss))
	}
}

func withUserAgent(ctx context.Context) context.Context {
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	if userAgents := meta.Get(userAgentHeader); len(userAgents) > 0 {
		return logger.WithUserAgent(ctx, userAgents[0])
	}

	return ctx
}
