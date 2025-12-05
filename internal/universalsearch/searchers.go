package universalsearch

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Factory functions for each resource type
func groupSearcher(service group.Service) searchFunc {
	return func(ctx context.Context, query string, limit int32) ([]*SearchResult, error) {
		sortBy := db.GroupSortableFieldGroupLevelAsc
		result, err := service.GetGroups(ctx, &group.GetGroupsInput{
			Search:            &query,
			PaginationOptions: &pagination.Options{First: &limit},
			Sort:              &sortBy,
		})
		if err != nil {
			return nil, err
		}

		results := make([]*SearchResult, len(result.Groups))
		for i, g := range result.Groups {
			results[i] = &SearchResult{Data: &g, Type: g.GetModelType()}
		}
		return results, nil
	}
}

func workspaceSearcher(service workspace.Service) searchFunc {
	return func(ctx context.Context, query string, limit int32) ([]*SearchResult, error) {
		sortBy := db.WorkspaceSortableFieldFullPathAsc
		result, err := service.GetWorkspaces(ctx, &workspace.GetWorkspacesInput{
			Search:            &query,
			PaginationOptions: &pagination.Options{First: &limit},
			Sort:              &sortBy,
		})
		if err != nil {
			return nil, err
		}

		results := make([]*SearchResult, len(result.Workspaces))
		for i, w := range result.Workspaces {
			results[i] = &SearchResult{Data: &w, Type: w.GetModelType()}
		}
		return results, nil
	}
}

func teamSearcher(service team.Service) searchFunc {
	return func(ctx context.Context, query string, limit int32) ([]*SearchResult, error) {
		sortBy := db.TeamSortableFieldNameAsc
		result, err := service.GetTeams(ctx, &team.GetTeamsInput{
			TeamNamePrefix:    &query,
			PaginationOptions: &pagination.Options{First: &limit},
			Sort:              &sortBy,
		})
		if err != nil {
			return nil, err
		}

		results := make([]*SearchResult, len(result.Teams))
		for i, t := range result.Teams {
			results[i] = &SearchResult{Data: &t, Type: t.GetModelType()}
		}
		return results, nil
	}
}

func moduleSearcher(service moduleregistry.Service) searchFunc {
	return func(ctx context.Context, query string, limit int32) ([]*SearchResult, error) {
		sortBy := db.TerraformModuleSortableFieldFieldGroupLevelAsc
		result, err := service.GetModules(ctx, &moduleregistry.GetModulesInput{
			Search:            &query,
			PaginationOptions: &pagination.Options{First: &limit},
			Sort:              &sortBy,
		})
		if err != nil {
			return nil, err
		}

		results := make([]*SearchResult, len(result.Modules))
		for i, m := range result.Modules {
			results[i] = &SearchResult{Data: &m, Type: m.GetModelType()}
		}
		return results, nil
	}
}

func providerSearcher(service providerregistry.Service) searchFunc {
	return func(ctx context.Context, query string, limit int32) ([]*SearchResult, error) {
		sortBy := db.TerraformProviderSortableFieldNameAsc
		result, err := service.GetProviders(ctx, &providerregistry.GetProvidersInput{
			Search:            &query,
			PaginationOptions: &pagination.Options{First: &limit},
			Sort:              &sortBy,
		})
		if err != nil {
			return nil, err
		}

		results := make([]*SearchResult, len(result.Providers))
		for i, p := range result.Providers {
			results[i] = &SearchResult{Data: &p, Type: p.GetModelType()}
		}
		return results, nil
	}
}
