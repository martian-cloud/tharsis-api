package namespace

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// Namespace represents a group or workspace
type Namespace interface {
	GetPath() string
	GetParentID() string
	ExpandPath() []string
	GetRunnerTags() []string
}

// RunnerTagsSetting contains tag settings, inherited or direct, returned to the group and workspace resolvers.
type RunnerTagsSetting struct {
	Inherited     bool
	NamespacePath string
	Value         []string
}

// InheritedSettingResolver is used to resolve inherited settings by searching the group hierarchy
type InheritedSettingResolver interface {
	GetRunnerTags(ctx context.Context, namespace Namespace) (*RunnerTagsSetting, error)
}

type getSettingFunc func(namespace Namespace) (any, bool)

type setting struct {
	inherited     bool
	namespacePath string
	value         any
}

type inheritedSettingsResolver struct {
	dbClient *db.Client
}

// NewInheritedSettingResolver creates a new instance
func NewInheritedSettingResolver(dbClient *db.Client) InheritedSettingResolver {
	return &inheritedSettingsResolver{
		dbClient: dbClient,
	}
}

func (r *inheritedSettingsResolver) GetRunnerTags(ctx context.Context, namespace Namespace) (*RunnerTagsSetting, error) {
	response, err := r.getInheritedSetting(ctx, namespace, func(namespace Namespace) (any, bool) {
		tags := namespace.GetRunnerTags()
		return tags, tags != nil
	})
	if err != nil {
		return nil, err
	}

	value := []string{}
	if response.value != nil {
		value = response.value.([]string)
	}

	return &RunnerTagsSetting{
		Inherited:     response.inherited,
		NamespacePath: response.namespacePath,
		Value:         value,
	}, nil
}

func (r *inheritedSettingsResolver) getInheritedSetting(ctx context.Context, namespace Namespace, getSetting getSettingFunc) (*setting, error) {
	// The group sets its own tags.
	if s, ok := getSetting(namespace); ok {
		return &setting{
			inherited:     false,
			namespacePath: namespace.GetPath(),
			value:         s,
		}, nil
	}

	// A root group has no ancestors.
	// To avoid false positives, don't look for ancestor groups.
	if namespace.GetParentID() == "" {
		// At this point, we know group setting is nil.
		return &setting{
			inherited:     false,
			namespacePath: namespace.GetPath(),
		}, nil
	}

	sortLowestToHighest := db.GroupSortableFieldFullPathDesc
	parentGroupsResult, err := r.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{
		Sort: &sortLowestToHighest,
		Filter: &db.GroupFilter{
			GroupPaths: namespace.ExpandPath()[1:],
		},
	})
	if err != nil {
		return nil, err
	}

	parentGroups := []*models.Group{}
	for _, g := range parentGroupsResult.Groups {
		copyGroup := g
		parentGroups = append(parentGroups, &copyGroup)
	}

	// Find the first/lowest group with the setting defined
	for _, g := range parentGroups {
		if s, ok := getSetting(g); ok {
			return &setting{
				inherited:     true,
				namespacePath: g.FullPath,
				value:         s,
			}, nil
		}
	}

	// No setting found in any ancestor group.
	// The last group in the list is a root group, so return its full path.
	return &setting{
		inherited:     true,
		namespacePath: parentGroups[len(parentGroups)-1].FullPath,
	}, nil
}
