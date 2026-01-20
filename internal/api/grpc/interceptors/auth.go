package interceptors

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthenticationUnary returns a UnaryServerInterceptor for token-based authentication.
// It looks for the 'authorization' field in the metadata of the request to locate the token.
func AuthenticationUnary(authenticator auth.Authenticator) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ctxWithCaller, err := withCaller(ctx, authenticator)
		if err != nil {
			return nil, err
		}

		return handler(ctxWithCaller, req)
	}
}

// AuthenticationStream returns a stream interceptor
func AuthenticationStream(authenticator auth.Authenticator) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := withCaller(ss.Context(), authenticator)
		if err != nil {
			return err
		}

		return handler(srv, newWrappedStream(ctx, ss))
	}
}

func withCaller(ctx context.Context, authenticator auth.Authenticator) (context.Context, error) {
	// Get the metadata from the request context.
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "failed to extract metadata from request")
	}

	// metadata keys are always lowercase.
	var token string
	if authHeader, ok := meta["authorization"]; ok {
		token = authHeader[0]
	}

	// Authenticate the caller.
	caller, err := authenticator.Authenticate(ctx, token, false)
	if err != nil && errors.ErrorCode(err) != errors.EUnauthorized {
		return nil, buildStatusError(err)
	}

	if caller != nil {
		ctx = auth.WithCaller(ctx, caller)
	}

	return ctx, nil
}
