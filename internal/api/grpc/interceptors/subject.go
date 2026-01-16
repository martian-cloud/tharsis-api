package interceptors

import (
	"context"
	"fmt"
	"net"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	log "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// SubjectUnary adds the subject to unary requests
func SubjectUnary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		return handler(withSubject(ctx), req)
	}
}

// SubjectStream adds the subject to stream requests
func SubjectStream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, newWrappedStream(withSubject(ss.Context()), ss))
	}
}

func withSubject(ctx context.Context) context.Context {
	var subject string

	caller := auth.GetCaller(ctx)
	if caller != nil {
		subject = caller.GetSubject()
	} else {
		p, ok := peer.FromContext(ctx)
		if !ok {
			return ctx
		}
		clientAddr := p.Addr.String()
		// Extract just the IP address (remove the port)
		ip, _, err := net.SplitHostPort(clientAddr)
		if err != nil {
			return ctx
		}

		subject = fmt.Sprintf("anonymous-%s", ip)
	}

	// Add subject to auth context
	ctx = auth.WithSubject(ctx, subject)
	// Add subject to logger context
	ctx = log.WithSubject(ctx, subject)

	return ctx
}
