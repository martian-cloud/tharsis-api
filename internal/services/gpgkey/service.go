// Package gpgkey package
package gpgkey

import (
	"context"
	"fmt"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
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

// CreateGPGKeyInput is the input for creating a GPG key
type CreateGPGKeyInput struct {
	GroupID    string
	ASCIIArmor string
}

// GetGPGKeysInput is the input for querying a list of gpg keys
type GetGPGKeysInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.GPGKeySortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// NamespacePath is the namespace to return gpg keys for
	NamespacePath string
	// IncludeInherited includes inherited gpg keys in the result
	IncludeInherited bool
}

// Service implements all gpg key related functionality
type Service interface {
	GetGPGKeyByID(ctx context.Context, id string) (*models.GPGKey, error)
	GetGPGKeys(ctx context.Context, input *GetGPGKeysInput) (*db.GPGKeysResult, error)
	GetGPGKeysByIDs(ctx context.Context, idList []string) ([]models.GPGKey, error)
	CreateGPGKey(ctx context.Context, input *CreateGPGKeyInput) (*models.GPGKey, error)
	DeleteGPGKey(ctx context.Context, gpgKey *models.GPGKey) error
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
	return newService(
		logger,
		dbClient,
		limitChecker,
		activityService,
	)
}

func newService(
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

func (s *service) GetGPGKeys(ctx context.Context, input *GetGPGKeysInput) (*db.GPGKeysResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetGPGKeys")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewGPGKeyPermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	filter := &db.GPGKeyFilter{}

	if input.IncludeInherited {
		pathParts := strings.Split(input.NamespacePath, "/")

		paths := []string{}
		for len(pathParts) > 0 {
			paths = append(paths, strings.Join(pathParts, "/"))
			// Remove last element
			pathParts = pathParts[:len(pathParts)-1]
		}

		filter.NamespacePaths = paths
	} else {
		// This will return an empty result for workspace namespaces because workspaces
		// don't have gpg keys directly associated (i.e. only group namespaces do)
		filter.NamespacePaths = []string{input.NamespacePath}
	}

	result, err := s.dbClient.GPGKeys.GetGPGKeys(ctx, &db.GetGPGKeysInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            filter,
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get GPG keys")
		return nil, err
	}
	return result, nil
}

func (s *service) GetGPGKeysByIDs(ctx context.Context, idList []string) ([]models.GPGKey, error) {
	ctx, span := tracer.Start(ctx, "svc.GetGPGKeysByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.GPGKeys.GetGPGKeys(ctx, &db.GetGPGKeysInput{
		Filter: &db.GPGKeyFilter{
			KeyIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get GPG keys by IDs")
		return nil, err
	}

	namespacePaths := []string{}
	for _, k := range result.GPGKeys {
		namespacePaths = append(namespacePaths, k.GetGroupPath())
	}

	if len(namespacePaths) > 0 {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.GPGKeyResourceType, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			tracing.RecordError(span, err, "inherited resource access check failed")
			return nil, err
		}
	}

	return result.GPGKeys, nil
}

func (s *service) DeleteGPGKey(ctx context.Context, gpgKey *models.GPGKey) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteGPGKey")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteGPGKeyPermission, auth.WithGroupID(gpgKey.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	s.logger.Infow("Requested deletion of a gpg key.",
		"caller", caller.GetSubject(),
		"groupID", gpgKey.GroupID,
		"gpgKeyID", gpgKey.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to start a DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteGPGKey: %v", txErr)
		}
	}()

	if err = s.dbClient.GPGKeys.DeleteGPGKey(txContext, gpgKey); err != nil {
		tracing.RecordError(span, err, "failed to delete a GPG key")
		return err
	}

	// Retrieve the group to get its path.
	group, err := s.dbClient.Groups.GetGroupByID(txContext, gpgKey.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to retrieve a GPG key's group")
		return err
	}
	if group == nil {
		tracing.RecordError(span, nil, "GPG key's group does not exist: %s", gpgKey.GroupID)
		return fmt.Errorf("group ID does not exist: %s", gpgKey.GroupID)
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetGroup,
			TargetID:      group.Metadata.ID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: gpgKey.GetHexGPGKeyID(),
				ID:   gpgKey.Metadata.ID,
				Type: string(models.TargetGPGKey),
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create an activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetGPGKeyByID(ctx context.Context, id string) (*models.GPGKey, error) {
	ctx, span := tracer.Start(ctx, "svc.GetGPGKeyByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get gpgKey from DB
	gpgKey, err := s.dbClient.GPGKeys.GetGPGKeyByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get GPG key by ID")
		return nil, err
	}

	if gpgKey == nil {
		tracing.RecordError(span, nil, "gpg key with ID %s not found", id)
		return nil, errors.New(errors.ENotFound, "gpg key with ID %s not found", id)
	}

	err = caller.RequireAccessToInheritableResource(ctx, permissions.GPGKeyResourceType, auth.WithGroupID(gpgKey.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "inheritable resource permission check failed")
		return nil, err
	}

	return gpgKey, nil
}

func (s *service) CreateGPGKey(ctx context.Context, input *CreateGPGKeyInput) (*models.GPGKey, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateGPGKey")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateGPGKeyPermission, auth.WithGroupID(input.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Read key to get GPG key ID and fingerprint
	entityList, err := openpgp.ReadArmoredKeyRing(strings.NewReader(input.ASCIIArmor))
	if err != nil {
		tracing.RecordError(span, err, "failed to read ascii key armor")
		return nil, errors.Wrap(err, errors.EInvalid, "failed to read ascii key armor")
	}

	if len(entityList) != 1 {
		tracing.RecordError(span, nil, "invalid number of public keys found, expected 1 but found %d", len(entityList))
		return nil, errors.New(errors.EInvalid, "invalid number of public keys found, expected 1 but found %d", len(entityList))
	}

	group, err := s.dbClient.Groups.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get GPG key's group by ID")
		return nil, err
	}
	if group == nil {
		tracing.RecordError(span, nil, "group ID does not exist: %s", input.GroupID)
		return nil, fmt.Errorf("group ID does not exist: %s", input.GroupID)
	}

	primaryKey := entityList[0].PrimaryKey
	toCreate := &models.GPGKey{
		GroupID:     input.GroupID,
		GPGKeyID:    primaryKey.KeyId,
		Fingerprint: fmt.Sprintf("%016X", primaryKey.Fingerprint),
		ASCIIArmor:  input.ASCIIArmor,
		CreatedBy:   caller.GetSubject(),
	}

	s.logger.Infow("Requested creation of a gpg key.",
		"caller", caller.GetSubject(),
		"groupID", input.GroupID,
		"gpgKeyID", toCreate.GetHexGPGKeyID(),
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin a DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateGPGKey: %v", txErr)
		}
	}()

	// Store gpg key in DB
	createdKey, err := s.dbClient.GPGKeys.CreateGPGKey(txContext, toCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to store a GPG key in the DB")
		return nil, err
	}

	// Get the number of GPG keys in the group to check whether we just violated the limit.
	newKeys, err := s.dbClient.GPGKeys.GetGPGKeys(txContext, &db.GetGPGKeysInput{
		Filter: &db.GPGKeyFilter{
			NamespacePaths: []string{group.FullPath},
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get group's GPG keys")
		return nil, err
	}

	if err = s.limitChecker.CheckLimit(txContext, limits.ResourceLimitGPGKeysPerGroup, newKeys.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetGPGKey,
			TargetID:      createdKey.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit a DB transaction")
		return nil, err
	}

	return createdKey, nil
}
