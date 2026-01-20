package interceptors

import (
	"context"
	goerrors "errors"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// errorToStatusCode maps from API errors to gRPC status code.
var errorToStatusCode = map[errors.CodeType]codes.Code{
	errors.EInternal:           codes.Internal,
	errors.ETooLarge:           codes.InvalidArgument,
	errors.EInvalid:            codes.InvalidArgument,
	errors.ENotImplemented:     codes.Unimplemented,
	errors.EConflict:           codes.AlreadyExists,
	errors.EOptimisticLock:     codes.Aborted,
	errors.ENotFound:           codes.NotFound,
	errors.EForbidden:          codes.PermissionDenied,
	errors.ETooManyRequests:    codes.ResourceExhausted,
	errors.EUnauthorized:       codes.Unauthenticated,
	errors.EServiceUnavailable: codes.Unavailable,
}

// ErrorHandlerUnary handles any errors that occur by returning the correct grpc error code
func ErrorHandlerUnary(logger logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Handle the rpc.
		resp, err := handler(ctx, req)
		if err != nil {
			return nil, handleError(ctx, err, logger)
		}

		return resp, nil
	}
}

// ErrorHandlerStream handles any errors that occur by returning the correct grpc error code
func ErrorHandlerStream(logger logger.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := handler(srv, ss); err != nil {
			return handleError(ss.Context(), err, logger)
		}

		return nil
	}
}

// buildStatusError builds a gRPC error from an API error.
func buildStatusError(err error) error {
	var grpcCode codes.Code
	var grpcMsg string

	switch {
	case errors.ErrorCode(err) == errors.EInternal:
		grpcCode = codes.Internal
		grpcMsg = errors.InternalErrorMessage
	case goerrors.Is(err, context.DeadlineExceeded):
		grpcCode = codes.DeadlineExceeded
		grpcMsg = err.Error()
	case errors.IsContextCanceledError(err):
		grpcCode = codes.Canceled
		grpcMsg = err.Error()
	default:
		grpcCode = errorToStatusCode[errors.ErrorCode(err)]
		grpcMsg = errors.ErrorMessage(err)
	}

	return status.Error(grpcCode, grpcMsg)
}

func handleError(ctx context.Context, err error, logger logger.Logger) error {
	// Log certain errors to make troubleshooting easier.
	switch errors.ErrorCode(err) {
	case errors.EInternal:
		// Don't log context deadline expired and context cancelled errors.
		if !(goerrors.Is(err, context.DeadlineExceeded) || errors.IsContextCanceledError(err)) {
			logger.WithContextFields(ctx).Errorf("Unexpected gRPC error occurred: %s", err.Error())
		}
	}
	// Convert error to gRPC equivalent.
	return buildStatusError(err)
}
