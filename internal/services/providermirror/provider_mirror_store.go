package providermirror

//go:generate go tool mockery --name TerraformProviderMirrorStore --inpackage --case underscore

import (
	"context"
	"fmt"
	"io"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// TerraformProviderMirrorStore is the interface for the terraform provider mirror store
type TerraformProviderMirrorStore interface {
	GetProviderPlatformPackagePresignedURL(ctx context.Context, platformMirrorID string) (string, error)
	UploadProviderPlatformPackage(ctx context.Context, platformMirrorID string, body io.Reader) error
}

type terraformProviderMirrorStore struct {
	objectStore objectstore.ObjectStore
}

// NewProviderMirrorStore creates an instance of the TerraformProviderMirrorStore interface
func NewProviderMirrorStore(objectStore objectstore.ObjectStore) TerraformProviderMirrorStore {
	return &terraformProviderMirrorStore{objectStore: objectStore}
}

// GetProviderPlatformPackagePresignedURL returns the presigned URL to download a provider package.
func (t *terraformProviderMirrorStore) GetProviderPlatformPackagePresignedURL(ctx context.Context, platformMirrorID string) (string, error) {
	return t.objectStore.GetPresignedURL(ctx, getPackageObjectKey(platformMirrorID))
}

// UploadProviderPlatformPackage uploads the terraform provider platform package.
func (t *terraformProviderMirrorStore) UploadProviderPlatformPackage(ctx context.Context, platformMirrorID string, body io.Reader) error {
	return t.objectStore.UploadObject(ctx, getPackageObjectKey(platformMirrorID), body)
}

// getPackageObjectKey returns the object key for the platform package.
func getPackageObjectKey(platformMirrorID string) string {
	return fmt.Sprintf("provider-mirror/providers/%s.zip", platformMirrorID)
}
