// Package packageregistry contains the shared, service-agnostic logic for the generic package
// registry: object-store access for package version content and stateless read/visibility helpers
// that both the package-registry service and policy-domain services build on. It must not import
// any internal/services/* package.
package packageregistry

//go:generate go tool mockery --name PackageStore --inpackage --case underscore

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// PackageStore encapsulates object-store access for package version content (tar.gz packages).
type PackageStore interface {
	UploadPackageVersion(ctx context.Context, packageVersion *models.PackageVersion, body io.Reader) (db.RetainObjectRefFunc, string, error)
	DownloadPackageVersion(ctx context.Context, packageVersion *models.PackageVersion, writer io.WriterAt) error
	GetPackageVersionPresignedURL(ctx context.Context, packageVersion *models.PackageVersion) (string, error)
}

type packageStore struct {
	objectStore     objectstore.ObjectStore
	objectStoreRefs db.ObjectStoreRefs
}

// NewPackageStore creates an instance of the PackageStore interface.
func NewPackageStore(objectStore objectstore.ObjectStore, objectStoreRefs db.ObjectStoreRefs) PackageStore {
	return &packageStore{objectStore: objectStore, objectStoreRefs: objectStoreRefs}
}

func (r *packageStore) UploadPackageVersion(ctx context.Context, packageVersion *models.PackageVersion, body io.Reader) (db.RetainObjectRefFunc, string, error) {
	// Mint a fresh UUID-based key on every upload (including re-uploads) so each upload writes a
	// distinct object; the caller persists the returned key on the package version. Objects from
	// superseded uploads stay linked to the version and are reclaimed by the janitor when it's deleted.
	key := getPackageVersionPackageObjectKey(packageVersion.PackageID, packageVersion.Metadata.ID, uuid.New().String())
	if err := r.objectStore.UploadObject(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return r.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerPackageVersion, ownerID)
	}, key, nil
}

func (r *packageStore) DownloadPackageVersion(ctx context.Context, packageVersion *models.PackageVersion, writer io.WriterAt) error {
	return r.objectStore.DownloadObject(ctx, ptr.ToString(packageVersion.ObjectStoreKey), writer, nil)
}

func (r *packageStore) GetPackageVersionPresignedURL(ctx context.Context, packageVersion *models.PackageVersion) (string, error) {
	return r.objectStore.GetPresignedURL(ctx, ptr.ToString(packageVersion.ObjectStoreKey))
}

func getPackageVersionPackageObjectKey(packageID, packageVersionID, id string) string {
	return fmt.Sprintf("packages/%s/%s/%s/package.tar.gz", packageID, packageVersionID, id)
}
