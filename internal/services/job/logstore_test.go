package job

import (
	"context"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/objectstore"
)

func TestGetLogs(t *testing.T) {
	// Test cases
	tests := []struct {
		retErr       error
		name         string
		expectedLogs string
		startOffset  int
		expectErr    bool
	}{
		{
			name:         "get all logs",
			expectedLogs: "this is a test",
			expectErr:    false,
		},
		{
			name:         "get partial logs",
			startOffset:  5,
			expectedLogs: "is a test",
			expectErr:    false,
		},
		{
			name: "logs not found",
			retErr: errors.NewError(
				errors.ENotFound,
				"Not Found",
			),
			expectErr: false,
		},
		{
			name: "unexpected error",
			retErr: errors.NewError(
				errors.EInternal,
				"",
			),
			expectErr: true,
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

			logStore := NewLogStore(&mockObjectStore, nil)

			logs, err := logStore.GetLogs(ctx, "workspace-1", "run-1", "plan-1", test.startOffset, 100)
			if err != nil {
				assert.True(t, test.expectErr, "Error was not expected")
				return
			}

			mockObjectStore.AssertExpectations(t)

			assert.Equal(t, test.expectedLogs, string(logs))
		})
	}
}

func TestSaveLogs(t *testing.T) {
	jobID := "job1"

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
			expectErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockDownloadResult := func(_ context.Context, _ string, w io.WriterAt, _ *objectstore.DownloadOptions) error {
				if test.existingLogs == "" {
					return errors.NewError(
						errors.ENotFound,
						"Not Found",
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

			mockJobs := db.MockJobs{}
			mockJobs.Test(t)

			mockJobs.On("GetJobLogDescriptorByJobID", mock.Anything, jobID).Return(func(_ context.Context, jobID string) *models.JobLogDescriptor {
				if len(test.existingLogs) > 0 {
					return &models.JobLogDescriptor{JobID: jobID}
				}
				return nil
			}, nil)

			if len(test.existingLogs) > 0 {
				mockJobs.On("UpdateJobLogDescriptor", mock.Anything, &models.JobLogDescriptor{
					JobID: jobID,
					Size:  test.startOffset + len(test.logsToUpload),
				}).Return(nil, nil)
			} else {
				mockJobs.On("CreateJobLogDescriptor", mock.Anything, &models.JobLogDescriptor{
					JobID: jobID,
					Size:  test.startOffset + len(test.logsToUpload),
				}).Return(nil, nil)
			}

			dbClient := db.Client{
				Jobs: &mockJobs,
			}

			logStore := NewLogStore(&mockObjectStore, &dbClient)

			err := logStore.SaveLogs(ctx, "workspace-1", "run-1", jobID, test.startOffset, []byte(test.logsToUpload))
			if err != nil {
				assert.True(t, test.expectErr, "Error was not expected")
				return
			}

			mockObjectStore.AssertExpectations(t)
			mockJobs.AssertExpectations(t)
		})
	}
}
