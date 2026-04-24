package agent

//go:generate go tool mockery --name Store --inpackage --case underscore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// Store interface encapsulates the logic for saving and retrieving agent data from object storage.
type Store interface {
	UploadToolContent(ctx context.Context, sessionID, messageID string, content json.RawMessage) error
	GetToolContent(ctx context.Context, sessionID, messageID string) (json.RawMessage, error)
	UploadTrace(ctx context.Context, sessionID, traceID string, data []byte) error
	GetTrace(ctx context.Context, sessionID, traceID string) ([]byte, error)
	SaveHistory(ctx context.Context, sessionID string, data []byte) error
	LoadHistory(ctx context.Context, sessionID string) ([]byte, error)
}

type agentStore struct {
	objectStore objectstore.ObjectStore
}

// NewAgentStore creates an instance of the Store interface.
func NewAgentStore(objectStore objectstore.ObjectStore) Store {
	return &agentStore{objectStore: objectStore}
}

// UploadToolContent uploads tool response content to object storage.
func (s *agentStore) UploadToolContent(ctx context.Context, sessionID, messageID string, content json.RawMessage) error {
	return s.objectStore.UploadObject(ctx, getToolContentKey(sessionID, messageID), bytes.NewReader(content))
}

// GetToolContent retrieves tool response content from object storage.
func (s *agentStore) GetToolContent(ctx context.Context, sessionID, messageID string) (json.RawMessage, error) {
	reader, err := s.objectStore.GetObjectStream(ctx, getToolContentKey(sessionID, messageID), nil)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func getToolContentKey(sessionID, messageID string) string {
	return fmt.Sprintf("agent-sessions/%s/messages/%s/tool-content", sessionID, messageID)
}

// UploadTrace uploads trace data to object storage.
func (s *agentStore) UploadTrace(ctx context.Context, sessionID, traceID string, data []byte) error {
	return s.objectStore.UploadObject(ctx, getTraceKey(sessionID, traceID), bytes.NewReader(data))
}

// GetTrace retrieves trace data from object storage.
func (s *agentStore) GetTrace(ctx context.Context, sessionID, traceID string) ([]byte, error) {
	reader, err := s.objectStore.GetObjectStream(ctx, getTraceKey(sessionID, traceID), nil)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func getTraceKey(sessionID, traceID string) string {
	return fmt.Sprintf("agent-sessions/%s/traces/%s", sessionID, traceID)
}

// SaveHistory saves conversation history to object storage.
func (s *agentStore) SaveHistory(ctx context.Context, sessionID string, data []byte) error {
	return s.objectStore.UploadObject(ctx, getHistoryKey(sessionID), bytes.NewReader(data))
}

// LoadHistory loads conversation history from object storage. Returns nil, nil if not found.
func (s *agentStore) LoadHistory(ctx context.Context, sessionID string) ([]byte, error) {
	exists, err := s.objectStore.DoesObjectExist(ctx, getHistoryKey(sessionID))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	reader, err := s.objectStore.GetObjectStream(ctx, getHistoryKey(sessionID), nil)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func getHistoryKey(sessionID string) string {
	return fmt.Sprintf("agent-sessions/%s/history", sessionID)
}
