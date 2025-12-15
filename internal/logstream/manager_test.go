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

	tests := []struct {
		name               string
		lastSeenLogSize    *int
		logStreamSize      int
		logStreamExists    bool
		expectErrorMessage string
		expectedEvents     []struct {
			size   int
			offset int
			logs   string
		}
		dbEvents []db.LogStreamEventData
	}{
		{
			name:            "successful subscription with no last seen size",
			logStreamExists: true,
			logStreamSize:   10,
			dbEvents: []db.LogStreamEventData{
				{Size: 13, Completed: false},
			},
			expectedEvents: []struct {
				size   int
				offset int
				logs   string
			}{
				{size: 13, offset: 10, logs: "new"},
			},
		},
		{
			name:            "successful subscription with last seen size",
			lastSeenLogSize: ptr.Int(5),
			logStreamExists: true,
			logStreamSize:   15,
			expectedEvents: []struct {
				size   int
				offset int
				logs   string
			}{
				{size: 10, offset: 5, logs: "chunk"},
				{size: 15, offset: 10, logs: "data2"},
			},
		},
		{
			name:               "log stream not found",
			logStreamExists:    false,
			expectErrorMessage: "log stream not found with ID",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			mockLogStreams := db.NewMockLogStreams(t)
			mockStore := NewMockStore(t)

			var mockEvents *db.MockEvents
			var eventChan chan db.Event
			var errorChan chan error

			if len(test.dbEvents) > 0 {
				mockEvents = db.NewMockEvents(t)
				eventChan = make(chan db.Event, 10)
				errorChan = make(chan error, 1)
				mockEvents.On("Listen", mock.Anything).Return((<-chan db.Event)(eventChan), (<-chan error)(errorChan))
			}

			var logStream *models.LogStream
			if test.logStreamExists {
				logStream = &models.LogStream{
					Metadata: models.ResourceMetadata{
						ID: streamID,
					},
					Size:      test.logStreamSize,
					Completed: false,
				}
			}

			mockLogStreams.On("GetLogStreamByID", mock.Anything, streamID).Return(logStream, nil)

			if test.logStreamExists && len(test.expectedEvents) > 0 {
				for _, event := range test.expectedEvents {
					offset := event.offset
					mockStore.On("ReadLogs", mock.Anything, streamID, offset, mock.AnythingOfType("int")).Return([]byte(event.logs), nil).Once()
				}
			}

			// Mock for db events
			if len(test.dbEvents) > 0 {
				mockStore.On("ReadLogs", mock.Anything, streamID, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return([]byte("new"), nil).Maybe()
			}

			dbClient := &db.Client{
				LogStreams: mockLogStreams,
				Events:     mockEvents,
			}

			logger, _ := logger.NewForTest()
			var eventManager *events.EventManager
			if mockEvents != nil {
				eventManager = events.NewEventManager(dbClient, logger)
				eventManager.Start(ctx)
			} else {
				eventManager = events.NewEventManager(nil, logger)
			}

			manager := New(mockStore, dbClient, eventManager, logger)

			options := &SubscriptionOptions{
				LogStreamID:     streamID,
				LastSeenLogSize: test.lastSeenLogSize,
			}

			channel, err := manager.Subscribe(ctx, options)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, channel)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, channel)

				// Send db events after subscription is established
				go func() {
					time.Sleep(10 * time.Millisecond) // Allow subscription to be established
					for _, dbEvent := range test.dbEvents {
						eventData, _ := json.Marshal(dbEvent)
						eventChan <- db.Event{
							Table:  "log_streams",
							Action: "UPDATE",
							ID:     streamID,
							Data:   eventData,
						}
					}
				}()

				// Verify we can receive expected events
				for i, expectedEvent := range test.expectedEvents {
					select {
					case event := <-channel:
						assert.Equal(t, expectedEvent.size, event.Size)
						require.NotNil(t, event.Data)
						assert.Equal(t, expectedEvent.offset, event.Data.Offset)
						assert.Equal(t, expectedEvent.logs, event.Data.Logs)
					case <-time.After(100 * time.Millisecond):
						t.Fatalf("expected to receive event %d", i+1)
					}
				}
			}
		})
	}
}
