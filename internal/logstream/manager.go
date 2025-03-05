// Package logstream provides functionality for saving and retrieving logs
package logstream

//go:generate go tool mockery --name Manager --inpackage --case underscore

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// SubscriptionOptions includes options for setting up a log event subscription
type SubscriptionOptions struct {
	LastSeenLogSize *int
	LogStreamID     string
}

// LogEvent represents a log stream event
type LogEvent struct {
	Size      int
	Completed bool
}

// Manager interface encapsulates the logic for saving and retrieving logs
type Manager interface {
	WriteLogs(ctx context.Context, logStreamID string, startOffset int, buffer []byte) (*models.LogStream, error)
	ReadLogs(ctx context.Context, logStreamID string, startOffset int, limit int) ([]byte, error)
	Subscribe(ctx context.Context, options *SubscriptionOptions) (<-chan *LogEvent, error)
}

type stream struct {
	store        Store
	dbClient     *db.Client
	eventManager *events.EventManager
	logger       logger.Logger
}

// New creates an instance of the Manager interface
func New(store Store, dbClient *db.Client, eventManager *events.EventManager, logger logger.Logger) Manager {
	return &stream{
		store:        store,
		dbClient:     dbClient,
		eventManager: eventManager,
		logger:       logger,
	}
}

// WriteLogs saves a chunk of logs to the store
func (s *stream) WriteLogs(ctx context.Context, logStreamID string, startOffset int, buffer []byte) (*models.LogStream, error) {
	stream, err := s.dbClient.LogStreams.GetLogStreamByID(ctx, logStreamID)
	if err != nil {
		return nil, err
	}

	if stream == nil {
		return nil, errors.New("log stream not found: %s", logStreamID)
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	stream.Size = startOffset + len(buffer)

	// Wrap update in transaction to ensure that the DB is not updated if the logs cannot be written to the store
	updatedStream, err := s.dbClient.LogStreams.UpdateLogStream(txContext, stream)
	if err != nil {
		return nil, err
	}

	// Write logs to store
	if err = s.store.WriteLogs(ctx, logStreamID, startOffset, buffer); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return updatedStream, nil
}

// ReadLogs returns a chunk of logs
func (s *stream) ReadLogs(ctx context.Context, logStreamID string, startOffset int, limit int) ([]byte, error) {
	return s.store.ReadLogs(ctx, logStreamID, startOffset, limit)
}

func (s *stream) Subscribe(ctx context.Context, options *SubscriptionOptions) (<-chan *LogEvent, error) {
	logStream, err := s.dbClient.LogStreams.GetLogStreamByID(ctx, options.LogStreamID)
	if err != nil {
		return nil, err
	}

	if logStream == nil {
		return nil, fmt.Errorf("log stream not found with ID: %s", options.LogStreamID)
	}

	subscription := events.Subscription{
		Type: events.LogStreamSubscription,
		ID:   logStream.Metadata.ID,
		Actions: []events.SubscriptionAction{
			events.UpdateAction,
		},
	}
	subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})

	outgoing := make(chan *LogEvent)
	go func() {
		// Defer close of outgoing channel
		defer close(outgoing)
		defer s.eventManager.Unsubscribe(subscriber)

		if options.LastSeenLogSize != nil {
			if logStream.Size != *options.LastSeenLogSize {
				select {
				case <-ctx.Done():
					return
				case outgoing <- &LogEvent{Size: logStream.Size}:
				}
			}
		}

		// Wait for log stream updates
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if !errors.IsContextCanceledError(err) {
					s.logger.Errorf("error occurred while waiting for log events: %v", err)
				}
				return
			}

			logStreamEventData, err := event.ToLogStreamEventData()
			if err != nil {
				s.logger.Errorf("failed to get log stream event data in log stream subscription, log event %s: %v", event.ID, err)
				return
			}

			select {
			case <-ctx.Done():
				return
			case outgoing <- &LogEvent{Size: logStreamEventData.Size, Completed: logStreamEventData.Completed}:
			}

			// Return from loop if log stream has been completed because there are no more logs to process
			if logStream.Completed {
				return
			}
		}
	}()

	return outgoing, nil
}
