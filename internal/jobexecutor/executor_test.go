package jobexecutor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/resource"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// fakeMonitor is a hand-rolled resource.Monitor that returns preset metrics on Stop.
type fakeMonitor struct {
	metrics   *resource.Metrics
	stopCalls int
}

func (m *fakeMonitor) Start()                           {}
func (m *fakeMonitor) Breaches() <-chan resource.Breach { return nil }
func (m *fakeMonitor) Stop() *resource.Metrics {
	m.stopCalls++
	return m.metrics
}

func TestSaveResourceUsage(t *testing.T) {
	testLogger, _ := logger.NewForTest()

	t.Run("persists collected metrics", func(t *testing.T) {
		client := &jobclient.MockClient{}
		client.On("SaveJobResourceUsage", mock.Anything, "job-1", mock.Anything, mock.Anything).Return(nil, nil).Once()

		j := &JobExecutor{
			cfg:         &JobConfig{JobID: "job-1"},
			client:      client,
			logger:      testLogger,
			resourceMon: &fakeMonitor{metrics: &resource.Metrics{}},
		}
		j.saveResourceUsage(context.Background())

		client.AssertExpectations(t)
	})

	t.Run("is idempotent so panic and normal paths never double-save", func(t *testing.T) {
		client := &jobclient.MockClient{}
		client.On("SaveJobResourceUsage", mock.Anything, "job-1", mock.Anything, mock.Anything).Return(nil, nil).Once()

		mon := &fakeMonitor{metrics: &resource.Metrics{}}
		j := &JobExecutor{
			cfg:         &JobConfig{JobID: "job-1"},
			client:      client,
			logger:      testLogger,
			resourceMon: mon,
		}
		j.saveResourceUsage(context.Background())
		j.saveResourceUsage(context.Background()) // second call must be a no-op

		client.AssertExpectations(t) // Once() fails if SaveJobResourceUsage ran twice
		if mon.stopCalls != 1 {
			t.Fatalf("expected monitor to be stopped once, got %d", mon.stopCalls)
		}
	})

	t.Run("does nothing when no monitor was started", func(t *testing.T) {
		client := &jobclient.MockClient{}

		j := &JobExecutor{
			cfg:    &JobConfig{JobID: "job-1"},
			client: client,
			logger: testLogger,
		}
		j.saveResourceUsage(context.Background())

		client.AssertNotCalled(t, "SaveJobResourceUsage")
	})

	t.Run("does nothing when no metrics were collected", func(t *testing.T) {
		client := &jobclient.MockClient{}

		j := &JobExecutor{
			cfg:         &JobConfig{JobID: "job-1"},
			client:      client,
			logger:      testLogger,
			resourceMon: &fakeMonitor{metrics: nil},
		}
		j.saveResourceUsage(context.Background())

		client.AssertNotCalled(t, "SaveJobResourceUsage")
	})

	t.Run("swallows save errors so a failed save never blocks job status", func(t *testing.T) {
		client := &jobclient.MockClient{}
		client.On("SaveJobResourceUsage", mock.Anything, "job-1", mock.Anything, mock.Anything).Return(nil, errors.New("grpc down")).Once()

		j := &JobExecutor{
			cfg:         &JobConfig{JobID: "job-1"},
			client:      client,
			logger:      testLogger,
			resourceMon: &fakeMonitor{metrics: &resource.Metrics{}},
		}
		j.saveResourceUsage(context.Background())

		client.AssertExpectations(t)
	})
}
