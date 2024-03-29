// Package cli package
package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	// hashicorpReleasesBaseURL is used to download Terraform CLI binary.
	hashicorpReleasesBaseURL = "https://releases.hashicorp.com"
)

// zipContentType represents the allowed mime types when downloading a zip archive.
var zipContentType = []string{
	"application/x-zip-compressed",
	"application/zip",
}

// TerraformCLIVersionsInput is the input for retrieving CLI versions.
type TerraformCLIVersionsInput struct {
	Version      string
	OS           string
	Architecture string
}

// TerraformCLIVersions represents Terraform CLI versions.
type TerraformCLIVersions []string

// Latest returns the latest version from the slice i.e. the last element.
func (v TerraformCLIVersions) Latest() string {
	return v[len(v)-1]
}

// Supported returns a Tharsis error if the supplied version is not supported.
func (v TerraformCLIVersions) Supported(wantVersion string) error {
	for _, supportedVersion := range v {
		if wantVersion == supportedVersion {
			return nil
		}
	}

	return errors.New("Unsupported Terraform version", errors.WithErrorCode(errors.EInvalid))
}

// Service encapsulates the logic for interacting with the CLI service.
type Service interface {
	GetTerraformCLIVersions(ctx context.Context) (TerraformCLIVersions, error)
	CreateTerraformCLIDownloadURL(ctx context.Context, input *TerraformCLIVersionsInput) (string, error)
}

type service struct {
	logger                        logger.Logger
	httpClient                    *http.Client
	taskManager                   asynctask.Manager
	cliStore                      TerraformCLIStore
	terraformCLIVersionConstraint string
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	httpClient *http.Client,
	taskManager asynctask.Manager,
	cliStore TerraformCLIStore,
	terraformCLIVersionConstraint string,
) Service {
	return &service{
		logger:                        logger,
		httpClient:                    httpClient,
		taskManager:                   taskManager,
		cliStore:                      cliStore,
		terraformCLIVersionConstraint: terraformCLIVersionConstraint,
	}
}

// GetTerraformCLIVersions returns all available Terraform CLI versions.
func (s *service) GetTerraformCLIVersions(ctx context.Context) (TerraformCLIVersions, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTerraformCLIVersions")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Returned versions should adhere to terraformCLIVersionConstraints.
	constraints, err := version.NewConstraint(s.terraformCLIVersionConstraint)
	if err != nil {
		tracing.RecordError(span, err, "failed to generate a Terraform CLI version constraint")
		return nil, err
	}

	versions := &releases.Versions{
		Product:     product.Terraform,
		Constraints: constraints,
	}

	// List all the versions that meet constraints above.
	versionSources, err := versions.List(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to list the versions that meet the specified constraints")
		return nil, err
	}

	// If the length here is zero, then the retrieval failed.
	if len(versionSources) == 0 {
		tracing.RecordError(span, nil, "failed to get a list of Terraform CLI versions")
		return nil, errors.New(
			"failed to get a list of Terraform CLI versions",
			errors.WithErrorCode(errors.EInternal))
	}

	var stringVersions TerraformCLIVersions

	// Convert version sources to their raw string version.
	for _, src := range versionSources {
		source := src.(*releases.ExactVersion)
		stringVersions = append(stringVersions, source.Version.String())
	}

	return stringVersions, nil
}

// CreateTerraformCLIDownloadURL
func (s *service) CreateTerraformCLIDownloadURL(ctx context.Context, input *TerraformCLIVersionsInput) (string, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateTerraformCLIDownloadURL")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return "", err
	}

	exists, err := s.cliStore.DoesTerraformCLIBinaryExist(ctx, input.Version, input.OS, input.Architecture)
	if err != nil {
		tracing.RecordError(span, err, "failed to check whether the Terraform CLI version binary exists")
		return "", err
	}

	// Attempt to download the CLI release in a goroutine if it doesn't exist.
	if !exists {
		s.taskManager.StartTask(func(taskCtx context.Context) {
			if err := s.downloadTerraformCLIRelease(taskCtx, input); err != nil {
				// Cannot trace an error, because the span will have already been ended.
				s.logger.Errorf("error while downloading Terraform CLI release: %v", err)
			}
		})

		return getTerraformCLIDownloadURL(input), nil
	}

	return s.cliStore.CreateTerraformCLIBinaryPresignedURL(ctx, input.Version, input.OS, input.Architecture)
}

func (s *service) downloadTerraformCLIRelease(ctx context.Context, input *TerraformCLIVersionsInput) error {
	response, err := s.httpClient.Get(getTerraformCLIDownloadURL(input))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download response status: %s", response.Status)
	}

	// Verify the mime type.
	mimeType := response.Header.Get("content-type")
	if !isZipMimeType(mimeType) {
		return fmt.Errorf("unexpected mime type: expected %v, got %s", zipContentType, mimeType)
	}

	return s.cliStore.UploadTerraformCLIBinary(ctx, input.Version, input.OS, input.Architecture, response.Body)
}

// getTerraformCLIDownloadURL returns the Hashicorp releases URL
// for a Terraform CLI binary.
func getTerraformCLIDownloadURL(input *TerraformCLIVersionsInput) string {
	fileName := strings.Join([]string{"terraform", input.Version, input.OS, input.Architecture}, "_") + ".zip"
	return fmt.Sprintf(
		"%s/terraform/%s/%s",
		hashicorpReleasesBaseURL,
		url.PathEscape(input.Version),
		url.PathEscape(fileName),
	)
}

// isZipMimeType verifies the mime type to be a zip equivalent.
func isZipMimeType(contentType string) bool {
	for _, mime := range zipContentType {
		if contentType == mime {
			return true
		}
	}

	return false
}
