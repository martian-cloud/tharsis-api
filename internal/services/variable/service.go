// Package variable package
package variable

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GetVariablesInput is the input for querying a list of variables
type GetVariablesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.VariableSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// NamespacePaths filters the variables by the specified paths
	NamespacePaths []string
	// Include Values
	IncludeValues bool
}

// SetVariablesInput is the input for setting namespace variables
type SetVariablesInput struct {
	NamespacePath string
	Category      models.VariableCategory
	Variables     []models.Variable
}

// Service implements all variable related functionality
type Service interface {
	GetVariables(ctx context.Context, namespacePath string) ([]models.Variable, error)
	GetVariableByID(ctx context.Context, id string) (*models.Variable, error)
	GetVariablesByIDs(ctx context.Context, ids []string) ([]models.Variable, error)
	SetVariables(ctx context.Context, input *SetVariablesInput) error
	CreateVariable(ctx context.Context, input *models.Variable) (*models.Variable, error)
	UpdateVariable(ctx context.Context, variable *models.Variable) (*models.Variable, error)
	DeleteVariable(ctx context.Context, variable *models.Variable) error
}

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	limitChecker    limits.LimitChecker
	activityService activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	activityService activityevent.Service,
) Service {
	return &service{
		logger:          logger,
		dbClient:        dbClient,
		limitChecker:    limitChecker,
		activityService: activityService,
	}
}

func (s *service) GetVariables(ctx context.Context, namespacePath string) ([]models.Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVariables")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Only include variable values if the caller has ViewVariableValuePermission on workspace.
	hasViewVariableValuePerm := false
	if err = caller.RequirePermission(ctx, permissions.ViewVariableValuePermission, auth.WithNamespacePath(namespacePath)); err == nil {
		hasViewVariableValuePerm = true
	} else if err = caller.RequirePermission(ctx, permissions.ViewVariablePermission, auth.WithNamespacePath(namespacePath)); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	pathParts := strings.Split(namespacePath, "/")

	namespacePaths := []string{}
	for len(pathParts) > 0 {
		// Add parent namespace
		namespacePaths = append(namespacePaths, strings.Join(pathParts, "/"))
		// Remove last element
		pathParts = pathParts[:len(pathParts)-1]
	}

	sortBy := db.VariableSortableFieldNamespacePathDesc
	dbInput := &db.GetVariablesInput{
		Sort: &sortBy,
		Filter: &db.VariableFilter{
			NamespacePaths: namespacePaths,
		},
	}

	result, err := s.dbClient.Variables.GetVariables(ctx, dbInput)
	if err != nil {
		tracing.RecordError(span, err, "failed to get variables")
		return nil, err
	}

	variables := []models.Variable{}

	seen := map[string]struct{}{}
	for _, v := range result.Variables {
		varCopy := v
		// Clear values if caller can't view values for namespace.
		if !hasViewVariableValuePerm {
			varCopy.Value = nil
		}

		keyAndCategory := fmt.Sprintf("%s::%s", varCopy.Key, varCopy.Category)
		if _, ok := seen[keyAndCategory]; !ok {
			variables = append(variables, varCopy)
			seen[keyAndCategory] = struct{}{}
		}
	}

	// Sort variable list
	sort.Slice(variables, func(i, j int) bool {
		v := strings.Compare(variables[j].NamespacePath, variables[i].NamespacePath)
		if v == 0 {
			return strings.Compare(variables[i].Key, variables[j].Key) < 0
		}
		return v < 0
	})

	return variables, nil
}

func (s *service) GetVariableByID(ctx context.Context, id string) (*models.Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVariableByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	variable, err := s.dbClient.Variables.GetVariableByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get variable by ID")
		return nil, err
	}

	if variable == nil {
		tracing.RecordError(span, nil, "variable with id %s not found", id)
		return nil, errors.New("variable with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, permissions.ViewVariableValuePermission, auth.WithNamespacePath(variable.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return variable, nil
}

func (s *service) GetVariablesByIDs(ctx context.Context, ids []string) ([]models.Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVariablesByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get variables from DB.
	resp, err := s.dbClient.Variables.GetVariables(ctx, &db.GetVariablesInput{
		Filter: &db.VariableFilter{
			VariableIDs: ids,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get variables")
		return nil, err
	}

	namespacePaths := []string{}
	for _, variable := range resp.Variables {
		namespacePaths = append(namespacePaths, variable.NamespacePath)
	}

	if len(namespacePaths) > 0 {
		err = caller.RequirePermission(ctx, permissions.ViewVariableValuePermission, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	}

	return resp.Variables, nil
}

func (s *service) SetVariables(ctx context.Context, input *SetVariablesInput) error {
	ctx, span := tracer.Start(ctx, "svc.SetVariables")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.CreateVariablePermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	// Start transaction
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for SetVariables: %v", txErr)
		}
	}()

	// Delete existing namespace variables
	if dErr := s.dbClient.Variables.DeleteVariables(txContext, input.NamespacePath, input.Category); dErr != nil {
		tracing.RecordError(span, dErr, "failed to begin DB transaction")
		return dErr
	}

	for _, v := range input.Variables {
		if input.Category != v.Category {
			tracing.RecordError(span, nil, "variable category does not match input")
			return errors.New("variable category does not match input")
		}

		if input.NamespacePath != v.NamespacePath {
			tracing.RecordError(span, nil, "variable namespace path does not match input")
			return errors.New("variable namespace path does not match input")
		}

		if input.Category == models.EnvironmentVariableCategory && v.Hcl {
			tracing.RecordError(span, nil, "HCL variables are not supported for the environment category")
			return errors.New("HCL variables are not supported for the environment category", errors.WithErrorCode(errors.EInvalid))
		}
	}

	if len(input.Variables) > 0 {
		if cErr := s.dbClient.Variables.CreateVariables(txContext, input.NamespacePath, input.Variables); cErr != nil {
			tracing.RecordError(span, cErr, "failed to create variables")
			return cErr
		}
	}

	targetType, targetID, err := s.getTargetTypeID(txContext, input.NamespacePath)
	if err != nil {
		tracing.RecordError(span, err, "failed to get target type by ID")
		return err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &input.NamespacePath,
			Action:        models.ActionSetVariables,
			TargetType:    targetType,
			TargetID:      targetID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) CreateVariable(ctx context.Context, input *models.Variable) (*models.Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateVariable")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateVariablePermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	if input.Category == models.EnvironmentVariableCategory && input.Hcl {
		tracing.RecordError(span, nil, "failed to commit DB transaction")
		return nil, errors.New("HCL variables are not supported for the environment category", errors.WithErrorCode(errors.EInvalid))
	}

	if input.Key == "" {
		tracing.RecordError(span, nil, "Key cannot be empty")
		return nil, errors.New("Key cannot be empty", errors.WithErrorCode(errors.EInvalid))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateVariable: %v", txErr)
		}
	}()

	variable, err := s.dbClient.Variables.CreateVariable(txContext, input)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Get the number of variables in the namespace to check whether we just violated the limit.
	newVariables, err := s.dbClient.Variables.GetVariables(txContext, &db.GetVariablesInput{
		Filter: &db.VariableFilter{
			NamespacePaths: []string{variable.NamespacePath},
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace's variables")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext, limits.ResourceLimitVariablesPerNamespace, newVariables.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &input.NamespacePath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetVariable,
			TargetID:      variable.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Created a new variable.",
		"caller", caller.GetSubject(),
		"namespacePath", input.NamespacePath,
		"variableID", variable.Metadata.ID,
	)

	return variable, nil
}

func (s *service) UpdateVariable(ctx context.Context, variable *models.Variable) (*models.Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateVariable")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateVariablePermission, auth.WithNamespacePath(variable.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	if variable.Category == models.EnvironmentVariableCategory && variable.Hcl {
		tracing.RecordError(span, nil, "HCL variables are not supported for the environment category")
		return nil, errors.New("HCL variables are not supported for the environment category", errors.WithErrorCode(errors.EInvalid))
	}

	if variable.Key == "" {
		tracing.RecordError(span, nil, "Key cannot be empty")
		return nil, errors.New("Key cannot be empty", errors.WithErrorCode(errors.EInvalid))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateVariable: %v", txErr)
		}
	}()

	updatedVariable, err := s.dbClient.Variables.UpdateVariable(txContext, variable)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &updatedVariable.NamespacePath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetVariable,
			TargetID:      updatedVariable.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Updated a variable.",
		"caller", caller.GetSubject(),
		"namespacePath", variable.NamespacePath,
		"variableID", updatedVariable.Metadata.ID,
	)
	return updatedVariable, nil
}

func (s *service) DeleteVariable(ctx context.Context, variable *models.Variable) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteVariable")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteVariablePermission, auth.WithNamespacePath(variable.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteVariable: %v", txErr)
		}
	}()

	s.logger.Infow("Requested deletion of a variable.",
		"caller", caller.GetSubject(),
		"namespacePath", variable.NamespacePath,
		"variableID", variable.Metadata.ID,
	)

	err = s.dbClient.Variables.DeleteVariable(txContext, variable)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	targetType, targetID, err := s.getTargetTypeID(txContext, variable.NamespacePath)
	if err != nil {
		tracing.RecordError(span, err, "failed to get target type ID")
		return err
	}

	// Record a DeleteChildResource activity event whether the variable was group-level or workspace-level.
	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &variable.NamespacePath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    targetType,
			TargetID:      targetID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: variable.Key,
				ID:   variable.Metadata.ID,
				Type: string(models.TargetVariable),
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

// getTargetTypeID returns the target type and the target ID, whether the target is a group or a workspace.
func (s *service) getTargetTypeID(ctx context.Context,
	namespacePath string,
) (models.ActivityEventTargetType, string, error) {
	var targetType models.ActivityEventTargetType
	targetID := ""
	group, gErr := s.dbClient.Groups.GetGroupByFullPath(ctx, namespacePath)
	if (gErr == nil) && (group != nil) {
		targetType = models.TargetGroup
		targetID = group.Metadata.ID
	} else {
		workspace, wErr := s.dbClient.Workspaces.GetWorkspaceByFullPath(ctx, namespacePath)
		if (wErr == nil) && (workspace != nil) {
			targetType = models.TargetWorkspace
			targetID = workspace.Metadata.ID
		}
	}
	if targetID == "" {
		return "", "", fmt.Errorf("failed to find group or workspace ID with path %s", namespacePath)
	}

	return targetType, targetID, nil
}
