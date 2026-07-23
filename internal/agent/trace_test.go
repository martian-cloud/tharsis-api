package agent

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem/trace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
)

func TestTraceRepository_Save(t *testing.T) {
	mockRefs := db.NewMockObjectStoreRefs(t)
	mockRefs.On("LinkRef", mock.Anything, traceObjectKey("session-1", "trace-1"), db.ObjectStoreRefOwnerAgentSession, "session-1").Return(nil)

	mockStore := NewMockStore(t)
	mockStore.On("UploadTrace", mock.Anything, "session-1", "trace-1", mock.Anything).Return(db.RetainObjectRefFunc(func(ctx context.Context, ownerID string) error {
		return mockRefs.LinkRef(ctx, traceObjectKey("session-1", "trace-1"), db.ObjectStoreRefOwnerAgentSession, ownerID)
	}), traceObjectKey("session-1", "trace-1"), nil)

	repo := newTraceRepository(mockStore, "session-1")
	err := repo.Save(context.Background(), &trace.Trace{TraceID: "trace-1"})
	require.Nil(t, err)
}
