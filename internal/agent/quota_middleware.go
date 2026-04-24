package agent

import (
	"context"
	"time"

	"github.com/m-mizutani/gollem"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/llm"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// quotaMiddleware checks and updates monthly LLM credit usage per user.
type quotaMiddleware struct {
	dbClient  *db.Client
	llmClient llm.Client
	session   *models.AgentSession
}

// Middleware returns a ContentBlockMiddleware that enforces credit limits.
func (q *quotaMiddleware) Middleware() gollem.ContentBlockMiddleware {
	return func(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
		return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
			quota, err := q.getOrCreateQuota(ctx)
			if err != nil {
				return nil, err
			}

			// Check limit — fetch the resource limit value directly and compare as float64
			limit, err := q.dbClient.ResourceLimits.GetResourceLimit(ctx, string(limits.ResourceLimitAgentCreditsPerUserPerMonth))
			if err != nil {
				return nil, errors.Wrap(err, "failed to get credit limit")
			}
			if limit != nil && quota.TotalCredits >= float64(limit.Value) {
				return nil, errors.New(
					"monthly AI credit limit reached, please try again next month",
					errors.WithErrorCode(errors.EForbidden),
				)
			}

			resp, err := next(ctx, req)
			if err != nil {
				return nil, err
			}

			// Calculate and persist credits used
			credits := q.llmClient.GetCreditCount(llm.CreditInput{
				InputTokens:  resp.InputToken,
				OutputTokens: resp.OutputToken,
			})
			if credits > 0 {
				if addErr := q.dbClient.AgentCreditQuotas.AddCredits(ctx, quota.Metadata.ID, credits); addErr != nil {
					return nil, errors.Wrap(addErr, "failed to add credits to quota")
				}

				q.session.TotalCredits += credits
				updated, updateErr := q.dbClient.AgentSessions.UpdateAgentSession(ctx, q.session)
				if updateErr != nil {
					return nil, errors.Wrap(updateErr, "failed to update session credits")
				}
				q.session = updated
			}

			return resp, nil
		}
	}
}

func (q *quotaMiddleware) getOrCreateQuota(ctx context.Context) (*models.AgentCreditQuota, error) {
	monthDate := beginningOfMonth(time.Now().UTC())

	quota, err := q.dbClient.AgentCreditQuotas.GetAgentCreditQuota(ctx, q.session.UserID, monthDate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get credit quota")
	}
	if quota != nil {
		return quota, nil
	}

	// Quota doesn't exist — create it
	quota, err = q.dbClient.AgentCreditQuotas.CreateAgentCreditQuota(ctx, &models.AgentCreditQuota{
		UserID:    q.session.UserID,
		MonthDate: monthDate,
	})
	if err != nil {
		// Another thread/process created it — re-query
		if errors.ErrorCode(err) == errors.EConflict {
			return q.dbClient.AgentCreditQuotas.GetAgentCreditQuota(ctx, q.session.UserID, monthDate)
		}
		return nil, errors.Wrap(err, "failed to create credit quota")
	}

	return quota, nil
}

func beginningOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
