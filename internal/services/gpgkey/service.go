package gpgkey

import (
	"context"
	"fmt"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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
	logger   logger.Logger
	dbClient *db.Client
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
) Service {
	return newService(
		logger,
		dbClient,
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
) Service {
	return &service{
		logger:   logger,
		dbClient: dbClient,
	}
}

func (s *service) GetGPGKeys(ctx context.Context, input *GetGPGKeysInput) (*db.GPGKeysResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToNamespace(ctx, input.NamespacePath, models.ViewerRole); err != nil {
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

	for _, k := range result.GPGKeys {
		if err := caller.RequireAccessToInheritedGroupResource(ctx, k.GroupID); err != nil {
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

	if err := caller.RequireAccessToGroup(ctx, gpgKey.GroupID, models.DeployerRole); err != nil {
		return err
	}

	s.logger.Infow("Requested deletion of a gpg key.",
		"caller", caller.GetSubject(),
		"groupID", gpgKey.GroupID,
		"gpgKeyID", gpgKey.Metadata.ID,
	)
	return s.dbClient.GPGKeys.DeleteGPGKey(ctx, gpgKey)
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

	if err := caller.RequireAccessToInheritedGroupResource(ctx, gpgKey.GroupID); err != nil {
		return nil, err
	}

	return gpgKey, nil
}

func (s *service) CreateGPGKey(ctx context.Context, input *CreateGPGKeyInput) (*models.GPGKey, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToGroup(ctx, input.GroupID, models.DeployerRole); err != nil {
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

	primaryKey := entityList[0].PrimaryKey

	gpgKey := &models.GPGKey{
		GroupID:     input.GroupID,
		GPGKeyID:    primaryKey.KeyId,
		Fingerprint: fmt.Sprintf("%016X", primaryKey.Fingerprint),
		ASCIIArmor:  input.ASCIIArmor,
		CreatedBy:   caller.GetSubject(),
	}

	s.logger.Infow("Requested creation of a gpg key.",
		"caller", caller.GetSubject(),
		"groupID", input.GroupID,
		"gpgKeyID", gpgKey.GetHexGPGKeyID(),
	)

	// Store gpg key in DB
	return s.dbClient.GPGKeys.CreateGPGKey(ctx, gpgKey)
}
