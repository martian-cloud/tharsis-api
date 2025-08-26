// Package events package
package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	// sendEventTimeout is the timeout for sending an event to a subscriber
	sendEventTimeout = time.Minute * 5
	// subscriberBufferSize is the buffer size for the subscriber channel
	subscriberBufferSize = 500
)

// SubscriptionType specifies the type of subscription
type SubscriptionType string

// SubscriptionType constants
const (
	JobSubscription             SubscriptionType = "jobs"
	LogStreamSubscription       SubscriptionType = "log_streams"
	RunSubscription             SubscriptionType = "runs"
	WorkspaceSubscription       SubscriptionType = "workspaces"
	RunnerSubscription          SubscriptionType = "runners"
	MaintenanceModeSubscription SubscriptionType = "maintenance_mode"
	RunnerSessionSubscription   SubscriptionType = "runner_sessions"
)

// SubscriptionAction type represents the available actions that can be subscribed type
type SubscriptionAction string

// SubscriptionAction constants
const (
	CreateAction SubscriptionAction = "INSERT"
	UpdateAction SubscriptionAction = "UPDATE"
	DeleteAction SubscriptionAction = "DELETE"
)

// Subscription includes the table to subscribe to
type Subscription struct {
	Type    SubscriptionType
	ID      string               // Optional ID of resource to subscribe to
	Actions []SubscriptionAction // Empty Actions list will subscribe to all action types
}

// Subscriber is used to subscribe to database events
type Subscriber struct {
	events        chan db.Event
	done          chan bool
	errors        chan error
	ID            string
	subscriptions []Subscription
}

// GetEvent blocks until an event is available
func (s *Subscriber) GetEvent(ctx context.Context) (*db.Event, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.done:
		return nil, fmt.Errorf("Subscriber has been unsubscribed")
	case event := <-s.events:
		return &event, nil
	case err := <-s.errors:
		return nil, err
	}
}

// EventManager is used to subscribe to database events
type EventManager struct {
	logger      logger.Logger
	dbClient    *db.Client
	subscribers []Subscriber
	lock        sync.RWMutex
}

// NewEventManager creates a new instance of EventManager
func NewEventManager(dbClient *db.Client, logger logger.Logger) *EventManager {
	return &EventManager{
		subscribers: []Subscriber{},
		dbClient:    dbClient,
		logger:      logger,
	}
}

// Start will start the event loop for listing to database events
func (e *EventManager) Start(ctx context.Context) {
	go func() {
		for {
			// Make channel
			ch, errorCh := e.dbClient.Events.Listen(ctx)
			// Wait for events
			for {
				exitLoop := false

				select {
				case <-ctx.Done():
					return
				case event := <-ch:
					e.notifyEvent(ctx, event)
				case err := <-errorCh:
					e.logger.WithContextFields(ctx).Errorf("received error when listening for db event: %v", err)

					// Notify subscribers of error
					e.notifyError(err)

					// Exit this for loop to setup a new DB listener connection
					exitLoop = true
				}

				if exitLoop {
					break
				}
			}

			// Wait before reconnecting
			time.Sleep(10 * time.Second)
		}
	}()
}

// Subscribe creates a Subscriber that will be notified of database
// events based on the specified subscriptions
func (e *EventManager) Subscribe(subscriptions []Subscription) *Subscriber {
	e.lock.Lock()
	defer e.lock.Unlock()

	subscriber := Subscriber{
		ID:            uuid.New().String(),
		subscriptions: subscriptions,
		events:        make(chan db.Event, subscriberBufferSize),
		errors:        make(chan error, 1),
		done:          make(chan bool, 1),
	}

	e.subscribers = append(e.subscribers, subscriber)

	return &subscriber
}

// Unsubscribe removes the subscriber
func (e *EventManager) Unsubscribe(subscriber *Subscriber) {
	e.lock.Lock()
	for i, s := range e.subscribers {
		if s.ID == subscriber.ID {
			e.subscribers = append(e.subscribers[:i], e.subscribers[i+1:]...)
			break
		}
	}
	e.lock.Unlock()

	// Use non-blocking send in case Unsubscribe is called multiple times
	select {
	case subscriber.done <- true:
	default:
	}
}

func (e *EventManager) notifyEvent(ctx context.Context, event db.Event) {
	matchedSubscribers := []*Subscriber{}

	e.lock.RLock()

	for _, subscriber := range e.subscribers {
		sub := subscriber
		if e.match(event, &sub) {
			matchedSubscribers = append(matchedSubscribers, &sub)
		}
	}

	e.lock.RUnlock()

	// Send events after lock has been released
	for _, subscriber := range matchedSubscribers {
		e.sendEventToSubscriber(ctx, subscriber, event)
	}
}

func (e *EventManager) sendEventToSubscriber(ctx context.Context, subscriber *Subscriber, event db.Event) {
	// Add timeout to send event, this should never happen but just in case this will prevent the
	// event manager from locking up
	sendContext, cancel := context.WithTimeout(ctx, sendEventTimeout)
	defer cancel()

	// Send event to subscriber
	select {
	case <-sendContext.Done():
		e.logger.WithContextFields(ctx).Error("event manager failed to send event to subscriber due to timeout")
		// unsubscribe subscriber here to prevent further issues
		e.Unsubscribe(subscriber)
	case subscriber.events <- event:
	case <-subscriber.done:
	}
}

func (e *EventManager) notifyError(err error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	for _, subscriber := range e.subscribers {
		// Send error to subscriber
		go func(ch chan error, done chan bool, e error) {
			select {
			case ch <- e:
			case <-done:
				return
			}
		}(subscriber.errors, subscriber.done, err)
	}
}

func (e *EventManager) match(event db.Event, subscriber *Subscriber) bool {
	for _, subscription := range subscriber.subscriptions {
		if subscription.Type != SubscriptionType(event.Table) {
			continue
		}

		if subscription.ID != "" && subscription.ID != event.ID {
			continue
		}

		if len(subscription.Actions) == 0 {
			return true
		}

		for _, action := range subscription.Actions {
			if action == SubscriptionAction(event.Action) {
				return true
			}
		}
	}

	return false
}
