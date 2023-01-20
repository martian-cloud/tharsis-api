package resolver

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"

// ProblemType represents the type of problem
type ProblemType string

// Problem constants
const (
	Conflict     ProblemType = "CONFLICT"
	BadRequest   ProblemType = "BAD_REQUEST"
	NotFound     ProblemType = "NOT_FOUND"
	Forbidden    ProblemType = "FORBIDDEN"
	Unauthorized ProblemType = "UNAUTHORIZED"
)

var tharsisErrorToProblemType = map[string]ProblemType{
	errors.EInvalid:      BadRequest,
	errors.EConflict:     Conflict,
	errors.ENotFound:     NotFound,
	errors.EForbidden:    Forbidden,
	errors.EUnauthorized: Unauthorized,
}

// Problem is used to represent a user facing issue
type Problem struct {
	Message string
	Field   *[]string
	Type    ProblemType
}

func buildProblem(err error) (*Problem, error) {
	code := errors.ErrorCode(err)
	pType, ok := tharsisErrorToProblemType[code]
	if ok {
		return &Problem{
			Message: errors.ErrorMessage(err),
			Field:   &[]string{},
			Type:    pType,
		}, nil
	}
	return nil, err
}
