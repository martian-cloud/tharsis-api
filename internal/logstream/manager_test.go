package logstream

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestWriteLogs(t *testing.T) {
	streamID := "stream1"

	// Test cases
	tests := []struct {
		name        string
		logs        string
		startOffset int
	}{
		{
			name:        "write logs with 0 offset",
			startOffset: 0,
			logs:        "this is a test log",
		},
		{
			name:        "write logs with offset greater than 0",
			startOffset: 100,
			logs:        "this is a test log",
		},
		{
			name:        "write an empty log",
			startOffset: 0,
			logs:        "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockTransactions := db.NewMockTransactions(t)

			mockLogStreams := db.NewMockLogStreams(t)
			mockStore := NewMockStore(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockLogStreams.On("GetLogStreamByID", mock.Anything, streamID).Return(&models.LogStream{
				Metadata: models.ResourceMetadata{
					ID: streamID,
				},
			}, nil)

			mockLogStreams.On("UpdateLogStream", mock.Anything, &models.LogStream{
				Metadata: models.ResourceMetadata{
					ID: streamID,
				},
				Size:      test.startOffset + len(test.logs),
				Completed: false,
			}).Return(&models.LogStream{
				Metadata: models.ResourceMetadata{
					ID: streamID,
				},
				Size: test.startOffset + len(test.logs),
			}, nil)

			mockStore.On("WriteLogs", mock.Anything, streamID, test.startOffset, []byte(test.logs)).Return(nil)

			dbClient := &db.Client{
				Transactions: mockTransactions,
				LogStreams:   mockLogStreams,
			}

			logger, _ := logger.NewForTest()

			manager := New(mockStore, dbClient, nil, logger)

			updatedLogStream, err := manager.WriteLogs(ctx, streamID, test.startOffset, []byte(test.logs))
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.startOffset+len(test.logs), updatedLogStream.Size)
		})
	}
}

func TestReadLogs(t *testing.T) {
	streamID := "stream1"
	startOffset := 0
	limit := 100
	logs := []byte("this is a test log")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockStore := NewMockStore(t)

	mockStore.On("ReadLogs", mock.Anything, streamID, startOffset, limit).Return(logs, nil)

	logger, _ := logger.NewForTest()

	manager := New(mockStore, nil, nil, logger)

	resp, err := manager.ReadLogs(ctx, streamID, startOffset, limit)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, logs, resp)
}

func TestSubscribe(t *testing.T) {
	streamID := "stream1"

	// Test cases
	tests := []struct {
		name           string
		lastSeenSize   *int
		sendEventData  []*db.LogStreamEventData
		expectedEvents []LogEvent
	}{
		{
			name: "stream 2 events with last seen size not set",
			sendEventData: []*db.LogStreamEventData{
				{Size: 5, Completed: false},
				{Size: 10, Completed: true},
			},
			expectedEvents: []LogEvent{
				{Size: 5, Completed: false},
				{Size: 10, Completed: true},
			},
		},
		{
			name: "last seen does not equal the current stream size",
			sendEventData: []*db.LogStreamEventData{
				{Size: 5, Completed: false},
				{Size: 10, Completed: true},
			},
			expectedEvents: []LogEvent{
				{Size: 2, Completed: false},
				{Size: 5, Completed: false},
				{Size: 10, Completed: true},
			},
			lastSeenSize: ptr.Int(3),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			mockStore := NewMockStore(t)
			mockLogStreams := db.NewMockLogStreams(t)
			mockEvents := db.NewMockEvents(t)

			mockEventChannel := make(chan db.Event, 1)
			var roEventChan <-chan db.Event = mockEventChannel
			mockEvents.On("Listen", mock.Anything).Return(roEventChan, make(<-chan error)).Maybe()

			mockLogStreams.On("GetLogStreamByID", mock.Anything, streamID).Return(&models.LogStream{
				Metadata: models.ResourceMetadata{
					ID: streamID,
				},
				Size: 2,
			}, nil).Once()

			dbClient := &db.Client{
				LogStreams: mockLogStreams,
				Events:     mockEvents,
			}

			logger, _ := logger.NewForTest()

			eventManager := events.NewEventManager(dbClient, logger)
			eventManager.Start(ctx)

			manager := New(mockStore, dbClient, eventManager, logger)

			eventChannel, err := manager.Subscribe(ctx, &SubscriptionOptions{
				LogStreamID:     streamID,
				LastSeenLogSize: test.lastSeenSize,
			})
			if err != nil {
				t.Fatal(err)
			}

			receivedEvents := []*LogEvent{}

			go func() {
				for _, d := range test.sendEventData {
					encoded, err := json.Marshal(d)
					require.Nil(t, err)

					mockEventChannel <- db.Event{
						Table:  "log_streams",
						Action: string(events.UpdateAction),
						ID:     streamID,
						Data:   encoded,
					}
				}
			}()

			for e := range eventChannel {
				eCopy := e
				receivedEvents = append(receivedEvents, eCopy)

				if len(receivedEvents) == len(test.expectedEvents) {
					break
				}
			}

			require.Equal(t, len(test.expectedEvents), len(receivedEvents))
			for i, e := range test.expectedEvents {
				assert.Equal(t, e, *receivedEvents[i])
			}
		})
	}
}
