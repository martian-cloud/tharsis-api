package graphql

import (
	graphqlgo "github.com/graph-gophers/graphql-go/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
)

var tharsisErrorToStatusCode = map[string]string{
	errors.EInternal:        "INTERNAL_SERVER_ERROR",
	errors.ETooLarge:        "INTERNAL_SERVER_ERROR",
	errors.EInvalid:         "BAD_REQUEST",
	errors.ENotImplemented:  "NOT_IMPLEMENTED",
	errors.EConflict:        "CONFLICT",
	errors.EOptimisticLock:  "OPTIMISTIC_LOCK",
	errors.ENotFound:        "NOT_FOUND",
	errors.EForbidden:       "FORBIDDEN",
	errors.ETooManyRequests: "RATE_LIMIT_EXCEEDED",
	errors.EUnauthorized:    "UNAUTHENTICATED",
}

func getErrExtensions(err error) map[string]interface{} {
	code := errors.ErrorCode(err)
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
				}

				if ic, ok := e.(indexedCauser); ok {
					qe.Path = append(qe.Path, ic.Index())
					qe.Message = ic.Cause().Error()
				}

				expanded = append(expanded, qe)
			}
		default:
			expanded = append(expanded, err)
		}
	}

	return expanded
}
