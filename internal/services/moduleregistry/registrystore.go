package moduleregistry

//go:generate mockery --name RegistryStore --inpackage --case underscore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/objectstore"
)

// RegistryStore interface encapsulates the logic for saving workspace registrys
type RegistryStore interface {
	UploadModulePackage(
		ctx context.Context,
		moduleVersion *models.TerraformModuleVersion,
		module *models.TerraformModule,
		body io.Reader,
	) error
	UploadModuleConfigurationDetails(
		ctx context.Context,
		metadata *ModuleConfigurationDetails,
		moduleVersion *models.TerraformModuleVersion,
		module *models.TerraformModule,
	) error
	GetModuleConfigurationDetails(
		ctx context.Context,
		moduleVersion *models.TerraformModuleVersion,
		module *models.TerraformModule,
		path string,
	) (io.ReadCloser, error)
	DownloadModulePackage(
		ctx context.Context,
		moduleVersion *models.TerraformModuleVersion,
		module *models.TerraformModule,
		writer io.WriterAt,
	) error
	GetModulePackagePresignedURL(
		ctx context.Context,
		moduleVersion *models.TerraformModuleVersion,
		module *models.TerraformModule,
	) (string, error)
}

type registryStore struct {
	objectStore objectstore.ObjectStore
}

// NewRegistryStore creates an instance of the RegistryStore interface
func NewRegistryStore(objectStore objectstore.ObjectStore) RegistryStore {
	return &registryStore{objectStore: objectStore}
}

func (r *registryStore) GetModuleConfigurationDetails(
	ctx context.Context,
	moduleVersion *models.TerraformModuleVersion,
	module *models.TerraformModule,
	path string,
) (io.ReadCloser, error) {
	return r.objectStore.GetObjectStream(
		ctx,
		getModuleConfigurationDetailsObjectKey(path, moduleVersion, module),
		nil,
	)
}

func (r *registryStore) UploadModulePackage(
	ctx context.Context,
	moduleVersion *models.TerraformModuleVersion,
	module *models.TerraformModule,
	body io.Reader,
) error {
	return r.upload(ctx, getModulePackageObjectKey(moduleVersion, module), body)
}

func (r *registryStore) UploadModuleConfigurationDetails(
	ctx context.Context,
	metadata *ModuleConfigurationDetails,
	moduleVersion *models.TerraformModuleVersion,
	module *models.TerraformModule,
) error {
	serializedMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	reader := bytes.NewReader(serializedMetadata)
	return r.upload(ctx, getModuleConfigurationDetailsObjectKey(metadata.Path, moduleVersion, module), reader)
}

func (r *registryStore) DownloadModulePackage(
	ctx context.Context,
	moduleVersion *models.TerraformModuleVersion,
	module *models.TerraformModule,
	writer io.WriterAt,
) error {
	return r.download(
		ctx,
		getModulePackageObjectKey(moduleVersion, module),
		writer,
	)
}

func (r *registryStore) GetModulePackagePresignedURL(
	ctx context.Context,
	moduleVersion *models.TerraformModuleVersion,
	module *models.TerraformModule,
) (string, error) {
	return r.objectStore.GetPresignedURL(ctx, getModulePackageObjectKey(moduleVersion, module))
}

func (r *registryStore) upload(ctx context.Context, key string, body io.Reader) error {
	return r.objectStore.UploadObject(ctx, key, body)
}

func (r *registryStore) download(ctx context.Context, key string, writer io.WriterAt) error {
	return r.objectStore.DownloadObject(
		ctx,
		key,
		writer,
		nil,
	)
}

func getModuleConfigurationDetailsObjectKey(path string, moduleVersion *models.TerraformModuleVersion, module *models.TerraformModule) string {
	return fmt.Sprintf("registry/modules/%s/%s/metadata/%s", module.Metadata.ID, moduleVersion.Metadata.ID, path)
}

func getModulePackageObjectKey(moduleVersion *models.TerraformModuleVersion, module *models.TerraformModule) string {
	return fmt.Sprintf(
		"registry/modules/%s/%s/package.tar.gz",
		module.Metadata.ID,
		moduleVersion.Metadata.ID,
	)
}
