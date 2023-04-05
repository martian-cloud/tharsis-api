// Package gpgkey package
package gpgkey

import (
	"context"
	"fmt"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
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
	PaginationOptions *db.PaginationOptions
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
	activityService activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	activityService activityevent.Service,
) Service {
	return newService(
		logger,
		dbClient,
		activityService,
	)
}

func newService(
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

func (s *service) GetGPGKeys(ctx context.Context, input *GetGPGKeysInput) (*db.GPGKeysResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewGPGKeyPermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
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
		return nil, err
	}
	return result, nil
}

func (s *service) GetGPGKeysByIDs(ctx context.Context, idList []string) ([]models.GPGKey, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.GPGKeys.GetGPGKeys(ctx, &db.GetGPGKeysInput{
		Filter: &db.GPGKeyFilter{
			KeyIDs: idList,
		},
	})
	if err != nil {
		return nil, err
	}

	namespacePaths := []string{}
	for _, k := range result.GPGKeys {
		namespacePaths = append(namespacePaths, k.GetGroupPath())
	}

	if len(namespacePaths) > 0 {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.GPGKeyResourceType, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			return nil, err
		}
	}

	return result.GPGKeys, nil
}

func (s *service) DeleteGPGKey(ctx context.Context, gpgKey *models.GPGKey) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteGPGKeyPermission, auth.WithGroupID(gpgKey.GroupID))
	if err != nil {
		return err
	}

	s.logger.Infow("Requested deletion of a gpg key.",
		"caller", caller.GetSubject(),
		"groupID", gpgKey.GroupID,
		"gpgKeyID", gpgKey.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteGPGKey: %v", txErr)
		}
	}()

	if err = s.dbClient.GPGKeys.DeleteGPGKey(txContext, gpgKey); err != nil {
		return err
	}

	// Retrieve the group to get its path.
	group, err := s.dbClient.Groups.GetGroupByID(txContext, gpgKey.GroupID)
	if err != nil {
		return err
	}
	if group == nil {
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
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetGPGKeyByID(ctx context.Context, id string) (*models.GPGKey, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get gpgKey from DB
	gpgKey, err := s.dbClient.GPGKeys.GetGPGKeyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if gpgKey == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("gpg key with ID %s not found", id))
	}

	err = caller.RequireAccessToInheritableResource(ctx, permissions.GPGKeyResourceType, auth.WithGroupID(gpgKey.GroupID))
	if err != nil {
		return nil, err
	}

	return gpgKey, nil
}

func (s *service) CreateGPGKey(ctx context.Context, input *CreateGPGKeyInput) (*models.GPGKey, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateGPGKeyPermission, auth.WithGroupID(input.GroupID))
	if err != nil {
		return nil, err
	}

	// Read key to get GPG key ID and fingerprint
	entityList, err := openpgp.ReadArmoredKeyRing(strings.NewReader(input.ASCIIArmor))
	if err != nil {
		return nil, errors.NewError(errors.EInvalid, fmt.Sprintf("failed to read ascii key armor: %v", err))
	}

	if len(entityList) != 1 {
		return nil, errors.NewError(errors.EInvalid, fmt.Sprintf("invalid number of public keys found, expected 1 but found %d", len(entityList)))
	}

	group, err := s.dbClient.Groups.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
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
		return nil, err
	}

	return createdKey, nil
}
