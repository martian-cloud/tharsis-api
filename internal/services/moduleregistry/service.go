package moduleregistry

//go:generate mockery --name Service --inpackage --case underscore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/go-slug"
	"github.com/hashicorp/go-version"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/semver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
)

// CreateModuleInput is the input for creating a terraform module
type CreateModuleInput struct {
	Name          string
	System        string
	GroupID       string
	RepositoryURL string
	Private       bool
}

// CreateModuleVersionInput is the input for creating a terraform module version
type CreateModuleVersionInput struct {
	SemanticVersion string
	ModuleID        string
	SHASum          []byte
}

// GetModulesInput is the input for getting a list of terraform modules
type GetModulesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.TerraformModuleSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *db.PaginationOptions
	// Group filters modules be the specified group
	Group *models.Group
	// Search filters module list by modules with a name that contains the search query
	Search *string
}

// GetModuleVersionsInput is the input for getting a list of module versions
type GetModuleVersionsInput struct {
	Sort              *db.TerraformModuleVersionSortableField
	PaginationOptions *db.PaginationOptions
	Status            *models.TerraformModuleVersionStatus
	SemanticVersion   *string
	Latest            *bool
	ModuleID          string
}

// Service implements all module registry functionality
type Service interface {
	GetModuleByID(ctx context.Context, id string) (*models.TerraformModule, error)
	GetModuleByPath(ctx context.Context, path string) (*models.TerraformModule, error)
	GetModuleByAddress(ctx context.Context, namespace string, name string, system string) (*models.TerraformModule, error)
	GetModulesByIDs(ctx context.Context, ids []string) ([]models.TerraformModule, error)
	GetModules(ctx context.Context, input *GetModulesInput) (*db.ModulesResult, error)
	CreateModule(ctx context.Context, input *CreateModuleInput) (*models.TerraformModule, error)
	UpdateModule(ctx context.Context, module *models.TerraformModule) (*models.TerraformModule, error)
	DeleteModule(ctx context.Context, module *models.TerraformModule) error
	GetModuleVersionByID(ctx context.Context, id string) (*models.TerraformModuleVersion, error)
	GetModuleVersions(ctx context.Context, input *GetModuleVersionsInput) (*db.ModuleVersionsResult, error)
	GetModuleVersionsByIDs(ctx context.Context, ids []string) ([]models.TerraformModuleVersion, error)
	CreateModuleVersion(ctx context.Context, input *CreateModuleVersionInput) (*models.TerraformModuleVersion, error)
	DeleteModuleVersion(ctx context.Context, moduleVersion *models.TerraformModuleVersion) error
	GetModuleConfigurationDetails(ctx context.Context, moduleVersion *models.TerraformModuleVersion, path string) (*ModuleConfigurationDetails, error)
	UploadModuleVersionPackage(ctx context.Context, moduleVersion *models.TerraformModuleVersion, reader io.Reader) error
	GetModuleVersionPackageDownloadURL(ctx context.Context, moduleVersion *models.TerraformModuleVersion) (string, error)
}

type handleCallerFunc func(
	ctx context.Context,
	userHandler func(ctx context.Context, caller *auth.UserCaller) error,
	serviceAccountHandler func(ctx context.Context, caller *auth.ServiceAccountCaller) error,
) error

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	registryStore   RegistryStore
	activityService activityevent.Service
	taskManager     asynctask.Manager
	handleCaller    handleCallerFunc
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	registryStore RegistryStore,
	activityService activityevent.Service,
	taskManager asynctask.Manager,
) Service {
	return newService(
		logger,
		dbClient,
		registryStore,
		activityService,
		taskManager,
		auth.HandleCaller,
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	registryStore RegistryStore,
	activityService activityevent.Service,
	taskManager asynctask.Manager,
	handleCaller handleCallerFunc,
) Service {
	return &service{
		logger,
		dbClient,
		registryStore,
		activityService,
		taskManager,
		handleCaller,
	}
}

func (s *service) GetModuleByID(ctx context.Context, id string) (*models.TerraformModule, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	module, err := s.getModuleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if module.Private {
		if err = caller.RequireAccessToInheritedGroupResource(ctx, module.GroupID); err != nil {
			return nil, err
		}
	}

	return module, nil
}

func (s *service) GetModuleByPath(ctx context.Context, path string) (*models.TerraformModule, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	module, err := s.dbClient.TerraformModules.GetModuleByPath(ctx, path)
	if err != nil {
		return nil, err
	}

	if module == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("module with path %s not found", path))
	}

	if module.Private {
		if err = caller.RequireAccessToInheritedGroupResource(ctx, module.GroupID); err != nil {
			return nil, err
		}
	}

	return module, nil
}

func (s *service) GetModuleByAddress(ctx context.Context, namespace string, name string, system string) (*models.TerraformModule, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	rootGroup, err := s.dbClient.Groups.GetGroupByFullPath(ctx, namespace)
	if err != nil {
		return nil, err
	}

	if rootGroup == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("namespace %s not found", namespace))
	}

	moduleResult, err := s.dbClient.TerraformModules.GetModules(ctx, &db.GetModulesInput{
		PaginationOptions: &db.PaginationOptions{First: ptr.Int32(1)},
		Filter: &db.TerraformModuleFilter{
			RootGroupID: &rootGroup.Metadata.ID,
			Name:        &name,
			System:      &system,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(moduleResult.Modules) == 0 {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("module with name %s and system %s not found in namespace %s", name, system, namespace))
	}

	module := moduleResult.Modules[0]

	if module.Private {
		if err = caller.RequireAccessToInheritedGroupResource(ctx, module.GroupID); err != nil {
			return nil, err
		}
	}

	return &module, nil
}

func (s *service) GetModules(ctx context.Context, input *GetModulesInput) (*db.ModulesResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	dbInput := db.GetModulesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.TerraformModuleFilter{
			Search: input.Search,
		},
	}

	if input.Group != nil {
		if err = caller.RequireAccessToNamespace(ctx, input.Group.FullPath, models.ViewerRole); err != nil {
			return nil, err
		}
		dbInput.Filter.GroupID = &input.Group.Metadata.ID
	} else {
		policy, napErr := caller.GetNamespaceAccessPolicy(ctx)
		if napErr != nil {
			return nil, napErr
		}

		if !policy.AllowAll {
			if err = s.handleCaller(
				ctx,
				func(_ context.Context, c *auth.UserCaller) error {
					dbInput.Filter.UserID = &c.User.Metadata.ID
					return nil
				},
				func(_ context.Context, c *auth.ServiceAccountCaller) error {
					dbInput.Filter.ServiceAccountID = &c.ServiceAccountID
					return nil
				},
			); err != nil {
				return nil, err
			}
		}
	}

	return s.dbClient.TerraformModules.GetModules(ctx, &dbInput)
}

func (s *service) UpdateModule(ctx context.Context, module *models.TerraformModule) (*models.TerraformModule, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToGroup(ctx, module.GroupID, models.DeployerRole); err != nil {
		return nil, err
	}

	if vErr := module.Validate(); vErr != nil {
		return nil, vErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateModule: %v", txErr)
		}
	}()

	updatedModule, err := s.dbClient.TerraformModules.UpdateModule(txContext, module)
	if err != nil {
		return nil, err
	}

	groupPath := updatedModule.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetTerraformModule,
			TargetID:      updatedModule.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return updatedModule, nil
}

func (s *service) CreateModule(ctx context.Context, input *CreateModuleInput) (*models.TerraformModule, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToGroup(ctx, input.GroupID, models.DeployerRole); err != nil {
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	var rootGroupID string
	if group.ParentID == "" {
		rootGroupID = input.GroupID
	} else {
		rootGroup, gErr := s.dbClient.Groups.GetGroupByFullPath(ctx, group.GetRootGroupPath())
		if gErr != nil {
			return nil, gErr
		}
		rootGroupID = rootGroup.Metadata.ID
	}

	moduleToCreate := &models.TerraformModule{
		Name:          input.Name,
		System:        input.System,
		GroupID:       input.GroupID,
		RootGroupID:   rootGroupID,
		Private:       input.Private,
		RepositoryURL: input.RepositoryURL,
		CreatedBy:     caller.GetSubject(),
	}

	if vErr := moduleToCreate.Validate(); vErr != nil {
		return nil, vErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateModule: %v", txErr)
		}
	}()

	createdModule, err := s.dbClient.TerraformModules.CreateModule(txContext, moduleToCreate)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetTerraformModule,
			TargetID:      createdModule.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return createdModule, nil
}

func (s *service) DeleteModule(ctx context.Context, module *models.TerraformModule) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if err = caller.RequireAccessToGroup(ctx, module.GroupID, models.DeployerRole); err != nil {
		return err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteModule: %v", txErr)
		}
	}()

	err = s.dbClient.TerraformModules.DeleteModule(txContext, module)
	if err != nil {
		return err
	}

	groupPath := module.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetGroup,
			TargetID:      module.GroupID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: module.Name,
				ID:   module.Metadata.ID,
				Type: string(models.TargetTerraformModule),
			},
		}); err != nil {
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetModulesByIDs(ctx context.Context, ids []string) ([]models.TerraformModule, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	response, err := s.dbClient.TerraformModules.GetModules(ctx, &db.GetModulesInput{
		Filter: &db.TerraformModuleFilter{
			TerraformModuleIDs: ids,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, module := range response.Modules {
		if module.Private {
			if err := caller.RequireAccessToInheritedGroupResource(ctx, module.GroupID); err != nil {
				return nil, err
			}
		}
	}

	return response.Modules, nil
}

func (s *service) GetModuleConfigurationDetails(ctx context.Context, moduleVersion *models.TerraformModuleVersion, path string) (*ModuleConfigurationDetails, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	module, err := s.getModuleByID(ctx, moduleVersion.ModuleID)
	if err != nil {
		return nil, err
	}

	if module.Private {
		if err = caller.RequireAccessToInheritedGroupResource(ctx, module.GroupID); err != nil {
			return nil, err
		}
	}

	reader, err := s.registryStore.GetModuleConfigurationDetails(ctx, moduleVersion, module, path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var moduleMetadata ModuleConfigurationDetails
	if err := json.NewDecoder(reader).Decode(&moduleMetadata); err != nil {
		return nil, err
	}

	return &moduleMetadata, nil
}

func (s *service) GetModuleVersionByID(ctx context.Context, id string) (*models.TerraformModuleVersion, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	moduleVersion, err := s.getModuleVersionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	module, err := s.getModuleByID(ctx, moduleVersion.ModuleID)
	if err != nil {
		return nil, err
	}

	if module.Private {
		if err = caller.RequireAccessToInheritedGroupResource(ctx, module.GroupID); err != nil {
			return nil, err
		}
	}

	return moduleVersion, nil
}

func (s *service) GetModuleVersions(ctx context.Context, input *GetModuleVersionsInput) (*db.ModuleVersionsResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	module, err := s.getModuleByID(ctx, input.ModuleID)
	if err != nil {
		return nil, err
	}

	if module.Private {
		if err = caller.RequireAccessToInheritedGroupResource(ctx, module.GroupID); err != nil {
			return nil, err
		}
	}

	dbInput := db.GetModuleVersionsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.TerraformModuleVersionFilter{
			ModuleID:        &input.ModuleID,
			Status:          input.Status,
			SemanticVersion: input.SemanticVersion,
			Latest:          input.Latest,
		},
	}

	return s.dbClient.TerraformModuleVersions.GetModuleVersions(ctx, &dbInput)

}

func (s *service) GetModuleVersionsByIDs(ctx context.Context, ids []string) ([]models.TerraformModuleVersion, error) {
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	response, err := s.dbClient.TerraformModuleVersions.GetModuleVersions(ctx, &db.GetModuleVersionsInput{
		Filter: &db.TerraformModuleVersionFilter{
			ModuleVersionIDs: ids,
		},
	})
	if err != nil {
		return nil, err
	}

	moduleIDList := []string{}

	for _, version := range response.ModuleVersions {
		moduleIDList = append(moduleIDList, version.ModuleID)
	}

	// GetModulesByIDs performs the authorization checks to verify that the subject
	// can view the requested modules
	if len(moduleIDList) > 0 {
		if _, err := s.GetModulesByIDs(ctx, moduleIDList); err != nil {
			return nil, err
		}
	}

	return response.ModuleVersions, nil
}

func (s *service) CreateModuleVersion(ctx context.Context, input *CreateModuleVersionInput) (*models.TerraformModuleVersion, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	module, err := s.getModuleByID(ctx, input.ModuleID)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToGroup(ctx, module.GroupID, models.DeployerRole); err != nil {
		return nil, err
	}

	// Verify semantic version is valid
	semVersion, err := version.NewSemver(input.SemanticVersion)
	if err != nil {
		return nil, errors.NewError(errors.EInvalid, fmt.Sprintf("invalid semantic version: %v", err))
	}

	// Check if this version is greater than the previous latest
	versionsResp, err := s.dbClient.TerraformModuleVersions.GetModuleVersions(ctx, &db.GetModuleVersionsInput{
		PaginationOptions: &db.PaginationOptions{
			First: ptr.Int32(1),
		},
		Filter: &db.TerraformModuleVersionFilter{
			ModuleID: &input.ModuleID,
			Latest:   ptr.Bool(true),
		},
	})
	if err != nil {
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for CreateModuleVersion: %v", txErr)
		}
	}()

	isLatest := false
	if len(versionsResp.ModuleVersions) > 0 {
		prevLatest := versionsResp.ModuleVersions[0]
		prevSemVersion, sErr := version.NewSemver(prevLatest.SemanticVersion)
		if sErr != nil {
			return nil, sErr
		}
		if semver.IsSemverGreaterThan(semVersion, prevSemVersion) {
			isLatest = true
			// Remove latest from version
			prevLatest.Latest = false
			if _, uErr := s.dbClient.TerraformModuleVersions.UpdateModuleVersion(txContext, &prevLatest); uErr != nil {
				return nil, uErr
			}
		}
	} else {
		// New version is the latest since it is the only version
		isLatest = true
	}

	moduleVersion, err := s.dbClient.TerraformModuleVersions.CreateModuleVersion(txContext, &models.TerraformModuleVersion{
		ModuleID:        input.ModuleID,
		SemanticVersion: semVersion.String(),
		Latest:          isLatest,
		SHASum:          input.SHASum,
		Status:          models.TerraformModuleVersionStatusPending,
		CreatedBy:       caller.GetSubject(),
	})
	if err != nil {
		return nil, err
	}

	groupPath := module.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetTerraformModuleVersion,
			TargetID:      moduleVersion.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	s.logger.Infow("Created a module version.",
		"caller", caller.GetSubject(),
		"moduleID", input.ModuleID,
		"moduleVersion", moduleVersion.SemanticVersion,
	)

	return moduleVersion, nil
}

func (s *service) DeleteModuleVersion(ctx context.Context, moduleVersion *models.TerraformModuleVersion) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	module, err := s.getModuleByID(ctx, moduleVersion.ModuleID)
	if err != nil {
		return err
	}

	if err = caller.RequireAccessToGroup(ctx, module.GroupID, models.DeployerRole); err != nil {
		return err
	}

	// Reset latest flag if we're deleting the latest version
	var newLatestVersion *models.TerraformModuleVersion
	if moduleVersion.Latest {
		versionsResp, gpErr := s.dbClient.TerraformModuleVersions.GetModuleVersions(ctx, &db.GetModuleVersionsInput{
			Filter: &db.TerraformModuleVersionFilter{
				ModuleID: &module.Metadata.ID,
			},
		})
		if gpErr != nil {
			return err
		}

		for _, v := range versionsResp.ModuleVersions {
			vCopy := v

			// Skip if this is the module version we're deleting
			if v.Metadata.ID == moduleVersion.Metadata.ID {
				continue
			}

			if newLatestVersion == nil {
				newLatestVersion = &vCopy
				continue
			}

			latestSemver, lsErr := version.NewSemver(newLatestVersion.SemanticVersion)
			if lsErr != nil {
				return lsErr
			}

			currentSemver, csErr := version.NewSemver(vCopy.SemanticVersion)
			if csErr != nil {
				return csErr
			}

			if semver.IsSemverGreaterThan(currentSemver, latestSemver) {
				newLatestVersion = &vCopy
			}
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for DeleteModuleVersion: %v", txErr)
		}
	}()

	// Delete module version from DB
	if err = s.dbClient.TerraformModuleVersions.DeleteModuleVersion(txContext, moduleVersion); err != nil {
		return err
	}

	if newLatestVersion != nil {
		s.logger.Infof(
			"Deleted latest module version, latest flag is being set to latest version %s for module %s",
			newLatestVersion.SemanticVersion,
			module.ResourcePath,
		)
		newLatestVersion.Latest = true
		if _, err = s.dbClient.TerraformModuleVersions.UpdateModuleVersion(txContext, newLatestVersion); err != nil {
			return err
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return err
	}

	s.logger.Infow("Deleted a module version.",
		"caller", caller.GetSubject(),
		"moduleID", module.Metadata.ID,
		"moduleVersion", moduleVersion.SemanticVersion,
	)

	return nil
}

func (s *service) UploadModuleVersionPackage(ctx context.Context, moduleVersion *models.TerraformModuleVersion, reader io.Reader) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	module, err := s.getModuleByID(ctx, moduleVersion.ModuleID)
	if err != nil {
		return err
	}

	if err = caller.RequireAccessToGroup(ctx, module.GroupID, models.DeployerRole); err != nil {
		return err
	}

	if moduleVersion.Status == models.TerraformModuleVersionStatusUploadInProgress {
		return errors.NewError(errors.EConflict, "module package upload is already in progress")
	}

	if moduleVersion.Status == models.TerraformModuleVersionStatusUploaded || moduleVersion.Status == models.TerraformModuleVersionStatusErrored {
		return errors.NewError(errors.EConflict, "module package already uploaded")
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	currentTime := time.Now()
	// Update DB before object storage. If the object storage write fails, the DB transaction will be rolled back
	moduleVersion.UploadStartedTimestamp = &currentTime
	moduleVersion.Status = models.TerraformModuleVersionStatusUploadInProgress
	updatedModuleVersion, err := s.dbClient.TerraformModuleVersions.UpdateModuleVersion(txContext, moduleVersion)
	if err != nil {
		return err
	}

	checksum := sha256.New()

	// Create Tee reader which will writer to the multi writer
	teeReader := io.TeeReader(reader, checksum)

	if err = s.registryStore.UploadModulePackage(ctx, updatedModuleVersion, module, teeReader); err != nil {
		return err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return err
	}

	s.logger.Infof("Uploaded module with sha checksum %s", hex.EncodeToString(checksum.Sum(nil)))

	// Verify checksum matches expected checksum
	shaSum := hex.EncodeToString(checksum.Sum(nil))
	if shaSum != updatedModuleVersion.GetSHASumHex() {
		if err = s.setModuleVersionError(ctx, moduleVersion.Metadata.ID, fmt.Sprintf("Expected checksum of %s does not match received checksum %s", updatedModuleVersion.GetSHASumHex(), shaSum)); err != nil {
			s.logger.Errorf("failed to set terraform module version status to errored %v", err)
		}
		return nil
	}

	// Start async task to extract module metadata
	s.taskManager.StartTask(func(taskCtx context.Context) {
		if err := s.uploadModuleMetadata(taskCtx, module, updatedModuleVersion); err != nil {
			if err = s.setModuleVersionError(taskCtx, moduleVersion.Metadata.ID, err.Error()); err != nil {
				s.logger.Errorf("failed to set terraform module version status to errored %v", err)
			}
		}
	})

	return nil
}

func (s *service) GetModuleVersionPackageDownloadURL(ctx context.Context, moduleVersion *models.TerraformModuleVersion) (string, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return "", err
	}

	module, err := s.getModuleByID(ctx, moduleVersion.ModuleID)
	if err != nil {
		return "", err
	}

	if module.Private {
		if err = caller.RequireAccessToInheritedGroupResource(ctx, module.GroupID); err != nil {
			return "", err
		}
	}

	downloadURL, err := s.registryStore.GetModulePackagePresignedURL(ctx, moduleVersion, module)
	if err != nil {
		return "", err
	}

	return downloadURL, nil
}

func (s *service) getModuleByID(ctx context.Context, id string) (*models.TerraformModule, error) {
	module, err := s.dbClient.TerraformModules.GetModuleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if module == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("module with id %s not found", id))
	}

	return module, nil
}

func (s *service) getModuleVersionByID(ctx context.Context, id string) (*models.TerraformModuleVersion, error) {
	version, err := s.dbClient.TerraformModuleVersions.GetModuleVersionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if version == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("module version with id %s not found", id))
	}

	return version, nil
}

func (s *service) uploadModuleMetadata(ctx context.Context, module *models.TerraformModule, moduleVersion *models.TerraformModuleVersion) error {
	slugFile, err := os.CreateTemp("", "terraform-slug.tgz")
	if err != nil {
		return err
	}
	defer slugFile.Close()
	defer os.Remove(slugFile.Name())

	if err = s.registryStore.DownloadModulePackage(ctx, moduleVersion, module, slugFile); err != nil {
		return err
	}

	moduleDir, err := os.MkdirTemp("", "unpacked-slug")
	if err != nil {
		return err
	}
	defer os.RemoveAll(moduleDir)

	s.logger.Infof("Unpacking slug to temp dir %s", moduleDir)

	// Unpack slug
	if err = slug.Unpack(slugFile, moduleDir); err != nil {
		return err
	}

	parseResponse, err := parseModule(moduleDir)
	if err != nil {
		return err
	}

	if err = s.registryStore.UploadModuleConfigurationDetails(ctx, parseResponse.Root, moduleVersion, module); err != nil {
		return err
	}

	moduleVersion.Submodules = []string{}
	for _, submodule := range parseResponse.Submodules {
		moduleVersion.Submodules = append(moduleVersion.Submodules, strings.Split(submodule.Path, "/")[1])

		submoduleCopy := submodule
		if err = s.registryStore.UploadModuleConfigurationDetails(ctx, &submoduleCopy, moduleVersion, module); err != nil {
			return err
		}
	}

	moduleVersion.Examples = []string{}
	for _, example := range parseResponse.Examples {
		moduleVersion.Examples = append(moduleVersion.Examples, strings.Split(example.Path, "/")[1])

		exampleCopy := example
		if err = s.registryStore.UploadModuleConfigurationDetails(ctx, &exampleCopy, moduleVersion, module); err != nil {
			return err
		}
	}

	if len(parseResponse.Diagnostics) > 0 {
		moduleDiagnosticsBuf, diagErr := json.Marshal(parseResponse.Diagnostics)
		if diagErr != nil {
			return diagErr
		}

		moduleVersion.Diagnostics = strings.ReplaceAll(string(moduleDiagnosticsBuf), moduleDir, "")
	}

	if parseResponse.Diagnostics.HasErrors() {
		moduleVersion.Error = "failed validation"
		moduleVersion.Status = models.TerraformModuleVersionStatusErrored
	} else {
		moduleVersion.Status = models.TerraformModuleVersionStatusUploaded
	}

	_, err = s.dbClient.TerraformModuleVersions.UpdateModuleVersion(ctx, moduleVersion)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) setModuleVersionError(ctx context.Context, moduleVersionID string, errorMsg string) error {
	mv, err := s.dbClient.TerraformModuleVersions.GetModuleVersionByID(ctx, moduleVersionID)
	if err != nil {
		return err
	}
	mv.Status = models.TerraformModuleVersionStatusErrored
	mv.Error = errorMsg
	_, err = s.dbClient.TerraformModuleVersions.UpdateModuleVersion(ctx, mv)
	if err != nil {
		return err
	}
	return nil
}
