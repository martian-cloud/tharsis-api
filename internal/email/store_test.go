package email

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

func TestStoreUploadPayload(t *testing.T) {
	payload := []byte(`{"hello":"world"}`)

	t.Run("uploads the payload and returns a retain func that links the ref", func(t *testing.T) {
		ctx := t.Context()

		mockObjectStore := objectstore.NewMockObjectStore(t)
		mockRefs := db.NewMockObjectStoreRefs(t)

		var uploadedKey string
		mockObjectStore.On("UploadObject", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
			Run(func(args mock.Arguments) {
				uploadedKey = args.Get(1).(string)
				body, err := io.ReadAll(args.Get(2).(io.Reader))
				require.NoError(t, err)
				assert.Equal(t, payload, body)
			}).Return(nil)

		s := NewStore(mockObjectStore, mockRefs)

		retain, key, err := s.UploadPayload(ctx, payload)
		require.NoError(t, err)

		// The returned key matches the key uploaded and follows the emails/<id>/payload.msgpack layout.
		assert.Equal(t, uploadedKey, key)
		assert.True(t, strings.HasPrefix(key, "emails/"))
		assert.True(t, strings.HasSuffix(key, "/payload.msgpack"))
		require.NotNil(t, retain)

		// The retain func links the uploaded object to the owning outbox under the same key.
		mockRefs.On("LinkRef", mock.Anything, key, db.ObjectStoreRefOwnerEmailOutboxItem, "outbox-1").Return(nil)
		require.NoError(t, retain(ctx, "outbox-1"))
	})

	t.Run("upload error is returned without a retain func", func(t *testing.T) {
		ctx := t.Context()

		mockObjectStore := objectstore.NewMockObjectStore(t)
		mockRefs := db.NewMockObjectStoreRefs(t)

		mockObjectStore.On("UploadObject", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("boom"))

		s := NewStore(mockObjectStore, mockRefs)

		retain, key, err := s.UploadPayload(ctx, payload)
		assert.Error(t, err)
		assert.Nil(t, retain)
		assert.Empty(t, key)
	})

	t.Run("retain func propagates a LinkRef error", func(t *testing.T) {
		ctx := t.Context()

		mockObjectStore := objectstore.NewMockObjectStore(t)
		mockRefs := db.NewMockObjectStoreRefs(t)

		mockObjectStore.On("UploadObject", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockRefs.On("LinkRef", mock.Anything, mock.Anything, db.ObjectStoreRefOwnerEmailOutboxItem, "outbox-1").
			Return(errors.New("link failed"))

		s := NewStore(mockObjectStore, mockRefs)

		retain, _, err := s.UploadPayload(ctx, payload)
		require.NoError(t, err)

		assert.Error(t, retain(ctx, "outbox-1"))
	})
}

func TestStoreGetPayload(t *testing.T) {
	const key = "emails/abc/payload.msgpack"

	t.Run("returns the object body", func(t *testing.T) {
		ctx := t.Context()

		mockObjectStore := objectstore.NewMockObjectStore(t)
		mockRefs := db.NewMockObjectStoreRefs(t)

		want := []byte(`{"hello":"world"}`)
		mockObjectStore.On("GetObjectStream", mock.Anything, key, (*objectstore.DownloadOptions)(nil)).
			Return(&objectstore.GetObjectStreamOutput{Body: io.NopCloser(bytes.NewReader(want))}, nil)

		s := NewStore(mockObjectStore, mockRefs)

		got, err := s.GetPayload(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("propagates a stream error", func(t *testing.T) {
		ctx := t.Context()

		mockObjectStore := objectstore.NewMockObjectStore(t)
		mockRefs := db.NewMockObjectStoreRefs(t)

		mockObjectStore.On("GetObjectStream", mock.Anything, key, (*objectstore.DownloadOptions)(nil)).
			Return(nil, errors.New("not found"))

		s := NewStore(mockObjectStore, mockRefs)

		got, err := s.GetPayload(ctx, key)
		assert.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestPayloadObjectKey(t *testing.T) {
	assert.Equal(t, "emails/abc-123/payload.msgpack", payloadObjectKey("abc-123"))
}
