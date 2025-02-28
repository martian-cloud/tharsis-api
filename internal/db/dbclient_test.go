//go:build integration

package db

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (

	// a contrived bogus ID
	nonExistentID = "12345678-1234-1234-1234-123456789abc"

	// an invalid ID
	invalidID = "not-a-valid-uuid"

	// must set max job duration when creating a workspace
	forTestMaxJobDuration = time.Hour * 12

	// Maximum number of DB connections--intended to mirror the CI environment
	maxConns = 4
)

var (

	// Used by NewClient to build the DB URI:
	// The script that runs the tests should set these via build flags.
	TestDBHost string
	TestDBPort string // contents of string must be numeric
	TestDBName string
	TestDBMode string
	TestDBUser string
	TestDBPass string

	// constants that cannot be constants

	// Map of names of tables that are excluded from being truncated.
	nonTruncateTables = map[string]interface{}{
		"schema_migrations": nil,
		"resource_limits":   nil,
	}

	// returned for resource version mismatch (or what the DB layer thinks is a version mismatch)
	resourceVersionMismatch = ptr.String(ErrOptimisticLockError.Error())

	// returned for some invalid UUID cases
	invalidUUIDMsg1 = ptr.String("ERROR: invalid input syntax for type uuid: \"" + invalidID + "\" (SQLSTATE 22P02)")

	// returned for some other invalid UUID cases
	invalidUUIDMsg2 = ptr.String(fmt.Sprintf("failed to scan query count result: %s", *invalidUUIDMsg1))

	// returned for some other invalid UUID cases
	emptyUUIDMsg2 = ptr.String("failed to scan query count result: ERROR: invalid input syntax for type uuid: \"\" (SQLSTATE 22P02)")

	// returned for some invalid UUID cases
	invalidUUIDMsg4 = ptr.String("ERROR: invalid input syntax for type uuid: \"\" (SQLSTATE 22P02)")
)

type testClient struct {
	logger logger.Logger
	client *Client
}

// time bounds for comparing object metadata
type timeBounds struct {
	createLow  *time.Time
	createHigh *time.Time
	updateLow  *time.Time
	updateHigh *time.Time
}

// newTestClient creates a new DB client to use for the DB integration tests.
// It also wipes all tables empty.
// Based on environment variables, the client could be for a local standalone DB server
// or one created inside the CI/CD pipeline.
func newTestClient(ctx context.Context, t *testing.T) *testClient {
	portNum, err := strconv.Atoi(TestDBPort)
	if err != nil {
		t.Fatal(err)
	}

	logger, _ := logger.NewForTest()

	client, err := NewClient(ctx, TestDBHost, portNum, TestDBName, TestDBMode, TestDBUser, TestDBPass, maxConns, true, logger)
	if err != nil {
		t.Fatal(err)
	}

	result := testClient{
		client: client,
		logger: logger,
	}

	err = result.wipeAllTables(ctx)
	if err != nil {
		t.Fatal(err)
	}

	return &result
}

// close closes the test client but does not terminate the local server
func (tc *testClient) close(ctx context.Context) {
	tc.client.Close(ctx)
}

func (tc *testClient) wipeAllTables(ctx context.Context) error {
	conn := tc.client.getConnection(ctx)

	// Get the names of all tables to wipe.  Sort them to ensure deterministic behavior.
	query := dialect.From(goqu.T("pg_tables")).
		Select("tablename").
		Where(goqu.I("schemaname").Eq("public")).
		Order(goqu.I("tablename").Asc())

	sql, _, err := query.ToSQL()
	if err != nil {
		return err
	}

	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return err
	}
	defer rows.Close()

	model := struct {
		tableName string
	}{}
	fields := []interface{}{
		&model.tableName,
	}
	tableNames := []interface{}{}
	for rows.Next() {
		err = rows.Scan(fields...)
		if err != nil {
			return err
		}
		// Exclude special tables from being wiped.
		if _, ok := nonTruncateTables[model.tableName]; !ok {
			tableNames = append(tableNames, model.tableName)
		}
	}

	if len(tableNames) == 0 {
		return fmt.Errorf("function wipeAllTables found no tables to truncate")
	}

	// Wipe all the tables.
	query2 := dialect.Truncate(tableNames...)

	sql2, _, err := query2.ToSQL()
	if err != nil {
		return err
	}

	_, err = conn.Exec(ctx, sql2)
	if err != nil {
		return err
	}

	return nil
}

type sortableField interface {
	getFieldDescriptor() *pagination.FieldDescriptor
	getSortDirection() pagination.SortDirection
	getValue() string
}

func testResourcePaginationAndSorting(
	ctx context.Context,
	t *testing.T,
	totalCount int,
	sortableFields []sortableField,
	getResourcesFunc func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error),
) {
	/* Test pagination in forward direction */
	defaultSortBy := sortableFields[0]

	middleIndex := totalCount / 2
	pageInfo, resources, err := getResourcesFunc(ctx, defaultSortBy, &pagination.Options{
		First: ptr.Int32(int32(middleIndex)),
	})
	require.Nil(t, err)

	assert.Equal(t, middleIndex, len(resources))
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)

	cursor, err := pageInfo.Cursor(resources[len(resources)-1])
	require.Nil(t, err)

	remaining := totalCount - middleIndex
	pageInfo, resources, err = getResourcesFunc(ctx, defaultSortBy, &pagination.Options{
		First: ptr.Int32(int32(remaining)),
		After: cursor,
	})
	require.Nil(t, err)

	assert.Equal(t, remaining, len(resources))
	assert.True(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)

	/* Test pagination in reverse direction */

	pageInfo, resources, err = getResourcesFunc(ctx, defaultSortBy, &pagination.Options{
		Last: ptr.Int32(int32(middleIndex)),
	})
	require.Nil(t, err)

	assert.Equal(t, middleIndex, len(resources))
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)

	cursor, err = pageInfo.Cursor(resources[len(resources)-1])
	require.Nil(t, err)

	remaining = totalCount - middleIndex
	pageInfo, resources, err = getResourcesFunc(ctx, defaultSortBy, &pagination.Options{
		Last:   ptr.Int32(int32(remaining)),
		Before: cursor,
	})
	require.Nil(t, err)

	assert.Equal(t, remaining, len(resources))
	assert.False(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)

	/* Test sorting */
	for _, sortByField := range sortableFields {
		_, resources, err = getResourcesFunc(ctx, sortByField, &pagination.Options{})
		require.Nil(t, err)

		values := []string{}
		for _, resource := range resources {
			value, err := resource.ResolveMetadata(sortByField.getFieldDescriptor().Key)
			require.Nil(t, err)
			values = append(values, value)
		}

		// Must detect whether values are iso8601 timestamps, which are similar enough to RFC 3339 to use that layout.
		// If they are, must convert them and sort as time.Time rather than as strings.
		// That is because truncated trailing zeros cause string comparison to be different vs. time value comparison.
		areAllTimestamps := true
		timeValues := []time.Time{}
		for _, value := range values {
			tv, err := time.Parse(time.RFC3339, value)
			if err != nil {
				areAllTimestamps = false
				break
			}
			timeValues = append(timeValues, tv)
		}

		if areAllTimestamps {
			// Time value sort/comparison.
			expectedTimes := []time.Time{}
			expectedTimes = append(expectedTimes, timeValues...)

			slices.SortFunc(expectedTimes, func(a, b time.Time) int {
				if sortByField.getSortDirection() == pagination.AscSort {
					return int(a.Sub(b)) // positive if a is later/greater than g
				}
				return int(b.Sub(a)) // positive if b is later/greater than a
			})

			assert.Equal(t, expectedTimes, timeValues, "resources are not sorted correctly when using sort by %s", sortByField.getValue())
		} else {
			// Ordinary string sort/comparison.
			expectedValues := []string{}
			expectedValues = append(expectedValues, values...)

			slices.SortFunc(expectedValues, func(a, b string) int {
				if sortByField.getSortDirection() == pagination.AscSort {
					return cmp.Compare(a, b)
				}
				return cmp.Compare(b, a)
			})

			assert.Equal(t, expectedValues, values, "resources are not sorted correctly when using sort by %s", sortByField.getValue())
		}
	}
}

//////////////////////////////////////////////////////////////////////////////

// Create initial objects to prepare to run tests.  These functions are called
// by several test modules.

// createInitialTeams creates some teams for a test.
func createInitialTeams(ctx context.Context, testClient *testClient,
	toCreate []models.Team) ([]models.Team, map[string]string, error) {
	result := []models.Team{}
	teamName2ID := make(map[string]string)

	for _, team := range toCreate {

		created, err := testClient.client.Teams.CreateTeam(ctx, &team)
		if err != nil {
			return nil, nil, err
		}

		result = append(result, *created)
		teamName2ID[created.Name] = created.Metadata.ID
	}

	return result, teamName2ID, nil
}

// createInitialUsers creates some users for a test.
func createInitialUsers(ctx context.Context, testClient *testClient,
	toCreate []models.User) ([]models.User, map[string]string, error) {
	result := []models.User{}
	username2ID := make(map[string]string)

	for _, user := range toCreate {

		created, err := testClient.client.Users.CreateUser(ctx, &user)
		if err != nil {
			return nil, nil, err
		}

		result = append(result, *created)
		username2ID[created.Username] = created.Metadata.ID
	}

	return result, username2ID, nil
}

// createInitialTeamMembers creates some team member relationships for a test.
func createInitialTeamMembers(ctx context.Context, testClient *testClient,
	teamMap, userMap map[string]string, toCreate []models.TeamMember) ([]models.TeamMember, error) {
	result := []models.TeamMember{}

	for _, input := range toCreate {

		created, err := testClient.client.TeamMembers.AddUserToTeam(ctx, &models.TeamMember{
			UserID: userMap[input.UserID],
			TeamID: teamMap[input.TeamID],
		})
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

// createInitialGroups creates some groups for a test.
//
// NOTE: Due to the need to supply the parent ID for non-top-level groups,
// the groups must be created in a top-down manner.
func createInitialGroups(ctx context.Context, testClient *testClient,
	toCreate []models.Group) ([]models.Group, map[string]string, error) {
	result := []models.Group{}
	fullPath2ID := make(map[string]string)

	for _, group := range toCreate {

		// Derive the parent ID and name from the full path.
		parentPath := fullPath2ParentPath(group.FullPath)
		if parentPath != "" {
			// Must check the parent path and set the Parent ID field.
			parentID, ok := fullPath2ID[parentPath]
			if !ok {
				return nil, nil, fmt.Errorf("Failed to look up parent path in createInitialGroups: %s", parentPath)
			}
			if group.ParentID == "" {
				group.ParentID = parentID
			}
		}
		if group.Name == "" {
			group.Name = fullPath2Name(group.FullPath)
		}

		created, err := testClient.client.Groups.CreateGroup(ctx, &group)
		if err != nil {
			return nil, nil, err
		}

		result = append(result, *created)
		fullPath2ID[created.FullPath] = created.Metadata.ID

	}

	return result, fullPath2ID, nil
}

// createInitialServiceAccounts creates some service accounts for a test.
func createInitialServiceAccounts(ctx context.Context, testClient *testClient, groupMap map[string]string,
	toCreate []models.ServiceAccount) ([]models.ServiceAccount, map[string]string, error) {
	result := []models.ServiceAccount{}
	serviceAccountName2ID := make(map[string]string)

	for _, input := range toCreate {

		created, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
			Name:        input.Name,
			Description: input.Description,
			GroupID:     groupMap[input.GroupID],
		})
		if err != nil {
			return nil, nil, err
		}

		result = append(result, *created)
		serviceAccountName2ID[created.Name] = created.Metadata.ID
	}

	return result, serviceAccountName2ID, nil
}

// createInitialWorkspaces creates some warmup workspaces for a test.
func createInitialWorkspaces(ctx context.Context, testClient *testClient, groupPath2ID map[string]string,
	newWorkspaces []models.Workspace) ([]models.Workspace, error) {

	resultWorkspaces := []models.Workspace{}
	for _, workspace := range newWorkspaces {

		// Derive the group ID and name from the full path.
		groupPath := fullPath2ParentPath(workspace.FullPath)
		parentID, ok := groupPath2ID[groupPath]
		if !ok {
			return nil, fmt.Errorf("Failed to look up parent path in createInitialWorkspaces: %s", groupPath)
		}
		if workspace.GroupID == "" {
			workspace.GroupID = parentID
		}
		if workspace.Name == "" {
			workspace.Name = fullPath2Name(workspace.FullPath)
		}

		// Must set the MaxJobDuration field.
		duration := int32(forTestMaxJobDuration.Minutes())
		workspace.MaxJobDuration = &duration

		created, err := testClient.client.Workspaces.CreateWorkspace(ctx, &workspace)
		if err != nil {
			return nil, err
		}

		resultWorkspaces = append(resultWorkspaces, *created)
	}

	return resultWorkspaces, nil
}

// createInitialNamespaceMemberships creates some warmup namespace memberships for a test.
func createInitialNamespaceMemberships(ctx context.Context, testClient *testClient,
	teamMap, userMap, groupMap, serviceAccountMap, rolesMap map[string]string,
	toCreate []CreateNamespaceMembershipInput) ([]models.NamespaceMembership, error) {
	result := []models.NamespaceMembership{}

	for _, input := range toCreate {

		translated := CreateNamespaceMembershipInput{
			NamespacePath: input.NamespacePath,
			RoleID:        rolesMap[input.RoleID],
		}
		if input.UserID != nil {
			translated.UserID = ptr.String(userMap[*input.UserID])
		}
		if input.ServiceAccountID != nil {
			translated.ServiceAccountID = ptr.String(serviceAccountMap[*input.ServiceAccountID])
		}
		if input.TeamID != nil {
			translated.TeamID = ptr.String(teamMap[*input.TeamID])
		}

		created, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx,
			&translated)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

// createInitialRuns creates some warmup runs for a test.
func createInitialRuns(ctx context.Context, testClient *testClient,
	toCreate []models.Run, workspaceID string) ([]models.Run, error) {
	result := []models.Run{}

	for _, input := range toCreate {
		input.WorkspaceID = workspaceID

		created, err := testClient.client.Runs.CreateRun(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	// In order to make the created-at and last-updated-at orders differ,
	// update every third run without changing any values.
	for ix, toUpdate := range result {
		if ix%3 == 0 {
			updated, err := testClient.client.Runs.UpdateRun(ctx, &toUpdate)
			if err != nil {
				return nil, err
			}
			result[ix] = *updated
		}
	}

	return result, nil
}

// createInitialPlans creates some warmup plans for a test.
func createInitialPlans(ctx context.Context, testClient *testClient,
	toCreate []models.Plan, workspaceID string) ([]models.Plan, error) {
	result := []models.Plan{}

	for _, input := range toCreate {

		input.WorkspaceID = workspaceID
		created, err := testClient.client.Plans.CreatePlan(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

// createInitialApplies creates some warmup applies for a test.
func createInitialApplies(ctx context.Context, testClient *testClient,
	toCreate []models.Apply, workspaceID string) ([]models.Apply, error) {
	result := []models.Apply{}

	for _, input := range toCreate {

		input.WorkspaceID = workspaceID
		created, err := testClient.client.Applies.CreateApply(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

// createInitialConfigurationVersions creates some warmup configuration versions for a test.
func createInitialConfigurationVersions(ctx context.Context, testClient *testClient,
	toCreate []models.ConfigurationVersion) ([]models.ConfigurationVersion, error) {
	result := []models.ConfigurationVersion{}

	for _, input := range toCreate {

		created, err := testClient.client.ConfigurationVersions.CreateConfigurationVersion(ctx, input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

// createInitialJobs creates some warmup jobs for a test.
// Workspace IDs are assigned by rotation.
func createInitialJobs(ctx context.Context, testClient *testClient,
	toCreate []models.Job, workspaceIDs []string, runID, runnerID string) ([]models.Job, error) {
	result := []models.Job{}
	nextWorkspaceIndex := 0

	for _, input := range toCreate {

		workspaceID := workspaceIDs[nextWorkspaceIndex]
		nextWorkspaceIndex = (nextWorkspaceIndex + 1) % len(workspaceIDs)

		input.WorkspaceID = workspaceID
		input.RunID = runID
		input.RunnerID = &runnerID
		if input.Tags == nil {
			input.Tags = []string{}
		}
		created, err := testClient.client.Jobs.CreateJob(ctx, &input)
		if err != nil {
			return nil, err
		}

		// In order to make the created-at and last-updated-at orders differ,
		// update every third run without changing any values.
		for ix, toUpdate := range result {
			if ix%3 == 0 {
				updated, err := testClient.client.Jobs.UpdateJob(ctx, &toUpdate)
				if err != nil {
					return nil, err
				}
				result[ix] = *updated
			}
		}

		result = append(result, *created)
	}

	return result, nil
}

// createInitialManagedIdentities creates some warmup managed identities for a test.
func createInitialManagedIdentities(ctx context.Context, testClient *testClient,
	groupMap map[string]string, toCreate []models.ManagedIdentity) (
	[]models.ManagedIdentity, error) {
	result := []models.ManagedIdentity{}

	for _, input := range toCreate {

		input.GroupID = groupMap[input.GroupID]
		created, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &input)
		if err != nil {
			return nil, fmt.Errorf("failed to create initial managed identity: %s", err)
		}

		result = append(result, *created)
	}

	// In order to make the created-at and last-updated-at orders differ,
	// update every third object without changing any values.
	for ix, toUpdate := range result {
		if ix%3 == 0 {
			updated, err := testClient.client.ManagedIdentities.UpdateManagedIdentity(ctx, &toUpdate)
			if err != nil {
				return nil, fmt.Errorf("failed to update initial managed identity: %s", err)
			}
			result[ix] = *updated
		}
	}

	return result, nil
}

// createInitialManagedIdentityAccessRules creates some warmup managed identity access rules for a test.
func createInitialManagedIdentityAccessRules(ctx context.Context, testClient *testClient,
	managedIdentity2ID, username2ID, serviceAccountName2ID, teamName2ID map[string]string,
	toCreate []models.ManagedIdentityAccessRule) ([]models.ManagedIdentityAccessRule, error) {
	result := []models.ManagedIdentityAccessRule{}

	for _, input := range toCreate {
		var ok bool
		var err error

		inputManagedIdentityID := input.ManagedIdentityID
		input.ManagedIdentityID, ok = managedIdentity2ID[inputManagedIdentityID]
		if !ok {
			return nil, fmt.Errorf("Failed to translate managed identity name to ID: %s", inputManagedIdentityID)
		}

		// translate username to ID
		input.AllowedUserIDs, err = translateNames2IDs(input.AllowedUserIDs, username2ID)
		if err != nil {
			return nil, err
		}

		// translate service account name to ID
		input.AllowedServiceAccountIDs, err = translateNames2IDs(input.AllowedServiceAccountIDs, serviceAccountName2ID)
		if err != nil {
			return nil, err
		}

		// translate name name to ID
		input.AllowedTeamIDs, err = translateNames2IDs(input.AllowedTeamIDs, teamName2ID)
		if err != nil {
			return nil, err
		}

		created, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

// createInitialStateVersions creates some warmup state versions for a test.
func createInitialStateVersions(ctx context.Context, testClient *testClient,
	workspace2ID, run2ID map[string]string,
	toCreate []models.StateVersion) ([]models.StateVersion, error) {
	result := []models.StateVersion{}

	for _, input := range toCreate {
		var ok bool
		var err error

		inputWorkspaceID := input.WorkspaceID
		input.WorkspaceID, ok = workspace2ID[inputWorkspaceID]
		if !ok {
			return nil, fmt.Errorf("Failed to translate workspace path to ID: %s", inputWorkspaceID)
		}

		inputRunID := input.RunID
		if inputRunID != nil {
			var temp string
			temp, ok = run2ID[*inputRunID]
			if !ok {
				return nil, fmt.Errorf("Failed to translate run specifier to ID: %s", *inputRunID)
			}
			input.RunID = &temp
		}

		created, err := testClient.client.StateVersions.CreateStateVersion(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

/*
createInitialStateVersionOutputs creates some warmup state version outputs for a test.

In the input, the state version ID is the workspace's full path, a colon, and the run's comment.
In the state version map, the key is the workspaces ID, a colon, and the run ID.
*/
func createInitialStateVersionOutputs(ctx context.Context, testClient *testClient,
	workspace2ID, run2ID, stateVersion2ID map[string]string,
	toCreate []models.StateVersionOutput) ([]models.StateVersionOutput, error) {
	result := []models.StateVersionOutput{}

	for _, input := range toCreate {

		parts := strings.Split(input.StateVersionID, ":")
		if len(parts) != 2 {
			return nil,
				fmt.Errorf("state version ID fed to createInitialStateVersionOutputs is invalid: %s", input.StateVersionID)
		}

		workspaceID, okw := workspace2ID[parts[0]]
		if !okw {
			return nil,
				fmt.Errorf("failed to look up workspace full path: %s", parts[0])
		}

		runID, okr := run2ID[parts[1]]
		if !okr {
			return nil,
				fmt.Errorf("failed to look up run comment: %s", parts[1])
		}

		var oks bool
		input.StateVersionID, oks = stateVersion2ID[fmt.Sprintf("%s:%s", workspaceID, runID)]
		if !oks {
			return nil,
				fmt.Errorf("failed to look up IDs based on: %s and %s", parts[0], parts[1])
		}

		created, err := testClient.client.StateVersionOutputs.CreateStateVersionOutput(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

// translateNames2IDs translates names to IDs based on the supplied map
func translateNames2IDs(names []string, mapper map[string]string) ([]string, error) {
	result := []string{}

	for _, name := range names {

		translated, ok := mapper[name]
		if !ok {
			return nil, fmt.Errorf("Failed to map name to ID: %s", name)
		}

		result = append(result, translated)
	}

	return result, nil
}

// createInitialVariables creates some warmup variables for a test.
func createInitialVariables(ctx context.Context, testClient *testClient,
	toCreate []models.Variable) ([]models.Variable, error) {
	result := []models.Variable{}

	for _, input := range toCreate {

		created, err := testClient.client.Variables.CreateVariable(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

// createInitialGPGKeys creates some warmup GPG keys for a test.
func createInitialGPGKeys(ctx context.Context, testClient *testClient,
	toCreate []models.GPGKey, groupPath2ID map[string]string) ([]models.GPGKey, error) {
	result := []models.GPGKey{}

	for _, input := range toCreate {

		input.GroupID = groupPath2ID[input.GroupID]
		created, err := testClient.client.GPGKeys.CreateGPGKey(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

// createInitialTerraformProviders creates some warmup Terraform providers for a test.
func createInitialTerraformProviders(ctx context.Context, testClient *testClient,
	toCreate []models.TerraformProvider, groupPath2ID map[string]string) (
	[]models.TerraformProvider, map[string]string, error) {
	result := []models.TerraformProvider{}
	resourcePath2ID := make(map[string]string)

	for _, input := range toCreate {

		rootGroupPath := input.RootGroupID
		rootGroupID, ok := groupPath2ID[rootGroupPath]
		if !ok {
			return nil, nil,
				fmt.Errorf("createInitialTerraformProviders failed to look up root group path: %s", rootGroupPath)
		}
		input.RootGroupID = rootGroupID

		groupPath := input.GroupID
		groupID, ok := groupPath2ID[groupPath]
		if !ok {
			return nil, nil,
				fmt.Errorf("createInitialTerraformProviders failed to look up group path: %s", groupPath)
		}
		input.GroupID = groupID

		created, err := testClient.client.TerraformProviders.CreateProvider(ctx, &input)
		if err != nil {
			return nil, nil, err
		}

		result = append(result, *created)
		resourcePath2ID[created.ResourcePath] = created.Metadata.ID
	}

	return result, resourcePath2ID, nil
}

// createInitialTerraformProviderVersions creates some warmup Terraform provider versions for a test.
func createInitialTerraformProviderVersions(ctx context.Context, testClient *testClient,
	toCreate []models.TerraformProviderVersion,
	providerResourcePath2ID map[string]string) ([]models.TerraformProviderVersion, map[string]string, error) {
	result := []models.TerraformProviderVersion{}
	versionSpecs2ID := make(map[string]string)

	for _, input := range toCreate {

		providerResourcePath := input.ProviderID
		providerID, ok := providerResourcePath2ID[providerResourcePath]
		if !ok {
			return nil, nil,
				fmt.Errorf("createInitialTerraformProviderVersions failed to look up provider resource path: %s",
					providerResourcePath)
		}
		input.ProviderID = providerID

		created, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &input)
		if err != nil {
			return nil, nil, err
		}

		result = append(result, *created)
		versionSpecs2ID[providerResourcePath+":"+created.SemanticVersion] = created.Metadata.ID
	}

	return result, versionSpecs2ID, nil
}

// createInitialTerraformProviderPlatforms creates some warmup Terraform provider platforms for a test.
func createInitialTerraformProviderPlatforms(ctx context.Context, testClient *testClient,
	toCreate []models.TerraformProviderPlatform, versionSpecs2ID map[string]string) ([]models.TerraformProviderPlatform, error) {
	result := []models.TerraformProviderPlatform{}

	for _, input := range toCreate {

		versionSpecs := input.ProviderVersionID
		versionID, ok := versionSpecs2ID[versionSpecs]
		if !ok {
			return nil,
				fmt.Errorf("createInitialTerraformProviderPlatforms failed to look up version specs: %s", versionSpecs)
		}
		input.ProviderVersionID = versionID

		created, err := testClient.client.TerraformProviderPlatforms.CreateProviderPlatform(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

// createInitialActivityEvents creates some warmup activity events for a test.
func createInitialActivityEvents(ctx context.Context, testClient *testClient,
	toCreate []models.ActivityEvent, userMap, serviceAccountMap map[string]string) ([]models.ActivityEvent, error) {
	result := []models.ActivityEvent{}

	for _, input := range toCreate {

		// Replace username with user ID.
		if input.UserID != nil {
			username := *input.UserID
			userID, ok := userMap[username]
			if !ok {
				return nil, fmt.Errorf("Failed to replace username with user ID: %s", username)
			}
			input.UserID = ptr.String(userID)
		}

		// Replace service account name with service account ID.
		if input.ServiceAccountID != nil {
			serviceAccountName := *input.ServiceAccountID
			serviceAccountID, ok := serviceAccountMap[serviceAccountName]
			if !ok {
				return nil,
					fmt.Errorf("Failed to replace service account name with service account ID: %s", serviceAccountName)
			}
			input.ServiceAccountID = ptr.String(serviceAccountID)
		}

		created, err := testClient.client.ActivityEvents.CreateActivityEvent(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

//////////////////////////////////////////////////////////////////////////////

// Other utility function(s):

// fullPath2ParentPath returns the parent path of the specified full path.
func fullPath2ParentPath(fullPath string) string {

	if strings.Contains(fullPath, "/") {
		// For a nested group, remove the last slash-separated segment.
		segments := strings.Split(fullPath, "/")
		return strings.Join(segments[:len(segments)-1], "/")
	}

	// For a top-level group, the parent path is the empty string.
	return ""
}

// fullPath2Name returns the name of the specified full path.
func fullPath2Name(fullPath string) string {

	if strings.Contains(fullPath, "/") {
		// For a nested group, remove the last slash-separated segment.
		segments := strings.Split(fullPath, "/")
		return segments[len(segments)-1]
	}

	// For a top-level group, the name is the full path.
	return fullPath
}

// reverseStringSlice returns a new []string in the reverse order of the original.
// The original is not modified.
// This is not optimized for speed.
// It is used by multiple test modules.
func reverseStringSlice(input []string) []string {
	result := []string{}

	for i := len(input) - 1; i >= 0; i-- {
		result = append(result, input[i])
	}

	return result
}

// Compare one actual time vs. an expect interval.
// Use the negative sense, because we want >= and <=, while time gives us > and <.
func compareTime(t *testing.T, expectedLow, expectedHigh, actual *time.Time) {
	assert.False(t, actual.Before(*expectedLow))
	assert.False(t, actual.After(*expectedHigh))
}

// Check whether any actual error contains what was expected.
// If an error was expected but did not occur, the test is terminated, and
// any subsequent test cases that should have run will not be attempted.
func checkError(t *testing.T, expectedMsg *string, actualError error) {
	if expectedMsg == nil {
		assert.Nil(t, actualError)
	} else {
		// Uses require rather than assert to avoid a nil pointer dereference.
		require.NotNil(t, actualError)
		assert.Contains(t, actualError.Error(), *expectedMsg)
	}
}

// The End.
