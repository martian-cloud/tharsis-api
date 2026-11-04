package graphql

import (
	"context"
	"fmt"
	"testing"

	grapherrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetErrExtensions(t *testing.T) {
	tests := []struct {
		name       string
		queryError *grapherrors.QueryError
		expectCode string
	}{
		{
			name:       "not found",
			queryError: &grapherrors.QueryError{Err: errors.New("missing", errors.WithErrorCode(errors.ENotFound))},
			expectCode: "NOT_FOUND",
		},
		{
			// Reported as an internal error so clients see no new code.
			name:       "data loss",
			queryError: &grapherrors.QueryError{Err: errors.New("gone", errors.WithErrorCode(errors.EDataLoss))},
			expectCode: "INTERNAL_SERVER_ERROR",
		},
		{
			name:       "unknown field",
			queryError: &grapherrors.QueryError{Rule: graphqlFieldsOnCorrectTypeError},
			expectCode: "NOT_IMPLEMENTED",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectCode, getErrExtensions(test.queryError)["code"])
		})
	}
}

// TestErrorCodesMapped guards against a code missing from the GraphQL map, which would give clients
// an empty error code.
func TestErrorCodesMapped(t *testing.T) {
	for _, code := range []errors.CodeType{
		errors.EInternal, errors.ENotImplemented, errors.ENotFound, errors.EConflict, errors.EOptimisticLock,
		errors.EInvalid, errors.EForbidden, errors.ETooManyRequests, errors.EUnauthorized, errors.ETooLarge,
		errors.EServiceUnavailable, errors.EDataLoss,
	} {
		assert.NotEmpty(t, tharsisErrorToStatusCode[code], string(code))
	}
}

func TestAddErrorCode(t *testing.T) {
	tests := []struct {
		name       string
		queryError *grapherrors.QueryError
		expectCode errors.CodeType
	}{
		{
			name:       "validation error",
			queryError: &grapherrors.QueryError{Message: "bad", Rule: "FieldsOnCorrectType"},
			expectCode: errors.EInvalid,
		},
		{
			// graphql-go raises these without a rule, e.g. for an empty document.
			name:       "request error without an underlying error",
			queryError: &grapherrors.QueryError{Message: "graphql-ws protocol header is missing"},
			expectCode: errors.EInvalid,
		},
		{
			name:       "request error wrapping a plain error",
			queryError: grapherrors.Errorf("%s", fmt.Errorf("no operation with name %q", "100%")),
			expectCode: errors.EInvalid,
		},
		{
			name:       "panic",
			queryError: (&panicHandler{}).MakePanicError(context.Background(), "boom"),
			expectCode: errors.EInternal,
		},
		{
			name:       "context canceled",
			queryError: grapherrors.Errorf("%s", context.Canceled),
			expectCode: errors.EInternal,
		},
		{
			name:       "context deadline exceeded",
			queryError: grapherrors.Errorf("%s", context.DeadlineExceeded),
			expectCode: errors.EInternal,
		},
		{
			name:       "execution error on a field",
			queryError: &grapherrors.QueryError{Message: `graphql: got nil for non-null "String!"`, Path: []any{"hello"}},
			expectCode: errors.EInternal,
		},
		{
			name: "resolver error",
			queryError: &grapherrors.QueryError{
				Err:           errors.New("missing", errors.WithErrorCode(errors.ENotFound)),
				Path:          []any{"node"},
				ResolverError: errors.New("missing", errors.WithErrorCode(errors.ENotFound)),
			},
			expectCode: errors.ENotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message := test.queryError.Message
			qe := addErrorCode(test.queryError)
			assert.Equal(t, tharsisErrorToStatusCode[test.expectCode], getErrExtensions(qe)["code"])
			assert.Equal(t, message, qe.Message)
		})
	}
}
