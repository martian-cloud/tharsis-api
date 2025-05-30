// Package db package
package db

//go:generate go tool mockery --name ActivityEvents --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// ActivityEvents encapsulates the logic to access activity events from the database
type ActivityEvents interface {
	GetActivityEvents(ctx context.Context, input *GetActivityEventsInput) (*ActivityEventsResult, error)
	CreateActivityEvent(ctx context.Context, input *models.ActivityEvent) (*models.ActivityEvent, error)
}

// ActivityEventSortableField represents the fields that an activity event can be sorted by
type ActivityEventSortableField string

// ActivityEventSortableField constants
const (
	ActivityEventSortableFieldCreatedAtAsc      ActivityEventSortableField = "CREATED_ASC"
	ActivityEventSortableFieldCreatedAtDesc     ActivityEventSortableField = "CREATED_DESC"
	ActivityEventSortableFieldNamespacePathAsc  ActivityEventSortableField = "NAMESPACE_PATH_ASC"
	ActivityEventSortableFieldNamespacePathDesc ActivityEventSortableField = "NAMESPACE_PATH_DESC"
	ActivityEventSortableFieldActionAsc         ActivityEventSortableField = "ACTION_ASC"
	ActivityEventSortableFieldActionDesc        ActivityEventSortableField = "ACTION_DESC"
)

func (sf ActivityEventSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case ActivityEventSortableFieldCreatedAtAsc, ActivityEventSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "activity_events", Col: "created_at"}
	case ActivityEventSortableFieldNamespacePathAsc, ActivityEventSortableFieldNamespacePathDesc:
		return &pagination.FieldDescriptor{Key: "namespace_path", Table: "namespaces", Col: "path"}
	case ActivityEventSortableFieldActionAsc, ActivityEventSortableFieldActionDesc:
		return &pagination.FieldDescriptor{Key: "action", Table: "activity_events", Col: "action"}
	default:
		return nil
	}
}

func (sf ActivityEventSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// ActivityEventNamespaceMembershipRequirement specifies the namespace requirements for returning
// activity events
type ActivityEventNamespaceMembershipRequirement struct {
	UserID           *string
	ServiceAccountID *string
}

// ActivityEventFilter contains the supported fields for filtering activity event resources
type ActivityEventFilter struct {
	TimeRangeEnd                   *time.Time
	UserID                         *string
	ServiceAccountID               *string
	NamespacePath                  *string
	TimeRangeStart                 *time.Time
	NamespaceMembershipRequirement *ActivityEventNamespaceMembershipRequirement
	ActivityEventIDs               []string
	Actions                        []models.ActivityEventAction
	TargetTypes                    []models.ActivityEventTargetType
	IncludeNested                  bool
}

// GetActivityEventsInput is the input for listing activity events
type GetActivityEventsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *ActivityEventSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter contains the supported fields for filtering ActivityEvent resources
	Filter *ActivityEventFilter
}

// ActivityEventsResult contains the response data and page information
type ActivityEventsResult struct {
	PageInfo       *pagination.PageInfo
	ActivityEvents []models.ActivityEvent
}

type activityEvents struct {
	dbClient *Client
}

// Where needed, namespaces.path will be added.
var activityEventFieldList = append(metadataFieldList,
	"user_id",
	"service_account_id",
	"action",
	"target_type",
	"gpg_key_target_id",
	"group_target_id",
	"managed_identity_target_id",
	"managed_identity_rule_target_id",
	"namespace_membership_target_id",
	"run_target_id",
	"service_account_target_id",
	"state_version_target_id",
	"team_target_id",
	"terraform_provider_target_id",
	"terraform_provider_version_target_id",
	"terraform_module_target_id",
	"terraform_module_version_target_id",
	"variable_target_id",
	"workspace_target_id",
	"payload",
	"vcs_provider_target_id",
	"role_target_id",
	"runner_target_id",
	"terraform_provider_version_mirror_target_id",
	"federated_registry_target_id",
)

// NewActivityEvents returns an instance of the ActivityEvents interface
func NewActivityEvents(dbClient *Client) ActivityEvents {
	return &activityEvents{dbClient: dbClient}
}

func (m *activityEvents) GetActivityEvents(ctx context.Context,
	input *GetActivityEventsInput,
) (*ActivityEventsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetActivityEvents")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()
	if input.Filter != nil {
		if input.Filter.ActivityEventIDs != nil {
			ex = ex.Append(goqu.I("activity_events.id").In(input.Filter.ActivityEventIDs))
		}
		if input.Filter.UserID != nil {
			ex = ex.Append(goqu.I("activity_events.user_id").Eq(input.Filter.UserID))
		}
		if input.Filter.ServiceAccountID != nil {
			ex = ex.Append(goqu.I("activity_events.service_account_id").Eq(input.Filter.ServiceAccountID))
		}
		if input.Filter.NamespacePath != nil {
			if input.Filter.IncludeNested {
				// Return activity events connected directly to the specified namespace
				// _OR_ to any namespace in/under the specified namespace.
				orex := goqu.Or()
				// Add both plain path and with slash anything else.
				orex = orex.Append(goqu.I("namespaces.path").Eq(input.Filter.NamespacePath),
					goqu.I("namespaces.path").Like(*input.Filter.NamespacePath+"/%"))
				ex = ex.Append(orex)
			} else {
				// Return only activity events connected directly to a specified namespace.
				ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePath))
			}
		}
		if input.Filter.TimeRangeStart != nil {
			// Must use UTC here otherwise, queries will return unexpected results.
			ex = ex.Append(goqu.I("activity_events.created_at").Gte(input.Filter.TimeRangeStart.UTC()))
		}
		if input.Filter.TimeRangeEnd != nil {
			// Must use UTC here otherwise, queries will return unexpected results.
			ex = ex.Append(goqu.I("activity_events.created_at").Lte(input.Filter.TimeRangeEnd.UTC()))
		}
		if input.Filter.Actions != nil {
			ex = ex.Append(goqu.I("activity_events.action").In(input.Filter.Actions))
		}
		if input.Filter.TargetTypes != nil {
			ex = ex.Append(goqu.I("activity_events.target_type").In(input.Filter.TargetTypes))
		}

		// This filters out any activity events related to any namespace to which a user or
		//  service account may have LOST membership after the activity events were created.
		if input.Filter.NamespaceMembershipRequirement != nil {
			ex = ex.Append(namespaceMembershipExpressionBuilder{
				userID:           input.Filter.NamespaceMembershipRequirement.UserID,
				serviceAccountID: input.Filter.NamespaceMembershipRequirement.ServiceAccountID,
			}.build())
		}
	}

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	// Do a join with the namespaces table in order to get the namespace path rather than just the ID.
	// Use a left join in order to get all the activity events, even those with no path.
	query := dialect.From("activity_events").
		LeftJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"activity_events.namespace_id": goqu.I("namespaces.id")})).
		Select(m.getSelectFields(true)...).
		Where(ex)

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "activity_events", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, m.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.ActivityEvent{}
	for rows.Next() {
		item, err := scanActivityEvent(rows, true)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan rows")
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		tracing.RecordError(span, err, "failed to finalize rows")
		return nil, err
	}

	result := ActivityEventsResult{
		PageInfo:       rows.GetPageInfo(),
		ActivityEvents: results,
	}

	return &result, nil
}

func (m *activityEvents) CreateActivityEvent(ctx context.Context, input *models.ActivityEvent) (*models.ActivityEvent, error) {
	ctx, span := tracer.Start(ctx, "db.CreateActivityEvent")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	var namespaceID *string
	if input.NamespacePath != nil {

		namespace, err := getNamespaceByPath(ctx, m.dbClient.getConnection(ctx), *input.NamespacePath)
		if err != nil {
			tracing.RecordError(span, err, "failed to get namespace by path")
			return nil, err
		}

		if namespace == nil {
			tracing.RecordError(span, nil, "Namespace not found")
			return nil, errors.New("Namespace not found", errors.WithErrorCode(errors.ENotFound))
		}

		namespaceID = &namespace.id
	}

	// Must use target type to fan out target ID to the various columns.
	var (
		gpgKeyTargetID                         *string
		groupTargetID                          *string
		managedIdentityTargetID                *string
		managedIdentityRuleTargetID            *string
		namespaceMembershipTargetID            *string
		runTargetID                            *string
		serviceAccountTargetID                 *string
		stateVersionTargetID                   *string
		teamTargetID                           *string
		terraformProviderTargetID              *string
		terraformProviderVersionTargetID       *string
		terraformModuleTargetID                *string
		terraformModuleVersionTargetID         *string
		variableTargetID                       *string
		workspaceTargetID                      *string
		vcsProviderTargetID                    *string
		roleTargetID                           *string
		runnerTargetID                         *string
		terraformProviderVersionMirrorTargetID *string
		federatedRegistryTargetID              *string
	)

	switch input.TargetType {
	case models.TargetGPGKey:
		gpgKeyTargetID = &input.TargetID
	case models.TargetGroup:
		groupTargetID = &input.TargetID
	case models.TargetManagedIdentity:
		managedIdentityTargetID = &input.TargetID
	case models.TargetManagedIdentityAccessRule:
		managedIdentityRuleTargetID = &input.TargetID
	case models.TargetNamespaceMembership:
		namespaceMembershipTargetID = &input.TargetID
	case models.TargetRun:
		runTargetID = &input.TargetID
	case models.TargetServiceAccount:
		serviceAccountTargetID = &input.TargetID
	case models.TargetStateVersion:
		stateVersionTargetID = &input.TargetID
	case models.TargetTeam:
		teamTargetID = &input.TargetID
	case models.TargetTerraformProvider:
		terraformProviderTargetID = &input.TargetID
	case models.TargetTerraformProviderVersion:
		terraformProviderVersionTargetID = &input.TargetID
	case models.TargetTerraformModule:
		terraformModuleTargetID = &input.TargetID
	case models.TargetTerraformModuleVersion:
		terraformModuleVersionTargetID = &input.TargetID
	case models.TargetVariable:
		variableTargetID = &input.TargetID
	case models.TargetWorkspace:
		workspaceTargetID = &input.TargetID
	case models.TargetVCSProvider:
		vcsProviderTargetID = &input.TargetID
	case models.TargetRole:
		roleTargetID = &input.TargetID
	case models.TargetRunner:
		runnerTargetID = &input.TargetID
	case models.TargetTerraformProviderVersionMirror:
		terraformProviderVersionMirrorTargetID = &input.TargetID
	case models.TargetFederatedRegistry:
		federatedRegistryTargetID = &input.TargetID
	default:
		// theoretically cannot happen, but in case of a rainy day
		tracing.RecordError(span, nil, "invalid target type: %s", input.TargetType)
		return nil, fmt.Errorf("invalid target type: %s", input.TargetType)
	}

	var payload interface{}
	if input.Payload != nil {
		payload = input.Payload
	}

	timestamp := currentTime()
	record := goqu.Record{
		"id":                                   newResourceID(),
		"version":                              initialResourceVersion,
		"created_at":                           timestamp,
		"updated_at":                           timestamp,
		"user_id":                              input.UserID,
		"service_account_id":                   input.ServiceAccountID,
		"namespace_id":                         namespaceID,
		"action":                               input.Action,
		"target_type":                          input.TargetType,
		"gpg_key_target_id":                    gpgKeyTargetID,
		"group_target_id":                      groupTargetID,
		"managed_identity_target_id":           managedIdentityTargetID,
		"managed_identity_rule_target_id":      managedIdentityRuleTargetID,
		"namespace_membership_target_id":       namespaceMembershipTargetID,
		"run_target_id":                        runTargetID,
		"service_account_target_id":            serviceAccountTargetID,
		"state_version_target_id":              stateVersionTargetID,
		"team_target_id":                       teamTargetID,
		"terraform_provider_target_id":         terraformProviderTargetID,
		"terraform_provider_version_target_id": terraformProviderVersionTargetID,
		"terraform_module_target_id":           terraformModuleTargetID,
		"terraform_module_version_target_id":   terraformModuleVersionTargetID,
		"variable_target_id":                   variableTargetID,
		"workspace_target_id":                  workspaceTargetID,
		"payload":                              payload,
		"vcs_provider_target_id":               vcsProviderTargetID,
		"role_target_id":                       roleTargetID,
		"runner_target_id":                     runnerTargetID,
		"terraform_provider_version_mirror_target_id": terraformProviderVersionMirrorTargetID,
		"federated_registry_target_id":                federatedRegistryTargetID,
	}

	sql, args, err := dialect.Insert("activity_events").
		Prepared(true).
		Rows(record).
		Returning(m.getSelectFields(false)...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to insert to table")
		return nil, err
	}

	createdActivityEvent, err := scanActivityEvent(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_activity_events_user_id":
					tracing.RecordError(span, nil, "user does not exist")
					return nil, errors.New("user does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_service_account_id":
					tracing.RecordError(span, nil, "service account does not exist")
					return nil, errors.New("service account does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_namespace_id":
					tracing.RecordError(span, nil, "namespace path does not exist")
					return nil, errors.New("namespace path does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_gpg_key_target_id":
					tracing.RecordError(span, nil, "GPG key does not exist")
					return nil, errors.New("GPG key does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_group_target_id":
					tracing.RecordError(span, nil, "group does not exist")
					return nil, errors.New("group does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_managed_identity_target_id":
					tracing.RecordError(span, nil, "managed identity does not exist")
					return nil, errors.New("managed identity does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_managed_identity_rule_target_id":
					tracing.RecordError(span, nil, "managed identity access rule does not exist")
					return nil, errors.New("managed identity access rule does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_namespace_membership_target_id":
					tracing.RecordError(span, nil, "namespace membership does not exist")
					return nil, errors.New("namespace membership does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_run_target_id":
					tracing.RecordError(span, nil, "run does not exist")
					return nil, errors.New("run does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_service_account_target_id":
					tracing.RecordError(span, nil, "service account does not exist")
					return nil, errors.New("service account does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_state_version_target_id":
					tracing.RecordError(span, nil, "state version does not exist")
					return nil, errors.New("state version does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_team_target_id":
					tracing.RecordError(span, nil, "team does not exist")
					return nil, errors.New("team does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_terraform_provider_target_id":
					tracing.RecordError(span, nil, "terraform provider does not exist")
					return nil, errors.New("terraform provider does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_terraform_provider_version_target_id":
					tracing.RecordError(span, nil, "terraform provider version does not exist")
					return nil, errors.New("terraform provider version does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_terraform_module_target_id":
					tracing.RecordError(span, nil, "terraform module does not exist")
					return nil, errors.New("terraform module does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_terraform_module_version_target_id":
					tracing.RecordError(span, nil, "terraform module version does not exist")
					return nil, errors.New("terraform module version does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_variable_target_id":
					tracing.RecordError(span, nil, "variable does not exist")
					return nil, errors.New("variable does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_workspace_target_id":
					tracing.RecordError(span, nil, "workspace does not exist")
					return nil, errors.New("workspace does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_vcs_providers_target_id":
					tracing.RecordError(span, nil, "vcs provider does not exist")
					return nil, errors.New("vcs provider does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_role_target_id":
					tracing.RecordError(span, nil, "role does not exist")
					return nil, errors.New("role does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_runner_target_id":
					tracing.RecordError(span, nil, "runner does not exist")
					return nil, errors.New("runner does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_terraform_provider_version_mirror_target_id":
					tracing.RecordError(span, nil, "terraform provider version mirror does not exist")
					return nil, errors.New("terraform provider version mirror does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_activity_events_federated_registry_target_id":
					return nil, errors.New("federated registry does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				}
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	// Set the namespace path
	createdActivityEvent.NamespacePath = input.NamespacePath

	return createdActivityEvent, nil
}

func (m *activityEvents) getSelectFields(addOtherTables bool) []interface{} {
	selectFields := []interface{}{}

	for _, field := range activityEventFieldList {
		selectFields = append(selectFields, fmt.Sprintf("activity_events.%s", field))
	}

	if addOtherTables {
		selectFields = append(selectFields, "namespaces.path")
	}

	return selectFields
}

func scanActivityEvent(row scanner, withOtherTables bool) (*models.ActivityEvent, error) {
	activityEvent := &models.ActivityEvent{}

	// Must use target type to fan out target ID to the various columns.
	var (
		gpgKeyTargetID                         *string
		groupTargetID                          *string
		managedIdentityTargetID                *string
		managedIdentityRuleTargetID            *string
		namespaceMembershipTargetID            *string
		runTargetID                            *string
		serviceAccountTargetID                 *string
		stateVersionTargetID                   *string
		teamTargetID                           *string
		terraformProviderTargetID              *string
		terraformProviderVersionTargetID       *string
		terraformModuleTargetID                *string
		terraformModuleVersionTargetID         *string
		variableTargetID                       *string
		workspaceTargetID                      *string
		vcsProviderTargetID                    *string
		roleTargetID                           *string
		runnerTargetID                         *string
		terraformProviderVersionMirrorTargetID *string
		federatedRegistryTargetID              *string
	)

	fields := []interface{}{
		&activityEvent.Metadata.ID,
		&activityEvent.Metadata.CreationTimestamp,
		&activityEvent.Metadata.LastUpdatedTimestamp,
		&activityEvent.Metadata.Version,
		&activityEvent.UserID,
		&activityEvent.ServiceAccountID,
		&activityEvent.Action,
		&activityEvent.TargetType,
		&gpgKeyTargetID,
		&groupTargetID,
		&managedIdentityTargetID,
		&managedIdentityRuleTargetID,
		&namespaceMembershipTargetID,
		&runTargetID,
		&serviceAccountTargetID,
		&stateVersionTargetID,
		&teamTargetID,
		&terraformProviderTargetID,
		&terraformProviderVersionTargetID,
		&terraformModuleTargetID,
		&terraformModuleVersionTargetID,
		&variableTargetID,
		&workspaceTargetID,
		&activityEvent.Payload,
		&vcsProviderTargetID,
		&roleTargetID,
		&runnerTargetID,
		&terraformProviderVersionMirrorTargetID,
		&federatedRegistryTargetID,
	}

	// Balance the number of selected fields and fields to scan out.
	if withOtherTables {
		fields = append(fields, &activityEvent.NamespacePath)
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	activityEvent.Metadata.TRN = types.ActivityEventModelType.BuildTRN(activityEvent.GetGlobalID())

	switch activityEvent.TargetType {
	case models.TargetGPGKey:
		activityEvent.TargetID = *gpgKeyTargetID
	case models.TargetGroup:
		activityEvent.TargetID = *groupTargetID
	case models.TargetManagedIdentity:
		activityEvent.TargetID = *managedIdentityTargetID
	case models.TargetManagedIdentityAccessRule:
		activityEvent.TargetID = *managedIdentityRuleTargetID
	case models.TargetNamespaceMembership:
		activityEvent.TargetID = *namespaceMembershipTargetID
	case models.TargetRun:
		activityEvent.TargetID = *runTargetID
	case models.TargetServiceAccount:
		activityEvent.TargetID = *serviceAccountTargetID
	case models.TargetStateVersion:
		activityEvent.TargetID = *stateVersionTargetID
	case models.TargetTeam:
		activityEvent.TargetID = *teamTargetID
	case models.TargetTerraformProvider:
		activityEvent.TargetID = *terraformProviderTargetID
	case models.TargetTerraformProviderVersion:
		activityEvent.TargetID = *terraformProviderVersionTargetID
	case models.TargetTerraformModule:
		activityEvent.TargetID = *terraformModuleTargetID
	case models.TargetTerraformModuleVersion:
		activityEvent.TargetID = *terraformModuleVersionTargetID
	case models.TargetVariable:
		activityEvent.TargetID = *variableTargetID
	case models.TargetWorkspace:
		activityEvent.TargetID = *workspaceTargetID
	case models.TargetVCSProvider:
		activityEvent.TargetID = *vcsProviderTargetID
	case models.TargetRole:
		activityEvent.TargetID = *roleTargetID
	case models.TargetRunner:
		activityEvent.TargetID = *runnerTargetID
	case models.TargetTerraformProviderVersionMirror:
		activityEvent.TargetID = *terraformProviderVersionMirrorTargetID
	case models.TargetFederatedRegistry:
		activityEvent.TargetID = *federatedRegistryTargetID
	default:
		// theoretically cannot happen, but in case of a rainy day
		return nil, fmt.Errorf("invalid target type: %s", activityEvent.TargetType)
	}

	return activityEvent, nil
}
