package cleanup

import (
	"context"
	"slices"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// runPruner deletes the runs that a run cleanup policy marks stale.
type runPruner struct {
	dbClient *db.Client
}

func (p *runPruner) prune(ctx context.Context, ns namespace.Namespace, policy *models.CleanupPolicy, onDelete onDeleteFn) error {
	workspace, ok := ns.(*models.Workspace)
	if !ok {
		return errors.New("run cleanup policy pruner was given a %s namespace %q, which is not a workspace", ns.GetModelType(), ns.GetPath())
	}

	matcher := newRunMatcher(policy.RunPolicyData.Rules)

	if matcher.empty() {
		return nil
	}

	var cursor *string

	for {
		result, err := p.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
			Sort:              new(db.RunSortableFieldCreatedAtDesc),
			PaginationOptions: &pagination.Options{First: new(int32(sweepPageSize)), After: cursor},
			Filter: &db.RunFilter{
				WorkspaceID:     &workspace.Metadata.ID,
				Statuses:        matcher.targetStatuses,
				HasStateVersion: new(false),
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to get runs in cleanup policy sweeper")
		}

		// Queried after the runs list, once per page, so the protected-run check stays as fresh as
		// possible right before matching this page — the assessment can change concurrently with the sweep.
		assessment, err := p.dbClient.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, workspace.Metadata.ID)
		if err != nil {
			return errors.Wrap(err, "failed to get workspace assessment in cleanup policy sweeper")
		}

		var protectedRunID *string
		if assessment != nil {
			protectedRunID = assessment.RunID
		}

		var candidates []*models.Run
		for _, run := range result.Runs {
			if matcher.match(run, protectedRunID) {
				candidates = append(candidates, run)
			}
		}

		if err = deleteInChunks(ctx, candidates,
			func(run *models.Run) string { return run.Metadata.ID },
			func(run *models.Run) string { return run.Metadata.TRN },
			func(ctx context.Context, chunk []*models.Run) ([]string, error) {
				return p.dbClient.Runs.DeleteRunBatch(ctx, &db.DeleteRunBatchInput{Runs: chunk})
			},
			onDelete,
		); err != nil {
			return errors.Wrap(err, "failed to delete runs in cleanup policy sweeper")
		}

		if !result.PageInfo.HasNextPage || len(result.Runs) == 0 {
			return nil
		}

		cursor, err = result.PageInfo.Cursor(result.Runs[len(result.Runs)-1])
		if err != nil {
			return errors.Wrap(err, "failed to get the next run cursor in cleanup policy sweeper")
		}

		select {
		case <-time.After(pageSleepInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// runMatcher matches a workspace's runs against a policy's rules, tracking how many runs each rule
// has kept so far.
type runMatcher struct {
	rules          []*models.RunCleanupRule
	kept           []int32
	targetStatuses []models.RunStatus
	now            time.Time
}

// newRunMatcher builds a matcher for the given rules and computes the set of statuses that at least
// one deleting rule targets, so the DB query can be narrowed before any matching begins.
func newRunMatcher(rules []*models.RunCleanupRule) *runMatcher {
	var targetStatuses []models.RunStatus
	for _, status := range models.CleanupCandidateRunStatuses {
		if slices.ContainsFunc(rules, func(rule *models.RunCleanupRule) bool {
			return rule.Strategy != models.StrategyProtect && status.In(rule.Status)
		}) {
			targetStatuses = append(targetStatuses, status)
		}
	}

	return &runMatcher{
		rules:          rules,
		kept:           make([]int32, len(rules)),
		targetStatuses: targetStatuses,
		now:            time.Now().UTC(),
	}
}

// empty reports whether no deleting rules apply, meaning there is nothing to prune.
func (m *runMatcher) empty() bool {
	return len(m.targetStatuses) == 0
}

// match reports whether the first rule matching the run deletes it. protectedRunID, if non-nil, is
// never matched regardless of rule strategy.
func (m *runMatcher) match(run *models.Run, protectedRunID *string) bool {
	if protectedRunID != nil && run.Metadata.ID == *protectedRunID {
		return false
	}

	for i, rule := range m.rules {
		if rule.Speculative != nil && *rule.Speculative != run.Speculative() {
			continue
		}

		if rule.Assessment != nil && *rule.Assessment != run.IsAssessmentRun {
			continue
		}

		if !run.Status.In(rule.Status) {
			continue
		}

		if rule.Strategy == models.StrategyProtect {
			return false
		}

		m.kept[i]++
		if m.kept[i] <= rule.KeepMin {
			return false
		}

		switch rule.Strategy {
		case models.StrategyCount:
			return true
		case models.StrategyAge:
			return run.Metadata.CreationTimestamp.Before(m.now.AddDate(0, 0, -int(rule.DeleteAfterDays)))
		default:
			// An unrecognized strategy never deletes.
			return false
		}
	}

	return false
}
