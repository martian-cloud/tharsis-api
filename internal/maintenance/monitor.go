// Package maintenance provides the maintenance mode monitor
package maintenance

//go:generate go tool mockery --name Monitor --inpackage --case underscore

import (
	"context"
	"sync"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// Monitor is used to monitor the maintenance mode state
type Monitor interface {
	// Start starts the maintenance mode monitor
	Start(ctx context.Context)
	// InMaintenanceMode returns true if the system is in maintenance mode
	InMaintenanceMode(ctx context.Context) (bool, error)
}

type monitor struct {
	logger            logger.Logger
	dbClient          *db.Client
	eventManager      *events.EventManager
	inMaintenanceMode *bool // Cache maintenance mode state.
	lock              sync.RWMutex
}

// NewMonitor returns a new instance of the maintenance mode monitor
func NewMonitor(
	logger logger.Logger,
	dbClient *db.Client,
	eventManager *events.EventManager,
) Monitor {
	return &monitor{
		logger:       logger,
		dbClient:     dbClient,
		eventManager: eventManager,
	}
}

// Start starts the maintenance mode monitor
func (m *monitor) Start(ctx context.Context) {
	go func() {
		subscription := []events.Subscription{
			{
				// We want to listen to all maintenance mode events
				Type: events.MaintenanceModeSubscription,
				Actions: []events.SubscriptionAction{
					events.CreateAction,
					events.DeleteAction,
				},
			},
		}

		subscriber := m.eventManager.Subscribe(subscription)
		defer m.eventManager.Unsubscribe(subscriber)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := m.waitForMaintenanceModeEvent(ctx, subscriber); err != nil && !errors.IsContextCanceledError(err) {
					m.logger.Errorf("Failed to wait for maintenance mode event: %v", err)
				}
			}
		}
	}()
}

// InMaintenanceMode returns true if the system is in maintenance mode
func (m *monitor) InMaintenanceMode(ctx context.Context) (bool, error) {
	m.lock.RLock()
	if m.inMaintenanceMode != nil {
		defer m.lock.RUnlock()
		return *m.inMaintenanceMode, nil
	}
	m.lock.RUnlock()

	return m.updateMaintenanceModeState(ctx)
}

// waitForMaintenanceModeEvent waits for maintenance mode events and updates the cached maintenance mode state
func (m *monitor) waitForMaintenanceModeEvent(ctx context.Context, subscriber *events.Subscriber) error {
	if _, err := subscriber.GetEvent(ctx); err != nil {
		return errors.Wrap(err, "failed to get maintenance mode event")
	}

	_, err := m.updateMaintenanceModeState(ctx)
	return err
}

// updateMaintenanceModeState queries and sets the maintenance
// mode state and returns the resulting value.
func (m *monitor) updateMaintenanceModeState(ctx context.Context) (bool, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	maintenanceMode, err := m.dbClient.MaintenanceModes.GetMaintenanceMode(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to get maintenance mode state")
	}

	// A record in the maintenance mode table means we are in maintenance mode
	m.inMaintenanceMode = ptr.Bool(maintenanceMode != nil)

	return *m.inMaintenanceMode, nil
}
