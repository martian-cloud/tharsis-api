// Package aws package
package aws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	tErrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/objectstore"
)

const (
	// For security reasons, this is not configurable.
	presignURLExpiration = 1 * time.Minute

	// defaultAWSPartitionID is used when nothing is specified
	// for the AWS partition ID in the endpoint resolver.
	defaultAWSPartitionID = "aws"
)

// ObjectStore implementation for AWS S3
type ObjectStore struct {
	logger     logger.Logger
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
	bucket     string
}

// New returns an S3 implementation of the ObjectStore interface
func New(ctx context.Context, logger logger.Logger, pluginData map[string]string) (*ObjectStore, error) {
	bucket, ok := pluginData["bucket"]
	if !ok {
		return nil, fmt.Errorf("s3 object store plugin is missing the 'bucket' field")
	}

	region, ok := pluginData["region"]
	if !ok {
		return nil, fmt.Errorf("s3 object store plugin is missing the 'region' field")
	}

	accessKeyID := pluginData["aws_access_key_id"]
	secretKey := pluginData["aws_secret_access_key"]

	// Make sure secretKey is specified when using accessKeyID.
	if accessKeyID != "" && secretKey == "" {
		return nil, fmt.Errorf("s3 object store plugin is missing 'aws_secret_access_key' field but using 'aws_access_key_id'")
	}

	// Use a custom endpoint resolver.
	endpointResolver := aws.EndpointResolverWithOptionsFunc(func(_, region string, _ ...interface{}) (aws.Endpoint, error) {
		partitionID := defaultAWSPartitionID // Default
		if id, ok := pluginData["aws_partition_id"]; ok {
			partitionID = id
		}

		if endpoint, ok := pluginData["endpoint"]; ok {
			return aws.Endpoint{
				PartitionID:       partitionID,
				SigningRegion:     region,
				URL:               endpoint,
				HostnameImmutable: true,
			}, nil
		}

		// Allows fallback to default resolution.
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	// Otherwise, use default config.
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithDefaultRegion(region), config.WithEndpointResolverWithOptions(endpointResolver))
	if err != nil {
		return nil, err
	}

	// Use custom credentials.
	if accessKeyID != "" {
		awsCfg.Credentials = credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, "")
	}

	client := s3.NewFromConfig(awsCfg)
	uploader := manager.NewUploader(client)
	downloader := manager.NewDownloader(client)

	return &ObjectStore{logger, client, uploader, downloader, bucket}, nil
}

// UploadObject uploads an object to the object store
func (s *ObjectStore) UploadObject(ctx context.Context, key string, body io.Reader) error {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   body,
	})
	if err != nil {
		s.logger.Errorf("Failed to upload file to location %v", err)
		return err
	}

	return nil
}

// DownloadObject downloads an object using a concurrent download
func (s *ObjectStore) DownloadObject(ctx context.Context, key string, w io.WriterAt, options *objectstore.DownloadOptions) error {
	s3Options := s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if options != nil {
		s3Options.Range = options.ContentRange
	}

	_, err := s.downloader.Download(ctx, w, &s3Options)
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return tErrors.NewError(tErrors.ENotFound, fmt.Sprintf("Key %s not found in bucket %s", key, s.bucket))
		}

		var ae smithy.APIError
		if errors.As(err, &ae) && ae.ErrorCode() == "InvalidRange" {
			return tErrors.NewError(tErrors.ENotFound, fmt.Sprintf("Range %s not found in %s", *options.ContentRange, key))
		}

		s.logger.Errorf("Failed to download file from key %s %v", key, err)
		return err
	}

	return nil
}

// GetObjectStream returns an object stream for the object at the specified key
func (s *ObjectStore) GetObjectStream(ctx context.Context, key string, options *objectstore.DownloadOptions) (io.ReadCloser, error) {
	s3Options := s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if options != nil {
		s3Options.Range = options.ContentRange
	}

	result, err := s.client.GetObject(ctx, &s3Options)
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, tErrors.NewError(tErrors.ENotFound, fmt.Sprintf("Key %s not found in bucket %s", key, s.bucket))
		}

		var ae smithy.APIError
		if errors.As(err, &ae) && ae.ErrorCode() == "InvalidRange" {
			return nil, tErrors.NewError(tErrors.ENotFound, fmt.Sprintf("Range %s not found in %s", *options.ContentRange, key))
		}

		s.logger.Errorf("Failed to get file from key %s %v", key, err)
		return nil, err
	}

	return result.Body, nil
}

// DoesObjectExist returns a boolean indicating an object's existence.
// It doesn't download the object itself but simply queries for it's metadata.
func (s *ObjectStore) DoesObjectExist(ctx context.Context, key string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if _, err := s.client.HeadObject(ctx, input); err != nil {
		var respErr *awshttp.ResponseError
		if errors.As(err, &respErr) && respErr.ResponseError.HTTPStatusCode() == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetPresignedURL returns a presigned URL which can be used to temporarily
// provide access to an object from object storage without requiring
// IAM or AWS credentials.
func (s *ObjectStore) GetPresignedURL(ctx context.Context, key string) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	presignClient := s3.NewPresignClient(s.client, s3.WithPresignExpires(presignURLExpiration))

	presignedReq, err := presignClient.PresignGetObject(ctx, input)
	if err != nil {
		return "", tErrors.NewError(tErrors.EInternal, fmt.Sprintf("Failed to create presigned URL: %s", err.Error()))
	}

	return presignedReq.URL, nil
}
