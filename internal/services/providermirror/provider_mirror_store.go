package providermirror

//go:generate go tool mockery --name TerraformProviderMirrorStore --inpackage --case underscore

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// TerraformProviderMirrorStore is the interface for the terraform provider mirror store
type TerraformProviderMirrorStore interface {
	GetProviderPlatformPackagePresignedURL(ctx context.Context, objectStoreKey string) (string, error)
	UploadProviderPlatformPackage(ctx context.Context, body io.Reader) (db.RetainObjectRefFunc, string, error)
}

type terraformProviderMirrorStore struct {
	objectStore     objectstore.ObjectStore
	objectStoreRefs db.ObjectStoreRefs
}

// NewProviderMirrorStore creates an instance of the TerraformProviderMirrorStore interface
func NewProviderMirrorStore(objectStore objectstore.ObjectStore, objectStoreRefs db.ObjectStoreRefs) TerraformProviderMirrorStore {
	return &terraformProviderMirrorStore{objectStore: objectStore, objectStoreRefs: objectStoreRefs}
}

// GetProviderPlatformPackagePresignedURL returns the presigned URL to download a provider package.
func (t *terraformProviderMirrorStore) GetProviderPlatformPackagePresignedURL(ctx context.Context, objectStoreKey string) (string, error) {
	return t.objectStore.GetPresignedURL(ctx, objectStoreKey)
}

// UploadProviderPlatformPackage uploads the terraform provider platform package and returns a retain
// callback, the generated object key, and any error.
func (t *terraformProviderMirrorStore) UploadProviderPlatformPackage(ctx context.Context, body io.Reader) (db.RetainObjectRefFunc, string, error) {
	key := providerMirrorPlatformObjectKey(uuid.New().String())
	if err := t.objectStore.UploadObject(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return t.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerProviderMirrorPlatform, ownerID)
	}, key, nil
}

func providerMirrorPlatformObjectKey(id string) string {
	return fmt.Sprintf("provider-mirror/providers/%s.zip", id)
}
