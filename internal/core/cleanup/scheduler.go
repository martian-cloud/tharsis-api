// Package cleanup deletes the resources that cleanup policies mark stale.
package cleanup

import (
	"context"
	"math/rand/v2"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	nsutils "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

var (
	sweeperAttempts        = metric.NewCounter("cleanup_sweeper_attempts", "Amount of cleanup sweeper attempts.")
	sweeperAttemptDuration = metric.NewHistogram("cleanup_sweeper_attempt_duration", "Amount of time a single cleanup sweeper pass took.", 1, 2, 12)
	sweeperDeletedCount    = metric.NewCounterVec("cleanup_sweeper_deleted_count", "Number of resources the cleanup sweeper deleted per kind.", []string{"kind"})
)

// How often a sweep runs and how long one is allowed to take.
const (
	// A sweep only deletes what a policy already allows deleting -- an AGE rule keeps a resource for at
	// least seven days -- so cleanup is not time sensitive and passes are spread out to keep the fixed
	// cost of a pass rare. Each pass sleeps a jittered amount between these two so that the instances do
	// not all wake at once.
	minSweepInterval = 30 * time.Minute
	maxSweepInterval = 60 * time.Minute
	// maxPassDuration bounds one pass, so a policy over a large namespace drains over several passes
	// instead of hammering the DB in one. A third of the shortest interval leaves a pass room to overrun
	// without ever running into the next one, and keeps a sweeper that dies mid-pass from holding its
	// policies for long relative to the interval.
	maxPassDuration = minSweepInterval / 3
)

// How much a sweep takes on at a time.
const (
	// policyClaimBatchSize is how many policies one claim takes. Claiming in batches and looping, rather
	// than taking everything due at once, keeps one instance from holding the whole table while the
	// others idle.
	policyClaimBatchSize = 20
)

// The write that releases a policy's claim once its sweep stops.
const (
	// finishSweepTimeout bounds the write, which has to happen after the pass has already run out of time
	finishSweepTimeout = 30 * time.Second
	// finishSweepOLEAttempts caps the retries when a policy is edited while it is being swept. The
	// default of a thousand suits a contended session row, whereas one policy sees at most a handful of
	// concurrent edits.
	finishSweepOLEAttempts = 5
)

// namespaceFetchResult holds a page of namespaces and the pagination info needed to advance the
// sweep cursor after each processed namespace.
type namespaceFetchResult struct {
	namespaces []namespace.Namespace
	pageInfo   *pagination.PageInfo
}

// namespaceFetcher returns the page of namespaces at or under namespacePath that follows cursor.
type namespaceFetcher func(ctx context.Context, namespacePath string, cursor *string) (*namespaceFetchResult, error)

// Scheduler periodically deletes the resources that each namespace's cleanup policies mark stale.
type Scheduler struct {
	dbClient           *db.Client
	logger             logger.Logger
	maintenanceMonitor maintenance.Monitor
	pruners            map[models.CleanupRuleKind]pruner
}

// NewScheduler creates a new cleanup policy sweeper.
func NewScheduler(
	dbClient *db.Client,
	logger logger.Logger,
	maintenanceMonitor maintenance.Monitor,
) *Scheduler {
	return &Scheduler{
		dbClient:           dbClient,
		logger:             logger,
		maintenanceMonitor: maintenanceMonitor,
		pruners: map[models.CleanupRuleKind]pruner{
			models.CleanupRuleKindRuns:               &runPruner{dbClient: dbClient},
			models.CleanupRuleKindTerraformModules:   &moduleVersionPruner{dbClient: dbClient},
			models.CleanupRuleKindTerraformProviders: &providerVersionPruner{dbClient: dbClient},
		},
	}
}

// Start starts the cleanup policy sweeper.
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("cleanup policy sweeper started")

	go func() {
		for {
			sweeperAttempts.Inc()

			// Use randomization for the sleep duration to prevent all nodes from checking the DB at the same time
			sleep := minSweepInterval + time.Duration(rand.Int64N(int64(maxSweepInterval-minSweepInterval)))

			select {
			case <-time.After(sleep):
				if err := s.sweep(ctx); errors.FilterContextError(err) != nil {
					s.logger.Error(err)
				}
			case <-ctx.Done():
				s.logger.Info("cleanup policy sweeper stopped")
				return
			}
		}
	}()
}

func (s *Scheduler) sweep(ctx context.Context) error {
	inMaintenance, err := s.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check for maintenance mode in cleanup policy sweeper")
	}

	if inMaintenance {
		return nil
	}

	start := time.Now()
	defer func() {
		sweeperAttemptDuration.Observe(time.Since(start).Seconds())
	}()

	ctx, cancel := context.WithTimeout(ctx, maxPassDuration)
	defer cancel()

	now := time.Now()

	for {
		policies, err := s.dbClient.CleanupPolicies.ClaimCleanupPoliciesForSweep(ctx, &db.ClaimCleanupPoliciesForSweepInput{
			ClaimedBefore: now.Add(-minSweepInterval),
			Limit:         policyClaimBatchSize,
		})
		if err != nil {
			return errors.Wrap(err, "failed to claim cleanup policies in cleanup policy sweeper")
		}

		if len(policies) > 0 {
			s.logger.WithContextFields(ctx).Debugw("cleanup sweeper: claimed policies for this batch", "count", len(policies))
		}

		for i := range policies {
			if ctx.Err() != nil {
				// The pass is out of time, so the policies it did not reach stay due for the next one.
				return nil
			}

			if err := s.sweepNamespaces(ctx, &policies[i]); errors.FilterContextError(err) != nil {
				s.logger.WithContextFields(ctx).Errorw("cleanup policy sweeper failed to sweep the namespaces of a policy",
					"policyTRN", policies[i].Metadata.TRN,
					"error", err,
				)
			}
		}

		// A short batch means there is nothing left that is due.
		if len(policies) < policyClaimBatchSize {
			return nil
		}
	}
}

// sweepNamespaces sweeps every namespace the policy governs: the namespace it is set on and, for a
// group policy, the namespaces under it that have not overridden it with a policy of their own kind.
func (s *Scheduler) sweepNamespaces(ctx context.Context, policy *models.CleanupPolicy) error {
	kindPruner, ok := s.pruners[policy.Kind]
	if !ok {
		return errors.New("no pruner is registered for cleanup policy kind %s", policy.Kind)
	}

	// Only workspaces have runs, and only groups have terraform modules and providers.
	var fetch namespaceFetcher = s.fetchWorkspaces
	if policy.Kind.RequiresGroup() {
		fetch = s.fetchGroups
	}

	// Ensure we continue from where we left off.
	cursor := policy.SweepCursor

	defer func() {
		if err := s.finishSweep(ctx, policy, cursor); err != nil {
			s.logger.WithContextFields(ctx).Errorw("cleanup policy sweeper failed to finish sweep",
				"policyTRN", policy.Metadata.TRN,
				"error", err,
			)
		}
	}()

	cache := newOverrideCache(s.dbClient, policy)
	policyNamespacePath := policy.NamespacePath()

	for {
		// Re-read the policy before each page so a disable or deletion takes effect within one page
		// interval rather than running to the end of the walk.
		latest, err := s.dbClient.CleanupPolicies.GetCleanupPolicyByID(ctx, policy.Metadata.ID)
		if err != nil {
			return errors.Wrap(err, "failed to re-read cleanup policy")
		}

		if latest == nil || latest.Disabled {
			return nil
		}

		result, err := fetch(ctx, policyNamespacePath, cursor)
		if err != nil {
			// The recorded page is no longer one this sweep can resume from, so the policy starts over.
			if cursor != nil && errors.ErrorCode(err) == errors.EInvalid {
				cursor = nil
				continue
			}

			return err
		}

		if len(result.namespaces) > 0 {
			s.logger.WithContextFields(ctx).Debugw("cleanup sweeper: fetched a page of namespaces the policy governs",
				"policyTRN", policy.Metadata.TRN,
				"count", len(result.namespaces),
				"hasNextPage", result.pageInfo.HasNextPage,
			)
		}

		if err = cache.updateFromPage(ctx, result.namespaces); err != nil {
			return errors.Wrap(err, "failed to check child policies in cleanup policy sweeper")
		}

		for _, ns := range result.namespaces {
			if ctx.Err() != nil {
				return nil
			}

			// Advance unconditionally so a timed-out sweep always resumes past the last seen
			// namespace, even if it was skipped rather than pruned.
			cursor, err = result.pageInfo.Cursor(ns)
			if err != nil {
				return errors.Wrap(err, "failed to advance sweep cursor")
			}

			nsPath := ns.GetPath()

			// Skip namespaces that have their own policy of this kind or are under one:
			// that policy governs them, not this one.
			if cache.isOverridden(nsPath) {
				s.logger.WithContextFields(ctx).Debugw("cleanup sweeper: skipping namespace governed by a closer policy",
					"policyTRN", policy.Metadata.TRN,
					"namespacePath", nsPath,
				)

				continue
			}

			if err = kindPruner.prune(ctx, ns, policy, func(deletedTRNs ...string) {
				for _, t := range deletedTRNs {
					s.logger.WithContextFields(ctx).Infow("cleanup sweeper deleted resource",
						"policyTRN", policy.Metadata.TRN,
						"namespacePath", nsPath,
						"kind", string(policy.Kind),
						"resourceTRN", t,
					)
				}

				sweeperDeletedCount.WithLabelValues(string(policy.Kind)).Add(float64(len(deletedTRNs)))
			}); errors.FilterContextError(err) != nil {
				s.logger.WithContextFields(ctx).Errorw("cleanup policy sweeper failed to prune a namespace",
					"namespacePath", nsPath,
					"policyTRN", policy.Metadata.TRN,
					"error", err,
				)
			}
		}

		if !result.pageInfo.HasNextPage {
			// Every namespace the policy governs has been swept, so the policy starts over next time.
			cursor = nil
			return nil
		}

		select {
		case <-time.After(pageSleepInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Scheduler) fetchGroups(ctx context.Context, namespacePath string, cursor *string) (*namespaceFetchResult, error) {
	groupSort := db.GroupSortableFieldFullPathAsc
	pageSize := int32(sweepPageSize)

	result, err := s.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{
		Sort:              &groupSort,
		PaginationOptions: &pagination.Options{First: &pageSize, After: cursor},
		Filter: &db.GroupFilter{
			NamespacePathPrefix: &namespacePath,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get groups in cleanup policy sweeper")
	}

	namespaces := make([]namespace.Namespace, len(result.Groups))
	for i := range result.Groups {
		namespaces[i] = &result.Groups[i]
	}

	return &namespaceFetchResult{namespaces: namespaces, pageInfo: result.PageInfo}, nil
}

func (s *Scheduler) fetchWorkspaces(ctx context.Context, namespacePath string, cursor *string) (*namespaceFetchResult, error) {
	workspaceSort := db.WorkspaceSortableFieldFullPathAsc
	pageSize := int32(sweepPageSize)

	result, err := s.dbClient.Workspaces.GetWorkspaces(ctx, &db.GetWorkspacesInput{
		Sort:              &workspaceSort,
		PaginationOptions: &pagination.Options{First: &pageSize, After: cursor},
		Filter: &db.WorkspaceFilter{
			NamespacePathPrefix: &namespacePath,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspaces in cleanup policy sweeper")
	}

	namespaces := make([]namespace.Namespace, len(result.Workspaces))
	for i := range result.Workspaces {
		namespaces[i] = &result.Workspaces[i]
	}

	return &namespaceFetchResult{namespaces: namespaces, pageInfo: result.PageInfo}, nil
}

// finishSweep releases the policy's claim, recording either that the sweep reached the end of the
// policy's namespaces or the page it stopped at.
func (s *Scheduler) finishSweep(ctx context.Context, policy *models.CleanupPolicy, cursor *string) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), finishSweepTimeout)
	defer cancel()

	// Taken once, before any retry, so a retried write still records when the sweep itself finished
	// rather than when the last retry happened to run.
	completedAt := time.Now().UTC()

	// The policy is read again inside the retry so that an edit made while it was being swept keeps the
	// caller's rules and only the sweeper's own columns are rewritten.
	return s.dbClient.RetryOnOLE(ctx, func() error {
		latest, err := s.dbClient.CleanupPolicies.GetCleanupPolicyByID(ctx, policy.Metadata.ID)
		if err != nil {
			return err
		}

		// The policy was deleted while it was being swept, so there is nothing left to record.
		if latest == nil {
			return nil
		}

		latest.SweepCursor = cursor

		// A sweep that stopped partway hands the rest of its work to the next pass by making the policy
		// due again, whereas one that reached the end leaves the claim to lapse on its own an interval
		// after it was taken.
		if cursor != nil {
			latest.SweepClaimedAt = nil
		} else {
			// The walk reached the end of every namespace this policy governs, so this pass -- not
			// necessarily this policy's earlier passes -- is what completed the sweep.
			latest.LastSweepCompletedAt = &completedAt
		}

		s.logger.WithContextFields(ctx).Debugw("cleanup sweeper: finishing sweep of a policy",
			"policyTRN", policy.Metadata.TRN,
			"resuming", cursor != nil,
		)

		_, err = s.dbClient.CleanupPolicies.UpdateCleanupPolicy(ctx, latest)

		return err
	}, db.WithRetryOnOLEAttempts(finishSweepOLEAttempts))
}

// overrideCache tracks the namespace paths that have their own policy of the kind being swept.
type overrideCache struct {
	dbClient           *db.Client
	policy             *models.CleanupPolicy
	overriddenPrefixes map[string]struct{}
}

func newOverrideCache(dbClient *db.Client, policy *models.CleanupPolicy) *overrideCache {
	return &overrideCache{
		dbClient:           dbClient,
		policy:             policy,
		overriddenPrefixes: map[string]struct{}{},
	}
}

// addOverride records nsPath as having its own policy.
func (c *overrideCache) addOverride(nsPath string) {
	c.overriddenPrefixes[nsPath] = struct{}{}
}

// updateFromPage queries for child policies among the namespaces in the page and records any
// that have their own policy so their subtrees can be skipped.
func (c *overrideCache) updateFromPage(ctx context.Context, namespaces []namespace.Namespace) error {
	policyNamespacePath := c.policy.NamespacePath()

	var pathsToCheck []string
	for _, ns := range namespaces {
		p := ns.GetPath()

		if p == policyNamespacePath || c.isOverridden(p) {
			continue
		}

		pathsToCheck = append(pathsToCheck, p)
	}

	if len(pathsToCheck) == 0 {
		return nil
	}

	childPolicies, err := c.dbClient.CleanupPolicies.GetCleanupPolicies(ctx, &db.GetCleanupPoliciesInput{
		NamespacePaths: pathsToCheck,
		Kind:           &c.policy.Kind,
	})
	if err != nil {
		return errors.Wrap(err, "failed to check fetch policies for override in sweep cache")
	}

	for _, cp := range childPolicies {
		c.addOverride(cp.NamespacePath())
	}

	return nil
}

// isOverridden reports whether nsPath has its own policy or is under a namespace that does.
func (c *overrideCache) isOverridden(nsPath string) bool {
	if _, own := c.overriddenPrefixes[nsPath]; own {
		return true
	}

	for prefix := range c.overriddenPrefixes {
		if nsutils.IsDescendantOfPath(nsPath, prefix) {
			return true
		}
	}

	return false
}
