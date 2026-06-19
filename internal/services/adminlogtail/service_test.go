package adminlogtail

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger/logstore"
)

// adminCallerContext attaches an admin (or non-admin) caller to ctx.
func adminCallerContext(ctx context.Context, isAdmin bool) context.Context {
	mockCaller := &auth.MockCaller{}
	mockCaller.On("IsAdminModeActivated", mock.Anything).Return(isAdmin).Maybe()
	return auth.WithCaller(ctx, mockCaller)
}

func TestGetEntries(t *testing.T) {
	t.Run("no caller is unauthorized", func(t *testing.T) {
		store := logstore.NewMockStore(t)
		svc := NewService(store)

		_, err := svc.GetEntries(t.Context(), &GetEntriesInput{})
		assert.Equal(t, errors.EUnauthorized, errors.ErrorCode(err))
	})

	t.Run("non-admin is forbidden", func(t *testing.T) {
		store := logstore.NewMockStore(t)
		svc := NewService(store)
		ctx := adminCallerContext(t.Context(), false)

		_, err := svc.GetEntries(ctx, &GetEntriesInput{})
		assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
	})

	t.Run("admin delegates to the store", func(t *testing.T) {
		store := logstore.NewMockStore(t)
		store.On("GetEntries", []string{"INFO"}, "boom", 10).
			Return([]*logstore.LogEntry{{Seq: 1, Message: "a"}}, nil)
		svc := NewService(store)
		ctx := adminCallerContext(t.Context(), true)

		entries, err := svc.GetEntries(ctx, &GetEntriesInput{Levels: []string{"INFO"}, Search: "boom", Limit: 10})
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, "a", entries[0].Message)
	})

	t.Run("propagates store error", func(t *testing.T) {
		store := logstore.NewMockStore(t)
		store.On("GetEntries", []string(nil), "", 0).
			Return(nil, errors.New("redis down", errors.WithErrorCode(errors.EInternal)))
		svc := NewService(store)
		ctx := adminCallerContext(t.Context(), true)

		_, err := svc.GetEntries(ctx, &GetEntriesInput{})
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})
}

func TestSubscribe(t *testing.T) {
	t.Run("no caller is unauthorized", func(t *testing.T) {
		store := logstore.NewMockStore(t)
		svc := NewService(store)

		_, err := svc.Subscribe(t.Context())
		assert.Equal(t, errors.EUnauthorized, errors.ErrorCode(err))
	})

	t.Run("non-admin is forbidden", func(t *testing.T) {
		store := logstore.NewMockStore(t)
		svc := NewService(store)
		ctx := adminCallerContext(t.Context(), false)

		_, err := svc.Subscribe(ctx)
		assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
	})

	t.Run("admin delegates to the store", func(t *testing.T) {
		bidi := make(chan *logstore.LogEntry)
		var ch <-chan *logstore.LogEntry = bidi
		store := logstore.NewMockStore(t)
		store.On("Subscribe", mock.Anything).Return(ch, nil)
		svc := NewService(store)
		ctx := adminCallerContext(t.Context(), true)

		got, err := svc.Subscribe(ctx)
		require.NoError(t, err)
		assert.NotNil(t, got)
	})

	t.Run("propagates store error", func(t *testing.T) {
		store := logstore.NewMockStore(t)
		store.On("Subscribe", mock.Anything).
			Return(nil, errors.New("no stream", errors.WithErrorCode(errors.EInternal)))
		svc := NewService(store)
		ctx := adminCallerContext(t.Context(), true)

		_, err := svc.Subscribe(ctx)
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})
}
