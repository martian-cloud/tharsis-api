package variable

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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
	SetVariables(ctx context.Context, input *SetVariablesInput) error
	CreateVariable(ctx context.Context, input *models.Variable) (*models.Variable, error)
	UpdateVariable(ctx context.Context, variable *models.Variable) (*models.Variable, error)
	DeleteVariable(ctx context.Context, variable *models.Variable) error
}

type service struct {
	logger   logger.Logger
	dbClient *db.Client
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
) Service {
	return &service{
		logger:   logger,
		dbClient: dbClient,
	}
}

func (s *service) GetVariables(ctx context.Context, namespacePath string) ([]models.Variable, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Only include variable values if the caller has the deployer role on the namespace
	hasDeployerRole := false
	if err = caller.RequireAccessToNamespace(ctx, namespacePath, models.DeployerRole); err == nil {
		hasDeployerRole = true
	} else if err = caller.RequireAccessToNamespace(ctx, namespacePath, models.ViewerRole); err != nil {
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

	seen := map[string]bool{}
	for _, v := range result.Variables {
		varCopy := v
		// Clear values if caller does not have deployer role on namespace
		if !hasDeployerRole {
			varCopy.Value = nil
		}

		keyAndCategory := fmt.Sprintf("%s::%s", varCopy.Key, varCopy.Category)
		if _, ok := seen[keyAndCategory]; !ok {
			variables = append(variables, varCopy)

			seen[keyAndCategory] = true
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

	if err := caller.RequireAccessToNamespace(ctx, variable.NamespacePath, models.DeployerRole); err != nil {
		return nil, err
	}

	return variable, nil
}

func (s *service) SetVariables(ctx context.Context, input *SetVariablesInput) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if err = caller.RequireAccessToNamespace(ctx, input.NamespacePath, models.DeployerRole); err != nil {
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
	if err := s.dbClient.Variables.DeleteVariables(txContext, input.NamespacePath, input.Category); err != nil {
		return err
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
		if err := s.dbClient.Variables.CreateVariables(txContext, input.NamespacePath, input.Variables); err != nil {
			return err
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return err
	}

	return nil
}

func (s *service) CreateVariable(ctx context.Context, input *models.Variable) (*models.Variable, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToNamespace(ctx, input.NamespacePath, models.DeployerRole); err != nil {
		return nil, err
	}

	if input.Category == models.EnvironmentVariableCategory && input.Hcl {
		return nil, errors.NewError(errors.EInvalid, "HCL variables are not supported for the environment category")
	}

	if input.Key == "" {
		return nil, errors.NewError(errors.EInvalid, "Key cannot be empty")
	}

	variable, err := s.dbClient.Variables.CreateVariable(ctx, input)
	if err != nil {
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

	if err = caller.RequireAccessToNamespace(ctx, variable.NamespacePath, models.DeployerRole); err != nil {
		return nil, err
	}

	if variable.Category == models.EnvironmentVariableCategory && variable.Hcl {
		return nil, errors.NewError(errors.EInvalid, "HCL variables are not supported for the environment category")
	}

	if variable.Key == "" {
		return nil, errors.NewError(errors.EInvalid, "Key cannot be empty")
	}

	updatedVariable, err := s.dbClient.Variables.UpdateVariable(ctx, variable)
	if err != nil {
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

	if err := caller.RequireAccessToNamespace(ctx, variable.NamespacePath, models.DeployerRole); err != nil {
		return err
	}

	s.logger.Infow("Requested deletion of a variable.",
		"caller", caller.GetSubject(),
		"namespacePath", variable.NamespacePath,
		"variableID", variable.Metadata.ID,
	)
	return s.dbClient.Variables.DeleteVariable(ctx, variable)
}
