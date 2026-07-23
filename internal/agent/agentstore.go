package agent

//go:generate go tool mockery --name Store --inpackage --case underscore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/uuid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// Store interface encapsulates the logic for saving and retrieving agent data from object storage.
type Store interface {
	UploadToolContent(ctx context.Context, sessionID string, content json.RawMessage) (db.RetainObjectRefFunc, string, error)
	GetToolContentForMessage(ctx context.Context, storedKey string) (json.RawMessage, error)
	UploadTrace(ctx context.Context, sessionID, traceID string, data []byte) (db.RetainObjectRefFunc, string, error)
	GetTrace(ctx context.Context, sessionID, traceID string) ([]byte, error)
	SaveHistory(ctx context.Context, sessionID string, data []byte) (db.RetainObjectRefFunc, string, error)
	LoadHistory(ctx context.Context, sessionID string) ([]byte, error)
}

type agentStore struct {
	objectStore     objectstore.ObjectStore
	objectStoreRefs db.ObjectStoreRefs
}

// NewAgentStore creates an instance of the Store interface.
func NewAgentStore(objectStore objectstore.ObjectStore, objectStoreRefs db.ObjectStoreRefs) Store {
	return &agentStore{objectStore: objectStore, objectStoreRefs: objectStoreRefs}
}

// UploadToolContent uploads tool response content to object storage and returns a link callback
// (to invoke inside a TX after the owning message row is created) and the generated object key.
func (s *agentStore) UploadToolContent(ctx context.Context, sessionID string, content json.RawMessage) (db.RetainObjectRefFunc, string, error) {
	key := toolContentObjectKey(sessionID, uuid.New().String())
	if err := s.objectStore.UploadObject(ctx, key, bytes.NewReader(content)); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return s.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerAgentSession, ownerID)
	}, key, nil
}

// GetToolContentForMessage retrieves tool response content for the given message.
func (s *agentStore) GetToolContentForMessage(ctx context.Context, storedKey string) (json.RawMessage, error) {
	result, err := s.objectStore.GetObjectStream(ctx, storedKey, nil)
	if err != nil {
		return nil, err
	}

	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

// UploadTrace uploads trace data to object storage and returns a link callback and the object key.
func (s *agentStore) UploadTrace(ctx context.Context, sessionID, traceID string, data []byte) (db.RetainObjectRefFunc, string, error) {
	key := traceObjectKey(sessionID, traceID)
	if err := s.objectStore.UploadObject(ctx, key, bytes.NewReader(data)); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return s.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerAgentSession, ownerID)
	}, key, nil
}

// GetTrace retrieves trace data from object storage.
func (s *agentStore) GetTrace(ctx context.Context, sessionID, traceID string) ([]byte, error) {
	result, err := s.objectStore.GetObjectStream(ctx, traceObjectKey(sessionID, traceID), nil)
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

// SaveHistory saves conversation history to object storage and returns a retain callback and the object key.
func (s *agentStore) SaveHistory(ctx context.Context, sessionID string, data []byte) (db.RetainObjectRefFunc, string, error) {
	key := historyObjectKey(sessionID)
	if err := s.objectStore.UploadObject(ctx, key, bytes.NewReader(data)); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return s.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerAgentSession, ownerID)
	}, key, nil
}

// LoadHistory loads conversation history from object storage. Returns nil, nil if not found.
func (s *agentStore) LoadHistory(ctx context.Context, sessionID string) ([]byte, error) {
	exists, err := s.objectStore.DoesObjectExist(ctx, historyObjectKey(sessionID))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	result, err := s.objectStore.GetObjectStream(ctx, historyObjectKey(sessionID), nil)
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

func toolContentObjectKey(sessionID, id string) string {
	return fmt.Sprintf("agent-sessions/%s/tool-content/%s", sessionID, id)
}

func traceObjectKey(sessionID, traceID string) string {
	return fmt.Sprintf("agent-sessions/%s/traces/%s", sessionID, traceID)
}

func historyObjectKey(sessionID string) string {
	return fmt.Sprintf("agent-sessions/%s/history", sessionID)
}
