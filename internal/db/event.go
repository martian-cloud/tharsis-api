package db

//go:generate mockery --name Events --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"sync"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// Events provides the ability to listen for async events from the database
type Events interface {
	// Listen for async events from the database
	Listen(ctx context.Context) (<-chan Event, <-chan error)
}

// Event contains information related to the database row that was changed
type Event struct {
	Table  string `json:"table"`
	Action string `json:"action"`
	ID     string `json:"id"`
}

type events struct {
	dbClient *Client
}

var dbEventCount = metric.NewCounter("db_event_count", "Amount of database events.")

// NewEvents returns an instance of the Events interface
func NewEvents(dbClient *Client) Events {
	return &events{dbClient: dbClient}
}

func (e *events) Listen(ctx context.Context) (<-chan Event, <-chan error) {
	ch := make(chan Event)
	fatalErrors := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer close(ch)
		defer close(fatalErrors)

		// Don't try to do 'defer wg.Done()' here, because it will try to make it negative.

		conn, err := e.dbClient.conn.Acquire(ctx)
		if err != nil {
			e.dbClient.logger.Errorf("failed to acquire db connection in db events module: %v", err)
			fatalErrors <- errors.Wrap(err, errors.EInternal, "failed to acquire db connection from pool")
			wg.Done()
			return
		}
		defer conn.Release()

		_, err = conn.Exec(ctx, "listen events")
		if err != nil {
			e.dbClient.logger.Errorf("failed to start listening for db events: %v", err)
			fatalErrors <- errors.Wrap(err, errors.EInternal, "error when listening on events channel")
			wg.Done()
			return
		}

		// Let Listen return to its caller, now that listening is fully active.
		wg.Done()

		for {
			notification, err := conn.Conn().WaitForNotification(ctx)

			// Check if context has been cancelled
			if ctx.Err() != nil {
				return
			}

			if err != nil {
				e.dbClient.logger.Errorf("received error when listening for db event: %v", err)
				fatalErrors <- errors.Wrap(err, errors.EInternal, "error waiting for db notification")
				return
			}

			var event Event
			if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
				e.dbClient.logger.Errorf("failed to unmarshal db event %v", err)
				continue
			}

			dbEventCount.Inc()
			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Recording errors to the tracing span above is okay because of this wait.
	wg.Wait()
	return ch, fatalErrors
}
