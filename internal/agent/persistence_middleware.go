package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/gollem"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// messagePersister saves agent messages to the DB in real-time as the agent runs.
type messagePersister struct {
	dbClient      *db.Client
	agentStore    Store
	logger        logger.Logger
	sessionID     string
	runID         string
	lastMessageID *string
	excludeTools  map[string]struct{}
}

func newMessagePersister(dbClient *db.Client, agentStore Store, logger logger.Logger, sessionID string, runID string, lastMessageID *string, excludeTools []string) *messagePersister {
	et := make(map[string]struct{}, len(excludeTools))
	for _, t := range excludeTools {
		et[t] = struct{}{}
	}
	return &messagePersister{
		dbClient:      dbClient,
		agentStore:    agentStore,
		logger:        logger,
		sessionID:     sessionID,
		runID:         runID,
		lastMessageID: lastMessageID,
		excludeTools:  et,
	}
}

// save persists a message and updates lastMessageID to the newly created message's ID.
// For tool role messages, content is stored in object storage instead of the DB.
func (p *messagePersister) save(ctx context.Context, role string, content json.RawMessage) error {
	parentID := p.lastMessageID

	dbContent := content
	isTool := role == string(gollem.RoleTool)
	if isTool {
		dbContent = nil
	}

	var createdID string

	if err := p.dbClient.RetryOnOLE(ctx, func() error {
		txCtx, err := p.dbClient.Transactions.BeginTx(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin tx for save message: %w", err)
		}

		defer func() {
			if txErr := p.dbClient.Transactions.RollbackTx(txCtx); txErr != nil {
				p.logger.Errorf("failed to rollback tx for save message: %v", txErr)
			}
		}()

		created, err := p.dbClient.AgentSessionMessages.CreateAgentSessionMessage(txCtx, &models.AgentSessionMessage{
			SessionID: p.sessionID,
			RunID:     p.runID,
			ParentID:  parentID,
			Role:      role,
			Content:   dbContent,
		})
		if err != nil {
			return fmt.Errorf("failed to save session message: %w", err)
		}

		if isTool {
			if err := p.agentStore.UploadToolContent(txCtx, p.sessionID, created.Metadata.ID, content); err != nil {
				return fmt.Errorf("failed to upload tool content to object store: %w", err)
			}
		}

		run, err := p.dbClient.AgentSessionRuns.GetAgentSessionRunByID(txCtx, p.runID)
		if err != nil {
			return fmt.Errorf("failed to fetch run for last message update: %w", err)
		}

		run.LastMessageID = &created.Metadata.ID
		if _, err := p.dbClient.AgentSessionRuns.UpdateAgentSessionRun(txCtx, run); err != nil {
			return fmt.Errorf("failed to update run last message ID: %w", err)
		}

		if err := p.dbClient.Transactions.CommitTx(txCtx); err != nil {
			return fmt.Errorf("failed to commit tx for save message: %w", err)
		}

		createdID = created.Metadata.ID
		return nil
	}); err != nil {
		return err
	}

	p.lastMessageID = &createdID

	return nil
}

// saveMessage persists a single gollem message to the DB.
func (p *messagePersister) saveMessage(ctx context.Context, role gollem.MessageRole, contents []gollem.MessageContent) error {
	contentJSON, err := json.Marshal(contents)
	if err != nil {
		return fmt.Errorf("failed to marshal message content: %w", err)
	}
	return p.save(ctx, string(role), contentJSON)
}

// saveUserInput saves the user's input message.
func (p *messagePersister) saveUserInput(ctx context.Context, text string) error {
	tc, err := gollem.NewTextContent(text)
	if err != nil {
		return fmt.Errorf("failed to create text content: %w", err)
	}
	return p.saveMessage(ctx, gollem.RoleUser, []gollem.MessageContent{tc})
}

// ContentBlockMiddleware returns a middleware that persists assistant messages in real-time.
func (p *messagePersister) ContentBlockMiddleware() gollem.ContentBlockMiddleware {
	return func(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
		return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
			resp, err := next(ctx, req)
			if err != nil || resp == nil || resp.Error != nil {
				return resp, err
			}

			var contents []gollem.MessageContent

			for _, t := range resp.Texts {
				tc, tcErr := gollem.NewTextContent(t)
				if tcErr != nil {
					return nil, fmt.Errorf("failed to create text content: %w", tcErr)
				}
				contents = append(contents, tc)
			}

			for _, fc := range resp.FunctionCalls {
				if _, excluded := p.excludeTools[fc.Name]; excluded {
					continue
				}
				tc, tcErr := gollem.NewToolCallContent(fc.ID, fc.Name, fc.Arguments)
				if tcErr != nil {
					return nil, fmt.Errorf("failed to create tool call content: %w", tcErr)
				}
				contents = append(contents, tc)
			}

			if len(contents) > 0 {
				if saveErr := p.saveMessage(ctx, gollem.RoleAssistant, contents); saveErr != nil {
					return nil, saveErr
				}
			}

			return resp, nil
		}
	}
}

// ToolMiddleware returns a middleware that persists tool result messages in real-time.
func (p *messagePersister) ToolMiddleware() gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			toolResp, err := next(ctx, req)
			if err != nil {
				return nil, err
			}

			if _, excluded := p.excludeTools[req.Tool.Name]; !excluded {
				result := toolResp.Result
				isError := toolResp.Error != nil
				if isError {
					result = map[string]any{"error": toolResp.Error.Error()}
				}

				tc, tcErr := gollem.NewToolResponseContent(req.Tool.ID, req.Tool.Name, result, isError)
				if tcErr != nil {
					return nil, fmt.Errorf("failed to create tool response content: %w", tcErr)
				}
				if saveErr := p.saveMessage(ctx, gollem.RoleTool, []gollem.MessageContent{tc}); saveErr != nil {
					return nil, saveErr
				}
			}

			return toolResp, err
		}
	}
}
