package resolver

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"

// ProblemType represents the type of problem
type ProblemType string

// Problem constants
const (
	Conflict           ProblemType = "CONFLICT"
	BadRequest         ProblemType = "BAD_REQUEST"
	NotFound           ProblemType = "NOT_FOUND"
	Forbidden          ProblemType = "FORBIDDEN"
	ServiceUnavailable ProblemType = "SERVICE_UNAVAILABLE"
	// Unauthorized ProblemType = "UNAUTHORIZED" // This error shouldn't be mapped, instead should bubble up
)

var tharsisErrorToProblemType = map[errors.CodeType]ProblemType{
	errors.EInvalid:            BadRequest,
	errors.EConflict:           Conflict,
	errors.ENotFound:           NotFound,
	errors.EForbidden:          Forbidden,
	errors.EServiceUnavailable: ServiceUnavailable,
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
