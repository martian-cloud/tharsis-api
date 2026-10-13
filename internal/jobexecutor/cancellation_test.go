package jobexecutor

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeCancellationStream replays recv results in order; a nil event with a nil error blocks until
// the stream context is done.
type fakeCancellationStream struct {
	grpc.ClientStream
	ctx     context.Context
	results []streamResult
}

type streamResult struct {
	event *pb.JobCancellationEvent
	err   error
}

func (s *fakeCancellationStream) Recv() (*pb.JobCancellationEvent, error) {
	if len(s.results) == 0 {
		<-s.ctx.Done()
		return nil, s.ctx.Err()
	}
	r := s.results[0]
	s.results = s.results[1:]
	return r.event, r.err
}

func cancelEvent(jobStatus pb.JobStatus, forceCanceled bool) streamResult {
	return streamResult{event: &pb.JobCancellationEvent{Job: &pb.Job{Status: jobStatus, ForceCanceled: forceCanceled}}}
}

// mockSubscriptions makes each SubscribeToJobCancellationEvent call return the next stream.
func mockSubscriptions(client *jobclient.MockClient, streams ...[]streamResult) {
	for _, results := range streams {
		client.On("SubscribeToJobCancellationEvent", mock.Anything, "job-1").
			Return(func(ctx context.Context, _ string) (grpc.ServerStreamingClient[pb.JobCancellationEvent], error) {
				return &fakeCancellationStream{ctx: ctx, results: results}, nil
			}).Once()
	}
}

func newCancellationTestExecutor(client jobclient.Client) *JobExecutor {
	cancellableCtx, cancel := context.WithCancel(context.Background())
	return &JobExecutor{
		cfg:            &JobConfig{JobID: "job-1"},
		client:         client,
		cancellableCtx: cancellableCtx,
		cancelFunc:     cancel,
		exit:           func(int) { panic("unexpected exit") },
	}
}

func TestJobExecutor_waitForJobCancellation(t *testing.T) {
	tests := []struct {
		name           string
		streams        [][]streamResult
		expectCanceled bool
	}{
		{
			name:           "a canceling job is reported as canceled",
			streams:        [][]streamResult{{cancelEvent(pb.JobStatus_canceling, false)}},
			expectCanceled: true,
		},
		{
			name:           "a canceled job is reported as canceled",
			streams:        [][]streamResult{{cancelEvent(pb.JobStatus_canceled, false)}},
			expectCanceled: true,
		},
		{
			name: "a cancellation after a reconnect is reported",
			streams: [][]streamResult{
				{{err: status.Error(codes.Unavailable, "draining")}},
				{cancelEvent(pb.JobStatus_canceling, false)},
			},
			expectCanceled: true,
		},
		{
			name: "a stream ending without a cancellation is not reported as canceled",
			streams: [][]streamResult{{
				cancelEvent(pb.JobStatus_running, false),
				{err: io.EOF},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			client := jobclient.NewMockClient(t)
			mockSubscriptions(client, test.streams...)

			j := newCancellationTestExecutor(client)

			canceled, err := j.waitForJobCancellation(ctx)
			require.NoError(t, err)
			assert.Equal(t, test.expectCanceled, canceled)
		})
	}
}

func TestJobExecutor_startCancellationMonitor_ExitsOnForceCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := jobclient.NewMockClient(t)
	mockSubscriptions(client, []streamResult{cancelEvent(pb.JobStatus_canceling, false)})
	// After the graceful cancel, the force cancel poll reads the job, which a finished job's token may still do.
	client.On("GetJob", mock.Anything, "job-1").Return(&pb.Job{Status: pb.JobStatus_canceled, ForceCanceled: true}, nil)

	jobLogger := joblogger.NewMockLogger(t)
	jobLogger.On("Warningf", "Received a graceful job cancel request").Once()
	jobLogger.On("Warningf", "Received force cancel request for this job").Once()

	j := newCancellationTestExecutor(client)

	var once sync.Once
	exited := make(chan int, 1)
	j.exit = func(code int) { once.Do(func() { exited <- code }) }

	j.startCancellationMonitor(ctx, jobLogger, &pb.Job{Metadata: &pb.ResourceMetadata{Id: "job-1"}, MaxJobDuration: 60})

	select {
	case code := <-exited:
		assert.Equal(t, 1, code)
	case <-ctx.Done():
		t.Fatal("executor did not exit on force cancel")
	}
}

func TestJobExecutor_execute_CanceledBeforeStart(t *testing.T) {
	tests := []struct {
		name                  string
		setStatusErr          error
		job                   *pb.Job
		getJobErr             error
		expectAlreadyCanceled bool
	}{
		{
			name:                  "an update refused because the job is final is detected as canceled",
			setStatusErr:          status.Error(codes.Unauthenticated, "job token is no longer valid: the job has finished"),
			job:                   &pb.Job{Status: pb.JobStatus_canceled},
			expectAlreadyCanceled: true,
		},
		{
			name:                  "an illegal transition from canceled is detected as canceled",
			setStatusErr:          status.Error(codes.InvalidArgument, "cannot set job status"),
			job:                   &pb.Job{Status: pb.JobStatus_canceled},
			expectAlreadyCanceled: true,
		},
		{
			name:         "an error for a job that isn't canceled is returned",
			setStatusErr: status.Error(codes.Unavailable, "down"),
			job:          &pb.Job{Status: pb.JobStatus_pending},
		},
		{
			name:         "the original error is returned when the job can't be read",
			setStatusErr: status.Error(codes.Unavailable, "down"),
			getJobErr:    status.Error(codes.Unavailable, "down"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := jobclient.NewMockClient(t)
			client.On("SetJobStatus", mock.Anything, "job-1", pb.JobStatus_running, mock.Anything).Return(nil, test.setStatusErr)
			client.On("GetJob", mock.Anything, "job-1").Return(test.job, test.getJobErr)

			j := newCancellationTestExecutor(client)

			err := j.execute(context.Background(), joblogger.NewMockLogger(t))
			require.Error(t, err)
			if test.expectAlreadyCanceled {
				assert.Equal(t, errJobAlreadyCanceled, err)
			} else {
				assert.NotEqual(t, errJobAlreadyCanceled, err)
				assert.Contains(t, err.Error(), "failed to set job status to running")
			}
		})
	}
}

func TestJobExecutor_startCancellationMonitor_BacksOffWhenStreamEnds(t *testing.T) {
	orig := waitCancelStreamEnded
	waitCancelStreamEnded = 100 * time.Millisecond
	t.Cleanup(func() { waitCancelStreamEnded = orig })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Every subscription ends immediately without a cancellation, as a server that keeps closing the
	// stream would. Without a backoff this would resubscribe in a tight loop.
	var subscribes atomic.Int32
	client := jobclient.NewMockClient(t)
	client.On("SubscribeToJobCancellationEvent", mock.Anything, "job-1").
		Return(func(ctx context.Context, _ string) (grpc.ServerStreamingClient[pb.JobCancellationEvent], error) {
			subscribes.Add(1)
			return &fakeCancellationStream{ctx: ctx, results: []streamResult{{err: io.EOF}}}, nil
		})

	j := newCancellationTestExecutor(client)
	j.startCancellationMonitor(ctx, joblogger.NewMockLogger(t), &pb.Job{Metadata: &pb.ResourceMetadata{Id: "job-1"}, MaxJobDuration: 60})

	time.Sleep(350 * time.Millisecond)
	cancel()
	// Let the monitor goroutine see the cancel and stop (before the cleanup restores the delay it reads).
	time.Sleep(50 * time.Millisecond)

	// About one subscription per backoff interval (~4 in 350ms), not thousands.
	got := subscribes.Load()
	assert.GreaterOrEqual(t, got, int32(2))
	assert.LessOrEqual(t, got, int32(6))

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, got, subscribes.Load(), "the monitor must stop resubscribing once the job shuts down")
}
