// Package eventhandlers provides handlers for run state change events.
package eventhandlers

import (
	"context"
	"regexp"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/ansi"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ErroredRunEmailHandler manages errored runs.
type ErroredRunEmailHandler struct {
	logger          logger.Logger
	dbClient        *db.Client
	runStateManager *state.RunStateManager
	emailClient     email.Client
}

var (
	// Remove certain Unicode characters the Terraform CLI adds to the error messages.
	removeUnicodeCharacters = regexp.MustCompile("[╷│╵]")
)

// NewErroredRunEmailHandler returns an instance of ErroredRunEmailHandler.
func NewErroredRunEmailHandler(
	logger logger.Logger,
	dbClient *db.Client,
	runStateManager *state.RunStateManager,
	emailClient email.Client,
) *ErroredRunEmailHandler {
	return &ErroredRunEmailHandler{
		logger:          logger,
		dbClient:        dbClient,
		runStateManager: runStateManager,
		emailClient:     emailClient,
	}
}

// RegisterHandlers registers any handlers with the run state manager used by the ErroredRunEmailManager.
func (t *ErroredRunEmailHandler) RegisterHandlers() {
	t.runStateManager.RegisterHandler(state.RunEventType, t.handleErroredRunEvent)
}

// handleErroredRunEvent handles task status going to and from approval pending.
// It traps and logs the error from the internal function.
// It always returns nil.
func (t *ErroredRunEmailHandler) handleErroredRunEvent(ctx context.Context, _ state.EventType, _ interface{}, new interface{}) error {
	err := t.internalHandleErroredRunEvent(ctx, new)
	if err != nil {
		t.logger.Errorf("Errored run email handler failed to handle event: %v", err)
	}

	return nil
}

// internalHandleErroredRunEvent handles task status going to and from approval pending.
func (t *ErroredRunEmailHandler) internalHandleErroredRunEvent(ctx context.Context, new interface{}) error {
	newRun := new.(*models.Run)

	// If this run did not error out, don't attempt to send email.
	if newRun.Status != models.RunErrored {
		return nil
	}

	// If this run was created by a service account rather than a human user, don't attempt to send email.
	if !strings.Contains(newRun.CreatedBy, "@") {
		return nil
	}

	// Get the user (for the ID) from the created-by email address.
	user, err := t.dbClient.Users.GetUserByEmail(ctx, newRun.CreatedBy)
	if err != nil {
		return errors.Wrap(err, "failed to get user by email address %s", newRun.CreatedBy)
	}
	if user == nil {
		return errors.Wrap(err, "user not found %s: %v", newRun.CreatedBy)
	}

	// Get the workspace to get the full path.
	workspace, err := t.dbClient.Workspaces.GetWorkspaceByID(ctx, newRun.WorkspaceID)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace by ID %s", newRun.WorkspaceID)
	}
	if workspace == nil {
		return errors.Wrap(err, "workspace not found %s: %v", newRun.WorkspaceID)
	}
	workspacePath := workspace.FullPath

	// Get the error message from the plan or apply.
	var errorMessage string
	var runStage builder.RunStage
	if newRun.ApplyID != "" {
		apply, aErr := t.dbClient.Applies.GetApply(ctx, newRun.ApplyID)
		if aErr != nil {
			return errors.Wrap(aErr, "failed to get apply by ID %s", newRun.ApplyID)
		}

		if apply == nil {
			return errors.Wrap(aErr, "apply not found by ID: %s", newRun.ApplyID)
		}

		if apply.ErrorMessage != nil {
			errorMessage = *apply.ErrorMessage
			runStage = builder.ApplyStage
		}
	}
	if errorMessage == "" && newRun.PlanID != "" {
		plan, pErr := t.dbClient.Plans.GetPlan(ctx, newRun.PlanID)
		if pErr != nil {
			return errors.Wrap(pErr, "failed to get plan by ID %s", newRun.PlanID)
		}

		if plan == nil {
			return errors.Wrap(pErr, "plan not found by ID: %s", newRun.PlanID)
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
	if newRun.Speculative() {
		subject = "speculative"
		if newRun.IsDestroy {
			subject += " destroy"
		}
		subject += " plan"
	} else if newRun.IsDestroy {
		subject = "destroy"
		if runStage == builder.PlanStage {
			subject += " " + string(builder.PlanStage)
		}
	} else {
		subject = string(runStage)
	}
	subject += " failed"

	t.emailClient.SendMail(ctx, &email.SendMailInput{
		UsersIDs: []string{user.Metadata.ID},
		Subject:  "Tharsis " + subject,
		Builder: &builder.FailedRunEmail{
			WorkspacePath: workspacePath,
			Title:         cases.Title(language.English, cases.Compact).String(subject),
			ModuleVersion: newRun.ModuleVersion,
			ModuleSource:  newRun.ModuleSource,
			CreatedBy:     newRun.CreatedBy,
			ErrorMessage:  errorMessage,
			RunID:         gid.NewGlobalID(gid.RunType, newRun.Metadata.ID).String(),
			RunStage:      runStage,
		},
	})

	return nil
}
