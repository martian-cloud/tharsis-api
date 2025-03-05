// Package providerregistry package
package providerregistry

//go:generate go tool mockery --name RegistryStore --inpackage --case underscore

import (
	"context"
	"fmt"
	"io"

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
	) error
	UploadProviderVersionReadme(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
		body io.Reader,
	) error
	UploadProviderVersionSHASums(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
		body io.Reader,
	) error
	UploadProviderVersionSHASumsSignature(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
		body io.Reader,
	) error
	GetProviderVersionReadme(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
	) (io.ReadCloser, error)
	GetProviderPlatformBinaryPresignedURL(
		ctx context.Context,
		providerPlatform *models.TerraformProviderPlatform,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
	) (string, error)
	GetProviderVersionSHASumsPresignedURL(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
	) (string, error)
	GetProviderVersionSHASumsSignaturePresignedURL(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
	) (string, error)
}

type registryStore struct {
	objectStore objectstore.ObjectStore
}

// NewRegistryStore creates an instance of the RegistryStore interface
func NewRegistryStore(objectStore objectstore.ObjectStore) RegistryStore {
	return &registryStore{objectStore: objectStore}
}

func (r *registryStore) GetProviderVersionReadme(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
) (io.ReadCloser, error) {
	return r.objectStore.GetObjectStream(
		ctx,
		getProviderVersionReadmeObjectKey(providerVersion, provider),
		nil,
	)
}

func (r *registryStore) UploadProviderPlatformBinary(
	ctx context.Context,
	providerPlatform *models.TerraformProviderPlatform,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) error {
	return r.upload(ctx, getProviderPlatformObjectKey(providerPlatform, providerVersion, provider), body)
}

func (r *registryStore) UploadProviderVersionReadme(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) error {
	return r.upload(ctx, getProviderVersionReadmeObjectKey(providerVersion, provider), body)
}

func (r *registryStore) UploadProviderVersionSHASums(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) error {
	return r.upload(ctx, getProviderVersionSHASumsObjectKey(providerVersion, provider), body)
}

func (r *registryStore) UploadProviderVersionSHASumsSignature(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) error {
	return r.upload(ctx, getProviderVersionSHASumsSignatureObjectKey(providerVersion, provider), body)
}

func (r *registryStore) GetProviderPlatformBinaryPresignedURL(
	ctx context.Context,
	providerPlatform *models.TerraformProviderPlatform,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
) (string, error) {
	return r.objectStore.GetPresignedURL(ctx, getProviderPlatformObjectKey(providerPlatform, providerVersion, provider))
}

func (r *registryStore) GetProviderVersionSHASumsPresignedURL(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
) (string, error) {
	return r.objectStore.GetPresignedURL(ctx, getProviderVersionSHASumsObjectKey(providerVersion, provider))
}

func (r *registryStore) GetProviderVersionSHASumsSignaturePresignedURL(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
) (string, error) {
	return r.objectStore.GetPresignedURL(ctx, getProviderVersionSHASumsSignatureObjectKey(providerVersion, provider))
}

func (r *registryStore) upload(ctx context.Context, key string, body io.Reader) error {
	return r.objectStore.UploadObject(ctx, key, body)
}

func getProviderVersionReadmeObjectKey(providerVersion *models.TerraformProviderVersion, provider *models.TerraformProvider) string {
	return fmt.Sprintf("registry/providers/%s/%s/README", provider.Metadata.ID, providerVersion.Metadata.ID)
}

func getProviderVersionSHASumsObjectKey(providerVersion *models.TerraformProviderVersion, provider *models.TerraformProvider) string {
	return fmt.Sprintf("registry/providers/%s/%s/SHA256SUMS", provider.Metadata.ID, providerVersion.Metadata.ID)
}

func getProviderVersionSHASumsSignatureObjectKey(providerVersion *models.TerraformProviderVersion, provider *models.TerraformProvider) string {
	return fmt.Sprintf("registry/providers/%s/%s/SHA256SUMS.sig", provider.Metadata.ID, providerVersion.Metadata.ID)
}

func getProviderPlatformObjectKey(providerPlatform *models.TerraformProviderPlatform, providerVersion *models.TerraformProviderVersion, provider *models.TerraformProvider) string {
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
