package email

//go:generate go tool mockery --name Store --inpackage --case underscore

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// Store saves and loads the rendered email payload (the marshaled builder) in object storage.
type Store interface {
	// UploadPayload stores the email payload and returns a retain callback.
	UploadPayload(ctx context.Context, data []byte) (db.RetainObjectRefFunc, string, error)
	// GetPayload fetches a previously uploaded email payload by key.
	GetPayload(ctx context.Context, key string) ([]byte, error)
}

type store struct {
	objectStore     objectstore.ObjectStore
	objectStoreRefs db.ObjectStoreRefs
}

// NewStore creates an email payload store.
func NewStore(objectStore objectstore.ObjectStore, objectStoreRefs db.ObjectStoreRefs) Store {
	return &store{objectStore: objectStore, objectStoreRefs: objectStoreRefs}
}

func (s *store) UploadPayload(ctx context.Context, data []byte) (db.RetainObjectRefFunc, string, error) {
	key := payloadObjectKey(uuid.New().String())
	if err := s.objectStore.UploadObject(ctx, key, bytes.NewReader(data)); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return s.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerEmailOutboxItem, ownerID)
	}, key, nil
}

func (s *store) GetPayload(ctx context.Context, key string) ([]byte, error) {
	result, err := s.objectStore.GetObjectStream(ctx, key, nil)
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

func payloadObjectKey(id string) string {
	return fmt.Sprintf("emails/%s/payload.msgpack", id)
}
