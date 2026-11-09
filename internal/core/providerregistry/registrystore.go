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

// RegistryStore encapsulates object-store access for the Terraform provider registry; it lives in core so the
// docs work-item consumer can share one store with the service without an import cycle.
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
	UploadProviderVersionDocsPackage(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
		body io.Reader,
	) (db.RetainObjectRefFunc, string, error)
	UploadProviderVersionDocFile(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
		category models.TerraformProviderDocItemCategory,
		name string,
		body io.Reader,
	) (db.RetainObjectRefFunc, string, error)
	GetProviderVersionReadme(ctx context.Context, objectKey string) (io.ReadCloser, error)
	GetProviderVersionDocsPackage(ctx context.Context, providerVersion *models.TerraformProviderVersion, provider *models.TerraformProvider) (io.ReadCloser, error)
	GetProviderVersionDocFile(
		ctx context.Context,
		providerVersion *models.TerraformProviderVersion,
		provider *models.TerraformProvider,
		category models.TerraformProviderDocItemCategory,
		name string,
	) (io.ReadCloser, error)
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
	return r.getObject(ctx, objectKey)
}

// GetProviderVersionDocsPackage streams a version's uploaded docs package; taking the models rather than a
// key keeps the docs key layout inside this store.
func (r *registryStore) GetProviderVersionDocsPackage(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
) (io.ReadCloser, error) {
	return r.getObject(ctx, providerVersionDocsPackageObjectKey(providerVersion, provider))
}

// GetProviderVersionDocFile streams a single doc file; the key is derived from the stored category and
// filename, never from client input.
func (r *registryStore) GetProviderVersionDocFile(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	category models.TerraformProviderDocItemCategory,
	name string,
) (io.ReadCloser, error) {
	return r.getObject(ctx, providerVersionDocFileObjectKey(providerVersion, provider, category, name))
}

func (r *registryStore) UploadProviderPlatformBinary(
	ctx context.Context,
	providerPlatform *models.TerraformProviderPlatform,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	key := providerPlatformObjectKey(providerPlatform, providerVersion, provider)
	return r.upload(ctx, key, db.ObjectStoreRefOwnerProviderPlatform, body)
}

func (r *registryStore) UploadProviderVersionReadme(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	key := providerVersionReadmeObjectKey(providerVersion, provider)
	return r.upload(ctx, key, db.ObjectStoreRefOwnerProviderVersion, body)
}

func (r *registryStore) UploadProviderVersionSHASums(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	key := providerVersionSHASumsObjectKey(providerVersion, provider)
	return r.upload(ctx, key, db.ObjectStoreRefOwnerProviderVersion, body)
}

func (r *registryStore) UploadProviderVersionSHASumsSignature(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	key := providerVersionSHASumsSignatureObjectKey(providerVersion, provider)
	return r.upload(ctx, key, db.ObjectStoreRefOwnerProviderVersion, body)
}

func (r *registryStore) UploadProviderVersionDocsPackage(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	key := providerVersionDocsPackageObjectKey(providerVersion, provider)
	return r.upload(ctx, key, db.ObjectStoreRefOwnerProviderVersion, body)
}

func (r *registryStore) UploadProviderVersionDocFile(
	ctx context.Context,
	providerVersion *models.TerraformProviderVersion,
	provider *models.TerraformProvider,
	category models.TerraformProviderDocItemCategory,
	name string,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	key := providerVersionDocFileObjectKey(providerVersion, provider, category, name)
	return r.upload(ctx, key, db.ObjectStoreRefOwnerProviderVersion, body)
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

// getObject streams an object by key; the caller closes the returned reader.
func (r *registryStore) getObject(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	result, err := r.objectStore.GetObjectStream(ctx, objectKey, nil)
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

// upload writes an object and returns a func that links its pending ref to the owner; the ref must be linked
// within the janitor's grace period, ideally in the owning row's transaction, or the object is reclaimed as an orphan.
func (r *registryStore) upload(
	ctx context.Context,
	key string,
	owner db.ObjectStoreRefOwner,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	if err := r.objectStore.UploadObject(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return r.objectStoreRefs.LinkRef(ctx, key, owner, ownerID)
	}, key, nil
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

// providerVersionDocsPackageObjectKey returns the docs-package storage key; the single source of truth for
// the docs key layout, shared by writer and reader.
func providerVersionDocsPackageObjectKey(providerVersion *models.TerraformProviderVersion, provider *models.TerraformProvider) string {
	return fmt.Sprintf("registry/providers/%s/%s/docs-package.tar.gz", provider.Metadata.ID, providerVersion.Metadata.ID)
}

// providerVersionDocFileObjectKey returns a doc file's storage key in the uniform docs/<category>/<name> layout.
func providerVersionDocFileObjectKey(providerVersion *models.TerraformProviderVersion, provider *models.TerraformProvider, category models.TerraformProviderDocItemCategory, name string) string {
	return fmt.Sprintf("registry/providers/%s/%s/docs/%s/%s", provider.Metadata.ID, providerVersion.Metadata.ID, category, name)
}
