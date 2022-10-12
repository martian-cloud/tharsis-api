package errors

import (
	"fmt"
	"strings"
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
	Err  error
	Code string
	Msg  string
}

// NewError creates an instance of TharsisError
func NewError(code string, msg string, options ...func(*TharsisError)) *TharsisError {
	err := &TharsisError{Code: code, Msg: msg}
	for _, o := range options {
		o(err)
	}

	return err
}

// WithErrorErr sets the err on the error.
func WithErrorErr(err error) func(*TharsisError) {
	return func(e *TharsisError) {
		e.Err = err
	}
}

// Error implements the error interface by writing out the recursive messages.
func (e *TharsisError) Error() string {
	if e.Msg != "" && e.Err != nil {
		var b strings.Builder
		b.WriteString(e.Msg)
		b.WriteString(": ")
		b.WriteString(e.Err.Error())
		return b.String()
	} else if e.Msg != "" {
		return e.Msg
	} else if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("<%s>", e.Code)
}

// ErrorCode returns the code of the root error, if available; otherwise returns EINTERNAL.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}

	e, ok := err.(*TharsisError)
	if !ok {
		return EInternal
	}

	if e == nil {
		return ""
	}

	if e.Code != "" {
		return e.Code
	}

	if e.Err != nil {
		return ErrorCode(e.Err)
	}

	return EInternal
}

// ErrorMessage returns the messages associated with the error
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	e, ok := err.(*TharsisError)
	if !ok {
		return "An internal error has occurred."
	}

	if e == nil {
		return ""
	}

	if e.Msg != "" {
		return e.Msg
	}

	if e.Err != nil {
		return ErrorMessage(e.Err)
	}

	return "An internal error has occurred."
}
