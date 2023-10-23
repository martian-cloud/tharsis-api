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

// Error code constants
const (
	EInternal        = "internal error"
	ENotImplemented  = "not implemented"
	ENotFound        = "not found"
	EConflict        = "conflict"
	EOptimisticLock  = "optimistic lock"
	EInvalid         = "invalid"
	EForbidden       = "forbidden"
	ETooManyRequests = "too many requests"
	EUnauthorized    = "unauthorized"
	ETooLarge        = "request too large"
)

// TharsisError is the internal error implementation for the Tharsis API
type TharsisError struct {
	err     error
	code    string
	message string
}

// New returns a new Tharsis error with the code and message fields set
func New(code string, format string, a ...any) *TharsisError {
	span, a := findSpan(a)
	resultError := &TharsisError{
		code:    code,
		message: fmt.Sprintf(format, a...),
	}
	if span != nil {
		span.RecordError(resultError)
		span.SetStatus(codes.Error, fmt.Sprintf(format, a...))
	}
	return resultError
}

// Wrap returns a new TharsisError which wraps an existing error
func Wrap(err error, code string, format string, a ...any) *TharsisError {
	span, a := findSpan(a)
	if span != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf(format, a...))
	}
	return &TharsisError{
		code:    code,
		message: fmt.Sprintf(format, a...),
		err:     err,
	}
}

// findSpan finds and extracts the first span in a list of arguments.
func findSpan(a []any) (trace.Span, []any) {
	var foundSpan trace.Span
	var others []any

	for _, arg := range a {
		if foundSpan == nil {
			if candidate, ok := arg.(trace.Span); ok {
				foundSpan = candidate
				continue
			}
		}
		others = append(others, arg)
	}

	return foundSpan, others
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
func ErrorCode(err error) string {
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
		return "An internal error has occurred."
	}

	if e == nil {
		return ""
	}

	if e.message != "" {
		// e.Error() returns the message and the wrapped error
		return e.Error()
	}

	if e.err != nil {
		return ErrorMessage(e.err)
	}

	return "An internal error has occurred."
}

// IsContextCanceledError returns true if the error is a context.Canceled error
func IsContextCanceledError(err error) bool {
	return errors.Is(err, context.Canceled)
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
