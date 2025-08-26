// Package email supports sending emails.
package email

//go:generate go tool mockery --name Client --inpackage --case underscore

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// SendMailInput is the input for sending an email
type SendMailInput struct {
	UsersIDs       []string
	TeamsIDs       []string
	ExcludeUserIDs []string
	Subject        string
	Builder        builder.EmailBuilder
}

// Client is used for sending emails
type Client interface {
	SendMail(ctx context.Context, input *SendMailInput)
}

type client struct {
	taskManager   asynctask.Manager
	dbClient      *db.Client
	emailProvider email.Provider
	logger        logger.Logger
	templateCtx   *builder.TemplateContext
}

// NewClient returns a new client
func NewClient(emailProvider email.Provider, taskManager asynctask.Manager, dbClient *db.Client, logger logger.Logger, frontendURL string, emailFooter string) Client {
	return &client{
		emailProvider: emailProvider,
		taskManager:   taskManager,
		dbClient:      dbClient,
		logger:        logger,
		templateCtx:   builder.NewTemplateContext(frontendURL, emailFooter),
	}
}

func (c *client) SendMail(_ context.Context, input *SendMailInput) {
	// Send emails using an async goroutine to avoid blocking the main thread
	c.taskManager.StartTask(func(ctx context.Context) {
		if err := c.sendMail(ctx, input); err != nil {
			// Log an error if the email(s) failed to send
			c.logger.WithContextFields(ctx).Errorf("failed to send email of type %s: %v", string(input.Builder.Type()), err)
		}
	})
}

func (c *client) sendMail(ctx context.Context, input *SendMailInput) error {
	// Build email template
	emailBody, err := input.Builder.Build(c.templateCtx)
	if err != nil {
		return err
	}

	userIDMap := make(map[string]struct{})
	excludeUserIDMap := make(map[string]struct{})

	for _, id := range input.UsersIDs {
		userIDMap[id] = struct{}{}
	}

	for _, id := range input.ExcludeUserIDs {
		excludeUserIDMap[id] = struct{}{}
	}

	if len(input.TeamsIDs) > 0 {
		resp, tErr := c.dbClient.TeamMembers.GetTeamMembers(ctx, &db.GetTeamMembersInput{
			Filter: &db.TeamMemberFilter{
				TeamIDs: input.TeamsIDs,
			},
		})
		if tErr != nil {
			return tErr
		}

		for _, member := range resp.TeamMembers {
			userIDMap[member.UserID] = struct{}{}
		}
	}

	userIDs := make([]string, 0, len(userIDMap))
	for id := range userIDMap {
		if _, ok := excludeUserIDMap[id]; !ok {
			userIDs = append(userIDs, id)
		}
	}

	addresses := []string{}

	if len(userIDs) > 0 {
		resp, uErr := c.dbClient.Users.GetUsers(ctx, &db.GetUsersInput{
			Filter: &db.UserFilter{
				UserIDs: userIDs,
			},
		})
		if uErr != nil {
			return uErr
		}

		for _, user := range resp.Users {
			addresses = append(addresses, user.Email)
		}
	}

	// Send email to each recipient, eventually this can be optimized to send each recipient in a separate goroutine
	for _, recipient := range addresses {
		if err = c.emailProvider.SendMail(ctx, []string{recipient}, input.Subject, emailBody); err != nil {
			c.logger.WithContextFields(ctx).Errorf("failed to send email to %s: %v", recipient, err)
		}
	}

	return nil
}
