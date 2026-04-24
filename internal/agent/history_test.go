package agent

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHistoryRepository_Load(t *testing.T) {
	mockStore := NewMockStore(t)

	data := []byte(`{"version":3,"messages":[]}`)
	mockStore.On("LoadHistory", mock.Anything, "session-1").Return(data, nil)

	repo := newHistoryRepository(mockStore)
	result, err := repo.Load(context.Background(), "session-1")
	require.Nil(t, err)
	assert.NotNil(t, result)
}

func TestHistoryRepository_LoadNil(t *testing.T) {
	mockStore := NewMockStore(t)
	mockStore.On("LoadHistory", mock.Anything, "session-1").Return(nil, nil)

	repo := newHistoryRepository(mockStore)
	result, err := repo.Load(context.Background(), "session-1")
	require.Nil(t, err)
	assert.Nil(t, result)
}

func TestHistoryRepository_Save(t *testing.T) {
	mockStore := NewMockStore(t)
	mockStore.On("SaveHistory", mock.Anything, "session-1", mock.Anything).Return(nil)

	repo := newHistoryRepository(mockStore)
	err := repo.Save(context.Background(), "session-1", &gollem.History{})
	require.Nil(t, err)
}

func TestHistoryRepository_LoadCorruptedData(t *testing.T) {
	mockStore := NewMockStore(t)
	mockStore.On("LoadHistory", mock.Anything, "session-1").Return([]byte(`not json`), nil)

	repo := newHistoryRepository(mockStore)
	_, err := repo.Load(context.Background(), "session-1")
	assert.NotNil(t, err)
}
