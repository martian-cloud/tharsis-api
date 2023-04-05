// Package variable package
package variable

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
)

// GetVariablesInput is the input for querying a list of variables
type GetVariablesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.VariableSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *db.PaginationOptions
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
	activityService activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	activityService activityevent.Service,
) Service {
	return &service{
		logger:          logger,
		dbClient:        dbClient,
		activityService: activityService,
	}
}

func (s *service) GetVariables(ctx context.Context, namespacePath string) ([]models.Variable, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Only include variable values if the caller has ViewVariableValuePermission on workspace.
	hasViewVariableValuePerm := false
	if err = caller.RequirePermission(ctx, permissions.ViewVariableValuePermission, auth.WithNamespacePath(namespacePath)); err == nil {
		hasViewVariableValuePerm = true
	} else if err = caller.RequirePermission(ctx, permissions.ViewVariablePermission, auth.WithNamespacePath(namespacePath)); err != nil {
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
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	variable, err := s.dbClient.Variables.GetVariableByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if variable == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Variable with id %s not found", id))
	}

	err = caller.RequirePermission(ctx, permissions.ViewVariableValuePermission, auth.WithNamespacePath(variable.NamespacePath))
	if err != nil {
		return nil, err
	}

	return variable, nil
}

func (s *service) GetVariablesByIDs(ctx context.Context, ids []string) ([]models.Variable, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get variables from DB.
	resp, err := s.dbClient.Variables.GetVariables(ctx, &db.GetVariablesInput{
		Filter: &db.VariableFilter{
			VariableIDs: ids,
		},
	})
	if err != nil {
		return nil, err
	}

	namespacePaths := []string{}
	for _, variable := range resp.Variables {
		namespacePaths = append(namespacePaths, variable.NamespacePath)
	}

	if len(namespacePaths) > 0 {
		err = caller.RequirePermission(ctx, permissions.ViewVariableValuePermission, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			return nil, err
		}
	}

	return resp.Variables, nil
}

func (s *service) SetVariables(ctx context.Context, input *SetVariablesInput) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.CreateVariablePermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		return err
	}

	// Start transaction
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for SetVariables: %v", txErr)
		}
	}()

	// Delete existing namespace variables
	if dErr := s.dbClient.Variables.DeleteVariables(txContext, input.NamespacePath, input.Category); dErr != nil {
		return dErr
	}

	for _, v := range input.Variables {
		if input.Category != v.Category {
			return errors.NewError(errors.EInternal, "variable category does not match input")
		}

		if input.NamespacePath != v.NamespacePath {
			return errors.NewError(errors.EInternal, "variable namespace path does not match input")
		}

		if input.Category == models.EnvironmentVariableCategory && v.Hcl {
			return errors.NewError(errors.EInvalid, "HCL variables are not supported for the environment category")
		}
	}

	if len(input.Variables) > 0 {
		if cErr := s.dbClient.Variables.CreateVariables(txContext, input.NamespacePath, input.Variables); cErr != nil {
			return cErr
		}
	}

	targetType, targetID, err := s.getTargetTypeID(txContext, input.NamespacePath)
	if err != nil {
		return err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &input.NamespacePath,
			Action:        models.ActionSetVariables,
			TargetType:    targetType,
			TargetID:      targetID,
		}); err != nil {
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) CreateVariable(ctx context.Context, input *models.Variable) (*models.Variable, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateVariablePermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		return nil, err
	}

	if input.Category == models.EnvironmentVariableCategory && input.Hcl {
		return nil, errors.NewError(errors.EInvalid, "HCL variables are not supported for the environment category")
	}

	if input.Key == "" {
		return nil, errors.NewError(errors.EInvalid, "Key cannot be empty")
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateVariable: %v", txErr)
		}
	}()

	variable, err := s.dbClient.Variables.CreateVariable(txContext, input)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &input.NamespacePath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetVariable,
			TargetID:      variable.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
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
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateVariablePermission, auth.WithNamespacePath(variable.NamespacePath))
	if err != nil {
		return nil, err
	}

	if variable.Category == models.EnvironmentVariableCategory && variable.Hcl {
		return nil, errors.NewError(errors.EInvalid, "HCL variables are not supported for the environment category")
	}

	if variable.Key == "" {
		return nil, errors.NewError(errors.EInvalid, "Key cannot be empty")
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateVariable: %v", txErr)
		}
	}()

	updatedVariable, err := s.dbClient.Variables.UpdateVariable(txContext, variable)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &updatedVariable.NamespacePath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetVariable,
			TargetID:      updatedVariable.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
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
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteVariablePermission, auth.WithNamespacePath(variable.NamespacePath))
	if err != nil {
		return err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
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
		return err
	}

	targetType, targetID, err := s.getTargetTypeID(txContext, variable.NamespacePath)
	if err != nil {
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
