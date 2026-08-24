package client

//go:generate go tool mockery --name RESTClient --inpackage --case underscore

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/provider"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/slug"
)

const (
	contentTypeOctetStream = "application/octet-stream"
)

// ErrPackageNotFound is returned by DownloadPackage when the requested package version cannot be
// found (it has been deleted, is not fully uploaded, or the caller cannot view it). Callers can
// detect it with errors.Is.
var ErrPackageNotFound = errors.New("package not found")

var _ RESTClient = (*restClient)(nil)

// RESTClientConfig holds configuration for creating a new REST client.
type RESTClientConfig struct {
	TokenResolver      TokenResolver
	Endpoint           string
	UserAgent          *string      // optional; added to requests when using the default HTTP client
	HTTPClient         *http.Client // optional override; if nil, a retryable client with user agent is created
	InsecureSkipVerify bool         // skip TLS certificate verification; ignored when HTTPClient is set
}

// UploadConfigurationVersionInput is the input for uploading a configuration version.
type UploadConfigurationVersionInput struct {
	WorkspaceID     string
	ConfigVersionID string
	DirectoryPath   string
}

// DownloadConfigurationVersionInput is the input for downloading a configuration version.
type DownloadConfigurationVersionInput struct {
	ConfigVersionID string
	Writer          io.Writer
}

// UploadModuleVersionInput is the input for uploading a module version.
type UploadModuleVersionInput struct {
	ModuleVersionID string
	PackagePath     string
}

// UploadProviderReadmeInput is the input for uploading a provider README.
type UploadProviderReadmeInput struct {
	ProviderVersionID string
	ReadmePath        string
}

// UploadProviderChecksumsInput is the input for uploading provider checksums.
type UploadProviderChecksumsInput struct {
	ProviderVersionID string
	ChecksumsPath     string
}

// UploadProviderChecksumSignatureInput is the input for uploading provider checksum signature.
type UploadProviderChecksumSignatureInput struct {
	ProviderVersionID string
	SignaturePath     string
}

// UploadProviderPlatformBinaryInput is the input for uploading a provider platform binary.
type UploadProviderPlatformBinaryInput struct {
	PlatformID string
	BinaryPath string
}

// UploadProviderPlatformPackageToMirrorInput is the input for uploading a provider platform package to mirror.
type UploadProviderPlatformPackageToMirrorInput struct {
	VersionMirrorID string
	OS              string
	Arch            string
	Reader          io.Reader
}

// UploadPlanCacheInput is the input for uploading a plan cache.
type UploadPlanCacheInput struct {
	PlanID string
	Reader io.Reader
}

// UploadPlanDataInput is the input for uploading plan data.
type UploadPlanDataInput struct {
	PlanID string
	Reader io.Reader
}

// DownloadStateVersionInput is the input for downloading a state version.
type DownloadStateVersionInput struct {
	StateVersionID string
	Writer         io.Writer
}

// DownloadPlanCacheInput is the input for downloading a plan cache.
type DownloadPlanCacheInput struct {
	PlanID string
	Writer io.Writer
}

// DownloadPlanJSONInput is the input for downloading a plan's JSON representation.
type DownloadPlanJSONInput struct {
	PlanID string
	Writer io.Writer
}

// DownloadPackageInput is the input for downloading a package version by its package version id.
type DownloadPackageInput struct {
	PackageVersionID string
	Writer           io.Writer
}

// RESTClient is the interface for REST client operations.
type RESTClient interface {
	UploadConfigurationVersion(ctx context.Context, input *UploadConfigurationVersionInput) error
	DownloadConfigurationVersion(ctx context.Context, input *DownloadConfigurationVersionInput) error
	UploadModuleVersion(ctx context.Context, input *UploadModuleVersionInput) error
	UploadProviderReadme(ctx context.Context, input *UploadProviderReadmeInput) error
	UploadProviderChecksums(ctx context.Context, input *UploadProviderChecksumsInput) error
	UploadProviderChecksumSignature(ctx context.Context, input *UploadProviderChecksumSignatureInput) error
	UploadProviderPlatformBinary(ctx context.Context, input *UploadProviderPlatformBinaryInput) error
	UploadProviderPlatformPackageToMirror(ctx context.Context, input *UploadProviderPlatformPackageToMirrorInput) error
	UploadPlanCache(ctx context.Context, input *UploadPlanCacheInput) error
	UploadPlanData(ctx context.Context, input *UploadPlanDataInput) error
	DownloadStateVersion(ctx context.Context, input *DownloadStateVersionInput) error
	DownloadPlanCache(ctx context.Context, input *DownloadPlanCacheInput) error
	DownloadPlanJSON(ctx context.Context, input *DownloadPlanJSONInput) error
	DownloadPackage(ctx context.Context, input *DownloadPackageInput) error
}

// restClient handles REST API calls to the upstream Terraform-compatible API.
type restClient struct {
	baseURL           *url.URL
	tokenResolver     TokenResolver
	httpClient        *http.Client
	serviceDiscoverer provider.ServiceDiscoverer
}

// NewRESTClient creates a new REST client for interacting with the upstream Terraform REST API.
func NewRESTClient(cfg *RESTClientConfig) (RESTClient, error) {
	baseURL, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = 3
		retryClient.Logger = nil
		retryClient.RetryWaitMin = 10 * time.Second
		retryClient.RetryWaitMax = 60 * time.Second

		if cfg.InsecureSkipVerify {
			baseTransport := http.DefaultTransport.(*http.Transport).Clone()
			baseTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
			retryClient.HTTPClient = &http.Client{Transport: baseTransport}
		}

		httpClient = retryClient.StandardClient()

		if cfg.UserAgent != nil {
			httpClient.Transport = &UserAgentTransport{
				UserAgent: *cfg.UserAgent,
				Base:      httpClient.Transport,
			}
		}
	}

	return &restClient{
		baseURL:           baseURL,
		tokenResolver:     cfg.TokenResolver,
		httpClient:        httpClient,
		serviceDiscoverer: provider.NewServiceDiscoverer(httpClient),
	}, nil
}

// UploadConfigurationVersion uploads a directory as a tar.gz file.
func (c *restClient) UploadConfigurationVersion(ctx context.Context, input *UploadConfigurationVersionInput) error {
	discovered, err := c.serviceDiscoverer.DiscoverTFEServices(ctx, c.baseURL.String())
	if err != nil {
		return fmt.Errorf("failed to discover tfe v2 service: %w", err)
	}

	serviceURL, ok := discovered.Services[provider.TFEServiceID]
	if !ok {
		return fmt.Errorf("service url for %q not found", provider.TFEServiceID)
	}

	s, err := slug.New(input.DirectoryPath)
	if err != nil {
		return fmt.Errorf("failed to create slug: %w", err)
	}
	defer os.Remove(s.SlugPath)

	fileInfo, err := os.Stat(s.SlugPath)
	if err != nil {
		return fmt.Errorf("failed to stat slug file: %w", err)
	}

	reader, err := s.Open()
	if err != nil {
		return fmt.Errorf("failed to open slug: %w", err)
	}
	defer reader.Close()

	uploadURL := serviceURL.JoinPath("workspaces", input.WorkspaceID, "configuration-versions", input.ConfigVersionID, "upload").String()

	return c.doPut(ctx, uploadURL, reader, fileInfo.Size())
}

// DownloadConfigurationVersion downloads a configuration version.
func (c *restClient) DownloadConfigurationVersion(ctx context.Context, input *DownloadConfigurationVersionInput) error {
	discovered, err := c.serviceDiscoverer.DiscoverTFEServices(ctx, c.baseURL.String())
	if err != nil {
		return fmt.Errorf("failed to discover tfe v2 service: %w", err)
	}

	serviceURL, ok := discovered.Services[provider.TFEServiceID]
	if !ok {
		return fmt.Errorf("service url for %q not found", provider.TFEServiceID)
	}

	downloadURL := serviceURL.JoinPath("configuration-versions", input.ConfigVersionID, "content").String()

	return c.doGet(ctx, downloadURL, input.Writer, contentTypeOctetStream)
}

// UploadModuleVersion uploads a module version package.
func (c *restClient) UploadModuleVersion(ctx context.Context, input *UploadModuleVersionInput) error {
	stat, err := os.Stat(input.PackagePath)
	if err != nil {
		return err
	}

	reader, err := os.Open(input.PackagePath) // nosemgrep: gosec.G304-1
	if err != nil {
		return err
	}
	defer reader.Close()

	uploadURL := c.baseURL.JoinPath("v1", "module-registry", "versions", input.ModuleVersionID, "upload").String()

	return c.doPut(ctx, uploadURL, reader, stat.Size())
}

// UploadProviderReadme uploads a provider README file.
func (c *restClient) UploadProviderReadme(ctx context.Context, input *UploadProviderReadmeInput) error {
	reader, err := os.Open(input.ReadmePath) // nosemgrep: gosec.G304-1
	if err != nil {
		return err
	}
	defer reader.Close()

	stat, err := reader.Stat()
	if err != nil {
		return err
	}

	uploadURL := c.baseURL.JoinPath("v1", "provider-registry", "versions", input.ProviderVersionID, "readme", "upload").String()

	return c.doPut(ctx, uploadURL, reader, stat.Size())
}

// UploadProviderChecksums uploads provider checksums file.
func (c *restClient) UploadProviderChecksums(ctx context.Context, input *UploadProviderChecksumsInput) error {
	reader, err := os.Open(input.ChecksumsPath) // nosemgrep: gosec.G304-1
	if err != nil {
		return err
	}
	defer reader.Close()

	stat, err := reader.Stat()
	if err != nil {
		return err
	}

	uploadURL := c.baseURL.JoinPath("v1", "provider-registry", "versions", input.ProviderVersionID, "checksums", "upload").String()

	return c.doPut(ctx, uploadURL, reader, stat.Size())
}

// UploadProviderChecksumSignature uploads provider checksum signature file.
func (c *restClient) UploadProviderChecksumSignature(ctx context.Context, input *UploadProviderChecksumSignatureInput) error {
	reader, err := os.Open(input.SignaturePath) // nosemgrep: gosec.G304-1
	if err != nil {
		return err
	}
	defer reader.Close()

	stat, err := reader.Stat()
	if err != nil {
		return err
	}

	uploadURL := c.baseURL.JoinPath("v1", "provider-registry", "versions", input.ProviderVersionID, "signature", "upload").String()

	return c.doPut(ctx, uploadURL, reader, stat.Size())
}

// UploadProviderPlatformBinary uploads a provider platform binary.
func (c *restClient) UploadProviderPlatformBinary(ctx context.Context, input *UploadProviderPlatformBinaryInput) error {
	reader, err := os.Open(input.BinaryPath) // nosemgrep: gosec.G304-1
	if err != nil {
		return err
	}
	defer reader.Close()

	stat, err := reader.Stat()
	if err != nil {
		return err
	}

	uploadURL := c.baseURL.JoinPath("v1", "provider-registry", "platforms", input.PlatformID, "upload").String()

	return c.doPut(ctx, uploadURL, reader, stat.Size())
}

// UploadProviderPlatformPackageToMirror uploads a provider platform package to mirror.
func (c *restClient) UploadProviderPlatformPackageToMirror(ctx context.Context, input *UploadProviderPlatformPackageToMirrorInput) error {
	uploadURL := c.baseURL.JoinPath("v1", "provider-mirror", "providers", input.VersionMirrorID, input.OS, input.Arch, "upload").String()

	return c.doPut(ctx, uploadURL, input.Reader, -1)
}

// UploadPlanCache uploads a plan cache binary.
func (c *restClient) UploadPlanCache(ctx context.Context, input *UploadPlanCacheInput) error {
	uploadURL := c.baseURL.JoinPath("v1", "plans", input.PlanID, "content").String()

	return c.doPut(ctx, uploadURL, input.Reader, -1)
}

// UploadPlanData uploads plan JSON data and provider schemas.
func (c *restClient) UploadPlanData(ctx context.Context, input *UploadPlanDataInput) error {
	uploadURL := c.baseURL.JoinPath("v1", "plans", input.PlanID, "content.json").String()

	return c.doPut(ctx, uploadURL, input.Reader, -1)
}

// DownloadStateVersion downloads a state version.
func (c *restClient) DownloadStateVersion(ctx context.Context, input *DownloadStateVersionInput) error {
	discovered, err := c.serviceDiscoverer.DiscoverTFEServices(ctx, c.baseURL.String())
	if err != nil {
		return fmt.Errorf("failed to discover tfe v2 service: %w", err)
	}

	serviceURL, ok := discovered.Services[provider.TFEServiceID]
	if !ok {
		return fmt.Errorf("service url for %q not found", provider.TFEServiceID)
	}

	downloadURL := serviceURL.JoinPath("state-versions", input.StateVersionID, "content").String()

	return c.doGet(ctx, downloadURL, input.Writer, contentTypeOctetStream)
}

// DownloadPlanCache downloads a plan cache binary.
func (c *restClient) DownloadPlanCache(ctx context.Context, input *DownloadPlanCacheInput) error {
	discovered, err := c.serviceDiscoverer.DiscoverTFEServices(ctx, c.baseURL.String())
	if err != nil {
		return fmt.Errorf("failed to discover tfe v2 service: %w", err)
	}

	serviceURL, ok := discovered.Services[provider.TFEServiceID]
	if !ok {
		return fmt.Errorf("service url for %q not found", provider.TFEServiceID)
	}

	downloadURL := serviceURL.JoinPath("plans", input.PlanID, "content").String()

	return c.doGet(ctx, downloadURL, input.Writer, contentTypeOctetStream)
}

// DownloadPlanJSON downloads a plan's JSON representation.
func (c *restClient) DownloadPlanJSON(ctx context.Context, input *DownloadPlanJSONInput) error {
	discovered, err := c.serviceDiscoverer.DiscoverTFEServices(ctx, c.baseURL.String())
	if err != nil {
		return fmt.Errorf("failed to discover tfe v2 service: %w", err)
	}

	serviceURL, ok := discovered.Services[provider.TFEServiceID]
	if !ok {
		return fmt.Errorf("service url for %q not found", provider.TFEServiceID)
	}

	downloadURL := serviceURL.JoinPath("plans", input.PlanID, "json-output").String()

	return c.doGet(ctx, downloadURL, input.Writer, contentTypeOctetStream)
}

// DownloadPackage streams a package version's tar.gz content into the input writer. It uses the
// package registry version-id download endpoint, which returns a presigned download URL in the
// X-Download-Url header (authorization handled by the package registry service); this client then
// fetches that URL directly. A version that cannot be found surfaces as ErrPackageNotFound.
func (c *restClient) DownloadPackage(ctx context.Context, input *DownloadPackageInput) error {
	endpoint := c.baseURL.JoinPath(
		"v1", "package-registry", "versions", input.PackageVersionID, "download",
	).String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	authToken, err := c.tokenResolver.Token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("%w: %s", ErrPackageNotFound, string(bodyBytes))
	}
	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("failed to get package download URL, status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	presignedURL := resp.Header.Get("X-Download-Url")
	if presignedURL == "" {
		return fmt.Errorf("package download response missing X-Download-Url header")
	}

	// The presigned URL is self-authorizing; fetch it without the bearer token.
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, presignedURL, nil)
	if err != nil {
		return err
	}

	dlResp, err := c.httpClient.Do(dlReq)
	if err != nil {
		return err
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(dlResp.Body, 1024))
		return fmt.Errorf("package download failed with status code %d: %s", dlResp.StatusCode, string(bodyBytes))
	}

	_, err = io.Copy(input.Writer, dlResp.Body)

	return err
}

func (c *restClient) doPut(ctx context.Context, url string, body io.Reader, length int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, body)
	if err != nil {
		return err
	}

	authToken, err := c.tokenResolver.Token(ctx)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", contentTypeOctetStream)

	if length >= 0 {
		req.ContentLength = length
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("upload failed with status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *restClient) doGet(ctx context.Context, url string, writer io.Writer, accept string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	authToken, err := c.tokenResolver.Token(ctx)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+authToken)

	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("download failed with status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	_, err = io.Copy(writer, resp.Body)

	return err
}
