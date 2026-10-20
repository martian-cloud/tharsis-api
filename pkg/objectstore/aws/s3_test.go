package aws

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"go.uber.org/zap/zapcore"
)

const accessDeniedBody = `<?xml version="1.0" encoding="UTF-8"?>
<Error><Code>AccessDenied</Code><Message>Access Denied</Message><RequestId>test</RequestId></Error>`

// logCounts returns how many entries were logged at exactly ERROR and WARN level.
type logCounts func() (errors int, warnings int)

// newTestObjectStore returns an ObjectStore pointed at a local fake S3 server that answers
// every request with 403 AccessDenied, plus a function for counting logged entries by level.
func newTestObjectStore(t *testing.T) (*ObjectStore, logCounts) {
	t.Helper()

	return newTestObjectStoreWithHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(accessDeniedBody))
	})
}

// newTestObjectStoreWithHandler is like newTestObjectStore but lets the test control the fake S3 responses.
func newTestObjectStoreWithHandler(t *testing.T, handler http.HandlerFunc) (*ObjectStore, logCounts) {
	t.Helper()

	// Keep the SDK away from any real AWS configuration on the machine.
	t.Setenv("AWS_CONFIG_FILE", "/dev/null")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/dev/null")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	testLogger, logs := logger.NewForTest()

	store, err := New(t.Context(), testLogger, map[string]string{
		"bucket":                "test-bucket",
		"region":                "us-east-1",
		"aws_access_key_id":     "test",
		"aws_secret_access_key": "test",
		"endpoint":              server.URL,
	})
	require.NoError(t, err)

	counts := func() (int, int) {
		return logs.FilterLevelExact(zapcore.ErrorLevel).Len(), logs.FilterLevelExact(zapcore.WarnLevel).Len()
	}

	return store, counts
}

func canceledContext(t *testing.T) context.Context {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

func expiredContext(t *testing.T) context.Context {
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	t.Cleanup(cancel)
	return ctx
}

func liveContext(t *testing.T) context.Context {
	return t.Context()
}

// logLevelTestCase is shared by the DownloadObject and GetObjectStream tests.
type logLevelTestCase struct {
	name         string
	ctx          func(t *testing.T) context.Context
	expectErrIs  error
	expectErrors int
	expectWarns  int
}

var logLevelTestCases = []logLevelTestCase{
	{
		name:         "canceled context is not logged",
		ctx:          canceledContext,
		expectErrIs:  context.Canceled,
		expectErrors: 0,
		expectWarns:  0,
	},
	{
		name:         "deadline exceeded is logged at warn, not error",
		ctx:          expiredContext,
		expectErrIs:  context.DeadlineExceeded,
		expectErrors: 0,
		expectWarns:  1,
	},
	{
		name:         "access denied is logged at error",
		ctx:          liveContext,
		expectErrors: 1,
		expectWarns:  0,
	},
}

func TestDownloadObjectLogLevels(t *testing.T) {
	for _, test := range logLevelTestCases {
		t.Run(test.name, func(t *testing.T) {
			store, counts := newTestObjectStore(t)

			err := store.DownloadObject(test.ctx(t), "some/key", manager.NewWriteAtBuffer([]byte{}), nil)
			require.Error(t, err)

			if test.expectErrIs != nil {
				assert.True(t, errors.Is(err, test.expectErrIs), "expected %v in error chain, got: %v", test.expectErrIs, err)
			}

			errorCount, warnCount := counts()
			assert.Equal(t, test.expectErrors, errorCount, "ERROR entries")
			assert.Equal(t, test.expectWarns, warnCount, "WARN entries")
		})
	}
}

func TestUploadObjectLogLevels(t *testing.T) {
	for _, test := range logLevelTestCases {
		t.Run(test.name, func(t *testing.T) {
			store, counts := newTestObjectStore(t)

			err := store.UploadObject(test.ctx(t), "some/key", strings.NewReader("data"))
			require.Error(t, err)

			if test.expectErrIs != nil {
				assert.True(t, errors.Is(err, test.expectErrIs), "expected %v in error chain, got: %v", test.expectErrIs, err)
			}

			errorCount, warnCount := counts()
			assert.Equal(t, test.expectErrors, errorCount, "ERROR entries")
			assert.Equal(t, test.expectWarns, warnCount, "WARN entries")
		})
	}
}

func TestDeleteObjectsLogLevels(t *testing.T) {
	for _, test := range logLevelTestCases {
		t.Run(test.name, func(t *testing.T) {
			store, counts := newTestObjectStore(t)

			err := store.DeleteObjects(test.ctx(t), []string{"some/key"})
			require.Error(t, err)

			if test.expectErrIs != nil {
				assert.True(t, errors.Is(err, test.expectErrIs), "expected %v in error chain, got: %v", test.expectErrIs, err)
			}

			errorCount, warnCount := counts()
			assert.Equal(t, test.expectErrors, errorCount, "ERROR entries")
			assert.Equal(t, test.expectWarns, warnCount, "WARN entries")
		})
	}
}

// Per-object delete failures are reported by S3 itself (not context errors), so they stay at ERROR.
func TestDeleteObjectsPerObjectErrorsLoggedAtError(t *testing.T) {
	const deleteResultWithError = `<?xml version="1.0" encoding="UTF-8"?>
<DeleteResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Error><Key>some/key</Key><Code>AccessDenied</Code><Message>Access Denied</Message></Error></DeleteResult>`

	store, counts := newTestObjectStoreWithHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(deleteResultWithError))
	})

	err := store.DeleteObjects(t.Context(), []string{"some/key"})
	require.Error(t, err)

	errorCount, warnCount := counts()
	assert.Equal(t, 1, errorCount, "ERROR entries")
	assert.Equal(t, 0, warnCount, "WARN entries")
}

func TestGetObjectStreamLogLevels(t *testing.T) {
	for _, test := range logLevelTestCases {
		t.Run(test.name, func(t *testing.T) {
			store, counts := newTestObjectStore(t)

			output, err := store.GetObjectStream(test.ctx(t), "some/key", nil)
			require.Error(t, err)
			assert.Nil(t, output)

			if test.expectErrIs != nil {
				assert.True(t, errors.Is(err, test.expectErrIs), "expected %v in error chain, got: %v", test.expectErrIs, err)
			}

			errorCount, warnCount := counts()
			assert.Equal(t, test.expectErrors, errorCount, "ERROR entries")
			assert.Equal(t, test.expectWarns, warnCount, "WARN entries")
		})
	}
}
