// Package eventhandlers provides handlers for run state change events.
package eventhandlers

import (
	"context"
	"regexp"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/ansi"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ErroredRunEmailHandler manages errored runs.
type ErroredRunEmailHandler struct {
	logger              logger.Logger
	dbClient            *db.Client
	runStateManager     state.RunStateManager
	emailClient         email.Client
	notificationManager namespace.NotificationManager
	asyncTaskManager    asynctask.Manager
}

var (
	// Remove certain Unicode characters the Terraform CLI adds to the error messages.
	removeUnicodeCharacters = regexp.MustCompile("[╷│╵]")
)

// NewErroredRunEmailHandler returns an instance of ErroredRunEmailHandler.
func NewErroredRunEmailHandler(
	logger logger.Logger,
	dbClient *db.Client,
	runStateManager state.RunStateManager,
	emailClient email.Client,
	notificationManager namespace.NotificationManager,
	asyncTaskManager asynctask.Manager,
) *ErroredRunEmailHandler {
	return &ErroredRunEmailHandler{
		logger:              logger,
		dbClient:            dbClient,
		runStateManager:     runStateManager,
		emailClient:         emailClient,
		notificationManager: notificationManager,
		asyncTaskManager:    asyncTaskManager,
	}
}

// RegisterHandlers registers any handlers with the run state manager used by the ErroredRunEmailManager.
func (t *ErroredRunEmailHandler) RegisterHandlers() {
	t.runStateManager.RegisterHandler(state.RunEventType, t.handleErroredRunEvent)
}

// handleErroredRunEvent handles task status going to and from approval pending.
// It traps and logs the error from the internal function.
// It always returns nil.
func (t *ErroredRunEmailHandler) handleErroredRunEvent(_ context.Context, eventType state.EventType, _ interface{}, newModel interface{}) error {
	if eventType == state.RunEventType {
		run, ok := newModel.(*models.Run)
		if !ok {
			t.logger.Errorf("Errored run email handler received unexpected type for new object: %T", newModel)
			return nil
		}

		// If this run did not error out, don't attempt to send email.
		if run.Status != models.RunErrored {
			return nil
		}

		t.asyncTaskManager.StartTask(func(ctx context.Context) {
			err := t.sendFailedRunEmail(ctx, run)
			if err != nil {
				t.logger.Errorf("Errored run email handler failed to handle event: %v", err)
			}
		})
	}

	return nil
}

func (t *ErroredRunEmailHandler) sendFailedRunEmail(ctx context.Context, run *models.Run) error {
	// Check if this run was created by a user
	participantIDs := []string{}
	if strings.Contains(run.CreatedBy, "@") {
		// Get the user (for the ID) from the created-by email address.
		user, err := t.dbClient.Users.GetUserByEmail(ctx, run.CreatedBy)
		if err != nil {
			return errors.Wrap(err, "failed to get user by email address %s", run.CreatedBy)
		}
		if user == nil {
			return errors.Wrap(err, "user not found %s: %v", run.CreatedBy)
		}
		participantIDs = append(participantIDs, user.Metadata.ID)
	}

	// Get the workspace to get the full path.
	workspace, err := t.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace by ID %s", run.WorkspaceID)
	}
	if workspace == nil {
		return errors.Wrap(err, "workspace not found %s: %v", run.WorkspaceID)
	}

	// Get users to be notified.
	usersToNotify, err := t.notificationManager.GetUsersToNotify(ctx, &namespace.GetUsersToNotifyInput{
		NamespacePath:      workspace.FullPath,
		ParticipantUserIDs: participantIDs,
		CustomEventCheck: func(events *models.NotificationPreferenceCustomEvents) bool {
			return events.FailedRun
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get users to notify")
	}

	if len(usersToNotify) == 0 {
		return nil
	}

	workspacePath := workspace.FullPath

	// Get the error message from the plan or apply.
	var errorMessage string
	var runStage builder.RunStage
	if run.ApplyID != "" {
		apply, aErr := t.dbClient.Applies.GetApplyByID(ctx, run.ApplyID)
		if aErr != nil {
			return errors.Wrap(aErr, "failed to get apply by ID %s", run.ApplyID)
		}

		if apply == nil {
			return errors.Wrap(aErr, "apply not found by ID: %s", run.ApplyID)
		}

		if apply.ErrorMessage != nil {
			errorMessage = *apply.ErrorMessage
			runStage = builder.ApplyStage
		}
	}
	if errorMessage == "" && run.PlanID != "" {
		plan, pErr := t.dbClient.Plans.GetPlanByID(ctx, run.PlanID)
		if pErr != nil {
			return errors.Wrap(pErr, "failed to get plan by ID %s", run.PlanID)
		}

		if plan == nil {
			return errors.Wrap(pErr, "plan not found by ID: %s", run.PlanID)
		}

		if plan.ErrorMessage != nil {
			errorMessage = *plan.ErrorMessage
		}
		runStage = builder.PlanStage
	}
	errorMessage = ansi.UnColorize(errorMessage)

	// Remove certain Unicode characters the Terraform CLI adds to the error messages.
	errorMessage = removeUnicodeCharacters.ReplaceAllString(errorMessage, "")

	var subject string
	if run.Speculative() {
		subject = "speculative"
		if run.IsDestroy {
			subject += " destroy"
		}
		subject += " plan"
	} else if run.IsDestroy {
		subject = "destroy"
		if runStage == builder.PlanStage {
			subject += " " + string(builder.PlanStage)
		}
	} else {
		subject = string(runStage)
	}
	subject += " failed"

	t.emailClient.SendMail(ctx, &email.SendMailInput{
		UsersIDs: usersToNotify,
		Subject:  "Tharsis " + subject,
		Builder: &builder.FailedRunEmail{
			WorkspacePath: workspacePath,
			Title:         cases.Title(language.English, cases.Compact).String(subject),
			ModuleVersion: run.ModuleVersion,
			ModuleSource:  run.ModuleSource,
			CreatedBy:     run.CreatedBy,
			ErrorMessage:  errorMessage,
			RunID:         run.GetGlobalID(),
			RunStage:      runStage,
		},
	})

	return nil
}
