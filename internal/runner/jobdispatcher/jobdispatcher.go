// Package jobdispatcher package
package jobdispatcher

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
)

// JobDispatcher dispatches jobs to a runtime environment and cleans up their runtimes afterward.
type JobDispatcher interface {
	// DispatchJob launches the job and returns opaque dispatcher data (e.g. a pod name or container ID)
	// that each plugin defines and later consumes in CleanupJob.
	DispatchJob(ctx context.Context, jobID string, token string) (map[string]string, error)
	// CleanupJob tears down the runtime for a finished job using the data DispatchJob returned. Dispatchers
	// whose runtimes self-clean implement this as a no-op.
	CleanupJob(ctx context.Context, jobID string, dispatcherData map[string]string) error
	// Limits returns the resource limits jobs run under, which the runner reports at dispatch time.
	Limits() *types.ResourceLimits
}
