package errors

import (
	"context"
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
