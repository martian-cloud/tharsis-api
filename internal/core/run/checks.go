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

var validCheckStatuses = []string{"pass", "fail", "error", "unknown"}

// NormalizeCheckStatus returns the status unchanged if it is a known Terraform check
// status value, or "unknown" for any unrecognized input.
func NormalizeCheckStatus(status string) string {
	if slices.Contains(validCheckStatuses, status) {
		return status
	}
	return "unknown"
}
