package namespace

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestNew(t *testing.T) {
	dbClient := &db.Client{}
	svc, ok := New(dbClient).(*service)
	require.True(t, ok)
	assert.Same(t, dbClient, svc.dbClient)
}

func TestGetNamespace(t *testing.T) {
	groupNamespace := &db.Namespace{Type: models.NamespaceTypeGroup, Group: &models.Group{FullPath: "top/group"}}
	workspaceNamespace := &db.Namespace{Type: models.NamespaceTypeWorkspace, Workspace: &models.Workspace{FullPath: "top/workspace"}}

	testCases := []struct {
		name               string
		fullPath           string
		noCaller           bool
		namespace          *db.Namespace
		dbError            error
		expectPermission   models.Permission
		authError          error
		expectNamespace    *db.Namespace
		expectErrorCode    errors.CodeType
		expectDBLookupMade bool
	}{
		{
			name:               "group requires the view group permission",
			fullPath:           "top/group",
			namespace:          groupNamespace,
			expectPermission:   models.ViewGroupPermission,
			expectNamespace:    groupNamespace,
			expectDBLookupMade: true,
		},
		{
			name:               "workspace requires the view workspace permission",
			fullPath:           "top/workspace",
			namespace:          workspaceNamespace,
			expectPermission:   models.ViewWorkspacePermission,
			expectNamespace:    workspaceNamespace,
			expectDBLookupMade: true,
		},
		{
			name:               "namespace not found",
			fullPath:           "top/missing",
			expectErrorCode:    errors.ENotFound,
			expectDBLookupMade: true,
		},
		{
			name:               "db error",
			fullPath:           "top/group",
			dbError:            errors.New("boom"),
			expectErrorCode:    errors.EInternal,
			expectDBLookupMade: true,
		},
		{
			name:               "group permission denied",
			fullPath:           "top/group",
			namespace:          groupNamespace,
			expectPermission:   models.ViewGroupPermission,
			authError:          errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode:    errors.EForbidden,
			expectDBLookupMade: true,
		},
		{
			name:               "workspace permission denied",
			fullPath:           "top/workspace",
			namespace:          workspaceNamespace,
			expectPermission:   models.ViewWorkspacePermission,
			authError:          errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode:    errors.EForbidden,
			expectDBLookupMade: true,
		},
		{
			name:            "without caller",
			fullPath:        "top/group",
			noCaller:        true,
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			mockNamespaces := db.NewMockNamespaces(t)

			if !test.noCaller {
				mockCaller := auth.NewMockCaller(t)
				ctx = auth.WithCaller(ctx, mockCaller)

				if test.expectDBLookupMade {
					mockNamespaces.On("GetNamespace", mock.Anything, test.fullPath).Return(test.namespace, test.dbError)
				}
				if test.expectPermission != (models.Permission{}) {
					mockCaller.On("RequirePermission", mock.Anything, test.expectPermission, mock.Anything).Return(test.authError)
				}
			}

			svc := New(&db.Client{Namespaces: mockNamespaces})
			actual, err := svc.GetNamespace(ctx, test.fullPath)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectNamespace, actual)
		})
	}
}

func TestGetNamespaces(t *testing.T) {
	parentID := "parent-group-id"
	parent := &models.Group{Metadata: models.ResourceMetadata{ID: parentID}, FullPath: "top/parent"}
	memberships := []models.MembershipNamespace{{Path: "top"}}
	sortDesc := db.NamespaceSortableFieldFullPathDesc
	dbResult := &db.NamespacesResult{Namespaces: []db.Namespace{{Type: models.NamespaceTypeGroup, Group: parent}}}

	testCases := []struct {
		name             string
		input            *GetNamespacesInput
		noCaller         bool
		admin            bool
		memberships      []models.MembershipNamespace
		membershipsError error
		permissionError  error
		parentLookup     *models.Group
		parentLookupErr  error
		dbError          error
		expectFilter     *db.NamespaceFilter
		expectSort       *db.NamespaceSortableField
		expectErrorCode  errors.CodeType
	}{
		{
			name:         "non-admin top level is filtered to the caller's memberships",
			input:        &GetNamespacesInput{TopLevel: true},
			memberships:  memberships,
			expectFilter: &db.NamespaceFilter{TopLevel: true, RootNamespaceMemberships: memberships},
		},
		{
			name:         "non-admin caller without memberships gets an empty, non-nil filter",
			input:        &GetNamespacesInput{},
			memberships:  nil,
			expectFilter: &db.NamespaceFilter{RootNamespaceMemberships: []models.MembershipNamespace{}},
		},
		{
			name:         "admin mode applies no membership filter",
			input:        &GetNamespacesInput{TopLevel: true},
			admin:        true,
			expectFilter: &db.NamespaceFilter{TopLevel: true},
		},
		{
			name:             "membership lookup error",
			input:            &GetNamespacesInput{},
			membershipsError: errors.New("boom"),
			expectErrorCode:  errors.EInternal,
		},
		{
			name:         "search and sort are passed through",
			input:        &GetNamespacesInput{Search: ptr.String("abc"), Sort: &sortDesc},
			admin:        true,
			expectFilter: &db.NamespaceFilter{Search: ptr.String("abc")},
			expectSort:   &sortDesc,
		},
		{
			name:         "parent group lists its direct children",
			input:        &GetNamespacesInput{ParentGroupID: &parentID},
			parentLookup: parent,
			expectFilter: &db.NamespaceFilter{ParentPath: ptr.String("top/parent")},
		},
		{
			name:         "parent group with nested includes every descendant",
			input:        &GetNamespacesInput{ParentGroupID: &parentID, IncludeNested: true, Search: ptr.String("x")},
			parentLookup: parent,
			expectFilter: &db.NamespaceFilter{ParentPath: ptr.String("top/parent"), IncludeNested: true, Search: ptr.String("x")},
		},
		{
			name:            "parent group permission denied",
			input:           &GetNamespacesInput{ParentGroupID: &parentID},
			permissionError: errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "parent group not found",
			input:           &GetNamespacesInput{ParentGroupID: &parentID},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "parent group lookup error",
			input:           &GetNamespacesInput{ParentGroupID: &parentID},
			parentLookupErr: errors.New("boom"),
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "db error",
			input:           &GetNamespacesInput{},
			admin:           true,
			dbError:         errors.New("boom"),
			expectFilter:    &db.NamespaceFilter{},
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "without caller",
			input:           &GetNamespacesInput{},
			noCaller:        true,
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			mockNamespaces := db.NewMockNamespaces(t)
			mockGroups := db.NewMockGroups(t)

			if !test.noCaller {
				mockCaller := auth.NewMockCaller(t)
				ctx = auth.WithCaller(ctx, mockCaller)

				if test.input.ParentGroupID != nil {
					mockCaller.On("RequirePermission", mock.Anything, models.ViewGroupPermission, mock.Anything).Return(test.permissionError)
					if test.permissionError == nil {
						mockGroups.On("GetGroupByID", mock.Anything, *test.input.ParentGroupID).Return(test.parentLookup, test.parentLookupErr)
					}
				} else {
					mockCaller.On("IsAdminModeActivated", mock.Anything).Return(test.admin)
					if !test.admin {
						mockCaller.On("GetRootNamespaceMemberships", mock.Anything).Return(test.memberships, test.membershipsError)
					}
				}
			}

			if test.expectFilter != nil {
				mockNamespaces.On("GetNamespaces", mock.Anything, mock.MatchedBy(func(in *db.GetNamespacesInput) bool {
					return assert.ObjectsAreEqual(test.expectFilter, in.Filter) && assert.ObjectsAreEqual(test.expectSort, in.Sort)
				})).Return(dbResult, test.dbError)
			}

			svc := New(&db.Client{Namespaces: mockNamespaces, Groups: mockGroups})
			actual, err := svc.GetNamespaces(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, dbResult, actual)
		})
	}
}
