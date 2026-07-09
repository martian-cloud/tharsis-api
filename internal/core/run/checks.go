// Package run contains shared domain types for Terraform run and check results.
package run

import "slices"

// CheckResult represents a check result from a Terraform plan or state.
type CheckResult struct {
	Name    string
	Status  string
	Objects []CheckResultObject
}

// CheckResultObject represents an individual checkable object instance within a check result.
type CheckResultObject struct {
	Address         string
	Status          string
	FailureMessages []string
}

// validCheckStatuses lists the status values accepted by the CheckResultStatus enum
// defined in the GraphQL schema (see run.graphql). Keep this in sync with that enum.
var validCheckStatuses = []string{"pass", "fail", "error", "unknown"}

// NormalizeCheckStatus ensures the status is one of the known values accepted by the
// GraphQL enum, defaulting any unrecognized value to "unknown".
func NormalizeCheckStatus(status string) string {
	if slices.Contains(validCheckStatuses, status) {
		return status
	}
	return "unknown"
}
