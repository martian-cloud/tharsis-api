package eventhandlers

import (
	"context"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/ansi"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// removeUnicodeCharacters strips the box-drawing characters the Terraform CLI adds
// to its error messages.
var removeUnicodeCharacters = regexp.MustCompile("[╷│╵]")

// FailedRunEmailHandler handles sending emails when a run fails.
type FailedRunEmailHandler struct {
	dbClient            *db.Client
	logger              logger.Logger
	emailClient         email.Client
	notificationManager namespace.NotificationManager
	taskManager         asynctask.Manager
}

// NewFailedRunEmailHandler creates a new FailedRunEmailHandler.
func NewFailedRunEmailHandler(
	logger logger.Logger,
	dbClient *db.Client,
	taskManager asynctask.Manager,
	emailClient email.Client,
	notificationManager namespace.NotificationManager,
) *FailedRunEmailHandler {
	return &FailedRunEmailHandler{
		dbClient:            dbClient,
		logger:              logger,
		emailClient:         emailClient,
		notificationManager: notificationManager,
		taskManager:         taskManager,
	}
}

// HandleRunChanges handles run events.
func (h *FailedRunEmailHandler) HandleRunChanges(_ context.Context, changes []types.RunChange) error {
	for _, failed := range getFailedRuns(changes) {
		// Skip assessment runs.
		if failed.run.IsAssessmentRun {
			continue
		}

		run, stage := failed.run, failed.stage
		h.taskManager.StartTask(func(ctx context.Context) {
			h.sendFailureEmail(ctx, run, stage)
		})
	}

	return nil
}

func (h *FailedRunEmailHandler) sendFailureEmail(ctx context.Context, run *models.Run, stage builder.RunStage) {
	ws, err := h.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil || ws == nil {
		h.logger.WithContextFields(ctx).Errorf("failed to get workspace for run %s: %v", run.Metadata.ID, err)
		return
	}

	// If the run was created by a user, resolve the created-by email to a user ID so
	// the creator is included as a participant (notification matching is by user ID).
	participantIDs := []string{}
	if strings.Contains(run.CreatedBy, "@") {
		user, uErr := h.dbClient.Users.GetUserByEmail(ctx, run.CreatedBy)
		switch {
		case uErr != nil:
			// A failure to resolve the creator shouldn't suppress notifications to
			// every other subscriber; log it and continue without the creator.
			h.logger.WithContextFields(ctx).Errorf("failed to get user by email %s for run %s; continuing without the creator as a participant: %v", run.CreatedBy, run.Metadata.ID, uErr)
		case user != nil:
			participantIDs = append(participantIDs, user.Metadata.ID)
		}
	}

	userIDs, err := h.notificationManager.GetUsersToNotify(ctx, &namespace.GetUsersToNotifyInput{
		NamespacePath:      ws.FullPath,
		ParticipantUserIDs: participantIDs,
		CustomEventCheck: func(events *models.NotificationPreferenceCustomEvents) bool {
			return events != nil && events.FailedRun
		},
	})
	if err != nil {
		h.logger.WithContextFields(ctx).Errorf("failed to get users to notify for run %s: %v", run.Metadata.ID, err)
		return
	}

	if len(userIDs) == 0 {
		return
	}

	var errorMessage string
	switch stage {
	case builder.PlanStage:
		if run.Plan.ErrorMessage != nil {
			errorMessage = *run.Plan.ErrorMessage
		}
	case builder.ApplyStage:
		if run.Apply != nil && run.Apply.ErrorMessage != nil {
			errorMessage = *run.Apply.ErrorMessage
		}
	}

	// Strip ANSI color codes and the box-drawing characters the Terraform CLI adds.
	errorMessage = ansi.UnColorize(errorMessage)
	errorMessage = removeUnicodeCharacters.ReplaceAllString(errorMessage, "")

	subject := failureSubject(run, stage)

	h.emailClient.SendMail(ctx, &email.SendMailInput{
		UsersIDs: userIDs,
		Subject:  "Tharsis " + subject,
		Builder: &builder.FailedRunEmail{
			WorkspacePath: ws.FullPath,
			Title:         cases.Title(language.English, cases.Compact).String(subject),
			ModuleSource:  run.ModuleSource,
			ModuleVersion: run.ModuleVersion,
			CreatedBy:     run.CreatedBy,
			ErrorMessage:  errorMessage,
			RunID:         run.GetGlobalID(),
			RunStage:      stage,
		},
	})
}

// failureSubject builds the run-failure subject text, distinguishing speculative,
// destroy, and plan/apply runs (e.g. "speculative destroy plan failed",
// "destroy plan failed", "apply failed").
func failureSubject(run *models.Run, stage builder.RunStage) string {
	var subject string
	switch {
	case run.Speculative():
		subject = "speculative"
		if run.IsDestroy {
			subject += " destroy"
		}
		subject += " plan"
	case run.IsDestroy:
		subject = "destroy"
		if stage == builder.PlanStage {
			subject += " " + string(builder.PlanStage)
		}
	default:
		subject = string(stage)
	}
	return subject + " failed"
}

// failedRunStage pairs a run that failed with the stage (plan/apply) it failed in.
type failedRunStage struct {
	run   *models.Run
	stage builder.RunStage
}

// getFailedRuns returns every run in the change batch whose plan or apply node
// transitioned to errored, so each failed run gets its own notification.
func getFailedRuns(changes []types.RunChange) []failedRunStage {
	var failed []failedRunStage
	for _, change := range changes {
		for _, statusChange := range change.NodeStatusChanges {
			switch statusChange.GetNodeType() {
			case statemachine.PlanNodeType:
				if pc, ok := statusChange.(statemachine.PlanStatusChange); ok && pc.NewStatus == models.PlanErrored {
					failed = append(failed, failedRunStage{run: change.Run, stage: builder.PlanStage})
				}
			case statemachine.ApplyNodeType:
				if ac, ok := statusChange.(statemachine.ApplyStatusChange); ok && ac.NewStatus == models.ApplyErrored {
					failed = append(failed, failedRunStage{run: change.Run, stage: builder.ApplyStage})
				}
			}
		}
	}
	return failed
}
