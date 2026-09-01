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

// moduleVersionPruner deletes the module versions that a terraform module cleanup policy marks stale.
type moduleVersionPruner struct {
	dbClient *db.Client
}

func (p *moduleVersionPruner) prune(ctx context.Context, ns namespace.Namespace, policy *models.CleanupPolicy, onDelete onDeleteFn) error {
	// Only groups have terraform modules. A TERRAFORM_MODULES-kind policy is only ever configurable on
	// a group, so getting anything else here is a bug in the caller, not a state to skip quietly.
	group, ok := ns.(*models.Group)
	if !ok {
		return errors.New("terraform module cleanup policy pruner was given a %s namespace %q, which is not a group", ns.GetModelType(), ns.GetPath())
	}

	var cursor *string

	for {
		result, err := p.dbClient.TerraformModules.GetModules(ctx, &db.GetModulesInput{
			Sort:              new(db.TerraformModuleSortableFieldCreatedAtDesc),
			PaginationOptions: &pagination.Options{First: new(int32(sweepPageSize)), After: cursor},
			Filter:            &db.TerraformModuleFilter{GroupID: &group.Metadata.ID},
		})
		if err != nil {
			return errors.Wrap(err, "failed to get terraform modules in cleanup policy sweeper")
		}

		for i := range result.Modules {
			if err = p.pruneVersions(ctx, &result.Modules[i], policy.TerraformModulePolicyData, onDelete); err != nil {
				return errors.Wrap(err, "failed to prune versions of terraform module %q", result.Modules[i].Name)
			}
		}

		if !result.PageInfo.HasNextPage || len(result.Modules) == 0 {
			return nil
		}

		cursor, err = result.PageInfo.Cursor(&result.Modules[len(result.Modules)-1])
		if err != nil {
			return errors.Wrap(err, "failed to get the next terraform module cursor in cleanup policy sweeper")
		}

		select {
		case <-time.After(pageSleepInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// pruneVersions deletes the stale versions of one module.
func (p *moduleVersionPruner) pruneVersions(ctx context.Context, module *models.TerraformModule, data *models.TerraformModuleCleanupPolicyData, onDelete onDeleteFn) error {
	matcher := newModuleVersionMatcher(module, data.Rules)

	if matcher.empty() {
		return nil
	}

	uploaded := models.TerraformModuleVersionStatusUploaded
	var cursor *string

	for {
		result, err := p.dbClient.TerraformModuleVersions.GetModuleVersions(ctx, &db.GetModuleVersionsInput{
			Sort:              new(db.TerraformModuleVersionSortableFieldCreatedAtDesc),
			PaginationOptions: &pagination.Options{First: new(int32(sweepPageSize)), After: cursor},
			Filter: &db.TerraformModuleVersionFilter{
				ModuleID: &module.Metadata.ID,
				Status:   &uploaded,
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to get terraform module versions in cleanup policy sweeper")
		}

		var candidates []*models.TerraformModuleVersion
		for i := range result.ModuleVersions {
			mv := &result.ModuleVersions[i]
			if matcher.match(mv) {
				candidates = append(candidates, mv)
			}
		}

		if err = deleteInChunks(ctx, candidates,
			func(mv *models.TerraformModuleVersion) string { return mv.Metadata.ID },
			func(mv *models.TerraformModuleVersion) string { return mv.Metadata.TRN },
			func(ctx context.Context, chunk []*models.TerraformModuleVersion) ([]string, error) {
				return p.dbClient.TerraformModuleVersions.DeleteModuleVersionBatch(ctx, &db.DeleteModuleVersionBatchInput{ModuleVersions: chunk})
			},
			onDelete,
		); err != nil {
			return errors.Wrap(err, "failed to delete terraform module versions in cleanup policy sweeper")
		}

		if !result.PageInfo.HasNextPage || len(result.ModuleVersions) == 0 {
			return nil
		}

		cursor, err = result.PageInfo.Cursor(&result.ModuleVersions[len(result.ModuleVersions)-1])
		if err != nil {
			return errors.Wrap(err, "failed to get the next terraform module version cursor in cleanup policy sweeper")
		}

		select {
		case <-time.After(pageSleepInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// moduleVersionMatcher matches a module's versions against the rules targeting that module.
// Version rules use AGE strategy only; the module's current latest version is never deleted.
type moduleVersionMatcher struct {
	rules []*models.TerraformModuleCleanupRule
	now   time.Time
}

// newModuleVersionMatcher filters rules to those whose NameGlob and SystemGlob match the given
// module, so match only needs to check VersionGlob and strategy.
func newModuleVersionMatcher(module *models.TerraformModule, rules []*models.TerraformModuleCleanupRule) *moduleVersionMatcher {
	var matched []*models.TerraformModuleCleanupRule
	for _, rule := range rules {
		if rule.NameGlob.Matches(module.Name) && rule.SystemGlob.Matches(module.System) {
			matched = append(matched, rule)
		}
	}

	return &moduleVersionMatcher{rules: matched, now: time.Now().UTC()}
}

// empty reports whether no rules apply to this resource.
func (m *moduleVersionMatcher) empty() bool {
	return len(m.rules) == 0
}

// match reports whether the first matching rule deletes the module version.
func (m *moduleVersionMatcher) match(moduleVersion *models.TerraformModuleVersion) bool {
	if moduleVersion.Latest {
		return false
	}

	for _, rule := range m.rules {
		if !rule.VersionGlob.Matches(moduleVersion.SemanticVersion) {
			continue
		}

		if rule.Strategy == models.StrategyProtect {
			return false
		}

		return moduleVersion.Metadata.CreationTimestamp.Before(m.now.AddDate(0, 0, -int(rule.DeleteAfterDays)))
	}

	return false
}
