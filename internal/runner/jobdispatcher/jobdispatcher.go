package jobdispatcher

import (
	"context"
)

// JobDispatcher is used to dispatch a job to various runtime environments
type JobDispatcher interface {
	DispatchJob(ctx context.Context, jobID string, token string) (string, error)
}
