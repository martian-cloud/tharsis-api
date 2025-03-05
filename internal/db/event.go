package db

//go:generate go tool mockery --name Events --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	eventTableJobs           = "jobs"
	eventTableLogStreams     = "log_streams"
	eventTableRuns           = "runs"
	eventTableRunnerSessions = "runner_sessions"
)

// Events provides the ability to listen for async events from the database
type Events interface {
	// Listen for async events from the database
	Listen(ctx context.Context) (<-chan Event, <-chan error)
}

// Event contains processed information related to the database row that was changed
// The ID field is needed for triage independent of the type of the event data.
type Event struct {
	Table  string          `json:"table"`
	Action string          `json:"action"`
	ID     string          `json:"id"`
	Data   json.RawMessage `json:"data"`
}

// JobEventData contains the event response data for a row from the jobs table.
type JobEventData struct {
	ID              string  `json:"id"`
	RunnerID        *string `json:"runner_id"`
	WorkspaceID     string  `json:"workspace_id"`
	CancelRequested bool    `json:"cancel_requested"`
}

// LogStreamEventData contains the event response data for a row from the log_streams table.
type LogStreamEventData struct {
	Size      int  `json:"size"`
	Completed bool `json:"completed"`
}

// RunEventData contains the event response data for a row from the runs table.
type RunEventData struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
}

// RunnerSessionEventData contains the event response data for a row from the runner_sessions table.
type RunnerSessionEventData struct {
	ID       string `json:"id"`
	RunnerID string `json:"runner_id"`
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
			fatalErrors <- errors.Wrap(err, "failed to acquire db connection from pool")
			wg.Done()
			return
		}
		defer conn.Release()

		_, err = conn.Exec(ctx, "listen events")
		if err != nil {
			e.dbClient.logger.Errorf("failed to start listening for db events: %v", err)
			fatalErrors <- errors.Wrap(err, "error when listening on events channel")
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
				fatalErrors <- errors.Wrap(err, "error waiting for db notification")
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

/////////////////////////////////////////////////////////////////////////////////
// Shorthand methods to convert an event to specific event data references.

// ToJobEventData is a shorthand method to return type-checked event data.
func (e *Event) ToJobEventData() (*JobEventData, error) {
	if e.Table != eventTableJobs {
		return nil, fmt.Errorf("invalid event table, expected '%s': %s", eventTableJobs, e.Table)
	}

	d := JobEventData{}
	if err := json.Unmarshal(e.Data, &d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal db jobs event data, %v", err)
	}

	return &d, nil
}

// ToLogStreamEventData is a shorthand method to return type-checked event data.
func (e *Event) ToLogStreamEventData() (*LogStreamEventData, error) {
	if e.Table != eventTableLogStreams {
		return nil, fmt.Errorf("invalid event table, expected '%s': %s", eventTableLogStreams, e.Table)
	}

	d := LogStreamEventData{}
	if err := json.Unmarshal(e.Data, &d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal db log_streams event data, %v", err)
	}

	return &d, nil
}

// ToRunEventData is a shorthand method to return type-checked event data.
func (e *Event) ToRunEventData() (*RunEventData, error) {
	if e.Table != eventTableRuns {
		return nil, fmt.Errorf("invalid event table, expected '%s': %s", eventTableRuns, e.Table)
	}

	d := RunEventData{}
	if err := json.Unmarshal(e.Data, &d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal db runs event data, %v", err)
	}

	return &d, nil
}

// ToRunnerSessionEventData is a shorthand method to return type-checked event data.
func (e *Event) ToRunnerSessionEventData() (*RunnerSessionEventData, error) {
	if e.Table != eventTableRunnerSessions {
		return nil, fmt.Errorf("invalid event table, expected '%s': %s", eventTableRunnerSessions, e.Table)
	}

	d := RunnerSessionEventData{}
	if err := json.Unmarshal(e.Data, &d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal db runner_sessions event data, %v", err)
	}

	return &d, nil
}
