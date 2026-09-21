package errors

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestIsClientError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"forbidden", New("nope", WithErrorCode(EForbidden)), true},
		{"unauthorized", New("nope", WithErrorCode(EUnauthorized)), true},
		{"not found", New("nope", WithErrorCode(ENotFound)), true},
		{"conflict", New("nope", WithErrorCode(EConflict)), true},
		{"optimistic lock", New("nope", WithErrorCode(EOptimisticLock)), true},
		{"invalid", New("nope", WithErrorCode(EInvalid)), true},
		{"too many requests", New("nope", WithErrorCode(ETooManyRequests)), true},
		{"too large", New("nope", WithErrorCode(ETooLarge)), true},
		{"internal", New("boom", WithErrorCode(EInternal)), false},
		{"not implemented", New("nyi", WithErrorCode(ENotImplemented)), false},
		{"service unavailable", New("down", WithErrorCode(EServiceUnavailable)), false},
		{"plain non-TharsisError", fmt.Errorf("raw"), false},
		{"wrapped client error via %w", fmt.Errorf("outer: %w", New("nope", WithErrorCode(EForbidden))), true},
		{"nil", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isClientError(tt.err))
		})
	}
}

func TestNewAndWrap_SpanStatus(t *testing.T) {
	tests := []struct {
		name       string
		build      func(span trace.Span) *TharsisError
		wantStatus codes.Code
		wantEvent  bool
	}{
		{
			name: "New EInternal marks span error",
			build: func(s trace.Span) *TharsisError {
				return New("boom", WithErrorCode(EInternal), WithSpan(s))
			},
			wantStatus: codes.Error,
			wantEvent:  true,
		},
		{
			name: "New EForbidden leaves span unset",
			build: func(s trace.Span) *TharsisError {
				return New("nope", WithErrorCode(EForbidden), WithSpan(s))
			},
			wantStatus: codes.Unset,
			wantEvent:  false,
		},
		{
			name: "Wrap of EUnauthorized leaves span unset",
			build: func(s trace.Span) *TharsisError {
				inner := New("auth failed", WithErrorCode(EUnauthorized))
				return Wrap(inner, "caller authorization failed", WithSpan(s))
			},
			wantStatus: codes.Unset,
			wantEvent:  false,
		},
		{
			name: "Wrap of plain error (defaults EInternal) marks span error",
			build: func(s trace.Span) *TharsisError {
				return Wrap(fmt.Errorf("db exploded"), "query failed", WithSpan(s))
			},
			wantStatus: codes.Error,
			wantEvent:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			_, span := tp.Tracer("test").Start(context.Background(), "op")

			tt.build(span)
			span.End()

			ended := sr.Ended()
			require.Len(t, ended, 1)
			assert.Equal(t, tt.wantStatus, ended[0].Status().Code)
			assert.Equal(t, tt.wantEvent, len(ended[0].Events()) > 0)
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		build       func() *TharsisError
		wantCode    CodeType
		wantMessage string
	}{
		{
			name:        "defaults to internal when no code is given",
			build:       func() *TharsisError { return New("boom") },
			wantCode:    EInternal,
			wantMessage: "boom",
		},
		{
			name:        "uses the supplied code",
			build:       func() *TharsisError { return New("nope", WithErrorCode(EForbidden)) },
			wantCode:    EForbidden,
			wantMessage: "nope",
		},
		{
			name:        "formats the message with args, ignoring options",
			build:       func() *TharsisError { return New("resource %s not found", "ws-1", WithErrorCode(ENotFound)) },
			wantCode:    ENotFound,
			wantMessage: "resource ws-1 not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.build()
			assert.Equal(t, tt.wantCode, err.code)
			assert.Equal(t, tt.wantMessage, err.message)
			assert.Nil(t, err.Unwrap())
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name        string
		build       func() *TharsisError
		wantCode    CodeType
		wantMessage string
	}{
		{
			name: "inherits the wrapped TharsisError code",
			build: func() *TharsisError {
				return Wrap(New("missing", WithErrorCode(ENotFound)), "lookup failed")
			},
			wantCode:    ENotFound,
			wantMessage: "lookup failed: missing",
		},
		{
			name: "explicit code overrides the wrapped code",
			build: func() *TharsisError {
				return Wrap(New("missing", WithErrorCode(ENotFound)), "rejected", WithErrorCode(EInvalid))
			},
			wantCode:    EInvalid,
			wantMessage: "rejected: missing",
		},
		{
			name: "wrapping a plain error defaults to internal",
			build: func() *TharsisError {
				return Wrap(fmt.Errorf("db exploded"), "query failed")
			},
			wantCode:    EInternal,
			wantMessage: "query failed: db exploded",
		},
		{
			name: "formats the message with args",
			build: func() *TharsisError {
				return Wrap(fmt.Errorf("io"), "failed to read %s", "file.txt")
			},
			wantCode:    EInternal,
			wantMessage: "failed to read file.txt: io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.build()
			assert.Equal(t, tt.wantCode, err.code)
			assert.Equal(t, tt.wantMessage, err.Error())
			assert.NotNil(t, err.Unwrap())
		})
	}
}

func TestTharsisErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  *TharsisError
		want string
	}{
		{
			name: "message and wrapped error are joined",
			err:  &TharsisError{message: "outer", err: fmt.Errorf("inner")},
			want: "outer: inner",
		},
		{
			name: "message only",
			err:  &TharsisError{message: "just a message"},
			want: "just a message",
		},
		{
			name: "wrapped error only",
			err:  &TharsisError{err: fmt.Errorf("wrapped only")},
			want: "wrapped only",
		},
		{
			name: "neither message nor error falls back to the code",
			err:  &TharsisError{code: EInternal},
			want: "<internal error>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestTharsisErrorUnwrap(t *testing.T) {
	inner := fmt.Errorf("inner")
	wrapped := Wrap(inner, "outer")
	assert.Same(t, inner, wrapped.Unwrap())

	// errors.Is traverses through the TharsisError chain.
	assert.True(t, errors.Is(wrapped, inner))

	assert.Nil(t, New("no wrapped error").Unwrap())
}

func TestErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want CodeType
	}{
		{
			name: "nil returns empty",
			err:  nil,
			want: "",
		},
		{
			name: "plain non-TharsisError returns internal",
			err:  fmt.Errorf("raw"),
			want: EInternal,
		},
		{
			name: "TharsisError returns its code",
			err:  New("nope", WithErrorCode(EForbidden)),
			want: EForbidden,
		},
		{
			name: "code is read from the outermost TharsisError",
			err:  Wrap(New("inner", WithErrorCode(ENotFound)), "outer", WithErrorCode(EConflict)),
			want: EConflict,
		},
		{
			name: "inherited code surfaces through a wrap",
			err:  Wrap(New("inner", WithErrorCode(ENotFound)), "outer"),
			want: ENotFound,
		},
		{
			name: "TharsisError wrapped by a standard %w is unwrapped",
			err:  fmt.Errorf("std: %w", New("nope", WithErrorCode(EInvalid))),
			want: EInvalid,
		},
		{
			name: "first error is taken from a multi-error unwrap",
			err:  errors.Join(New("first", WithErrorCode(ETooManyRequests)), New("second", WithErrorCode(EForbidden))),
			want: ETooManyRequests,
		},
		{
			name: "empty code with a wrapped error recurses to the inner code",
			err:  &TharsisError{err: New("inner", WithErrorCode(ENotFound))},
			want: ENotFound,
		},
		{
			name: "empty code with no wrapped error falls back to internal",
			err:  &TharsisError{message: "no code"},
			want: EInternal,
		},
		{
			name: "empty multi-error unwrap is treated as non-TharsisError",
			err:  emptyJoinError{},
			want: EInternal,
		},
		{
			// A typed-nil *TharsisError satisfies the type assertion, so unwrap returns (nil, true).
			name: "typed-nil TharsisError returns empty",
			err:  (*TharsisError)(nil),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ErrorCode(tt.err))
		})
	}
}

// emptyJoinError implements the multi-error Unwrap contract with no errors, exercising the
// empty-slice branch of unwrapTharsisError.
type emptyJoinError struct{}

func (emptyJoinError) Error() string   { return "empty join" }
func (emptyJoinError) Unwrap() []error { return nil }

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil returns empty",
			err:  nil,
			want: "",
		},
		{
			name: "plain non-TharsisError is masked as internal",
			err:  fmt.Errorf("secret db details"),
			want: InternalErrorMessage,
		},
		{
			name: "internal TharsisError is masked",
			err:  New("secret db details", WithErrorCode(EInternal)),
			want: InternalErrorMessage,
		},
		{
			name: "client error message is surfaced",
			err:  New("resource not found", WithErrorCode(ENotFound)),
			want: "resource not found",
		},
		{
			name: "wrapped client error surfaces the full chain message",
			err:  Wrap(New("missing", WithErrorCode(ENotFound)), "lookup failed"),
			want: "lookup failed: missing",
		},
		{
			name: "message is read through a standard %w wrap",
			err:  fmt.Errorf("std: %w", New("bad input", WithErrorCode(EInvalid))),
			want: "bad input",
		},
		{
			name: "empty code with a wrapped client error recurses to the inner message",
			err:  &TharsisError{err: New("inner not found", WithErrorCode(ENotFound))},
			want: "inner not found",
		},
		{
			name: "empty code with no message and no wrapped error falls back to internal",
			err:  &TharsisError{},
			want: InternalErrorMessage,
		},
		{
			name: "typed-nil TharsisError returns empty",
			err:  (*TharsisError)(nil),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ErrorMessage(tt.err))
		})
	}
}

func TestIsContextCanceledError(t *testing.T) {
	assert.True(t, IsContextCanceledError(context.Canceled))
	assert.True(t, IsContextCanceledError(Wrap(context.Canceled, "wrapped")))
	assert.False(t, IsContextCanceledError(context.DeadlineExceeded))
	assert.False(t, IsContextCanceledError(fmt.Errorf("other")))
	assert.False(t, IsContextCanceledError(nil))
}

func TestIsDeadlineExceededError(t *testing.T) {
	assert.True(t, IsDeadlineExceededError(context.DeadlineExceeded))
	assert.True(t, IsDeadlineExceededError(Wrap(context.DeadlineExceeded, "wrapped")))
	assert.False(t, IsDeadlineExceededError(context.Canceled))
	assert.False(t, IsDeadlineExceededError(fmt.Errorf("other")))
	assert.False(t, IsDeadlineExceededError(nil))
}

func TestFilterContextError(t *testing.T) {
	other := fmt.Errorf("real error")

	tests := []struct {
		name    string
		err     error
		wantNil bool
	}{
		{
			name:    "nil stays nil",
			err:     nil,
			wantNil: true,
		},
		{
			name:    "context canceled is filtered to nil",
			err:     context.Canceled,
			wantNil: true,
		},
		{
			name:    "deadline exceeded is filtered to nil",
			err:     context.DeadlineExceeded,
			wantNil: true,
		},
		{
			name:    "wrapped context cancellation is filtered to nil",
			err:     Wrap(context.Canceled, "shutting down"),
			wantNil: true,
		},
		{
			name:    "an unrelated error is returned unchanged",
			err:     other,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterContextError(tt.err)
			if tt.wantNil {
				assert.NoError(t, got)
			} else {
				assert.Same(t, other, got)
			}
		})
	}
}
