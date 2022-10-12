package events

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
)

func TestGetEvent(t *testing.T) {
	// Test cases
	tests := []struct {
		name          string
		subscriptions []Subscription
		events        []db.Event
		expectEvents  []db.Event
	}{
		{
			name: "multiple subscriptions",
			subscriptions: []Subscription{
				{
					Type:    JobSubscription,
					Actions: []SubscriptionAction{CreateAction, UpdateAction},
				},
				{
					Type:    JobSubscription,
					Actions: []SubscriptionAction{DeleteAction},
				},
			},
			events: []db.Event{
				{Table: "jobs", Action: "INSERT", ID: "1"},
				{Table: "jobs", Action: "UPDATE", ID: "1"},
				{Table: "jobs", Action: "DELETE", ID: "1"},
			},
			expectEvents: []db.Event{
				{Table: "jobs", Action: "INSERT", ID: "1"},
				{Table: "jobs", Action: "UPDATE", ID: "1"},
				{Table: "jobs", Action: "DELETE", ID: "1"},
			},
		},
		{
			name: "subscribe to specific resource ID",
			subscriptions: []Subscription{
				{
					Type:    JobSubscription,
					Actions: []SubscriptionAction{UpdateAction},
					ID:      "1",
				},
			},
			events: []db.Event{
				{Table: "jobs", Action: "UPDATE", ID: "1"},
				{Table: "jobs", Action: "UPDATE", ID: "2"},
				{Table: "jobs", Action: "UPDATE", ID: "3"},
			},
			expectEvents: []db.Event{
				{Table: "jobs", Action: "UPDATE", ID: "1"},
			},
		},
		{
			name: "subscribe to all actions",
			subscriptions: []Subscription{
				{
					Type: JobSubscription,
				},
			},
			events: []db.Event{
				{Table: "jobs", Action: "INSERT", ID: "1"},
				{Table: "jobs", Action: "UPDATE", ID: "2"},
				{Table: "jobs", Action: "DELETE", ID: "3"},
				{Table: "test", Action: "DELETE", ID: "3"},
			},
			expectEvents: []db.Event{
				{Table: "jobs", Action: "INSERT", ID: "1"},
				{Table: "jobs", Action: "UPDATE", ID: "2"},
				{Table: "jobs", Action: "DELETE", ID: "3"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*1))
			defer cancel()

			eventChan := make(chan db.Event)
			errorChan := make(chan error)

			mockEvents := createMockEvents(eventChan, errorChan)

			client := db.Client{
				Events: mockEvents,
			}

			manager := NewEventManager(&client)
			manager.Start(ctx)

			subscriber := manager.Subscribe(test.subscriptions)

			// Send events
			for _, event := range test.events {
				eventChan <- event
			}

			foundEvents := 0

			// Wait for all expected events
			for foundEvents < len(test.expectEvents) {
				event, err := subscriber.GetEvent(ctx)
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				assert.True(t, containsEvent(test.expectEvents, *event), "Unexpected event received %v", *event)

				foundEvents++
			}
		})
	}
}

func TestUnsubscribe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*20))
	defer cancel()

	eventChan := make(chan db.Event)
	errorChan := make(chan error)

	mockEvents := createMockEvents(eventChan, errorChan)

	client := db.Client{
		Events: mockEvents,
	}

	manager := NewEventManager(&client)
	manager.Start(ctx)

	subscriber := manager.Subscribe([]Subscription{
		{
			Type: JobSubscription,
		},
	})

	eventChan <- db.Event{Table: "jobs", Action: "CREATE", ID: "1"}

	event, err := subscriber.GetEvent(ctx)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	assert.NotNil(t, event)

	manager.Unsubscribe(subscriber)
	assert.Equal(t, 0, len(manager.subscribers))

	// This should return a context cancelled since the subscriber has been removed
	_, err = subscriber.GetEvent(ctx)
	assert.EqualError(t, err, "Subscriber has been unsubscribed")
}

func TestGetEventOnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*1))
	defer cancel()

	eventChan := make(chan db.Event)
	errorChan := make(chan error)

	mockEvents := createMockEvents(eventChan, errorChan)

	client := db.Client{
		Events: mockEvents,
	}

	manager := NewEventManager(&client)
	manager.Start(ctx)

	subscriber := manager.Subscribe([]Subscription{})

	// Send Error
	errorChan <- fmt.Errorf("error occurred while listening")

	_, err := subscriber.GetEvent(ctx)
	assert.EqualError(t, err, "error occurred while listening")
}

func TestGetEventContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventChan := make(chan db.Event)
	errorChan := make(chan error)

	mockEvents := createMockEvents(eventChan, errorChan)

	client := db.Client{
		Events: mockEvents,
	}

	manager := NewEventManager(&client)
	manager.Start(ctx)

	subscriber := manager.Subscribe([]Subscription{})

	// Cancel context
	cancel()

	_, err := subscriber.GetEvent(ctx)
	assert.EqualError(t, err, "context canceled")
}

func createMockEvents(eventChan chan db.Event, errorChan chan error) *db.MockEvents {
	var roEventChan <-chan db.Event
	var roErrorChan <-chan error
	roEventChan = eventChan
	roErrorChan = errorChan

	mockEvents := db.MockEvents{}
	mockEvents.On("Listen", mock.Anything).Return(roEventChan, roErrorChan)

	return &mockEvents
}

func containsEvent(events []db.Event, event db.Event) bool {
	for _, e := range events {
		if event == e {
			return true
		}
	}
	return false
}
