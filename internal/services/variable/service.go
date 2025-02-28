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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/secret"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GetVariableVersionsInput is the input for querying a list of variable versions
type GetVariableVersionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.VariableVersionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// VariableID is the ID of the variable to query versions for
	VariableID string
}

// CreateVariableInput is the input for creating a variable
type CreateVariableInput struct {
	Value         string
	Category      models.VariableCategory
	NamespacePath string
	Key           string
	Hcl           bool
	Sensitive     bool
}

// UpdateVariableInput is the input for updating a variable
type UpdateVariableInput struct {
	MetadataVersion *int
	ID              string
	Value           string
	Key             string
	Hcl             bool
}

// DeleteVariableInput is the input for deleting a variable
type DeleteVariableInput struct {
	MetadataVersion *int
	ID              string
}

// SetVariablesInputVariable is an input variable for setting namespace variables
type SetVariablesInputVariable struct {
	Value     string
	Key       string
	Hcl       bool
	Sensitive bool
}

// SetVariablesInput is the input for setting namespace variables
type SetVariablesInput struct {
	NamespacePath string
	Category      models.VariableCategory
	Variables     []*SetVariablesInputVariable
}

// Service implements all variable related functionality
type Service interface {
	GetVariables(ctx context.Context, namespacePath string) ([]models.Variable, error)
	GetVariableByID(ctx context.Context, id string) (*models.Variable, error)
	GetVariableVersionByID(ctx context.Context, id string, includeSensitiveValue bool) (*models.VariableVersion, error)
	GetVariablesByIDs(ctx context.Context, ids []string) ([]models.Variable, error)
	SetVariables(ctx context.Context, input *SetVariablesInput) error
	CreateVariable(ctx context.Context, input *CreateVariableInput) (*models.Variable, error)
	UpdateVariable(ctx context.Context, variable *UpdateVariableInput) (*models.Variable, error)
	DeleteVariable(ctx context.Context, variable *DeleteVariableInput) error
	GetVariableVersions(ctx context.Context, input *GetVariableVersionsInput) (*db.VariableVersionResult, error)
}

type service struct {
	logger                          logger.Logger
	dbClient                        *db.Client
	limitChecker                    limits.LimitChecker
	activityService                 activityevent.Service
	secretManager                   secret.Manager
	disableSensitiveVariableFeature bool
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	activityService activityevent.Service,
	secretManager secret.Manager,
	disableSensitiveVariableFeature bool,
) Service {
	return &service{
		logger:                          logger,
		dbClient:                        dbClient,
		limitChecker:                    limitChecker,
		activityService:                 activityService,
		secretManager:                   secretManager,
		disableSensitiveVariableFeature: disableSensitiveVariableFeature,
	}
}

func (s *service) GetVariableVersionByID(ctx context.Context, id string, includeSensitiveValue bool) (*models.VariableVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVariableVersionByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	version, err := s.dbClient.VariableVersions.GetVariableVersionByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get variable version by ID", errors.WithSpan(span))
	}

	if version == nil {
		return nil, errors.New("variable version with id %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	variable, err := s.getVariableByID(ctx, version.VariableID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get variable by ID", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, permissions.ViewVariableValuePermission, auth.WithNamespacePath(variable.NamespacePath))
	if err != nil {
		return nil, err
	}

	if variable.Sensitive && includeSensitiveValue {
		// Get the secret value from the secret manager plugin
		secret, err := s.secretManager.Get(ctx, version.Key, version.SecretData)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get secret value for variable version %q", version.Metadata.ID, errors.WithSpan(span))
		}
		version.Value = &secret
	}

	return version, nil
}

func (s *service) GetVariableVersions(ctx context.Context, input *GetVariableVersionsInput) (*db.VariableVersionResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVariableVersions")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get variable
	variable, err := s.getVariableByID(ctx, input.VariableID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get variable by ID", errors.WithSpan(span))
	}

	if err = caller.RequirePermission(ctx, permissions.ViewVariableValuePermission, auth.WithNamespacePath(variable.NamespacePath)); err != nil {
		return nil, err
	}

	dbInput := &db.GetVariableVersionsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.VariableVersionFilter{
			VariableID: &input.VariableID,
		},
	}

	result, err := s.dbClient.VariableVersions.GetVariableVersions(ctx, dbInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get variable versions for variable %s", variable.Metadata.ID, errors.WithSpan(span))
	}

	return result, nil
}

func (s *service) GetVariables(ctx context.Context, namespacePath string) ([]models.Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVariables")
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

	variable, err := s.getVariableByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get variable by ID")
		return nil, err
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

	namespacesAllowedToViewValue := map[string]struct{}{}

	for _, namespacePath := range namespacePaths {
		err = caller.RequirePermission(ctx, permissions.ViewVariableValuePermission, auth.WithNamespacePath(namespacePath))
		if err != nil {
			err = caller.RequirePermission(ctx, permissions.ViewVariablePermission, auth.WithNamespacePath(namespacePath))
			if err != nil {
				tracing.RecordError(span, err, "permission check failed")
				return nil, err
			}
		} else {
			namespacesAllowedToViewValue[namespacePath] = struct{}{}
		}
	}

	// Filter out variable values that the caller is not allowed to view
	for i := range resp.Variables {
		if _, ok := namespacesAllowedToViewValue[resp.Variables[i].NamespacePath]; !ok {
			resp.Variables[i].Value = nil
		}
	}

	return resp.Variables, nil
}

func (s *service) SetVariables(ctx context.Context, input *SetVariablesInput) error {
	ctx, span := tracer.Start(ctx, "svc.SetVariables")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.CreateVariablePermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		return err
	}

	// Check if any variables have duplicate keys
	seen := map[string]struct{}{}
	for _, v := range input.Variables {
		if _, ok := seen[v.Key]; ok {
			return errors.New("duplicate variable key found", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}
		seen[v.Key] = struct{}{}

		if input.Category == models.EnvironmentVariableCategory && v.Hcl {
			return errors.New("HCL variables are not supported for the environment category", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}

		if v.Sensitive && v.Value == "" {
			return errors.New("value cannot be empty for sensitive variable", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}

		if v.Sensitive && s.disableSensitiveVariableFeature {
			return errors.New("support for sensitive variables is currently disabled", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}
	}

	// Get existing variables for the specified namespace and category
	existingVariables, err := s.dbClient.Variables.GetVariables(ctx, &db.GetVariablesInput{
		Filter: &db.VariableFilter{
			NamespacePaths: []string{input.NamespacePath},
			Category:       &input.Category,
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get existing variables", errors.WithSpan(span))
	}

	// Create map of existing variables by key
	existingVariablesByKey := map[string]*models.Variable{}
	for _, v := range existingVariables.Variables {
		v := v
		existingVariablesByKey[fmt.Sprintf("%s:%t", v.Key, v.Sensitive)] = &v
	}

	// Filter variables into three groups: to create, to update, and to delete
	variablesToCreate := []*models.Variable{}
	variablesToUpdate := []*models.Variable{}
	variablesToDelete := []*models.Variable{}

	for _, v := range input.Variables {
		v := v

		mapKey := fmt.Sprintf("%s:%t", v.Key, v.Sensitive)
		if existingVariable, ok := existingVariablesByKey[mapKey]; ok {
			var value string

			if v.Sensitive {
				// We need to get the senstivie value to determine if it has changed
				value, err = s.secretManager.Get(ctx, v.Key, existingVariable.SecretData)
				if err != nil {
					return errors.Wrap(err, "failed to get secret for variable %q", v.Key, errors.WithSpan(span))
				}
			} else {
				// Existing value should never be nil since this is not a sensitive variable
				if existingVariable.Value == nil {
					return errors.New("failed to set variables because existing value for variable %q is undefined", existingVariable.Key, errors.WithSpan(span))
				}
				value = *existingVariable.Value
			}

			if v.Value != value || v.Hcl != existingVariable.Hcl {
				if v.Sensitive {
					secretData, err := s.secretManager.Update(ctx, v.Key, existingVariable.SecretData, v.Value)
					if err != nil {
						return errors.Wrap(err, "failed to update secret for variable %q", v.Key, errors.WithSpan(span))
					}
					existingVariable.SecretData = secretData
				} else {
					existingVariable.Value = &v.Value
				}

				existingVariable.Hcl = v.Hcl
				variablesToUpdate = append(variablesToUpdate, existingVariable)
			}

			// Remove from existing variables map
			delete(existingVariablesByKey, mapKey)
		} else {
			variableToCreate := &models.Variable{
				Category:      input.Category,
				NamespacePath: input.NamespacePath,
				Key:           v.Key,
				Hcl:           v.Hcl,
				Sensitive:     v.Sensitive,
			}

			if v.Sensitive {
				secretData, err := s.secretManager.Create(ctx, v.Key, v.Value)
				if err != nil {
					return errors.Wrap(err, "failed to create secret for variable %q", v.Key, errors.WithSpan(span))
				}
				variableToCreate.SecretData = secretData
			} else {
				variableToCreate.Value = &v.Value
			}
			// Create variable
			variablesToCreate = append(variablesToCreate, variableToCreate)
		}
	}

	// Any remaining variables in existingVariablesByKey should be deleted
	for _, v := range existingVariablesByKey {
		variablesToDelete = append(variablesToDelete, v)
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

	// Delete variables not in the input
	for _, v := range variablesToDelete {
		if err = s.dbClient.Variables.DeleteVariable(txContext, v); err != nil {
			return errors.Wrap(err, "failed to delete variable", errors.WithSpan(span))
		}
	}

	// Update existing variables
	for _, v := range variablesToUpdate {
		if _, err = s.dbClient.Variables.UpdateVariable(txContext, v); err != nil {
			return errors.Wrap(err, "failed to update variable", errors.WithSpan(span))
		}
	}

	// Create new variables
	if len(variablesToCreate) > 0 {
		if err = s.dbClient.Variables.CreateVariables(txContext, input.NamespacePath, variablesToCreate); err != nil {
			return errors.Wrap(err, "failed to create variable", errors.WithSpan(span))
		}
	}

	targetType, targetID, err := s.getTargetTypeID(txContext, input.NamespacePath)
	if err != nil {
		return errors.Wrap(err, "failed to get target type ID", errors.WithSpan(span))
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &input.NamespacePath,
			Action:        models.ActionSetVariables,
			TargetType:    targetType,
			TargetID:      targetID,
		}); err != nil {
		return errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) CreateVariable(ctx context.Context, input *CreateVariableInput) (*models.Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateVariable")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateVariablePermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		return nil, err
	}

	if input.Category == models.EnvironmentVariableCategory && input.Hcl {
		return nil, errors.New("HCL variables are not supported for the environment category", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	if input.Key == "" {
		return nil, errors.New("key cannot be empty", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	if input.Sensitive && s.disableSensitiveVariableFeature {
		return nil, errors.New("support for sensitive variables is currently disabled", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	// Check if variable with this key and category already exists in the namespace
	existingVariables, err := s.dbClient.Variables.GetVariables(ctx, &db.GetVariablesInput{
		Filter: &db.VariableFilter{
			NamespacePaths: []string{input.NamespacePath},
			Key:            &input.Key,
			Category:       &input.Category,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to query variables in namespace", errors.WithSpan(span))
	}

	if existingVariables.PageInfo.TotalCount > 0 {
		return nil, errors.New("variable with key %q and category %q already exists in namespace", input.Key, input.Category, errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
	}

	variableToCreate := &models.Variable{
		Category:      input.Category,
		NamespacePath: input.NamespacePath,
		Key:           input.Key,
		Hcl:           input.Hcl,
		Sensitive:     input.Sensitive,
	}

	// If this is a sensitive variable we need to use the secret handler to handle the value.
	if input.Sensitive {
		if input.Value == "" {
			return nil, errors.New("value cannot be empty for sensitive variable", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}
		secretData, err := s.secretManager.Create(ctx, input.Key, input.Value)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create secret for variable %q", input.Key, errors.WithSpan(span))
		}
		variableToCreate.SecretData = secretData
	} else {
		variableToCreate.Value = &input.Value
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin db transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateVariable: %v", txErr)
		}
	}()

	variable, err := s.dbClient.Variables.CreateVariable(txContext, variableToCreate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create variable in db", errors.WithSpan(span))
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
		return nil, errors.Wrap(err, "failed to query variables in namespace", errors.WithSpan(span))
	}
	if err = s.limitChecker.CheckLimit(txContext, limits.ResourceLimitVariablesPerNamespace, newVariables.PageInfo.TotalCount); err != nil {
		return nil, errors.Wrap(err, "failed to check limit for variables per namespace", errors.WithSpan(span))
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
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.Infow("Created a new variable.",
		"caller", caller.GetSubject(),
		"namespacePath", input.NamespacePath,
		"variableID", variable.Metadata.ID,
	)

	return variable, nil
}

func (s *service) UpdateVariable(ctx context.Context, input *UpdateVariableInput) (*models.Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateVariable")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	variable, err := s.getVariableByID(ctx, input.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query variable", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, permissions.UpdateVariablePermission, auth.WithNamespacePath(variable.NamespacePath))
	if err != nil {
		return nil, err
	}

	if variable.Category == models.EnvironmentVariableCategory && input.Hcl {
		tracing.RecordError(span, nil, "HCL variables are not supported for the environment category")
		return nil, errors.New("HCL variables are not supported for the environment category", errors.WithErrorCode(errors.EInvalid))
	}

	if input.Key == "" {
		tracing.RecordError(span, nil, "Key cannot be empty")
		return nil, errors.New("Key cannot be empty", errors.WithErrorCode(errors.EInvalid))
	}

	// Check the metadata version if it's specified
	if input.MetadataVersion != nil && *input.MetadataVersion != variable.Metadata.Version {
		return nil, errors.New("metadata version mismatch", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
	}

	// If key is changed, check if a variable with the new key already exists in the namespace
	if input.Key != variable.Key {
		existingVariables, err := s.dbClient.Variables.GetVariables(ctx, &db.GetVariablesInput{
			Filter: &db.VariableFilter{
				NamespacePaths: []string{variable.NamespacePath},
				Key:            &input.Key,
				Category:       &variable.Category,
			},
			PaginationOptions: &pagination.Options{
				First: ptr.Int32(0),
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to query variables in namespace", errors.WithSpan(span))
		}

		if existingVariables.PageInfo.TotalCount > 0 {
			return nil, errors.New("variable with key %q and category %q already exists in namespace", input.Key, variable.Category, errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
		}
	}

	// Set updated fields
	variable.Key = input.Key
	variable.Hcl = input.Hcl

	if variable.Sensitive {
		if input.Value == "" {
			return nil, errors.New("value cannot be empty for sensitive variable", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}
		// Update the secret value using the secret manager plugin
		newSecretData, err := s.secretManager.Update(ctx, input.Key, variable.SecretData, input.Value)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update secret for variable %q", input.Key, errors.WithSpan(span))
		}
		variable.SecretData = newSecretData
	} else {
		variable.Value = &input.Value
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

func (s *service) DeleteVariable(ctx context.Context, input *DeleteVariableInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteVariable")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	variable, err := s.getVariableByID(ctx, input.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get variable by ID", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, permissions.DeleteVariablePermission, auth.WithNamespacePath(variable.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	// Check the metadata version if it's specified
	if input.MetadataVersion != nil && *input.MetadataVersion != variable.Metadata.Version {
		return errors.New("metadata version mismatch", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
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

func (s *service) getVariableByID(ctx context.Context, variableID string) (*models.Variable, error) {
	variable, err := s.dbClient.Variables.GetVariableByID(ctx, variableID)
	if err != nil {
		return nil, err
	}

	if variable == nil {
		return nil, errors.New("variable with ID %s not found", variableID, errors.WithErrorCode(errors.ENotFound))
	}

	return variable, nil
}
