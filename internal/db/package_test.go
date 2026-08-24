//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TestPackages_GetPackages_RootNamespaceMembershipsFilter verifies the global (cross-group)
// listing filter used when no group is specified: a caller sees global packages everywhere,
// anything within their membership subtrees (covers private), and root_group packages that
// share a root group with one of their memberships. A nil filter returns everything (admin);
// a non-nil empty filter returns only global packages.
func TestPackages_GetPackages_RootNamespaceMembershipsFilter(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	rootA, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "pkg-root-a",
		FullPath:  "pkg-root-a",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	rootB, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "pkg-root-b",
		FullPath:  "pkg-root-b",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	// name -> visibility, in the given group.
	type pkgSpec struct {
		name       string
		groupID    string
		visibility models.PackageVisibility
	}
	specs := []pkgSpec{
		{"a-private", rootA.Metadata.ID, models.PackageVisibilityPrivate},
		{"a-rootgroup", rootA.Metadata.ID, models.PackageVisibilityRootGroup},
		{"a-global", rootA.Metadata.ID, models.PackageVisibilityGlobal},
		{"b-private", rootB.Metadata.ID, models.PackageVisibilityPrivate},
		{"b-rootgroup", rootB.Metadata.ID, models.PackageVisibilityRootGroup},
		{"b-global", rootB.Metadata.ID, models.PackageVisibilityGlobal},
	}
	for _, s := range specs {
		_, cErr := testClient.client.Packages.CreatePackage(ctx, &models.Package{
			Name:        s.name,
			GroupID:     s.groupID,
			RootGroupID: s.groupID, // each test package is created directly in a root group
			Kind:        models.PackageKindOPAPolicy,
			Visibility:  s.visibility,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, cErr)
	}

	names := func(res *PackagesResult) map[string]struct{} {
		out := map[string]struct{}{}
		for _, p := range res.Packages {
			out[p.Name] = struct{}{}
		}
		return out
	}

	t.Run("member of root A sees A packages plus all global", func(t *testing.T) {
		res, gErr := testClient.client.Packages.GetPackages(ctx, &GetPackagesInput{
			Filter: &PackageFilter{
				RootNamespaceMemberships: []models.MembershipNamespace{{Path: "pkg-root-a"}},
			},
		})
		require.NoError(t, gErr)
		got := names(res)

		// Visible: everything in root A + global packages anywhere.
		require.Contains(t, got, "a-private")
		require.Contains(t, got, "a-rootgroup")
		require.Contains(t, got, "a-global")
		require.Contains(t, got, "b-global")
		// Not visible: root B's private and root_group packages.
		require.NotContains(t, got, "b-private")
		require.NotContains(t, got, "b-rootgroup")
	})

	t.Run("nil memberships returns all packages (admin)", func(t *testing.T) {
		res, gErr := testClient.client.Packages.GetPackages(ctx, &GetPackagesInput{
			Filter: &PackageFilter{},
		})
		require.NoError(t, gErr)
		got := names(res)
		for _, s := range specs {
			require.Contains(t, got, s.name)
		}
	})

	t.Run("empty memberships returns only global packages", func(t *testing.T) {
		res, gErr := testClient.client.Packages.GetPackages(ctx, &GetPackagesInput{
			Filter: &PackageFilter{
				RootNamespaceMemberships: []models.MembershipNamespace{},
			},
		})
		require.NoError(t, gErr)
		got := names(res)
		require.Contains(t, got, "a-global")
		require.Contains(t, got, "b-global")
		require.NotContains(t, got, "a-private")
		require.NotContains(t, got, "a-rootgroup")
		require.NotContains(t, got, "b-private")
		require.NotContains(t, got, "b-rootgroup")
	})
}

// TestPackages_GetPackages_RootGroupVisibilityAcrossSubtree verifies that a caller whose membership
// is a non-root subgroup (e.g. root/x) still sees root_group-visibility packages owned elsewhere in
// the same root group (e.g. root/y) — the case the root_group branch of the filter exists for —
// while private packages outside the caller's own subtree stay hidden.
func TestPackages_GetPackages_RootGroupVisibilityAcrossSubtree(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	root, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "pkg-deep-root", CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	// Two sibling subgroups under the same root. CreateGroup derives their full paths from the parent.
	groupX, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "x", ParentID: root.Metadata.ID, CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	groupY, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "y", ParentID: root.Metadata.ID, CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	// All packages share the same root group but live in sibling subgroups x and y.
	type pkgSpec struct {
		name       string
		groupID    string
		visibility models.PackageVisibility
	}
	specs := []pkgSpec{
		{"y-rootgroup", groupY.Metadata.ID, models.PackageVisibilityRootGroup},
		{"y-private", groupY.Metadata.ID, models.PackageVisibilityPrivate},
		{"x-private", groupX.Metadata.ID, models.PackageVisibilityPrivate},
	}
	for _, s := range specs {
		_, cErr := testClient.client.Packages.CreatePackage(ctx, &models.Package{
			Name:        s.name,
			GroupID:     s.groupID,
			RootGroupID: root.Metadata.ID, // every package's root is the shared root group
			Kind:        models.PackageKindOPAPolicy,
			Visibility:  s.visibility,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, cErr)
	}

	// Caller is a member of root/x only (a non-root subgroup).
	res, err := testClient.client.Packages.GetPackages(ctx, &GetPackagesInput{
		Filter: &PackageFilter{
			RootNamespaceMemberships: []models.MembershipNamespace{{Path: groupX.FullPath}},
		},
	})
	require.NoError(t, err)

	got := map[string]struct{}{}
	for _, p := range res.Packages {
		got[p.Name] = struct{}{}
	}

	// Visible: the root_group package in the sibling subtree (shares the root group) and the
	// private package in the caller's own subtree.
	require.Contains(t, got, "y-rootgroup")
	require.Contains(t, got, "x-private")
	// Not visible: a private package in the sibling subtree the caller is not a member of.
	require.NotContains(t, got, "y-private")
}

// TestPackages_GetPackages_NamespacePathsFilter verifies the scoped group listing with inherited
// packages: passing a group's ancestor-or-self namespace paths returns packages owned by the group
// and its ancestors regardless of visibility, and does NOT surface packages (even global ones)
// owned by groups outside that ancestry.
func TestPackages_GetPackages_NamespacePathsFilter(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	root, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "pkg-scope-root", CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	child, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "child", ParentID: root.Metadata.ID, CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	// An unrelated root group with a global package, to confirm the inherited listing does not
	// surface packages owned outside the target group's ancestry.
	other, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "pkg-scope-other", CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	mustCreate := func(name, groupID, rootGroupID string, v models.PackageVisibility) {
		_, cErr := testClient.client.Packages.CreatePackage(ctx, &models.Package{
			Name:        name,
			GroupID:     groupID,
			RootGroupID: rootGroupID,
			Kind:        models.PackageKindOPAPolicy,
			Visibility:  v,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, cErr)
	}
	mustCreate("root-private", root.Metadata.ID, root.Metadata.ID, models.PackageVisibilityPrivate)
	mustCreate("child-private", child.Metadata.ID, root.Metadata.ID, models.PackageVisibilityPrivate)
	mustCreate("other-global", other.Metadata.ID, other.Metadata.ID, models.PackageVisibilityGlobal)

	// List the child group including inherited packages: pass its ancestor-or-self paths.
	res, err := testClient.client.Packages.GetPackages(ctx, &GetPackagesInput{
		Filter: &PackageFilter{
			NamespacePaths: []string{child.FullPath, root.FullPath},
		},
	})
	require.NoError(t, err)

	got := map[string]struct{}{}
	for _, p := range res.Packages {
		got[p.Name] = struct{}{}
	}

	// Visible: the child's own package and the inherited ancestor package.
	require.Contains(t, got, "child-private")
	require.Contains(t, got, "root-private")
	// Not visible: a global package owned by an unrelated group is not inherited by this group.
	require.NotContains(t, got, "other-global")
}

// TestPackages_GetPackages_VisibleToGroupFilter verifies the "everything this group may reference"
// listing, which is the superset of the inherited scope above: it adds root_group packages owned
// anywhere in the group's own root group and global packages owned anywhere at all, while still
// excluding private packages outside the group's ancestry — including its own descendants, since
// private means the defining group and its *subgroups*. It is the SQL mirror of
// core/packageregistry.IsPackageVisibleToNamespace.
func TestPackages_GetPackages_VisibleToGroupFilter(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	root, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "pkg-vis-root", CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	// The group under test.
	child, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "child", ParentID: root.Metadata.ID, CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	// A descendant of the group under test, to prove visibility does not flow upwards.
	grandchild, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "grandchild", ParentID: child.Metadata.ID, CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	// A sibling subtree sharing the same root group.
	sibling, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "sibling", ParentID: root.Metadata.ID, CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	// An unrelated root group, so root_group visibility can be told apart from global.
	other, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "pkg-vis-other", CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	mustCreate := func(name, groupID, rootGroupID string, v models.PackageVisibility) {
		_, cErr := testClient.client.Packages.CreatePackage(ctx, &models.Package{
			Name:        name,
			GroupID:     groupID,
			RootGroupID: rootGroupID,
			Kind:        models.PackageKindOPAPolicy,
			Visibility:  v,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, cErr)
	}
	mustCreate("root-private", root.Metadata.ID, root.Metadata.ID, models.PackageVisibilityPrivate)
	mustCreate("child-private", child.Metadata.ID, root.Metadata.ID, models.PackageVisibilityPrivate)
	mustCreate("grandchild-private", grandchild.Metadata.ID, root.Metadata.ID, models.PackageVisibilityPrivate)
	mustCreate("sibling-rootgroup", sibling.Metadata.ID, root.Metadata.ID, models.PackageVisibilityRootGroup)
	mustCreate("sibling-private", sibling.Metadata.ID, root.Metadata.ID, models.PackageVisibilityPrivate)
	mustCreate("other-global", other.Metadata.ID, other.Metadata.ID, models.PackageVisibilityGlobal)
	mustCreate("other-rootgroup", other.Metadata.ID, other.Metadata.ID, models.PackageVisibilityRootGroup)
	mustCreate("other-private", other.Metadata.ID, other.Metadata.ID, models.PackageVisibilityPrivate)

	names := func(res *PackagesResult) map[string]struct{} {
		out := map[string]struct{}{}
		for _, p := range res.Packages {
			out[p.Name] = struct{}{}
		}
		return out
	}

	t.Run("returns every package visible to the group", func(t *testing.T) {
		res, gErr := testClient.client.Packages.GetPackages(ctx, &GetPackagesInput{
			Filter: &PackageFilter{VisibleToGroup: &child.FullPath},
		})
		require.NoError(t, gErr)
		got := names(res)

		// The inherited scope: the group's own package and its ancestor's.
		require.Contains(t, got, "child-private")
		require.Contains(t, got, "root-private")
		// What this filter adds over the inherited scope.
		require.Contains(t, got, "sibling-rootgroup")
		require.Contains(t, got, "other-global")

		// Private packages outside the ancestry stay hidden, in both directions.
		require.NotContains(t, got, "sibling-private")
		require.NotContains(t, got, "grandchild-private")
		require.NotContains(t, got, "other-private")
		// root_group visibility does not reach across root groups; only global does.
		require.NotContains(t, got, "other-rootgroup")
	})

	t.Run("scopes a root group to its own path", func(t *testing.T) {
		// A single-segment path: the ancestry expands to just itself, and it is its own root group. The
		// filter takes only the path, so this is where that derivation is exercised.
		res, gErr := testClient.client.Packages.GetPackages(ctx, &GetPackagesInput{
			Filter: &PackageFilter{VisibleToGroup: &root.FullPath},
		})
		require.NoError(t, gErr)
		got := names(res)

		require.Contains(t, got, "root-private")
		require.Contains(t, got, "sibling-rootgroup")
		require.Contains(t, got, "other-global")
		// Its own descendants' private packages are not visible to it: private reaches down, not up.
		require.NotContains(t, got, "child-private")
		require.NotContains(t, got, "sibling-private")
		require.NotContains(t, got, "other-rootgroup")
	})

	t.Run("search matches the fully-qualified package source", func(t *testing.T) {
		// A group-path substring that appears in no package name. Under the old name-only predicate
		// this returned nothing.
		search := sibling.FullPath
		res, gErr := testClient.client.Packages.GetPackages(ctx, &GetPackagesInput{
			Filter: &PackageFilter{VisibleToGroup: &child.FullPath, Search: &search},
		})
		require.NoError(t, gErr)
		got := names(res)

		require.Contains(t, got, "sibling-rootgroup")
		// Still scoped by visibility: the sibling's private package does not become visible.
		require.NotContains(t, got, "sibling-private")
		// And unrelated visible packages are filtered out by the search.
		require.NotContains(t, got, "child-private")
		require.NotContains(t, got, "other-global")
	})
}
