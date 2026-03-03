package filesystem

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

func TestNew(t *testing.T) {
	testLogger, _ := logger.NewForTest()

	t.Run("valid configuration", func(t *testing.T) {
		store, err := New(testLogger, "http://localhost:8000", map[string]string{
			"directory": t.TempDir(),
		})
		require.NoError(t, err)
		assert.NotNil(t, store)
		assert.NotEmpty(t, store.secretKey)
	})

	t.Run("with provided secret key", func(t *testing.T) {
		store, err := New(testLogger, "http://localhost:8000", map[string]string{
			"directory":  t.TempDir(),
			"secret_key": "my-secret",
		})
		require.NoError(t, err)
		assert.Equal(t, "my-secret", store.secretKey)
	})

	t.Run("missing directory field", func(t *testing.T) {
		_, err := New(testLogger, "http://localhost:8000", map[string]string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "directory")
	})
}

func TestUploadObject(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	store, err := New(testLogger, "http://localhost:8000", map[string]string{"directory": t.TempDir()})
	require.NoError(t, err)
	ctx := t.Context()

	t.Run("creates file with nested directories", func(t *testing.T) {
		err := store.UploadObject(ctx, "a/b/c/file.txt", bytes.NewReader([]byte("content")))
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(store.basePath, "a/b/c/file.txt"))
		require.NoError(t, err)
		assert.Equal(t, []byte("content"), data)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		key := "overwrite.txt"
		require.NoError(t, store.UploadObject(ctx, key, bytes.NewReader([]byte("first"))))
		require.NoError(t, store.UploadObject(ctx, key, bytes.NewReader([]byte("second"))))

		data, err := os.ReadFile(filepath.Join(store.basePath, key))
		require.NoError(t, err)
		assert.Equal(t, []byte("second"), data)
	})

	t.Run("normalizes path traversal attempts", func(t *testing.T) {
		// filepath.Clean normalizes "../escape.txt" to "escape.txt"
		err := store.UploadObject(ctx, "../escape.txt", bytes.NewReader([]byte("data")))
		require.NoError(t, err)

		// File should be created at normalized path within base
		data, err := os.ReadFile(filepath.Join(store.basePath, "escape.txt"))
		require.NoError(t, err)
		assert.Equal(t, []byte("data"), data)
	})

	t.Run("handles read error from body", func(t *testing.T) {
		err := store.UploadObject(ctx, "error.txt", &errorReader{})
		require.Error(t, err)
	})
}

func TestDownloadObject(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	store, err := New(testLogger, "http://localhost:8000", map[string]string{"directory": t.TempDir()})
	require.NoError(t, err)
	ctx := t.Context()
	content := []byte("test content for download")
	require.NoError(t, store.UploadObject(ctx, "download.txt", bytes.NewReader(content)))

	t.Run("full content", func(t *testing.T) {
		buf := &writeAtBuffer{}
		err := store.DownloadObject(ctx, "download.txt", buf, nil)
		require.NoError(t, err)
		assert.Equal(t, content, buf.data)
	})

	t.Run("range download", func(t *testing.T) {
		buf := &writeAtBuffer{}
		err := store.DownloadObject(ctx, "download.txt", buf, &objectstore.DownloadOptions{
			ContentRange: aws.String("bytes=5-11"),
		})
		require.NoError(t, err)
		assert.Equal(t, []byte("content"), buf.data)
	})

	t.Run("not found", func(t *testing.T) {
		err := store.DownloadObject(ctx, "nonexistent.txt", &writeAtBuffer{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid range", func(t *testing.T) {
		err := store.DownloadObject(ctx, "download.txt", &writeAtBuffer{}, &objectstore.DownloadOptions{
			ContentRange: aws.String("invalid"),
		})
		require.Error(t, err)
	})
}

func TestGetObjectStream(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	store, err := New(testLogger, "http://localhost:8000", map[string]string{"directory": t.TempDir()})
	require.NoError(t, err)
	ctx := t.Context()
	content := []byte("stream content")
	require.NoError(t, store.UploadObject(ctx, "stream.txt", bytes.NewReader(content)))

	t.Run("full content", func(t *testing.T) {
		stream, err := store.GetObjectStream(ctx, "stream.txt", nil)
		require.NoError(t, err)
		defer stream.Close()

		data, err := io.ReadAll(stream)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("range download", func(t *testing.T) {
		stream, err := store.GetObjectStream(ctx, "stream.txt", &objectstore.DownloadOptions{
			ContentRange: aws.String("bytes=0-5"),
		})
		require.NoError(t, err)
		defer stream.Close()

		data, err := io.ReadAll(stream)
		require.NoError(t, err)
		assert.Equal(t, []byte("stream"), data)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.GetObjectStream(ctx, "nonexistent.txt", nil)
		require.Error(t, err)
	})

	t.Run("invalid range closes file", func(t *testing.T) {
		_, err := store.GetObjectStream(ctx, "stream.txt", &objectstore.DownloadOptions{
			ContentRange: aws.String("bad"),
		})
		require.Error(t, err)
	})
}

func TestDoesObjectExist(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	store, err := New(testLogger, "http://localhost:8000", map[string]string{"directory": t.TempDir()})
	require.NoError(t, err)
	ctx := t.Context()
	require.NoError(t, store.UploadObject(ctx, "exists.txt", bytes.NewReader([]byte("data"))))

	t.Run("exists", func(t *testing.T) {
		exists, err := store.DoesObjectExist(ctx, "exists.txt")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("does not exist", func(t *testing.T) {
		exists, err := store.DoesObjectExist(ctx, "missing.txt")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("normalizes path traversal attempts", func(t *testing.T) {
		// filepath.Clean normalizes "../etc/passwd" - file doesn't exist so returns false
		exists, err := store.DoesObjectExist(ctx, "../etc/passwd")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestGetPresignedURL(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	store, err := New(testLogger, "http://localhost:8000", map[string]string{"directory": t.TempDir()})
	require.NoError(t, err)
	ctx := t.Context()
	require.NoError(t, store.UploadObject(ctx, "presign.txt", bytes.NewReader([]byte("data"))))

	t.Run("generates valid URL", func(t *testing.T) {
		ctxWithSubject := auth.WithSubject(ctx, "user@example.com")
		presignedURL, err := store.GetPresignedURL(ctxWithSubject, "presign.txt")
		require.NoError(t, err)

		parsed, err := url.Parse(presignedURL)
		require.NoError(t, err)
		assert.Equal(t, "http", parsed.Scheme)
		assert.Equal(t, "localhost:8000", parsed.Host)
		assert.Contains(t, parsed.Path, "/v1/objectstore/presign.txt")

		query := parsed.Query()
		assert.NotEmpty(t, query.Get("X-Amz-Algorithm"))
		assert.NotEmpty(t, query.Get("X-Amz-Credential"))
		assert.NotEmpty(t, query.Get("X-Amz-Date"))
		assert.NotEmpty(t, query.Get("X-Amz-Expires"))
		assert.NotEmpty(t, query.Get("X-Amz-Signature"))
	})

	t.Run("missing subject", func(t *testing.T) {
		_, err := store.GetPresignedURL(ctx, "presign.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no subject")
	})
}

func TestVerifyPresignedURL(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	store, err := New(testLogger, "http://localhost:8000", map[string]string{"directory": t.TempDir()})
	require.NoError(t, err)
	ctx := t.Context()
	require.NoError(t, store.UploadObject(ctx, "verify.txt", bytes.NewReader([]byte("data"))))

	ctxWithSubject := auth.WithSubject(ctx, "user@example.com")
	validURL, err := store.GetPresignedURL(ctxWithSubject, "verify.txt")
	require.NoError(t, err)

	t.Run("valid URL", func(t *testing.T) {
		parsed, _ := url.Parse(validURL)
		key, err := store.VerifyPresignedURL(ctx, parsed.RequestURI())
		require.NoError(t, err)
		assert.Equal(t, "verify.txt", key)
	})

	t.Run("missing signature parameters", func(t *testing.T) {
		_, err := store.VerifyPresignedURL(ctx, "/v1/objectstore/test.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required signature")
	})

	t.Run("invalid URL format", func(t *testing.T) {
		_, err := store.VerifyPresignedURL(ctx, "://invalid")
		require.Error(t, err)
	})

	t.Run("invalid date format", func(t *testing.T) {
		parsed, _ := url.Parse(validURL)
		query := parsed.Query()
		query.Set("X-Amz-Date", "invalid-date")
		parsed.RawQuery = query.Encode()

		_, err := store.VerifyPresignedURL(ctx, parsed.RequestURI())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid date")
	})

	t.Run("invalid expires value", func(t *testing.T) {
		parsed, _ := url.Parse(validURL)
		query := parsed.Query()
		query.Set("X-Amz-Expires", "not-a-number")
		parsed.RawQuery = query.Encode()

		_, err := store.VerifyPresignedURL(ctx, parsed.RequestURI())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid expires")
	})

	t.Run("expired URL", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, store.apiURL+pathPrefix+"verify.txt", nil)
		query := req.URL.Query()
		query.Set("X-Amz-Expires", "60")
		req.URL.RawQuery = query.Encode()

		expiredTime := time.Now().Add(-2 * time.Minute)
		signedURI, _, _ := store.signer.PresignHTTP(ctx, aws.Credentials{
			AccessKeyID:     "user@example.com",
			SecretAccessKey: store.secretKey,
		}, req, unsignedPayload, awsService, "", expiredTime)

		parsed, _ := url.Parse(signedURI)
		_, err := store.VerifyPresignedURL(ctx, parsed.RequestURI())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("tampered signature", func(t *testing.T) {
		parsed, _ := url.Parse(validURL)
		query := parsed.Query()
		sig := query.Get("X-Amz-Signature")
		// Flip the last character to ensure it's different from original
		lastChar := sig[len(sig)-1]
		if lastChar == '0' {
			query.Set("X-Amz-Signature", sig[:len(sig)-1]+"1")
		} else {
			query.Set("X-Amz-Signature", sig[:len(sig)-1]+"0")
		}
		parsed.RawQuery = query.Encode()

		_, err := store.VerifyPresignedURL(ctx, parsed.RequestURI())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signature mismatch")
	})

	t.Run("tampered expiration", func(t *testing.T) {
		parsed, _ := url.Parse(validURL)
		query := parsed.Query()
		query.Set("X-Amz-Expires", "9999")
		parsed.RawQuery = query.Encode()

		_, err := store.VerifyPresignedURL(ctx, parsed.RequestURI())
		require.Error(t, err)
	})

	t.Run("tampered credential", func(t *testing.T) {
		parsed, _ := url.Parse(validURL)
		query := parsed.Query()
		query.Set("X-Amz-Credential", "attacker/"+query.Get("X-Amz-Credential"))
		parsed.RawQuery = query.Encode()

		_, err := store.VerifyPresignedURL(ctx, parsed.RequestURI())
		require.Error(t, err)
	})

	t.Run("missing credential", func(t *testing.T) {
		parsed, _ := url.Parse(validURL)
		query := parsed.Query()
		query.Del("X-Amz-Credential")
		parsed.RawQuery = query.Encode()

		_, err := store.VerifyPresignedURL(ctx, parsed.RequestURI())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing credential")
	})

	t.Run("invalid credential format", func(t *testing.T) {
		parsed, _ := url.Parse(validURL)
		query := parsed.Query()
		query.Set("X-Amz-Credential", "no-slash-here")
		parsed.RawQuery = query.Encode()

		_, err := store.VerifyPresignedURL(ctx, parsed.RequestURI())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credential")
	})

	t.Run("invalid path prefix", func(t *testing.T) {
		parsed, _ := url.Parse(validURL)
		parsed.Path = "/wrong/path/verify.txt"
		_, err := store.VerifyPresignedURL(ctx, parsed.RequestURI())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid path")
	})

	t.Run("path traversal normalized in URL", func(t *testing.T) {
		// Create file that would be the normalized target
		require.NoError(t, store.UploadObject(ctx, "etc/passwd", bytes.NewReader([]byte("fake"))))

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, store.apiURL+pathPrefix+"../etc/passwd", nil)
		query := req.URL.Query()
		query.Set("X-Amz-Expires", "60")
		req.URL.RawQuery = query.Encode()

		signedURI, _, _ := store.signer.PresignHTTP(ctx, aws.Credentials{
			AccessKeyID:     "user@example.com",
			SecretAccessKey: store.secretKey,
		}, req, unsignedPayload, awsService, "", time.Now())

		parsed, _ := url.Parse(signedURI)
		// Returns the raw key from URL, sanitizeKey normalizes it internally
		key, err := store.VerifyPresignedURL(ctx, parsed.RequestURI())
		require.NoError(t, err)
		assert.Equal(t, "../etc/passwd", key)
	})
}

func TestSymlinkHandling(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	store, err := New(testLogger, "http://localhost:8000", map[string]string{"directory": t.TempDir()})
	require.NoError(t, err)
	ctx := t.Context()

	t.Run("allows symlink within base", func(t *testing.T) {
		// Create real file
		realDir := filepath.Join(store.basePath, "real")
		require.NoError(t, os.MkdirAll(realDir, 0750))
		require.NoError(t, os.WriteFile(filepath.Join(realDir, "file.txt"), []byte("real content"), 0640))

		// Create symlink within base
		linkPath := filepath.Join(store.basePath, "link")
		require.NoError(t, os.Symlink(realDir, linkPath))

		stream, err := store.GetObjectStream(ctx, "link/file.txt", nil)
		require.NoError(t, err)
		defer stream.Close()

		data, err := io.ReadAll(stream)
		require.NoError(t, err)
		assert.Equal(t, []byte("real content"), data)
	})

	t.Run("rejects symlink escaping base", func(t *testing.T) {
		// Create file outside base
		outsideDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0640))

		// Create symlink pointing outside
		linkPath := filepath.Join(store.basePath, "escape")
		require.NoError(t, os.Symlink(outsideDir, linkPath))

		_, err := store.GetObjectStream(ctx, "escape/secret.txt", nil)
		require.Error(t, err)
	})
}

type writeAtBuffer struct {
	data []byte
}

func (w *writeAtBuffer) WriteAt(p []byte, off int64) (int, error) {
	end := int(off) + len(p)
	if end > len(w.data) {
		newData := make([]byte, end)
		copy(newData, w.data)
		w.data = newData
	}
	copy(w.data[off:], p)
	return len(p), nil
}

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read error")
}
