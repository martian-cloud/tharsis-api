package cli

import (
	"context"
	"fmt"
	"io"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/objectstore"
)

//go:generate mockery --name TerraformCLIStore --inpackage --case underscore

// TerraformCLIStore interface encapsulates the logic for saving Terraform CLI binaries.
type TerraformCLIStore interface {
	CreateTerraformCLIBinaryPresignedURL(ctx context.Context, version, os, architecture string) (string, error)
	UploadTerraformCLIBinary(ctx context.Context, version, os, architecture string, body io.Reader) error
	DoesTerraformCLIBinaryExist(ctx context.Context, version, os, architecture string) (bool, error)
}

type terraformCLIStore struct {
	objectStore objectstore.ObjectStore
}

// NewCLIStore creates an instance of the CLIStore interface
func NewCLIStore(objectStore objectstore.ObjectStore) TerraformCLIStore {
	return &terraformCLIStore{objectStore: objectStore}
}

// CreateTerraformCLIPresignedURL creates a presigned URL that can
// be used to download a specific version of the Terraform CLI directly
// from object storage.
func (c *terraformCLIStore) CreateTerraformCLIBinaryPresignedURL(ctx context.Context, version, os, architecture string) (string, error) {
	return c.objectStore.GetPresignedURL(ctx, getTerraformCLIBinaryObjectKey(version, os, architecture))
}

// UploadTerraformCLI uploads the respective Terraform CLI binary to object storage.
func (c *terraformCLIStore) UploadTerraformCLIBinary(ctx context.Context, version, os, architecture string, body io.Reader) error {
	return c.objectStore.UploadObject(ctx, getTerraformCLIBinaryObjectKey(version, os, architecture), body)
}

// DoesTerraformCLIBinaryExist returns a boolean indicating
// the existence of a Terraform CLI binary in object storage.
func (c *terraformCLIStore) DoesTerraformCLIBinaryExist(ctx context.Context, version, os, architecture string) (bool, error) {
	return c.objectStore.DoesObjectExist(ctx, getTerraformCLIBinaryObjectKey(version, os, architecture))
}

func getTerraformCLIBinaryObjectKey(version, os, architecture string) string {
	return fmt.Sprintf(
		"terraform/%s/terraform_%s_%s_%s.zip",
		version,
		version,
		os,
		architecture,
	)
}
