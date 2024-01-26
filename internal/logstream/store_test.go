package logstream

import (
	"context"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

func TestReadLogsFromStore(t *testing.T) {
	logStreamID := "logstream-1"

	// Test cases
	tests := []struct {
		retErr       error
		name         string
		expectedLogs string
		startOffset  int
		logStream    *models.LogStream
		expectErr    bool
	}{
		{
			name:         "get all logs",
			expectedLogs: "this is a test",
			expectErr:    false,
			logStream: &models.LogStream{
				Metadata: models.ResourceMetadata{
					ID: logStreamID,
				},
			},
		},
		{
			name:         "get partial logs",
			startOffset:  5,
			expectedLogs: "is a test",
			expectErr:    false,
			logStream: &models.LogStream{
				Metadata: models.ResourceMetadata{
					ID: logStreamID,
				},
			},
		},
		{
			name: "logs not found",
			retErr: errors.New(
				"Not Found",
				errors.WithErrorCode(errors.ENotFound),
			),
			expectErr: false,
			logStream: &models.LogStream{
				Metadata: models.ResourceMetadata{
					ID: logStreamID,
				},
			},
		},
		{
			name: "unexpected error",
			retErr: errors.New(
				"internal error",
			),
			expectErr: true,
			logStream: &models.LogStream{
				Metadata: models.ResourceMetadata{
					ID: logStreamID,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockResult := func(_ context.Context, _ string, w io.WriterAt, option *objectstore.DownloadOptions) error {
				if test.retErr != nil {
					return test.retErr
				}

				content := "this is a test"

				if option != nil && option.ContentRange != nil {
					selection := strings.Split(strings.Split(*option.ContentRange, "=")[1], "-")
					start, _ := strconv.Atoi(selection[0])
					end, _ := strconv.Atoi(selection[1])
					if end > len(content) {
						end = len(content)
					}
					content = content[start:end]
				}
				_, _ = w.WriteAt([]byte(content), 0)

				return nil
			}

			mockObjectStore := objectstore.MockObjectStore{}
			mockObjectStore.On("DownloadObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockResult)

			mockLogStreams := db.NewMockLogStreams(t)
			mockJobs := db.NewMockJobs(t)

			mockLogStreams.On("GetLogStreamByID", mock.Anything, test.logStream.Metadata.ID).Return(test.logStream, nil).Maybe()

			mockDBClient := db.Client{
				LogStreams: mockLogStreams,
				Jobs:       mockJobs,
			}

			logStore := NewLogStore(&mockObjectStore, &mockDBClient)

			logs, err := logStore.ReadLogs(ctx, test.logStream.Metadata.ID, test.startOffset, 100)
			if err != nil {
				assert.True(t, test.expectErr, "Error was not expected: %v", err)
				return
			}

			assert.False(t, test.expectErr, "An error was expected")

			mockObjectStore.AssertExpectations(t)

			assert.Equal(t, test.expectedLogs, string(logs))
		})
	}
}

func TestWriteLogsToStore(t *testing.T) {
	// Test cases
	tests := []struct {
		retErr         error
		name           string
		existingLogs   string
		logsToUpload   string
		expectedUpload string
		startOffset    int
		expectErr      bool
	}{
		{
			name:           "upload new logs",
			existingLogs:   "",
			logsToUpload:   "this is a test",
			expectedUpload: "this is a test",
			expectErr:      false,
		},
		{
			name:           "append to existing logs",
			existingLogs:   "this",
			logsToUpload:   " is a test",
			expectedUpload: "this is a test",
			startOffset:    len("this"),
			expectErr:      false,
		},
		{
			name:         "start offset past eof",
			existingLogs: "this",
			logsToUpload: " is a test",
			startOffset:  100,
			expectErr:    true,
		},
		{
			name:           "truncate file",
			existingLogs:   "this is a test that will be truncated",
			logsToUpload:   " is a test",
			expectedUpload: "this is a test",
			startOffset:    len("this"),
			expectErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockDownloadResult := func(_ context.Context, _ string, w io.WriterAt, _ *objectstore.DownloadOptions) error {
				if test.existingLogs == "" {
					return errors.New(
						"Not Found",
						errors.WithErrorCode(errors.ENotFound),
					)
				}

				_, _ = w.WriteAt([]byte(test.existingLogs), 0)

				return nil
			}

			mockObjectStore := objectstore.MockObjectStore{}
			mockObjectStore.On("DownloadObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockDownloadResult)

			bodyMatcher := mock.MatchedBy(func(r io.Reader) bool {
				body, _ := io.ReadAll(r)
				str := string(body)
				return str == test.expectedUpload
			})

			mockObjectStore.On("UploadObject", mock.Anything, mock.Anything, bodyMatcher).Return(test.retErr)

			logStore := NewLogStore(&mockObjectStore, nil)

			err := logStore.WriteLogs(ctx, "stream-123", test.startOffset, []byte(test.logsToUpload))
			if err != nil {
				assert.True(t, test.expectErr, "Error was not expected %v", err)
				return
			}

			assert.False(t, test.expectErr, "An error was expected")

			mockObjectStore.AssertExpectations(t)
		})
	}
}
