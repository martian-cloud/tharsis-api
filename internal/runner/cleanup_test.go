package runner

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jobtypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// fakeCleanupClient is a hand-rolled Client that records cleanup interactions.
type fakeCleanupClient struct {
	Client
	claimJobs      []*CleanupJobInfo
	claimErr       error
	markedCleanArg []string
	markCleanErr   error
	sessionErrs    []error
}

func (f *fakeCleanupClient) ClaimJobsForCleanup(_ context.Context, _ *ClaimJobsForCleanupInput) ([]*CleanupJobInfo, error) {
	return f.claimJobs, f.claimErr
}

func (f *fakeCleanupClient) MarkJobsCleanedUp(_ context.Context, _ string, jobIDs []string) error {
	f.markedCleanArg = append(f.markedCleanArg, jobIDs...)
	return f.markCleanErr
}

func (f *fakeCleanupClient) CreateRunnerSessionError(_ context.Context, _ string, err error) error {
	f.sessionErrs = append(f.sessionErrs, err)
	return nil
}

// fakeCleanupDispatcher is a hand-rolled JobDispatcher that records cleaned resource names.
type fakeCleanupDispatcher struct {
	cleaned    []string
	cleanupErr error
}

func (f *fakeCleanupDispatcher) DispatchJob(_ context.Context, _ string, _ string) (map[string]string, error) {
	return nil, nil
}

func (f *fakeCleanupDispatcher) CleanupJob(_ context.Context, _ string, dispatcherData map[string]string) error {
	if f.cleanupErr != nil {
		return f.cleanupErr
	}
	f.cleaned = append(f.cleaned, dispatcherData["resourceName"])
	return nil
}

func (f *fakeCleanupDispatcher) Limits() *jobtypes.ResourceLimits {
	return nil
}

func TestCleanupWorker_cleanupJobs(t *testing.T) {
	testLogger, _ := logger.NewForTest()

	t.Run("cleans up each claimed job and marks it done", func(t *testing.T) {
		client := &fakeCleanupClient{
			claimJobs: []*CleanupJobInfo{
				{JobID: "job-1", DispatcherData: map[string]string{"resourceName": "pod-1"}},
				{JobID: "job-2", DispatcherData: map[string]string{"resourceName": "pod-2"}},
			},
		}
		dispatcher := &fakeCleanupDispatcher{}
		w := newCleanupWorker("runner-1", "session-1", client, dispatcher, testLogger)

		require.NoError(t, w.cleanupJobs(context.Background()))

		assert.Equal(t, []string{"pod-1", "pod-2"}, dispatcher.cleaned)
		assert.Equal(t, []string{"job-1", "job-2"}, client.markedCleanArg)
	})

	t.Run("a cleanup failure still marks the job done (best-effort)", func(t *testing.T) {
		client := &fakeCleanupClient{
			claimJobs: []*CleanupJobInfo{{JobID: "job-1", DispatcherData: map[string]string{"resourceName": "pod-1"}}},
		}
		dispatcher := &fakeCleanupDispatcher{cleanupErr: fmt.Errorf("boom")}
		w := newCleanupWorker("runner-1", "session-1", client, dispatcher, testLogger)

		require.NoError(t, w.cleanupJobs(context.Background()))

		// The dispatcher errored so nothing was recorded as cleaned, but the job is still marked done.
		assert.Empty(t, dispatcher.cleaned)
		assert.Equal(t, []string{"job-1"}, client.markedCleanArg)
		// The failure is reported as a runner session error.
		assert.Len(t, client.sessionErrs, 1)
	})

	t.Run("a claim failure is surfaced", func(t *testing.T) {
		client := &fakeCleanupClient{claimErr: fmt.Errorf("claim failed")}
		dispatcher := &fakeCleanupDispatcher{}
		w := newCleanupWorker("runner-1", "session-1", client, dispatcher, testLogger)

		assert.Error(t, w.cleanupJobs(context.Background()))
	})

	t.Run("a mark-cleaned-up failure is surfaced", func(t *testing.T) {
		client := &fakeCleanupClient{
			claimJobs:    []*CleanupJobInfo{{JobID: "job-1", DispatcherData: map[string]string{"resourceName": "pod-1"}}},
			markCleanErr: fmt.Errorf("mark failed"),
		}
		dispatcher := &fakeCleanupDispatcher{}
		w := newCleanupWorker("runner-1", "session-1", client, dispatcher, testLogger)

		assert.Error(t, w.cleanupJobs(context.Background()))
	})
}
