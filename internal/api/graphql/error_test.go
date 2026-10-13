package graphql

import (
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
