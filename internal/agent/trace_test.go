package agent

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem/trace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTraceRepository_Save(t *testing.T) {
	mockStore := NewMockStore(t)
	mockStore.On("UploadTrace", mock.Anything, "session-1", "trace-1", mock.Anything).Return(nil)

	repo := newTraceRepository(mockStore, "session-1")
	err := repo.Save(context.Background(), &trace.Trace{TraceID: "trace-1"})
	require.Nil(t, err)
}
