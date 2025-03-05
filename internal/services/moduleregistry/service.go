package moduleregistry

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
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
	"github.com/in-toto/in-toto-golang/in_toto"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/semver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
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

// CreateModuleAttestationInput is the input for creating a terraform module attestation
type CreateModuleAttestationInput struct {
	ModuleID        string
	Description     string
	AttestationData string
}

// GetModulesInput is the input for getting a list of terraform modules
type GetModulesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.TerraformModuleSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Group filters modules be the specified group
	Group *models.Group
	// Search filters module list by modules with a name that contains the search query
	Search *string
}

// GetModuleVersionsInput is the input for getting a list of module versions
type GetModuleVersionsInput struct {
	Sort              *db.TerraformModuleVersionSortableField
	PaginationOptions *pagination.Options
	Status            *models.TerraformModuleVersionStatus
	SemanticVersion   *string
	Latest            *bool
	ModuleID          string
	Search            *string
}

// GetModuleAttestationsInput is the input for getting a list of module attestations
type GetModuleAttestationsInput struct {
	Sort              *db.TerraformModuleAttestationSortableField
	PaginationOptions *pagination.Options
	Digest            *string
	ModuleID          string
}

const (
	// IntotoPayloadType is the type identifier for the in-toto format
	IntotoPayloadType = "application/vnd.in-toto+json"
	// MaxModuleAttestationSize is the max size in bytes for a module attestation
	MaxModuleAttestationSize = 1024 * 10
)

var (
	// SupportedIntotoStatementTypes contains a list of in-toto statement types that are
	// supported for module attestations
	SupportedIntotoStatementTypes = []string{"https://in-toto.io/Statement/v0.1"}
)

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
	GetModuleAttestationByID(ctx context.Context, id string) (*models.TerraformModuleAttestation, error)
	GetModuleAttestations(ctx context.Context, input *GetModuleAttestationsInput) (*db.ModuleAttestationsResult, error)
	CreateModuleAttestation(ctx context.Context, input *CreateModuleAttestationInput) (*models.TerraformModuleAttestation, error)
	UpdateModuleAttestation(ctx context.Context, attestation *models.TerraformModuleAttestation) (*models.TerraformModuleAttestation, error)
	DeleteModuleAttestation(ctx context.Context, attestation *models.TerraformModuleAttestation) error
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

// dsseEnvelope captures for signing format described in the following specification:
// https://github.com/secure-systems-lab/signing-spec/blob/master/envelope.md
type dsseEnvelope struct {
	PayloadType string          `json:"payloadType"`
	Payload     string          `json:"payload"`
	Signatures  []dsseSignature `json:"signatures"`
}

// dssSignature represents an in-toto signature from the DSSE specification
type dsseSignature struct {
	KeyID string `json:"keyid"`
	Sig   string `json:"sig"`
}

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	limitChecker    limits.LimitChecker
	registryStore   RegistryStore
	activityService activityevent.Service
	taskManager     asynctask.Manager
	handleCaller    handleCallerFunc
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	registryStore RegistryStore,
	activityService activityevent.Service,
	taskManager asynctask.Manager,
) Service {
	return newService(
		logger,
		dbClient,
		limitChecker,
		registryStore,
		activityService,
		taskManager,
		auth.HandleCaller,
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	registryStore RegistryStore,
	activityService activityevent.Service,
	taskManager asynctask.Manager,
	handleCaller handleCallerFunc,
) Service {
	return &service{
		logger,
		dbClient,
		limitChecker,
		registryStore,
		activityService,
		taskManager,
		handleCaller,
	}
}

func (s *service) GetModuleByID(ctx context.Context, id string) (*models.TerraformModule, error) {
	ctx, span := tracer.Start(ctx, "svc.GetModuleByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	module, err := s.getModuleByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return nil, err
	}

	if module.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformModuleResourceType, auth.WithGroupID(module.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return module, nil
}

func (s *service) GetModuleByPath(ctx context.Context, path string) (*models.TerraformModule, error) {
	ctx, span := tracer.Start(ctx, "svc.GetModuleByPath")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	module, err := s.dbClient.TerraformModules.GetModuleByPath(ctx, path)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by path")
		return nil, err
	}

	if module == nil {
		return nil, errors.New("module with path %s not found", path, errors.WithErrorCode(errors.ENotFound))
	}

	if module.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformModuleResourceType, auth.WithGroupID(module.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return module, nil
}

func (s *service) GetModuleByAddress(ctx context.Context, namespace string, name string, system string) (*models.TerraformModule, error) {
	ctx, span := tracer.Start(ctx, "svc.GetModuleByAddress")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	rootGroup, err := s.dbClient.Groups.GetGroupByFullPath(ctx, namespace)
	if err != nil {
		tracing.RecordError(span, err, "failed to get group by full path")
		return nil, err
	}

	if rootGroup == nil {
		return nil, errors.New("namespace %s not found", namespace, errors.WithErrorCode(errors.ENotFound))
	}

	moduleResult, err := s.dbClient.TerraformModules.GetModules(ctx, &db.GetModulesInput{
		PaginationOptions: &pagination.Options{First: ptr.Int32(1)},
		Filter: &db.TerraformModuleFilter{
			RootGroupID: &rootGroup.Metadata.ID,
			Name:        &name,
			System:      &system,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get modules")
		return nil, err
	}

	if len(moduleResult.Modules) == 0 {
		return nil, errors.New("module with name %s and system %s not found in namespace %s", name, system, namespace, errors.WithErrorCode(errors.ENotFound))
	}

	module := moduleResult.Modules[0]

	if module.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformModuleResourceType, auth.WithGroupID(module.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return &module, nil
}

func (s *service) GetModules(ctx context.Context, input *GetModulesInput) (*db.ModulesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetModules")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
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
		err = caller.RequirePermission(ctx, permissions.ViewTerraformModulePermission, auth.WithNamespacePath(input.Group.FullPath))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
		dbInput.Filter.GroupID = &input.Group.Metadata.ID
	} else {
		policy, napErr := caller.GetNamespaceAccessPolicy(ctx)
		if napErr != nil {
			tracing.RecordError(span, napErr, "failed to get namespace access policy")
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
				tracing.RecordError(span, err, "failed to set filters for non-admin access")
				return nil, err
			}
		}
	}

	return s.dbClient.TerraformModules.GetModules(ctx, &dbInput)
}

func (s *service) UpdateModule(ctx context.Context, module *models.TerraformModule) (*models.TerraformModule, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateModule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformModulePermission, auth.WithGroupID(module.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	if vErr := module.Validate(); vErr != nil {
		tracing.RecordError(span, vErr, "failed to validate terraform module model")
		return nil, vErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateModule: %v", txErr)
		}
	}()

	updatedModule, err := s.dbClient.TerraformModules.UpdateModule(txContext, module)
	if err != nil {
		tracing.RecordError(span, err, "failed to update module")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedModule, nil
}

func (s *service) CreateModuleAttestation(ctx context.Context, input *CreateModuleAttestationInput) (*models.TerraformModuleAttestation, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateModuleAttestation")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	module, err := s.getModuleByID(ctx, input.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformModulePermission, auth.WithGroupID(module.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	hash := sha256.New()

	// Compute the checksum.
	size, err := io.Copy(hash, strings.NewReader(input.AttestationData))
	if err != nil {
		tracing.RecordError(span, err, "failed to get reader to compute checksum")
		return nil, err
	}

	// Verify the module attestation data is below the size limit
	if size > MaxModuleAttestationSize {
		return nil, errors.New("module attestation of size %d exceeds max size limit of %d bytes", size, MaxModuleAttestationSize, errors.WithErrorCode(errors.EInvalid))
	}

	decodedSig, err := base64.StdEncoding.DecodeString(input.AttestationData)
	if err != nil {
		tracing.RecordError(span, err, "failed to decode base64 string")
		return nil, errors.Wrap(err, "failed to decode attestation data", errors.WithErrorCode(errors.EInvalid))
	}

	// Decode DSSE Envelope
	env := dsseEnvelope{}
	if err = json.Unmarshal(decodedSig, &env); err != nil {
		tracing.RecordError(span, err, "failed to unmarshal DSEE attestation data")
		return nil, errors.Wrap(err, "attestation data is not in dsse format", errors.WithErrorCode(errors.EInvalid))
	}

	if env.PayloadType != IntotoPayloadType {
		return nil, errors.New("invalid payloadType %s on envelope; expected %s", env.PayloadType, IntotoPayloadType, errors.WithErrorCode(errors.EInvalid))
	}

	// Get the expected digest from the attestation
	decodedPredicate, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		tracing.RecordError(span, err, "failed to decode base64 string")
		return nil, errors.Wrap(err, "decoding dsse envelope payload", errors.WithErrorCode(errors.EInvalid))
	}
	var statement in_toto.Statement
	if err = json.Unmarshal(decodedPredicate, &statement); err != nil {
		tracing.RecordError(span, err, "failed to unmarshal the in-toto statement")
		return nil, errors.Wrap(err, "decoding predicate", errors.WithErrorCode(errors.EInvalid))
	}

	foundSupportedType := false
	for _, statementType := range SupportedIntotoStatementTypes {
		if statementType == statement.Type {
			foundSupportedType = true
			break
		}
	}

	if !foundSupportedType {
		return nil, errors.New(
			"in-toto statement type %s not supported; expected one of %s",
			statement.Type,
			strings.Join(SupportedIntotoStatementTypes, ", "),
			errors.WithErrorCode(errors.EInvalid),
		)
	}

	// Compare the actual and expected
	if statement.Subject == nil || len(statement.Subject) == 0 {
		return nil, errors.New("in-toto statement is missing subject", errors.WithErrorCode(errors.EInvalid))
	}

	digests := []string{}
	for _, subject := range statement.Subject {
		digest, ok := subject.Digest["sha256"]
		if !ok {
			return nil, errors.New("subject %s is missing sha256 digest", subject.Name, errors.WithErrorCode(errors.EInvalid))
		}
		digests = append(digests, digest)
	}

	attestationToCreate := models.TerraformModuleAttestation{
		ModuleID:      input.ModuleID,
		Description:   input.Description,
		Data:          input.AttestationData,
		DataSHASum:    hash.Sum(nil),
		SchemaType:    statement.Type,
		PredicateType: statement.PredicateType,
		Digests:       digests,
		CreatedBy:     caller.GetSubject(),
	}

	if err = attestationToCreate.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate terraform module model")
		return nil, err
	}

	// Need to use a transaction so we can roll it back if resource limits are violated.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateModuleAttestation: %v", txErr)
		}
	}()

	createdAttestation, err := s.dbClient.TerraformModuleAttestations.CreateModuleAttestation(txContext, &attestationToCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create module attestation")
		return nil, err
	}

	// Get the number of attestations on this module to check whether we just violated the limit.
	newAttestations, err := s.dbClient.TerraformModuleAttestations.GetModuleAttestations(txContext, &db.GetModuleAttestationsInput{
		Filter: &db.TerraformModuleAttestationFilter{
			TimeRangeStart: ptr.Time(createdAttestation.Metadata.CreationTimestamp.Add(-limits.ResourceLimitTimePeriod)),
			ModuleID:       &createdAttestation.ModuleID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get module's attestations")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitAttestationsPerTerraformModulePerTimePeriod, newAttestations.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Created a module attestation.",
		"caller", caller.GetSubject(),
		"moduleID", input.ModuleID,
		"modulePath", module.ResourcePath,
		"moduleAttestationID", createdAttestation.Metadata.ID,
	)

	return createdAttestation, nil
}

func (s *service) UpdateModuleAttestation(ctx context.Context, attestation *models.TerraformModuleAttestation) (*models.TerraformModuleAttestation, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateModuleAttestation")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	module, err := s.getModuleByID(ctx, attestation.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformModulePermission, auth.WithGroupID(module.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	updatedAttestation, err := s.dbClient.TerraformModuleAttestations.UpdateModuleAttestation(ctx, attestation)
	if err != nil {
		tracing.RecordError(span, err, "failed to update module attestation")
		return nil, err
	}

	s.logger.Infow("Updated module attestation.",
		"caller", caller.GetSubject(),
		"moduleID", module.Metadata.ID,
		"modulePath", module.ResourcePath,
		"moduleAttestationID", attestation.Metadata.ID,
	)

	return updatedAttestation, nil
}

func (s *service) GetModuleAttestationByID(ctx context.Context, id string) (*models.TerraformModuleAttestation, error) {
	ctx, span := tracer.Start(ctx, "svc.GetModuleAttestationByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	moduleAttestation, err := s.dbClient.TerraformModuleAttestations.GetModuleAttestationByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module attestation by ID")
		return nil, err
	}

	if moduleAttestation == nil {
		return nil, errors.New("module with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	module, err := s.getModuleByID(ctx, moduleAttestation.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return nil, err
	}

	if module.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformModuleResourceType, auth.WithGroupID(module.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return moduleAttestation, nil
}

func (s *service) GetModuleAttestations(ctx context.Context, input *GetModuleAttestationsInput) (*db.ModuleAttestationsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetModuleAttestations")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	module, err := s.getModuleByID(ctx, input.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return nil, err
	}

	if module.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformModuleResourceType, auth.WithGroupID(module.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	dbInput := db.GetModuleAttestationsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.TerraformModuleAttestationFilter{
			ModuleID: &input.ModuleID,
			Digest:   input.Digest,
		},
	}

	return s.dbClient.TerraformModuleAttestations.GetModuleAttestations(ctx, &dbInput)
}

func (s *service) DeleteModuleAttestation(ctx context.Context, attestation *models.TerraformModuleAttestation) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteModuleAttestation")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	module, err := s.getModuleByID(ctx, attestation.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformModulePermission, auth.WithGroupID(module.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	err = s.dbClient.TerraformModuleAttestations.DeleteModuleAttestation(ctx, attestation)
	if err != nil {
		tracing.RecordError(span, err, "failed to delete module attestation")
		return err
	}

	s.logger.Infow("Deleted module attestation.",
		"caller", caller.GetSubject(),
		"moduleID", module.Metadata.ID,
		"modulePath", module.ResourcePath,
		"moduleAttestationID", attestation.Metadata.ID,
	)

	return nil
}

func (s *service) CreateModule(ctx context.Context, input *CreateModuleInput) (*models.TerraformModule, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateModule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateTerraformModulePermission, auth.WithGroupID(input.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get group by ID")
		return nil, err
	}

	if group == nil {
		return nil, fmt.Errorf("group with id %s not found", input.GroupID)
	}

	var rootGroupID string
	if group.ParentID == "" {
		rootGroupID = input.GroupID
	} else {
		rootGroup, gErr := s.dbClient.Groups.GetGroupByFullPath(ctx, group.GetRootGroupPath())
		if gErr != nil {
			tracing.RecordError(span, gErr, "failed to get group by full path")
			return nil, gErr
		}

		if rootGroup == nil {
			return nil, fmt.Errorf("group with path %s not found", group.GetRootGroupPath())
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
		tracing.RecordError(span, vErr, "failed to validate terraform module model")
		return nil, vErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateModule: %v", txErr)
		}
	}()

	createdModule, err := s.dbClient.TerraformModules.CreateModule(txContext, moduleToCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create module")
		return nil, err
	}

	// Get the number of modules in this group to check whether we just violated the limit.
	newModules, err := s.dbClient.TerraformModules.GetModules(txContext, &db.GetModulesInput{
		Filter: &db.TerraformModuleFilter{
			GroupID: &createdModule.GroupID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get group's modules")
		return nil, err
	}

	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitTerraformModulesPerGroup, newModules.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetTerraformModule,
			TargetID:      createdModule.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return createdModule, nil
}

func (s *service) DeleteModule(ctx context.Context, module *models.TerraformModule) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteModule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteTerraformModulePermission, auth.WithGroupID(module.GroupID))
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
			s.logger.Errorf("failed to rollback tx for service layer DeleteModule: %v", txErr)
		}
	}()

	err = s.dbClient.TerraformModules.DeleteModule(txContext, module)
	if err != nil {
		tracing.RecordError(span, err, "failed to delete module")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetModulesByIDs(ctx context.Context, ids []string) ([]models.TerraformModule, error) {
	ctx, span := tracer.Start(ctx, "svc.GetModulesByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	response, err := s.dbClient.TerraformModules.GetModules(ctx, &db.GetModulesInput{
		Filter: &db.TerraformModuleFilter{
			TerraformModuleIDs: ids,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get modules")
		return nil, err
	}

	namespacePaths := []string{}
	for _, module := range response.Modules {
		if module.Private {
			namespacePaths = append(namespacePaths, module.GetGroupPath())
		}
	}

	if len(namespacePaths) > 0 {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformModuleResourceType, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return response.Modules, nil
}

func (s *service) GetModuleConfigurationDetails(ctx context.Context, moduleVersion *models.TerraformModuleVersion, path string) (*ModuleConfigurationDetails, error) {
	ctx, span := tracer.Start(ctx, "svc.GetModuleConfigurationDetails")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	module, err := s.getModuleByID(ctx, moduleVersion.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return nil, err
	}

	if module.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformModuleResourceType, auth.WithGroupID(module.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	reader, err := s.registryStore.GetModuleConfigurationDetails(ctx, moduleVersion, module, path)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module configuration details")
		return nil, err
	}
	defer reader.Close()

	var moduleMetadata ModuleConfigurationDetails
	if err := json.NewDecoder(reader).Decode(&moduleMetadata); err != nil {
		tracing.RecordError(span, err, "failed to decode module metadata")
		return nil, err
	}

	return &moduleMetadata, nil
}

func (s *service) GetModuleVersionByID(ctx context.Context, id string) (*models.TerraformModuleVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetModuleVersionByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	moduleVersion, err := s.getModuleVersionByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module version by ID")
		return nil, err
	}

	module, err := s.getModuleByID(ctx, moduleVersion.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return nil, err
	}

	if module.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformModuleResourceType, auth.WithGroupID(module.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return moduleVersion, nil
}

func (s *service) GetModuleVersions(ctx context.Context, input *GetModuleVersionsInput) (*db.ModuleVersionsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetModuleVersions")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	module, err := s.getModuleByID(ctx, input.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return nil, err
	}

	if module.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformModuleResourceType, auth.WithGroupID(module.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
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
			Search:          input.Search,
		},
	}

	return s.dbClient.TerraformModuleVersions.GetModuleVersions(ctx, &dbInput)

}

func (s *service) GetModuleVersionsByIDs(ctx context.Context, ids []string) ([]models.TerraformModuleVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetModuleVersionsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	response, err := s.dbClient.TerraformModuleVersions.GetModuleVersions(ctx, &db.GetModuleVersionsInput{
		Filter: &db.TerraformModuleVersionFilter{
			ModuleVersionIDs: ids,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get module versions")
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
			tracing.RecordError(span, err, "failed to get modules by IDs")
			return nil, err
		}
	}

	return response.ModuleVersions, nil
}

func (s *service) CreateModuleVersion(ctx context.Context, input *CreateModuleVersionInput) (*models.TerraformModuleVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateModuleVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	module, err := s.getModuleByID(ctx, input.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformModulePermission, auth.WithGroupID(module.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Verify semantic version is valid
	semVersion, err := version.NewSemver(input.SemanticVersion)
	if err != nil {
		tracing.RecordError(span, err, "failed to verify semantic version")
		return nil, errors.Wrap(err, "invalid semantic version", errors.WithErrorCode(errors.EInvalid))
	}

	// Check if this version is greater than the previous latest
	versionsResp, err := s.dbClient.TerraformModuleVersions.GetModuleVersions(ctx, &db.GetModuleVersionsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		Filter: &db.TerraformModuleVersionFilter{
			ModuleID: &input.ModuleID,
			Latest:   ptr.Bool(true),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get module versions")
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
			tracing.RecordError(span, sErr, "semver validation failed")
			return nil, sErr
		}
		if semver.IsSemverGreaterThan(semVersion, prevSemVersion) {
			isLatest = true
			// Remove latest from version
			prevLatest.Latest = false
			if _, uErr := s.dbClient.TerraformModuleVersions.UpdateModuleVersion(txContext, &prevLatest); uErr != nil {
				tracing.RecordError(span, uErr, "failed to update module version")
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
		tracing.RecordError(span, err, "failed to create module version")
		return nil, err
	}

	groupPath := module.GetGroupPath()

	// Get the number of versions of this module to check whether we just violated the limit.
	newVersions, err := s.dbClient.TerraformModuleVersions.GetModuleVersions(txContext, &db.GetModuleVersionsInput{
		Filter: &db.TerraformModuleVersionFilter{
			TimeRangeStart: ptr.Time(moduleVersion.Metadata.CreationTimestamp.Add(-limits.ResourceLimitTimePeriod)),
			ModuleID:       &moduleVersion.ModuleID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get module's versions")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitVersionsPerTerraformModulePerTimePeriod, newVersions.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetTerraformModuleVersion,
			TargetID:      moduleVersion.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
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
	ctx, span := tracer.Start(ctx, "svc.DeleteModuleVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	module, err := s.getModuleByID(ctx, moduleVersion.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformModulePermission, auth.WithGroupID(module.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
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
			tracing.RecordError(span, err, "failed to get module version")
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
				tracing.RecordError(span, lsErr, "failed to validate semver")
				return lsErr
			}

			currentSemver, csErr := version.NewSemver(vCopy.SemanticVersion)
			if csErr != nil {
				tracing.RecordError(span, csErr, "failed to validate semver")
				return csErr
			}

			if semver.IsSemverGreaterThan(currentSemver, latestSemver) {
				newLatestVersion = &vCopy
			}
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for DeleteModuleVersion: %v", txErr)
		}
	}()

	// Delete module version from DB
	if err = s.dbClient.TerraformModuleVersions.DeleteModuleVersion(txContext, moduleVersion); err != nil {
		tracing.RecordError(span, err, "failed to delete module version")
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
			tracing.RecordError(span, err, "failed to update module version")
			return err
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
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
	ctx, span := tracer.Start(ctx, "svc.UploadModuleVersionPackage")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	module, err := s.getModuleByID(ctx, moduleVersion.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformModulePermission, auth.WithGroupID(module.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	if moduleVersion.Status == models.TerraformModuleVersionStatusUploadInProgress {
		return errors.New("module package upload is already in progress", errors.WithErrorCode(errors.EConflict))
	}

	if moduleVersion.Status == models.TerraformModuleVersionStatusUploaded || moduleVersion.Status == models.TerraformModuleVersionStatusErrored {
		return errors.New("module package already uploaded", errors.WithErrorCode(errors.EConflict))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
		tracing.RecordError(span, err, "failed to update module version")
		return err
	}

	checksum := sha256.New()

	// Create Tee reader which will writer to the multi writer
	teeReader := io.TeeReader(reader, checksum)

	if err = s.registryStore.UploadModulePackage(ctx, updatedModuleVersion, module, teeReader); err != nil {
		tracing.RecordError(span, err, "failed to upload module package")
		return err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return err
	}

	s.logger.Infof("Uploaded module with sha checksum %s", hex.EncodeToString(checksum.Sum(nil)))

	// Verify checksum matches expected checksum
	shaSum := hex.EncodeToString(checksum.Sum(nil))
	if shaSum != updatedModuleVersion.GetSHASumHex() {
		if err = s.setModuleVersionError(ctx, moduleVersion.Metadata.ID, fmt.Sprintf("Expected checksum of %s does not match received checksum %s", updatedModuleVersion.GetSHASumHex(), shaSum)); err != nil {
			tracing.RecordError(span, err, "failed to set module version status to errored")
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
	ctx, span := tracer.Start(ctx, "svc.GetModuleVersionPackageDownloadURL")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return "", err
	}

	module, err := s.getModuleByID(ctx, moduleVersion.ModuleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module by ID")
		return "", err
	}

	if module.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformModuleResourceType, auth.WithGroupID(module.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return "", err
		}
	}

	downloadURL, err := s.registryStore.GetModulePackagePresignedURL(ctx, moduleVersion, module)
	if err != nil {
		tracing.RecordError(span, err, "failed to get module package presigned URL")
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
		return nil, errors.New("module with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	return module, nil
}

func (s *service) getModuleVersionByID(ctx context.Context, id string) (*models.TerraformModuleVersion, error) {
	version, err := s.dbClient.TerraformModuleVersions.GetModuleVersionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if version == nil {
		return nil, errors.New("module version with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
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
