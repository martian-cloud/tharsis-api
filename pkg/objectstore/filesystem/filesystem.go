// Package filesystem package
package filesystem

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

const (
	// presignURLExpiration is the duration for which presigned URLs are valid
	presignURLExpiration = 1 * time.Minute
	// AWS constants for SigV4
	awsService         = "s3"
	unsignedPayload    = "UNSIGNED-PAYLOAD"
	amzDateFormat      = "20060102T150405Z"
	pathPrefix         = "/v1/objectstore/"
	amzDateParam       = "X-Amz-Date"
	amzExpiresParam    = "X-Amz-Expires"
	amzSignatureParam  = "X-Amz-Signature"
	amzCredentialParam = "X-Amz-Credential"
)

// sectionReadCloser wraps a SectionReader with a closer
type sectionReadCloser struct {
	*io.SectionReader
	closer io.Closer
}

func (s *sectionReadCloser) Close() error {
	return s.closer.Close()
}

// ObjectStore implementation for local filesystem
type ObjectStore struct {
	logger    logger.Logger
	basePath  string
	apiURL    string
	signer    *v4.Signer
	secretKey string
}

// New returns a filesystem implementation of the ObjectStore interface
func New(logger logger.Logger, apiURL string, pluginData map[string]string) (*ObjectStore, error) {
	basePath, ok := pluginData["directory"]
	if !ok {
		return nil, errors.New("filesystem object store plugin is missing the 'directory' field")
	}

	// Auto-generate secret key if not provided
	secretKey := pluginData["secret_key"]
	if secretKey == "" {
		generated, err := generateSecretKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate secret key: %w", err)
		}
		secretKey = generated
	}

	return &ObjectStore{
		logger:    logger,
		basePath:  basePath,
		apiURL:    apiURL,
		signer:    v4.NewSigner(),
		secretKey: secretKey,
	}, nil
}

// UploadObject uploads an object to the filesystem
func (f *ObjectStore) UploadObject(ctx context.Context, key string, body io.Reader) error {
	fullPath, err := f.sanitizeKey(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
		f.logger.WithContextFields(ctx).Errorf("failed to create directory for key %s: %v", key, err)
		return err
	}

	file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		f.logger.WithContextFields(ctx).Errorf("failed to create file for key %s: %v", key, err)
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			f.logger.WithContextFields(ctx).Errorf("failed to close file for key %s: %v", key, closeErr)
		}
	}()

	if _, err := io.Copy(file, body); err != nil {
		f.logger.WithContextFields(ctx).Errorf("failed to write file for key %s: %v", key, err)
		return err
	}

	return nil
}

// DownloadObject downloads an object from the filesystem
func (f *ObjectStore) DownloadObject(ctx context.Context, key string, w io.WriterAt, options *objectstore.DownloadOptions) error {
	fullPath, err := f.sanitizeKey(key)
	if err != nil {
		return err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return te.New("key %s not found", key, te.WithErrorCode(te.ENotFound))
		}
		f.logger.WithContextFields(ctx).Errorf("failed to open file for key %s: %v", key, err)
		return err
	}
	defer file.Close()

	var reader io.Reader = file

	if options != nil && options.ContentRange != nil {
		offset, length, err := parseRange(*options.ContentRange)
		if err != nil {
			return te.New("invalid range %s for key %s", *options.ContentRange, key, te.WithErrorCode(te.EInvalid))
		}
		reader = io.NewSectionReader(file, offset, length)
	}

	buf := make([]byte, 32*1024)
	var offset int64
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			if _, writeErr := w.WriteAt(buf[:n], offset); writeErr != nil {
				return writeErr
			}
			offset += int64(n)
		}

		if readErr == io.EOF {
			break
		}

		if readErr != nil {
			return readErr
		}
	}

	return nil
}

// GetObjectStream returns an object stream for the object at the specified key
func (f *ObjectStore) GetObjectStream(ctx context.Context, key string, options *objectstore.DownloadOptions) (io.ReadCloser, error) {
	fullPath, err := f.sanitizeKey(key)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, te.New("key %s not found", key, te.WithErrorCode(te.ENotFound))
		}
		f.logger.WithContextFields(ctx).Errorf("failed to open file for key %s: %v", key, err)
		return nil, err
	}

	if options != nil && options.ContentRange != nil {
		offset, length, err := parseRange(*options.ContentRange)
		if err != nil {
			file.Close()
			return nil, te.New("invalid range %s for key %s", *options.ContentRange, key, te.WithErrorCode(te.EInvalid))
		}
		return &sectionReadCloser{io.NewSectionReader(file, offset, length), file}, nil
	}

	return file, nil
}

// DoesObjectExist returns a boolean indicating an object's existence
func (f *ObjectStore) DoesObjectExist(_ context.Context, key string) (bool, error) {
	fullPath, err := f.sanitizeKey(key)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(fullPath)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// GetPresignedURL generates a presigned URL using AWS Signature V4
func (f *ObjectStore) GetPresignedURL(ctx context.Context, key string) (string, error) {
	if _, err := f.sanitizeKey(key); err != nil {
		return "", err
	}

	subject := auth.GetSubject(ctx)
	if subject == nil {
		return "", te.New("no subject found in context", te.WithErrorCode(te.EUnauthorized))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.apiURL+pathPrefix+key, nil)
	if err != nil {
		return "", te.Wrap(err, "failed to create request", te.WithErrorCode(te.EInternal))
	}

	// Set expiration in query params before signing
	query := req.URL.Query()
	query.Set(amzExpiresParam, fmt.Sprintf("%d", int64(presignURLExpiration.Seconds())))
	req.URL.RawQuery = query.Encode()

	signedURI, _, err := f.signer.PresignHTTP(ctx, aws.Credentials{
		AccessKeyID:     *subject,
		SecretAccessKey: f.secretKey,
	}, req, unsignedPayload, awsService, "", time.Now())
	if err != nil {
		return "", te.Wrap(err, "failed to sign request", te.WithErrorCode(te.EInternal))
	}

	return signedURI, nil
}

// VerifyPresignedURL verifies an AWS SigV4 presigned URL
func (f *ObjectStore) VerifyPresignedURL(ctx context.Context, urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", te.New("invalid URL format", te.WithErrorCode(te.EInvalid))
	}

	query := parsedURL.Query()
	amzDate := query.Get(amzDateParam)
	expires := query.Get(amzExpiresParam)
	providedSignature := query.Get(amzSignatureParam)

	if amzDate == "" || expires == "" || providedSignature == "" {
		return "", te.New("missing required signature parameters", te.WithErrorCode(te.EInvalid))
	}

	requestTime, err := time.Parse(amzDateFormat, amzDate)
	if err != nil {
		return "", te.New("invalid date format", te.WithErrorCode(te.EInvalid))
	}

	// Parse expiration
	var expiresSeconds int64
	if _, err := fmt.Sscanf(expires, "%d", &expiresSeconds); err != nil {
		return "", te.New("invalid expires value", te.WithErrorCode(te.EInvalid))
	}

	// Check if expired
	if time.Now().UTC().After(requestTime.Add(time.Duration(expiresSeconds) * time.Second)) {
		return "", te.New("presigned URL has expired", te.WithErrorCode(te.EForbidden))
	}

	// Extract credential and get access key (subject)
	credential := query.Get(amzCredentialParam)
	if credential == "" {
		return "", te.New("missing credential", te.WithErrorCode(te.EInvalid))
	}

	accessKey, _, found := strings.Cut(credential, "/")
	if !found {
		return "", te.New("invalid credential format", te.WithErrorCode(te.EInvalid))
	}

	// Extract key from path
	key, found := strings.CutPrefix(parsedURL.Path, pathPrefix)
	if !found {
		return "", te.New("invalid path", te.WithErrorCode(te.EInvalid))
	}

	// Parse API URL to get scheme and host
	apiURL, err := url.Parse(f.apiURL)
	if err != nil {
		return "", te.New("invalid API URL configuration", te.WithErrorCode(te.EInternal))
	}

	// Build the original request URL using API URL's scheme/host and incoming path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.Scheme+"://"+apiURL.Host+parsedURL.Path, nil)
	if err != nil {
		return "", te.Wrap(err, "failed to create request", te.WithErrorCode(te.EInternal))
	}

	// Include query params that were part of the original signature
	reqQuery := req.URL.Query()
	reqQuery.Set(amzExpiresParam, expires)
	req.URL.RawQuery = reqQuery.Encode()

	// Sign the request with the same timestamp
	signedURI, _, err := f.signer.PresignHTTP(ctx, aws.Credentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: f.secretKey,
	}, req, unsignedPayload, awsService, "", requestTime)
	if err != nil {
		return "", te.Wrap(err, "failed to verify signature", te.WithErrorCode(te.EForbidden))
	}

	// Parse signed URI and extract signature
	signedURL, err := url.Parse(signedURI)
	if err != nil {
		return "", te.Wrap(err, "failed to parse signed URL", te.WithErrorCode(te.EInternal))
	}

	expectedSignature := signedURL.Query().Get(amzSignatureParam)

	// Compare signatures using constant-time comparison
	if subtle.ConstantTimeCompare([]byte(expectedSignature), []byte(providedSignature)) != 1 {
		return "", te.New("signature mismatch", te.WithErrorCode(te.EForbidden))
	}

	return key, nil
}

// sanitizeKey validates and sanitizes the key to prevent path traversal
func (f *ObjectStore) sanitizeKey(key string) (string, error) {
	fullPath := filepath.Join(f.basePath, filepath.Clean("/"+key))

	// Validate symlinks resolve within base (only if path exists)
	safe, err := isSafePath(f.basePath, fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist, skip symlink validation
			return fullPath, nil
		}
		return "", err
	}
	if !safe {
		return "", te.New("invalid key", te.WithErrorCode(te.EInvalid))
	}

	return fullPath, nil
}

// isSafePath checks if userPath resolves to a location within base directory
func isSafePath(base, userPath string) (bool, error) {
	evaluatedUserPath, err := filepath.EvalSymlinks(userPath)
	if err != nil {
		return false, err
	}

	absUserPath, err := filepath.Abs(evaluatedUserPath)
	if err != nil {
		return false, err
	}

	evaluatedBasePath, err := filepath.EvalSymlinks(base)
	if err != nil {
		return false, err
	}

	absBasePath, err := filepath.Abs(evaluatedBasePath)
	if err != nil {
		return false, err
	}

	// Check prefix (after cleaning)
	rel, err := filepath.Rel(absBasePath, absUserPath)
	if err != nil {
		return false, err
	}

	return !strings.HasPrefix(rel, ".."), nil
}

// generateSecretKey generates a random secret key
func generateSecretKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// parseRange parses a Content-Range header value
func parseRange(rangeStr string) (offset, length int64, err error) {
	after, found := strings.CutPrefix(rangeStr, "bytes=")
	if !found {
		return 0, 0, errors.New("invalid range format")
	}

	startStr, endStr, found := strings.Cut(after, "-")
	if !found {
		return 0, 0, errors.New("invalid range format")
	}

	var start, end int64
	if _, err := fmt.Sscanf(startStr, "%d", &start); err != nil {
		return 0, 0, errors.New("invalid range format")
	}

	if _, err := fmt.Sscanf(endStr, "%d", &end); err != nil {
		return 0, 0, errors.New("invalid range format")
	}

	return start, end - start + 1, nil
}
