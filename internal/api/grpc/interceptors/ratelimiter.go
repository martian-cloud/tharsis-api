package interceptors

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/ratelimitstore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"google.golang.org/grpc"
)

// RateLimiterUnary enforces rate limiting for unary requests
func RateLimiterUnary(store ratelimitstore.Store) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if err := rateLimit(ctx, store); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// RateLimiterStream enforces rate limiting for stream requests
func RateLimiterStream(store ratelimitstore.Store) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := rateLimit(ss.Context(), store); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func rateLimit(ctx context.Context, store ratelimitstore.Store) error {
	// Use the subject string set by the ResolveSubject middleware.
	subject := auth.GetSubject(ctx)
	if subject == nil {
		return errors.New("subject not found for grpc request", errors.WithErrorCode(errors.EInvalid))
	}

	// Check whether rate limit has been exceeded.
	tokenLimit, _, _, ok, err := store.TakeMany(ctx, "http-"+*subject, uint64(1))
	if err != nil {
		return errors.New("failed to check grpc rate limit", errors.WithErrorCode(errors.EInternal))
	}

	if !ok {
		return errors.New("rate limit exceeded: limit=%d req/sec", tokenLimit, errors.WithErrorCode(errors.ETooManyRequests))
	}

	return nil
}
