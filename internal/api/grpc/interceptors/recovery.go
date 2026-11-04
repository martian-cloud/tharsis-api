package interceptors

import (
	"context"
	"runtime/debug"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryUnary recovers from a panic in the rpc and returns an internal error. Unlike net/http,
// grpc-go doesn't recover handler panics, so without this a single panic would crash the API.
func RecoveryUnary(logger logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = handlePanic(ctx, logger, info.FullMethod, r)
			}
		}()

		return handler(ctx, req)
	}
}

// RecoveryStream recovers from a panic in the rpc and returns an internal error.
func RecoveryStream(logger logger.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = handlePanic(ss.Context(), logger, info.FullMethod, r)
			}
		}()

		return handler(srv, ss)
	}
}

// handlePanic logs the recovered panic with its stack trace and returns a sanitized internal error.
func handlePanic(ctx context.Context, logger logger.Logger, method string, value any) error {
	logger.WithContextFields(ctx).Errorw("grpc panic recovered", "method", method, "panic", value, "stack", string(debug.Stack()))
	return status.Error(codes.Internal, errors.InternalErrorMessage)
}
