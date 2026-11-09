package providerregistry

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// fakeMaintenanceMonitor is a minimal maintenance.Monitor stub for consumer tests.
type fakeMaintenanceMonitor struct {
	inMaintenance bool
	err           error
}

func (f *fakeMaintenanceMonitor) Start(context.Context) {}
func (f *fakeMaintenanceMonitor) InMaintenanceMode(context.Context) (bool, error) {
	return f.inMaintenance, f.err
}

// TestHandleWorkItemDispatchesProcessTFProviderDocs verifies a docs work item is routed to the
// processor with the payload's provider version ID.
func TestHandleWorkItemDispatchesProcessTFProviderDocs(t *testing.T) {
	ctx := context.Background()

	mockProcessor := NewMockDocsProcessor(t)
	mockProcessor.On("ProcessProviderVersionDocs", mock.Anything, "pv-1", false).Return(nil).Once()

	log, _ := logger.NewForTest()
	s := NewWorkItemConsumer(log, nil, nil, nil, mockProcessor)

	err := s.handleWorkItem(ctx, &db.WorkItem{
		ID:      "wi-docs",
		Type:    db.ProcessTFProviderDocsType,
		Payload: &db.ProcessTFProviderDocsPayload{ProviderVersionID: "pv-1"},
	})

	assert.NoError(t, err)
	mockProcessor.AssertCalled(t, "ProcessProviderVersionDocs", mock.Anything, "pv-1", false)
}

// TestClaimAndProcess_AcknowledgesWhenHandlerSucceeds verifies a nil handler result acknowledges the item.
// A nil (invalid) payload is the cheapest nil-result path: handleWorkItem logs it and returns nil without
// invoking the processor, so the item is acked rather than redelivered.
func TestClaimAndProcess_AcknowledgesWhenHandlerSucceeds(t *testing.T) {
	ctx := context.Background()

	workItem := db.WorkItem{
		ID:      "wi-1",
		Type:    db.ProcessTFProviderDocsType,
		Payload: nil, // not a *ProcessTFProviderDocsPayload, so handleWorkItem returns nil without the processor
	}

	mockQueue := db.NewMockWorkItemsQueue(t)
	mockQueue.On("ClaimWorkItems", mock.Anything, mock.MatchedBy(func(in *db.ClaimWorkItemsInput) bool {
		return in.Type == db.ProcessTFProviderDocsType
	})).Return([]db.WorkItem{workItem}, nil)
	mockQueue.On("AcknowledgeWorkItem", mock.Anything, "wi-1").Return(nil).Once()

	// No processor expectation: an invalid payload must be dropped before the processor is invoked.
	mockProcessor := NewMockDocsProcessor(t)

	dbClient := &db.Client{WorkItemsQueue: mockQueue}
	log, _ := logger.NewForTest()
	s := NewWorkItemConsumer(log, dbClient, nil, &fakeMaintenanceMonitor{}, mockProcessor)

	s.claimAndProcess(ctx)

	mockQueue.AssertCalled(t, "AcknowledgeWorkItem", mock.Anything, "wi-1")
}

// TestClaimAndProcess_DoesNotAcknowledgeWhenHandlingFails verifies a failed item is left unacknowledged for redelivery.
func TestClaimAndProcess_DoesNotAcknowledgeWhenHandlingFails(t *testing.T) {
	ctx := context.Background()

	workItem := db.WorkItem{
		ID:      "wi-1",
		Type:    db.ProcessTFProviderDocsType,
		Payload: &db.ProcessTFProviderDocsPayload{ProviderVersionID: "pv-1"},
	}

	mockQueue := db.NewMockWorkItemsQueue(t)
	mockQueue.On("ClaimWorkItems", mock.Anything, mock.Anything).Return([]db.WorkItem{workItem}, nil)
	// No AcknowledgeWorkItem expectation: the mock fails the test if called, asserting no ack on failure.

	mockProcessor := NewMockDocsProcessor(t)
	mockProcessor.On("ProcessProviderVersionDocs", mock.Anything, "pv-1", false).
		Return(errors.New("boom", errors.WithErrorCode(errors.EInternal))).Once()

	dbClient := &db.Client{WorkItemsQueue: mockQueue}
	log, _ := logger.NewForTest()
	s := NewWorkItemConsumer(log, dbClient, nil, &fakeMaintenanceMonitor{}, mockProcessor)

	s.claimAndProcess(ctx)

	mockQueue.AssertNotCalled(t, "AcknowledgeWorkItem", mock.Anything, mock.Anything)
}

// TestClaimAndProcess_SkipsClaimWhileInMaintenance verifies claiming is skipped during maintenance mode.
func TestClaimAndProcess_SkipsClaimWhileInMaintenance(t *testing.T) {
	ctx := context.Background()

	// No ClaimWorkItems expectation: the mock fails the test if called, asserting no claim in maintenance.
	mockQueue := db.NewMockWorkItemsQueue(t)
	dbClient := &db.Client{WorkItemsQueue: mockQueue}

	log, _ := logger.NewForTest()
	s := NewWorkItemConsumer(log, dbClient, nil, &fakeMaintenanceMonitor{inMaintenance: true}, nil)

	s.claimAndProcess(ctx)
}

// TestMatchDocsWorkItemEvent verifies the subscription filter only matches this consumer's own work-item type.
func TestMatchDocsWorkItemEvent(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		match bool
	}{
		{name: "own type", data: `{"type":"PROCESS_TF_PROVIDER_DOCS"}`, match: true},
		{name: "run engine type", data: `{"type":"QUEUE_PENDING_RUNS_FOR_WORKSPACE"}`, match: false},
		{name: "unknown type", data: `{"type":"SOMETHING_ELSE"}`, match: false},
		{name: "invalid json", data: `not-json`, match: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.match, matchDocsWorkItemEvent(json.RawMessage(test.data)))
		})
	}
}
