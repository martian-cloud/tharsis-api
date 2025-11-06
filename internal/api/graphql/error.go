// Package graphql package
package graphql

import (
	grapherrors "github.com/graph-gophers/graphql-go/errors"
	graphqlgo "github.com/graph-gophers/graphql-go/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// This is the graphql error rule name for when a query requests fields that don't exist
// on the graphql type
const graphqlFieldsOnCorrectTypeError = "FieldsOnCorrectType"

var tharsisErrorToStatusCode = map[errors.CodeType]string{
	errors.EInternal:           "INTERNAL_SERVER_ERROR",
	errors.ETooLarge:           "INTERNAL_SERVER_ERROR",
	errors.EInvalid:            "BAD_REQUEST",
	errors.ENotImplemented:     "NOT_IMPLEMENTED",
	errors.EConflict:           "CONFLICT",
	errors.EOptimisticLock:     "OPTIMISTIC_LOCK",
	errors.ENotFound:           "NOT_FOUND",
	errors.EForbidden:          "FORBIDDEN",
	errors.ETooManyRequests:    "RATE_LIMIT_EXCEEDED",
	errors.EUnauthorized:       "UNAUTHENTICATED",
	errors.EServiceUnavailable: "SERVICE_UNAVAILABLE",
}

func getErrExtensions(queryError *grapherrors.QueryError) map[string]interface{} {
	code := errors.EInternal
	if queryError.Err != nil {
		code = errors.ErrorCode(queryError.Err)
	} else if queryError.Rule == graphqlFieldsOnCorrectTypeError {
		// Return the not implemented code here because the client is requesting a graphql field
		// which doesn't exist. This can occur during rolling updates when a newer UI version attempts to
		// query a field that doesn't exist on the older API version.
		code = errors.ENotImplemented
	}
	return map[string]interface{}{
		"code": tharsisErrorToStatusCode[code],
	}
}

type slicer interface {
	Slice() []error
}

type indexedCauser interface {
	Index() int
	Cause() error
}

func expandResolverErrors(errs []*graphqlgo.QueryError) []*graphqlgo.QueryError {
	expanded := make([]*graphqlgo.QueryError, 0, len(errs))

	for _, err := range errs {
		switch t := err.ResolverError.(type) {
		case slicer:
			for _, e := range t.Slice() {
				qe := &graphqlgo.QueryError{
					Message:   err.Message,
					Locations: err.Locations,
					Path:      err.Path,
					Err:       err.Err,
				}

				if ic, ok := e.(indexedCauser); ok {
					qe.Path = append(qe.Path, ic.Index())
					qe.Message = ic.Cause().Error()
				}

				expanded = append(expanded, addErrorCode(qe))
			}
		default:
			expanded = append(expanded, addErrorCode(err))
		}
	}

	return expanded
}

func addErrorCode(qe *graphqlgo.QueryError) *graphqlgo.QueryError {
	if qe.Rule != "" {
		// If the rule is set, we assume this is a bad request
		if qe.Err == nil {
			qe.Err = errors.New(qe.Message, errors.WithErrorCode(errors.EInvalid))
		} else {
			qe.Err = errors.Wrap(qe.Err, "invalid query", errors.WithErrorCode(errors.EInvalid))
		}
	}
	return qe
}
