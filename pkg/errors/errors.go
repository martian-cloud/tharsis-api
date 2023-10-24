// Package errors provides an interface for all
// errors returned from Tharsis.
package errors

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const internalErrorMessage = "An internal error has occurred."

// CodeType is used to specify the type of error
type CodeType string

// Error code constants
const (
	EInternal        CodeType = "internal error"
	ENotImplemented  CodeType = "not implemented"
	ENotFound        CodeType = "not found"
	EConflict        CodeType = "conflict"
	EOptimisticLock  CodeType = "optimistic lock"
	EInvalid         CodeType = "invalid"
	EForbidden       CodeType = "forbidden"
	ETooManyRequests CodeType = "too many requests"
	EUnauthorized    CodeType = "unauthorized"
	ETooLarge        CodeType = "request too large"
)

type config struct {
	span      trace.Span
	errorCode CodeType
}

// Option is is used to configure a TharsisError.
type Option func(*config)

// WithErrorCode sets the error code on the TharsisError.
func WithErrorCode(code CodeType) Option {
	return func(c *config) {
		c.errorCode = code
	}
}

// WithSpan records the error on the span and sets the status to Error.
func WithSpan(span trace.Span) Option {
	return func(c *config) {
		c.span = span
	}
}

// TharsisError is the internal error implementation for the Tharsis API
type TharsisError struct {
	err     error
	code    CodeType
	message string
}

// New returns a new Tharsis error with the code and message fields set
func New(format string, a ...any) *TharsisError {
	msg, cfg := interpretArgs(format, a...)

	code := cfg.errorCode
	if code == "" {
		// Code defaults to internal if one is not specified
		code = EInternal
	}

	resultError := &TharsisError{
		code:    code,
		message: msg,
	}

	if cfg.span != nil {
		cfg.span.RecordError(resultError)
		cfg.span.SetStatus(codes.Error, msg)
	}

	return resultError
}

// Wrap returns a new TharsisError which wraps an existing error
func Wrap(err error, format string, a ...any) *TharsisError {
	msg, cfg := interpretArgs(format, a...)
	if cfg.span != nil {
		cfg.span.RecordError(err)
		cfg.span.SetStatus(codes.Error, msg)
	}

	code := cfg.errorCode
	if code == "" {
		// Get code from wrapped error if one is not specified
		code = ErrorCode(err)
	}

	return &TharsisError{
		code:    code,
		message: msg,
		err:     err,
	}
}

// Error implements the error interface by writing out the recursive messages.
func (e *TharsisError) Error() string {
	if e.message != "" && e.err != nil {
		var b strings.Builder
		b.WriteString(e.message)
		b.WriteString(": ")
		b.WriteString(e.err.Error())
		return b.String()
	} else if e.message != "" {
		return e.message
	} else if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("<%s>", e.code)
}

// ErrorCode returns the code of the root error, if available; otherwise returns EINTERNAL.
func ErrorCode(err error) CodeType {
	if err == nil {
		return ""
	}

	e, ok := unwrapTharsisError(err)
	if !ok {
		return EInternal
	}

	if e == nil {
		return ""
	}

	if e.code != "" {
		return e.code
	}

	if e.err != nil {
		return ErrorCode(e.err)
	}

	return EInternal
}

// ErrorMessage returns the messages associated with the error
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	e, ok := unwrapTharsisError(err)
	if !ok {
		return internalErrorMessage
	}

	if e == nil {
		return ""
	}

	if e.code == EInternal {
		return internalErrorMessage
	}

	if e.message != "" {
		// e.Error() returns the message and the wrapped error
		return e.Error()
	}

	if e.err != nil {
		return ErrorMessage(e.err)
	}

	return internalErrorMessage
}

// IsContextCanceledError returns true if the error is a context.Canceled error
func IsContextCanceledError(err error) bool {
	return errors.Is(err, context.Canceled)
}

func interpretArgs(msg string, raw ...interface{}) (string, *config) {
	// Build our args and options
	var args []interface{}
	var opts []Option
	for _, r := range raw {
		if opt, ok := r.(Option); ok {
			opts = append(opts, opt)
		} else {
			args = append(args, r)
		}
	}

	// Build message
	msg = fmt.Sprintf(msg, args...)

	// Build config
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	return msg, cfg
}

func unwrapTharsisError(err error) (*TharsisError, bool) {
	for {
		if err == nil {
			return nil, false
		}

		tErr, ok := err.(*TharsisError)
		if ok {
			return tErr, true
		}

		err = errors.Unwrap(err)
	}
}
