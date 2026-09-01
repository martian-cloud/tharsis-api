package cleanup

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// providerVersionPruner deletes the provider versions that a terraform provider cleanup policy marks stale.
type providerVersionPruner struct {
	dbClient *db.Client
}

func (p *providerVersionPruner) prune(ctx context.Context, ns namespace.Namespace, policy *models.CleanupPolicy, onDelete onDeleteFn) error {
	// Only groups have terraform providers. A TERRAFORM_PROVIDERS-kind policy is only ever configurable
	// on a group, so getting anything else here is a bug in the caller, not a state to skip quietly.
	group, ok := ns.(*models.Group)
	if !ok {
		return errors.New("terraform provider cleanup policy pruner was given a %s namespace %q, which is not a group", ns.GetModelType(), ns.GetPath())
	}

	var cursor *string

	for {
		result, err := p.dbClient.TerraformProviders.GetProviders(ctx, &db.GetProvidersInput{
			Sort:              new(db.TerraformProviderSortableFieldCreatedAtDesc),
			PaginationOptions: &pagination.Options{First: new(int32(sweepPageSize)), After: cursor},
			Filter:            &db.TerraformProviderFilter{GroupID: &group.Metadata.ID},
		})
		if err != nil {
			return errors.Wrap(err, "failed to get terraform providers in cleanup policy sweeper")
		}

		for i := range result.Providers {
			if err = p.pruneVersions(ctx, &result.Providers[i], policy.TerraformProviderPolicyData, onDelete); err != nil {
				return errors.Wrap(err, "failed to prune versions of terraform provider %q", result.Providers[i].Name)
			}
		}

		if !result.PageInfo.HasNextPage || len(result.Providers) == 0 {
			return nil
		}

		cursor, err = result.PageInfo.Cursor(&result.Providers[len(result.Providers)-1])
		if err != nil {
			return errors.Wrap(err, "failed to get the next terraform provider cursor in cleanup policy sweeper")
		}

		select {
		case <-time.After(pageSleepInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// pruneVersions deletes the stale versions of one provider.
func (p *providerVersionPruner) pruneVersions(ctx context.Context, provider *models.TerraformProvider, data *models.TerraformProviderCleanupPolicyData, onDelete onDeleteFn) error {
	matcher := newProviderVersionMatcher(provider, data.Rules)

	if matcher.empty() {
		return nil
	}

	var cursor *string

	for {
		result, err := p.dbClient.TerraformProviderVersions.GetProviderVersions(ctx, &db.GetProviderVersionsInput{
			Sort:              new(db.TerraformProviderVersionSortableFieldCreatedAtDesc),
			PaginationOptions: &pagination.Options{First: new(int32(sweepPageSize)), After: cursor},
			Filter: &db.TerraformProviderVersionFilter{
				ProviderID: &provider.Metadata.ID,
				// Versions still in progress (missing either) are not yet usable and must not be deleted.
				SHASumsUploaded:          new(true),
				SHASumsSignatureUploaded: new(true),
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to get terraform provider versions in cleanup policy sweeper")
		}

		var candidates []*models.TerraformProviderVersion
		for i := range result.ProviderVersions {
			pv := &result.ProviderVersions[i]
			if matcher.match(pv) {
				candidates = append(candidates, pv)
			}
		}

		if err = deleteInChunks(ctx, candidates,
			func(mv *models.TerraformProviderVersion) string { return mv.Metadata.ID },
			func(mv *models.TerraformProviderVersion) string { return mv.Metadata.TRN },
			func(ctx context.Context, chunk []*models.TerraformProviderVersion) ([]string, error) {
				return p.dbClient.TerraformProviderVersions.DeleteProviderVersionBatch(ctx, &db.DeleteProviderVersionBatchInput{ProviderVersions: chunk})
			},
			onDelete,
		); err != nil {
			return errors.Wrap(err, "failed to delete terraform provider versions in cleanup policy sweeper")
		}

		if !result.PageInfo.HasNextPage || len(result.ProviderVersions) == 0 {
			return nil
		}

		cursor, err = result.PageInfo.Cursor(&result.ProviderVersions[len(result.ProviderVersions)-1])
		if err != nil {
			return errors.Wrap(err, "failed to get the next terraform provider version cursor in cleanup policy sweeper")
		}

		select {
		case <-time.After(pageSleepInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// providerVersionMatcher matches a provider's versions against the rules targeting that provider.
// Version rules use AGE strategy only; the latest version is protected at the DB level.
type providerVersionMatcher struct {
	rules []*models.TerraformProviderCleanupRule
	now   time.Time
}

// newProviderVersionMatcher filters rules to those whose NameGlob matches the given provider,
// so match only needs to check VersionGlob and strategy.
func newProviderVersionMatcher(provider *models.TerraformProvider, rules []*models.TerraformProviderCleanupRule) *providerVersionMatcher {
	var matched []*models.TerraformProviderCleanupRule
	for _, rule := range rules {
		if rule.NameGlob.Matches(provider.Name) {
			matched = append(matched, rule)
		}
	}

	return &providerVersionMatcher{rules: matched, now: time.Now().UTC()}
}

// empty reports whether no rules apply to this resource.
func (m *providerVersionMatcher) empty() bool {
	return len(m.rules) == 0
}

// match reports whether the first matching rule deletes the provider version.
func (m *providerVersionMatcher) match(providerVersion *models.TerraformProviderVersion) bool {
	if providerVersion.Latest {
		return false
	}

	for _, rule := range m.rules {
		if !rule.VersionGlob.Matches(providerVersion.SemanticVersion) {
			continue
		}

		if rule.Strategy == models.StrategyProtect {
			return false
		}

		return providerVersion.Metadata.CreationTimestamp.Before(m.now.AddDate(0, 0, -int(rule.DeleteAfterDays)))
	}

	return false
}
