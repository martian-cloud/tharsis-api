package moduleregistry

//go:generate go tool mockery --name RegistryStore --inpackage --case underscore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// RegistryStore interface encapsulates the logic for saving workspace registrys
type RegistryStore interface {
	UploadModulePackage(
		ctx context.Context,
		moduleVersion *models.TerraformModuleVersion,
		module *models.TerraformModule,
		body io.Reader,
	) (db.RetainObjectRefFunc, string, error)
	UploadModuleConfigurationDetails(
		ctx context.Context,
		metadata *ModuleConfigurationDetails,
		moduleVersion *models.TerraformModuleVersion,
		module *models.TerraformModule,
	) (db.RetainObjectRefFunc, string, error)
	GetModuleConfigurationDetails(
		ctx context.Context,
		moduleVersion *models.TerraformModuleVersion,
		module *models.TerraformModule,
		path string,
	) (io.ReadCloser, error)
	DownloadModulePackage(ctx context.Context, objectKey string, writer io.WriterAt) error
	GetModulePackagePresignedURL(ctx context.Context, objectKey string) (string, error)
}

type registryStore struct {
	objectStore     objectstore.ObjectStore
	objectStoreRefs db.ObjectStoreRefs
}

// NewRegistryStore creates an instance of the RegistryStore interface
func NewRegistryStore(objectStore objectstore.ObjectStore, objectStoreRefs db.ObjectStoreRefs) RegistryStore {
	return &registryStore{objectStore: objectStore, objectStoreRefs: objectStoreRefs}
}

func (r *registryStore) GetModuleConfigurationDetails(
	ctx context.Context,
	moduleVersion *models.TerraformModuleVersion,
	module *models.TerraformModule,
	path string,
) (io.ReadCloser, error) {
	result, err := r.objectStore.GetObjectStream(
		ctx,
		moduleConfigurationDetailsObjectKey(path, moduleVersion, module),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func (r *registryStore) UploadModulePackage(
	ctx context.Context,
	moduleVersion *models.TerraformModuleVersion,
	module *models.TerraformModule,
	body io.Reader,
) (db.RetainObjectRefFunc, string, error) {
	key := modulePackageObjectKey(moduleVersion, module)
	if err := r.objectStore.UploadObject(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return r.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerModuleVersion, ownerID)
	}, key, nil
}

func (r *registryStore) UploadModuleConfigurationDetails(
	ctx context.Context,
	metadata *ModuleConfigurationDetails,
	moduleVersion *models.TerraformModuleVersion,
	module *models.TerraformModule,
) (db.RetainObjectRefFunc, string, error) {
	serializedMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, "", err
	}

	key := moduleConfigurationDetailsObjectKey(metadata.Path, moduleVersion, module)
	if err := r.objectStore.UploadObject(ctx, key, bytes.NewReader(serializedMetadata)); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return r.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerModuleVersion, ownerID)
	}, key, nil
}

func (r *registryStore) DownloadModulePackage(ctx context.Context, objectKey string, writer io.WriterAt) error {
	return r.objectStore.DownloadObject(ctx, objectKey, writer, nil)
}

func (r *registryStore) GetModulePackagePresignedURL(ctx context.Context, objectKey string) (string, error) {
	return r.objectStore.GetPresignedURL(ctx, objectKey)
}

func moduleConfigurationDetailsObjectKey(path string, moduleVersion *models.TerraformModuleVersion, module *models.TerraformModule) string {
	return fmt.Sprintf("registry/modules/%s/%s/metadata/%s", module.Metadata.ID, moduleVersion.Metadata.ID, path)
}

func modulePackageObjectKey(moduleVersion *models.TerraformModuleVersion, module *models.TerraformModule) string {
	return fmt.Sprintf(
		"registry/modules/%s/%s/package.tar.gz",
		module.Metadata.ID,
		moduleVersion.Metadata.ID,
	)
}
