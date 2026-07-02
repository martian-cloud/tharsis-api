package logstream

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

func TestReadRange(t *testing.T) {
	const content = "this is a test"

	tests := []struct {
		retErr       error
		name         string
		expectedLogs string
		offset       int
		length       int
		expectErr    bool
	}{
		{
			name:         "read all",
			offset:       0,
			length:       100,
			expectedLogs: content,
		},
		{
			name:         "read partial",
			offset:       5,
			length:       100,
			expectedLogs: "is a test",
		},
		{
			name:      "zero length returns empty without reading",
			offset:    0,
			length:    0,
			expectErr: false,
		},
		{
			name:      "not found error is propagated",
			offset:    0,
			length:    100,
			retErr:    errors.New("Not Found", errors.WithErrorCode(errors.ENotFound)),
			expectErr: true,
		},
		{
			name:      "unexpected error is propagated",
			offset:    0,
			length:    100,
			retErr:    errors.New("internal error"),
			expectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockObjectStore := objectstore.MockObjectStore{}

			if test.length > 0 {
				mockObjectStore.On("GetObjectStream", mock.Anything, mock.Anything, mock.Anything).Return(
					func(_ context.Context, _ string, option *objectstore.DownloadOptions) (*objectstore.GetObjectStreamOutput, error) {
						if test.retErr != nil {
							return nil, test.retErr
						}

						out := content
						if option != nil && option.ContentRange != nil {
							selection := strings.Split(strings.Split(*option.ContentRange, "=")[1], "-")
							start, _ := strconv.Atoi(selection[0])
							end, _ := strconv.Atoi(selection[1]) // inclusive
							if start > len(out) {
								start = len(out)
							}
							if end+1 > len(out) {
								end = len(out) - 1
							}
							out = out[start : end+1]
						}
						return &objectstore.GetObjectStreamOutput{
							Body:          io.NopCloser(strings.NewReader(out)),
							ContentLength: int64(len(out)),
						}, nil
					},
				)
			}

			logStore := NewLogStore(&mockObjectStore)

			reader, err := logStore.ReadRange(ctx, "logstreams/stream-1/chunk-1.txt", test.offset, test.length)
			if err != nil {
				assert.True(t, test.expectErr, "Error was not expected: %v", err)
				return
			}

			logs, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.NoError(t, reader.Close())

			assert.False(t, test.expectErr, "An error was expected")
			mockObjectStore.AssertExpectations(t)
			assert.Equal(t, test.expectedLogs, string(logs))
		})
	}
}

func TestWriteChunk(t *testing.T) {
	tests := []struct {
		retErr         error
		name           string
		existingLogs   string
		logsToUpload   string
		expectedUpload string
		byteOffset     int
		expectErr      bool
		expectedCode   errors.CodeType
	}{
		{
			name:           "write new chunk object",
			existingLogs:   "",
			logsToUpload:   "this is a test",
			expectedUpload: "this is a test",
		},
		{
			name:           "append to existing chunk",
			existingLogs:   "this",
			logsToUpload:   " is a test",
			expectedUpload: "this is a test",
			byteOffset:     len("this"),
		},
		{
			// byteOffset is the committed chunk size, so an object shorter than it is object-store data
			// loss, not a client gap: it must surface as an internal error.
			name:         "object shorter than committed size is an internal error",
			existingLogs: "this",
			logsToUpload: " is a test",
			byteOffset:   100,
			expectErr:    true,
			expectedCode: errors.EInternal,
		},
		{
			name:           "overwrite truncates chunk",
			existingLogs:   "this is a test that will be truncated",
			logsToUpload:   " is a test",
			expectedUpload: "this is a test",
			byteOffset:     len("this"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// WriteChunk reads the preserved prefix [0, byteOffset) via a ranged GetObjectStream. The
			// store returns only the bytes that actually exist in the range, so a too-large offset yields
			// a short read (or ENotFound for an empty/missing object).
			mockGetResult := func(_ context.Context, _ string, opts *objectstore.DownloadOptions) (*objectstore.GetObjectStreamOutput, error) {
				if test.existingLogs == "" {
					return nil, errors.New("Not Found", errors.WithErrorCode(errors.ENotFound))
				}
				data := []byte(test.existingLogs)
				if opts != nil && opts.ContentRange != nil {
					selection := strings.Split(strings.Split(*opts.ContentRange, "=")[1], "-")
					start, _ := strconv.Atoi(selection[0])
					end, _ := strconv.Atoi(selection[1]) // inclusive
					if start > len(data) {
						start = len(data)
					}
					if end+1 > len(data) {
						end = len(data) - 1
					}
					data = data[start : end+1]
				}
				return &objectstore.GetObjectStreamOutput{
					Body:          io.NopCloser(bytes.NewReader(data)),
					ContentLength: int64(len(data)),
				}, nil
			}

			mockObjectStore := objectstore.MockObjectStore{}
			mockObjectStore.On("GetObjectStream", mock.Anything, mock.Anything, mock.Anything).Return(mockGetResult).Maybe()

			bodyMatcher := mock.MatchedBy(func(r io.Reader) bool {
				body, _ := io.ReadAll(r)
				return string(body) == test.expectedUpload
			})
			mockObjectStore.On("UploadObject", mock.Anything, mock.Anything, bodyMatcher).Return(test.retErr).Maybe()

			logStore := NewLogStore(&mockObjectStore)

			err := logStore.WriteChunk(ctx, "logstreams/stream-1/chunk-1.txt", test.byteOffset, []byte(test.logsToUpload))
			if err != nil {
				assert.True(t, test.expectErr, "Error was not expected %v", err)
				if test.expectedCode != "" {
					assert.Equal(t, test.expectedCode, errors.ErrorCode(err))
				}
				return
			}

			require.False(t, test.expectErr, "An error was expected")
			mockObjectStore.AssertExpectations(t)
		})
	}
}

func TestWriteObject(t *testing.T) {
	t.Run("streams the reader through to the object store", func(t *testing.T) {
		ctx := context.Background()

		mockObjectStore := objectstore.MockObjectStore{}
		bodyMatcher := mock.MatchedBy(func(r io.Reader) bool {
			body, _ := io.ReadAll(r)
			return string(body) == "consolidated logs"
		})
		mockObjectStore.On("UploadObject", mock.Anything, "logstreams/stream-1.txt", bodyMatcher).Return(nil)

		logStore := NewLogStore(&mockObjectStore)
		err := logStore.WriteObject(ctx, "logstreams/stream-1.txt", strings.NewReader("consolidated logs"))
		require.NoError(t, err)
		mockObjectStore.AssertExpectations(t)
	})

	t.Run("propagates an upload error", func(t *testing.T) {
		ctx := context.Background()

		mockObjectStore := objectstore.MockObjectStore{}
		mockObjectStore.On("UploadObject", mock.Anything, mock.Anything, mock.Anything).
			Return(errors.New("object store down"))

		logStore := NewLogStore(&mockObjectStore)
		err := logStore.WriteObject(ctx, "logstreams/stream-1.txt", strings.NewReader("data"))
		require.Error(t, err)
	})
}

func TestWriteChunkEdgeCases(t *testing.T) {
	t.Run("a zero-length write at offset 0 uploads an empty object without reading a prefix", func(t *testing.T) {
		ctx := context.Background()

		mockObjectStore := objectstore.MockObjectStore{}
		bodyMatcher := mock.MatchedBy(func(r io.Reader) bool {
			body, _ := io.ReadAll(r)
			return len(body) == 0
		})
		mockObjectStore.On("UploadObject", mock.Anything, mock.Anything, bodyMatcher).Return(nil)

		logStore := NewLogStore(&mockObjectStore)
		err := logStore.WriteChunk(ctx, "logstreams/stream-1/c0.txt", 0, []byte{})
		require.NoError(t, err)
		// byteOffset 0 skips the prefix read entirely.
		mockObjectStore.AssertNotCalled(t, "GetObjectStream", mock.Anything, mock.Anything, mock.Anything)
		mockObjectStore.AssertExpectations(t)
	})

	t.Run("a write at offset 0 replaces the whole object without reading the existing content", func(t *testing.T) {
		ctx := context.Background()

		mockObjectStore := objectstore.MockObjectStore{}
		// The replacement is exactly the new buffer — no prefix is preserved when byteOffset is 0.
		bodyMatcher := mock.MatchedBy(func(r io.Reader) bool {
			body, _ := io.ReadAll(r)
			return string(body) == "fresh"
		})
		mockObjectStore.On("UploadObject", mock.Anything, mock.Anything, bodyMatcher).Return(nil)

		logStore := NewLogStore(&mockObjectStore)
		err := logStore.WriteChunk(ctx, "logstreams/stream-1/c0.txt", 0, []byte("fresh"))
		require.NoError(t, err)
		mockObjectStore.AssertNotCalled(t, "GetObjectStream", mock.Anything, mock.Anything, mock.Anything)
		mockObjectStore.AssertExpectations(t)
	})
}

func TestReadRangeNegativeArgs(t *testing.T) {
	ctx := context.Background()

	t.Run("negative offset is rejected without touching the object store", func(t *testing.T) {
		mockObjectStore := objectstore.MockObjectStore{}
		logStore := NewLogStore(&mockObjectStore)

		_, err := logStore.ReadRange(ctx, "logstreams/stream-1/c0.txt", -1, 10)
		require.Error(t, err)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
		mockObjectStore.AssertNotCalled(t, "GetObjectStream", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("negative length is rejected without touching the object store", func(t *testing.T) {
		mockObjectStore := objectstore.MockObjectStore{}
		logStore := NewLogStore(&mockObjectStore)

		_, err := logStore.ReadRange(ctx, "logstreams/stream-1/c0.txt", 0, -1)
		require.Error(t, err)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
		mockObjectStore.AssertNotCalled(t, "GetObjectStream", mock.Anything, mock.Anything, mock.Anything)
	})
}
