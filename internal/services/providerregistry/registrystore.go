// Package providerregistry package
package providerregistry

//go:generate go tool mockery --name RegistryStore --inpackage --case underscore

import (
	"context"
	"fmt"
	"io"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// RegistryStore interface encapsulates the logic for saving workspace registrys
type RegistryStore interface {
	UploadProviderPlatformBinary(
		ctx context.Context,
		providerPlatform *models.TerraformProviderPlatform,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
		body io.Reader,
	) (db.RetainObjectRefFunc, string, error)
	UploadProviderVersionReadme(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
		body io.Reader,
	) (db.RetainObjectRefFunc, string, error)
	UploadProviderVersionSHASums(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
		body io.Reader,
	) (db.RetainObjectRefFunc, string, error)
	UploadProviderVersionSHASumsSignature(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
		body io.Reader,
	) (db.RetainObjectRefFunc, string, error)
	GetProviderVersionReadme(ctx context.Context, objectKey string) (io.ReadCloser, error)
	GetProviderPlatformBinaryPresignedURL(ctx context.Context, objectKey string) (string, error)
	GetProviderVersionSHASumsPresignedURL(ctx context.Context, objectKey string) (string, error)
	GetProviderVersionSHASumsSignaturePresignedURL(ctx context.Context, objectKey string) (string, error)
}

type registryStore struct {
	objectStore     objectstore.ObjectStore
	objectStoreRefs db.ObjectStoreRefs
}

// NewRegistryStore creates an instance of the RegistryStore interface
func NewRegistryStore(objectStore objectstore.ObjectStore, objectStoreRefs db.ObjectStoreRefs) RegistryStore {
	return &registryStore{objectStore: objectStore, objectStoreRefs: objectStoreRefs}
}

func (r *registryStore) GetProviderVersionReadme(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	result, err := r.objectStore.GetObjectStream(ctx, objectKey, nil)
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func (r *registryStore) UploadProviderPlatformBinary(
	ctx context.Context,
	providerPlatform *models.TerraformProviderPlatform,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	key := providerPlatformObjectKey(providerPlatform, providerVersion, provider)
	if err := r.objectStore.UploadObject(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return r.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerProviderPlatform, ownerID)
	}, key, nil
}

func (r *registryStore) UploadProviderVersionReadme(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	key := providerVersionReadmeObjectKey(providerVersion, provider)
	if err := r.objectStore.UploadObject(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return r.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerProviderVersion, ownerID)
	}, key, nil
}

func (r *registryStore) UploadProviderVersionSHASums(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	key := providerVersionSHASumsObjectKey(providerVersion, provider)
	if err := r.objectStore.UploadObject(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return r.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerProviderVersion, ownerID)
	}, key, nil
}

func (r *registryStore) UploadProviderVersionSHASumsSignature(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	key := providerVersionSHASumsSignatureObjectKey(providerVersion, provider)
	if err := r.objectStore.UploadObject(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return r.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerProviderVersion, ownerID)
	}, key, nil
}

func (r *registryStore) GetProviderPlatformBinaryPresignedURL(ctx context.Context, objectKey string) (string, error) {
	return r.objectStore.GetPresignedURL(ctx, objectKey)
}

func (r *registryStore) GetProviderVersionSHASumsPresignedURL(ctx context.Context, objectKey string) (string, error) {
	return r.objectStore.GetPresignedURL(ctx, objectKey)
}

func (r *registryStore) GetProviderVersionSHASumsSignaturePresignedURL(ctx context.Context, objectKey string) (string, error) {
	return r.objectStore.GetPresignedURL(ctx, objectKey)
}

func providerVersionReadmeObjectKey(providerVersion *models.TerraformProviderVersion, provider *models.TerraformProvider) string {
	return fmt.Sprintf("registry/providers/%s/%s/README", provider.Metadata.ID, providerVersion.Metadata.ID)
}

func providerVersionSHASumsObjectKey(providerVersion *models.TerraformProviderVersion, provider *models.TerraformProvider) string {
	return fmt.Sprintf("registry/providers/%s/%s/SHA256SUMS", provider.Metadata.ID, providerVersion.Metadata.ID)
}

func providerVersionSHASumsSignatureObjectKey(providerVersion *models.TerraformProviderVersion, provider *models.TerraformProvider) string {
	return fmt.Sprintf("registry/providers/%s/%s/SHA256SUMS.sig", provider.Metadata.ID, providerVersion.Metadata.ID)
}

func providerPlatformObjectKey(providerPlatform *models.TerraformProviderPlatform, providerVersion *models.TerraformProviderVersion, provider *models.TerraformProvider) string {
	return fmt.Sprintf(
		"registry/providers/%s/%s/platforms/%s_%s/terraform-provider-%s_%s_%s_%s.zip",
		provider.Metadata.ID,
		providerVersion.Metadata.ID,
		providerPlatform.OperatingSystem,
		providerPlatform.Architecture,
		provider.Name,
		providerVersion.SemanticVersion,
		providerPlatform.OperatingSystem,
		providerPlatform.Architecture,
	)
}
