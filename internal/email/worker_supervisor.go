package email

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	emailplugin "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// eventWaitErrorBackoff is how long a worker's event-wait loop pauses after a non-context error so a persistent subscription failure doesn't spin.
const eventWaitErrorBackoff = 5 * time.Second

// asyncWorkerConfig configures a background worker that runs on a jittered fallback interval and also wakes immediately when a matching email outbox item event arrives.
type asyncWorkerConfig struct {
	workerName          string
	wakeOnAction        events.SubscriptionAction
	wakeOnStatus        models.EmailOutboxItemStatus
	minFallbackInterval time.Duration
	maxFallbackInterval time.Duration
	// run does one pass and reports whether more work remains, in which case the worker re-runs immediately instead of sleeping.
	run func(context.Context) (bool, error)
}

// fallbackInterval returns the next jittered sleep, guarding against a non-positive span so equal or inverted interval constants don't panic rand.Int64N.
func (c asyncWorkerConfig) fallbackInterval() time.Duration {
	span := c.maxFallbackInterval - c.minFallbackInterval
	if span <= 0 {
		return c.minFallbackInterval
	}

	return c.minFallbackInterval + time.Duration(rand.Int64N(int64(span)))
}

// WorkerSupervisorInput is the input for building the email delivery background workers.
type WorkerSupervisorInput struct {
	DBClient           *db.Client
	Store              Store
	Provider           emailplugin.Provider
	Logger             logger.Logger
	MaintenanceMonitor maintenance.Monitor
	EventManager       *events.EventManager
	FrontendURL        string
	EmailFooter        string
	// EphemeralRetentionDays is how many days a completed ephemeral email is kept before the cleaner reclaims it.
	EphemeralRetentionDays int
}

// WorkerSupervisor builds and starts the email delivery background workers: the recipient materializer, the sender, the delivery-feedback consumer (when the provider supports it), and the ephemeral-outbox cleaner.
type WorkerSupervisor struct {
	logger             logger.Logger
	eventManager       *events.EventManager
	maintenanceMonitor maintenance.Monitor
	materializer       *Materializer
	sender             *Sender
	cleaner            *Cleaner
	// feedback is nil when the provider does not support delivery feedback.
	feedback *FeedbackManager
}

// NewWorkerSupervisor builds the email delivery workers from the given input.
func NewWorkerSupervisor(input *WorkerSupervisorInput) (*WorkerSupervisor, error) {
	cleaner, err := NewCleaner(
		input.DBClient,
		input.EphemeralRetentionDays,
	)
	if err != nil {
		return nil, err
	}

	s := &WorkerSupervisor{
		logger:             input.Logger,
		eventManager:       input.EventManager,
		maintenanceMonitor: input.MaintenanceMonitor,
		materializer: NewMaterializer(
			input.DBClient,
			input.Logger,
		),
		sender: NewSender(
			input.DBClient,
			input.Logger,
			input.Store,
			input.Provider,
			input.FrontendURL,
			input.EmailFooter,
		),
		cleaner: cleaner,
	}

	// A provider that delivers asynchronous feedback (SES) gets a consumer that applies delivery/bounce/complaint events.
	if feedbackProvider, ok := input.Provider.(emailplugin.FeedbackProvider); ok {
		s.feedback = NewFeedbackManager(input.DBClient, input.Logger, input.MaintenanceMonitor, feedbackProvider)
	}

	return s, nil
}

// Start starts all email delivery background workers.
func (s *WorkerSupervisor) Start(ctx context.Context) {
	s.runAsyncWorker(ctx, asyncWorkerConfig{
		workerName:          "email materializer",
		wakeOnAction:        events.CreateAction,
		wakeOnStatus:        models.EmailOutboxItemStatusPreparing,
		minFallbackInterval: minMaterializeInterval,
		maxFallbackInterval: maxMaterializeInterval,
		run:                 s.materializer.materializeBatch,
	})

	s.runAsyncWorker(ctx, asyncWorkerConfig{
		workerName:          "email sender",
		wakeOnAction:        events.UpdateAction,
		wakeOnStatus:        models.EmailOutboxItemStatusReady,
		minFallbackInterval: minSendInterval,
		maxFallbackInterval: maxSendInterval,
		run: func(ctx context.Context) (bool, error) {
			senderAttempts.Inc()
			return s.sender.send(ctx)
		},
	})

	// The cleaner has no wake trigger since reclaiming finished ephemeral rows isn't event-driven.
	s.runAsyncWorker(ctx, asyncWorkerConfig{
		workerName:          "email cleanup worker",
		minFallbackInterval: minCleanupInterval,
		maxFallbackInterval: maxCleanupInterval,
		run:                 s.cleaner.cleanBatch,
	})

	if s.feedback != nil {
		s.feedback.Start(ctx)
	}
}

// runAsyncWorker launches a worker that runs on a jittered fallback interval and wakes early on a matching outbox item event; a zero wakeOnStatus disables the event wake.
func (s *WorkerSupervisor) runAsyncWorker(ctx context.Context, cfg asyncWorkerConfig) {
	// A size-1 buffer coalesces a burst of events (or one arriving mid-run) into a single follow-up run.
	wake := make(chan struct{}, 1)

	if cfg.wakeOnStatus != "" {
		subscription := events.Subscription{
			Type:    events.EmailOutboxItemSubscription,
			Actions: []events.SubscriptionAction{cfg.wakeOnAction},
			Filter: func(data json.RawMessage) bool {
				var d struct {
					Status string `json:"status"`
				}
				return json.Unmarshal(data, &d) == nil && d.Status == string(cfg.wakeOnStatus)
			},
		}

		go func() {
			subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})
			defer s.eventManager.Unsubscribe(subscriber)

			for {
				if _, err := subscriber.GetEvent(ctx); err != nil {
					if te.IsContextCanceledError(err) {
						return
					}

					s.logger.Errorf("%s failed waiting for outbox item event: %v", cfg.workerName, err)

					// Back off so a persistent subscription error doesn't spin a hot loop; the fallback poll still runs the worker.
					select {
					case <-time.After(eventWaitErrorBackoff):
					case <-ctx.Done():
						return
					}

					continue
				}

				select {
				case wake <- struct{}{}:
				default:
				}
			}
		}()
	}

	go func() {
		s.logger.Infof("%s started", cfg.workerName)

		for {
			timer := time.NewTimer(cfg.fallbackInterval())

			select {
			case <-timer.C:
			case <-wake:
				timer.Stop()
			case <-ctx.Done():
				timer.Stop()
				s.logger.Infof("%s stopped", cfg.workerName)
				return
			}

			// Skip the pass while the system is in maintenance mode.
			inMaintenance, err := s.maintenanceMonitor.InMaintenanceMode(ctx)
			if err != nil {
				if err = te.FilterContextError(err); err != nil {
					s.logger.Errorf("%s failed to check maintenance mode: %v", cfg.workerName, err)
				}

				continue
			}

			if inMaintenance {
				continue
			}

			moreWork, err := cfg.run(ctx)
			if err = te.FilterContextError(err); err != nil {
				s.logger.Error(err)
			}

			// The pass left work behind (time cap or full batch); re-run immediately instead of sleeping.
			// The size-1 buffer coalesces this with any event wake, so it can't pile up.
			if moreWork {
				select {
				case wake <- struct{}{}:
				default:
				}
			}
		}
	}()
}
