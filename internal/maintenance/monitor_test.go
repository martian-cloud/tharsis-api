package maintenance

import (
	context "context"
	"testing"
	"time"

	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestWaitForMaintenanceModeEvent(t *testing.T) {
	sampleMaintenanceMode := &models.MaintenanceMode{
		Metadata: models.ResourceMetadata{
			ID: "1",
		},
		CreatedBy: "test-user",
	}

	type testCase struct {
		maintenanceMode            *models.MaintenanceMode
		name                       string
		event                      db.Event
		expectMaintenanceModeState bool
	}

	testCases := []testCase{
		{
			name: "received maintenance mode create event",
			event: db.Event{
				Table:  "maintenance_mode",
				Action: string(events.CreateAction),
				ID:     "1",
			},
			maintenanceMode:            sampleMaintenanceMode,
			expectMaintenanceModeState: true,
		},
		{
			name: "received maintenance mode delete event",
			event: db.Event{
				Table:  "maintenance_mode",
				Action: string(events.DeleteAction),
				ID:     "1",
			},
			expectMaintenanceModeState: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()

			mockEvents := db.NewMockEvents(t)
			mockMaintenanceModes := db.NewMockMaintenanceModes(t)

			mockEventChannel := make(chan db.Event, 1)
			var roEventChan <-chan db.Event = mockEventChannel
			mockEvents.On("Listen", mock.Anything).Return(roEventChan, make(<-chan error))

			mockMaintenanceModes.On("GetMaintenanceMode", mock.Anything).Return(test.maintenanceMode, nil)

			dbClient := &db.Client{
				Events:           mockEvents,
				MaintenanceModes: mockMaintenanceModes,
			}

			logger, _ := logger.NewForTest()

			eventManager := events.NewEventManager(dbClient, logger)
			eventManager.Start(ctx)

			subscription := []events.Subscription{
				{
					Type: events.MaintenanceModeSubscription,
					Actions: []events.SubscriptionAction{
						events.CreateAction,
						events.DeleteAction,
					},
				},
			}

			subscriber := eventManager.Subscribe(subscription)
			defer eventManager.Unsubscribe(subscriber)

			monitor := &monitor{
				logger:       logger,
				dbClient:     dbClient,
				eventManager: eventManager,
			}

			go func() {
				mockEventChannel <- test.event
			}()

			require.NoError(t, monitor.waitForMaintenanceModeEvent(ctx, subscriber))

			require.NotNil(t, monitor.inMaintenanceMode)
			require.Equal(t, test.expectMaintenanceModeState, *monitor.inMaintenanceMode)
		})
	}
}

func TestInMaintenanceMode(t *testing.T) {
	sampleMaintenanceMode := &models.MaintenanceMode{
		Metadata: models.ResourceMetadata{
			ID: "1",
		},
		CreatedBy: "test-user",
	}

	type testCase struct {
		maintenanceMode            *models.MaintenanceMode
		name                       string
		expectMaintenanceModeState bool
		cachedState                bool
	}

	testCases := []testCase{
		{
			name:                       "maintenance mode is enabled",
			maintenanceMode:            sampleMaintenanceMode,
			expectMaintenanceModeState: true,
		},
		{
			name: "maintenance mode is disabled",
		},
		{
			name:                       "maintenance mode state is cached",
			cachedState:                true,
			expectMaintenanceModeState: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()

			mockMaintenanceModes := &db.MockMaintenanceModes{}
			mockMaintenanceModes.Test(t)

			mockMaintenanceModes.On("GetMaintenanceMode", mock.Anything).Return(test.maintenanceMode, nil)

			dbClient := &db.Client{
				MaintenanceModes: mockMaintenanceModes,
			}

			monitor := &monitor{
				dbClient: dbClient,
			}

			if test.cachedState {
				monitor.inMaintenanceMode = &test.cachedState
				mockMaintenanceModes.AssertNotCalled(t, "GetMaintenanceMode", mock.Anything)
			}

			inMaintenanceMode, err := monitor.InMaintenanceMode(ctx)
			require.NoError(t, err)

			require.Equal(t, test.expectMaintenanceModeState, inMaintenanceMode)
		})
	}
}
