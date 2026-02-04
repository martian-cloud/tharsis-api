// Package provider provides clients for interacting with upstream provider registries
// using Terraform's Provider Registry Protocol.
package provider

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/apparentlymart/go-versions/versions"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
)

const (
	// TerraformPublicRegistryHost is the public Terraform registry that doesn't require authentication.
	TerraformPublicRegistryHost = "registry.terraform.io"
	// ProvidersServiceID is the Terraform service discovery ID for the provider registry protocol.
	ProvidersServiceID = "providers.v1"
)

const (
	// maxErrorBodyBytes is the maximum size of error response bodies to read.
	maxErrorBodyBytes = 1024 * 1024 // 1 MiB
	// acceptJSON is the Accept header for JSON responses.
	acceptJSON = "application/json"
	// acceptZipArchive is the Accept header for provider package downloads.
	acceptZipArchive = "application/zip, application/x-zip-compressed, application/octet-stream"
	// acceptText is the Accept header for checksums and signature files.
	acceptText = "text/plain, application/octet-stream"
)

// Provider represents a Terraform provider with its registry location.
type Provider struct {
	Hostname  string
	Namespace string
	Type      string
}

// String returns the provider address. The public Terraform registry hostname
// is omitted since it's the default registry (e.g., "hashicorp/aws").
func (p *Provider) String() string {
	if p.Hostname == TerraformPublicRegistryHost {
		return fmt.Sprintf("%s/%s", p.Namespace, p.Type)
	}

	return fmt.Sprintf("%s/%s/%s", p.Hostname, p.Namespace, p.Type)
}

// VersionInfo contains information about a provider version.
type VersionInfo struct {
	Version   string
	Platforms []Platform
}

// Platform represents an OS/arch combination.
type Platform struct {
	OS   string
	Arch string
}

// PackageInfo is the response returned when querying for a particular
// installation package. It is used to find the SHA256SUMS, SHA256SUMS.sig files
// and the associated key files needed to verify their authenticity.
// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#find-a-provider-package
type PackageInfo struct {
	DownloadURL         string
	SHASumsURL          string
	SHASumsSignatureURL string
	GPGASCIIArmors      []string
}

// RequestOption is a functional option for requests.
type RequestOption func(*requestOptions)

type requestOptions struct {
	token string
}

// WithToken sets the authentication token for the request.
func WithToken(token string) RequestOption {
	return func(o *requestOptions) {
		o.token = token
	}
}

type listVersionsResponse struct {
	Versions []struct {
		Version   string `json:"version"`
		Platforms []struct {
			OS   string `json:"os"`
			Arch string `json:"arch"`
		} `json:"platforms"`
	} `json:"versions"`
	Warnings []string `json:"warnings"`
}

type packageQueryResponse struct {
	DownloadURL         string `json:"download_url"`
	SHASumsURL          string `json:"shasums_url"`
	SHASumsSignatureURL string `json:"shasums_signature_url"`
	SigningKeys         struct {
		GPGPublicKeys []struct {
			ASCIIArmor string `json:"ascii_armor"`
		} `json:"gpg_public_keys"`
	} `json:"signing_keys"`
}

// Checksums holds verified SHA256 checksums for provider packages.
// Keys are package filenames (e.g., "terraform-provider-aws_5.0.0_linux_amd64.zip").
type Checksums map[string][]byte

// GetZipHash returns the zh: formatted hash for a provider package.
// Hash format complies with Terraform's zip hash format:
// https://github.com/hashicorp/terraform/blob/d49e991c3c33c10b26c120465466d41f96e073de/internal/getproviders/hash.go#L330
func (c Checksums) GetZipHash(filename string) (string, bool) {
	checksum, ok := c[filename]
	if !ok {
		return "", false
	}

	return fmt.Sprintf("zh:%x", checksum), true
}

//go:generate go tool mockery --name RegistryProtocol --inpackage --case underscore

// RegistryProtocol is the interface for interacting with upstream provider registries
// using Terraform's Provider Registry Protocol.
type RegistryProtocol interface {
	ListVersions(ctx context.Context, provider *Provider, opts ...RequestOption) ([]VersionInfo, error)
	GetPackageInfo(ctx context.Context, provider *Provider, version, os, arch string, opts ...RequestOption) (*PackageInfo, error)
	DownloadPackage(ctx context.Context, downloadURL string) (io.ReadCloser, int64, error)
	GetChecksums(ctx context.Context, packageInfo *PackageInfo) (Checksums, error)
}

type registryClient struct {
	httpClient *http.Client
	discovery  serviceDiscoverer
}

// NewRegistryClient creates a new provider registry client.
func NewRegistryClient(httpClient *http.Client) RegistryProtocol {
	return &registryClient{
		httpClient: httpClient,
		discovery:  &quietDisco{inner: disco.New()},
	}
}

// ListVersions lists the available provider versions and platforms they support
// by contacting the Terraform Registry API the provider is associated with.
// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#list-available-versions
func (c *registryClient) ListVersions(ctx context.Context, provider *Provider, opts ...RequestOption) ([]VersionInfo, error) {
	options := &requestOptions{}
	for _, opt := range opts {
		opt(options)
	}

	serviceURL, err := c.discovery.DiscoverServiceURL(svchost.Hostname(provider.Hostname), ProvidersServiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to discover provider registry's service URL: %w", err)
	}

	endpoint, err := url.Parse(path.Join(provider.Namespace, provider.Type, "versions"))
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, http.MethodGet, serviceURL.ResolveReference(endpoint).String(), options.token, acceptJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to perform http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		return nil, fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, string(body))
	}

	var response listVersionsResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if len(response.Warnings) > 0 {
		return nil, fmt.Errorf("provider versions endpoint returned warnings: %s", strings.Join(response.Warnings, "; "))
	}

	if len(response.Versions) == 0 {
		return nil, fmt.Errorf("no versions found for provider %s", provider)
	}

	versionInfos := make([]VersionInfo, len(response.Versions))
	for i, v := range response.Versions {
		platforms := make([]Platform, len(v.Platforms))
		for j, p := range v.Platforms {
			platforms[j] = Platform{OS: p.OS, Arch: p.Arch}
		}
		versionInfos[i] = VersionInfo{
			Version:   v.Version,
			Platforms: platforms,
		}
	}

	return versionInfos, nil
}

// GetPackageInfo attempts to locate the provider package at the provider's registry.
// It visits the endpoint for the target provider and parses the JSON response, which should
// give us access to the SHA256SUMS, SHA256SUMS.sig and GPG key used to sign the checksums file.
// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#find-a-provider-package
func (c *registryClient) GetPackageInfo(ctx context.Context, provider *Provider, version, os, arch string, opts ...RequestOption) (*PackageInfo, error) {
	options := &requestOptions{}
	for _, opt := range opts {
		opt(options)
	}

	serviceURL, err := c.discovery.DiscoverServiceURL(svchost.Hostname(provider.Hostname), ProvidersServiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to discover provider registry's service URL: %w", err)
	}

	// Build the URL to the provider's download endpoint which will give us access to the
	// SHASUMS, SHASUMS.sig and GPG key used to sign the checksums file. These are generally
	// hosted in a different location than the provider's registry.
	endpoint, err := url.Parse(path.Join(provider.Namespace, provider.Type, version, "download", os, arch))
	if err != nil {
		return nil, fmt.Errorf("failed to build package download URL: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, serviceURL.ResolveReference(endpoint).String(), options.token, acceptJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get package download URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		return nil, fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, string(body))
	}

	var response packageQueryResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode download package response body: %w", err)
	}

	gpgKeys := make([]string, len(response.SigningKeys.GPGPublicKeys))
	for i, k := range response.SigningKeys.GPGPublicKeys {
		gpgKeys[i] = k.ASCIIArmor
	}

	return &PackageInfo{
		DownloadURL:         response.DownloadURL,
		SHASumsURL:          response.SHASumsURL,
		SHASumsSignatureURL: response.SHASumsSignatureURL,
		GPGASCIIArmors:      gpgKeys,
	}, nil
}

// DownloadPackage downloads a provider package from the given URL.
// Note: Token is not passed as download URLs are typically presigned and already contain authentication.
func (c *registryClient) DownloadPackage(ctx context.Context, downloadURL string) (io.ReadCloser, int64, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, downloadURL, "", acceptZipArchive)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to download package: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		resp.Body.Close()
		return nil, 0, fmt.Errorf("download failed with status: %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, resp.ContentLength, nil
}

// GetChecksums downloads and verifies the checksums file from the provider registry.
// Note: Token is not passed for checksum downloads as these URLs are typically presigned
// and already contain authentication.
func (c *registryClient) GetChecksums(ctx context.Context, packageInfo *PackageInfo) (Checksums, error) {
	checksumResp, err := c.doRequest(ctx, http.MethodGet, packageInfo.SHASumsURL, "", acceptText)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksums file: %w", err)
	}
	defer checksumResp.Body.Close()

	if checksumResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(checksumResp.Body, maxErrorBodyBytes))
		return nil, fmt.Errorf("unexpected status returned from checksums download: %d: %s", checksumResp.StatusCode, string(body))
	}

	signatureResp, err := c.doRequest(ctx, http.MethodGet, packageInfo.SHASumsSignatureURL, "", acceptText)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksums signature file: %w", err)
	}
	defer signatureResp.Body.Close()

	if signatureResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(signatureResp.Body, maxErrorBodyBytes))
		return nil, fmt.Errorf("unexpected status returned from checksums signature download: %d: %s", signatureResp.StatusCode, string(body))
	}

	var buffer bytes.Buffer
	sigReader := io.TeeReader(checksumResp.Body, &buffer)

	if err = verifySumsSignature(sigReader, signatureResp.Body, packageInfo.GPGASCIIArmors); err != nil {
		return nil, fmt.Errorf("failed to verify checksum signature: %w", err)
	}

	checksums := make(Checksums)
	scanner := bufio.NewScanner(&buffer)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 {
			return nil, fmt.Errorf("unexpected checksum line format: %s", scanner.Text())
		}

		hexBytes, err := hex.DecodeString(parts[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse checksums: %w", err)
		}

		if len(hexBytes) != sha256.Size {
			return nil, fmt.Errorf("unexpected checksum size: expected %d, got %d", sha256.Size, len(hexBytes))
		}

		checksums[parts[1]] = hexBytes
	}

	if len(checksums) == 0 {
		return nil, fmt.Errorf("no checksums found after parsing response")
	}

	return checksums, nil
}

func (c *registryClient) doRequest(ctx context.Context, method, url, token, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.httpClient.Do(req)
}

// NewProvider parses a provider from the input. It validates
// all components to make sure they comply with Terraform's standards.
func NewProvider(hostname, namespace, providerType string) (*Provider, error) {
	if _, err := tfaddr.ParseProviderPart(namespace); err != nil {
		return nil, fmt.Errorf("invalid registry namespace: %w", err)
	}

	if _, err := tfaddr.ParseProviderPart(providerType); err != nil {
		return nil, fmt.Errorf("invalid provider type: %w", err)
	}

	if _, err := svchost.ForComparison(hostname); err != nil {
		return nil, fmt.Errorf("invalid registry hostname: %w", err)
	}

	return &Provider{
		Hostname:  hostname,
		Namespace: namespace,
		Type:      providerType,
	}, nil
}

// GetPlatformForVersion finds the target version in the list and returns a platform it supports.
func GetPlatformForVersion(targetVersion string, versionInfos []VersionInfo) (*Platform, error) {
	for _, vi := range versionInfos {
		if vi.Version == targetVersion && len(vi.Platforms) > 0 {
			return &vi.Platforms[0], nil
		}
	}

	return nil, fmt.Errorf("no supported platforms found or provider version not supported")
}

// GetPackageName returns the package name in Terraform's format:
// terraform-provider-<provider_type>_<version>_<os>_<arch>.zip
func GetPackageName(providerType, version, os, arch string) string {
	return fmt.Sprintf("terraform-provider-%s_%s_%s_%s.zip", providerType, version, os, arch)
}

// FindLatestVersion finds the latest semantic version from a list of version infos.
func FindLatestVersion(versionInfos []VersionInfo) (string, error) {
	if len(versionInfos) == 0 {
		return "", fmt.Errorf("no versions provided")
	}

	versionsList := make(versions.List, len(versionInfos))
	for i, vi := range versionInfos {
		v, err := versions.ParseVersion(vi.Version)
		if err != nil {
			return "", fmt.Errorf("failed to parse provider version %q: %w", vi.Version, err)
		}
		versionsList[i] = v
	}

	return versionsList.Newest().String(), nil
}

func verifySumsSignature(checksums, signature io.Reader, gpgKeys []string) error {
	var matchFound bool
	for _, key := range gpgKeys {
		entityList, err := openpgp.ReadArmoredKeyRing(strings.NewReader(key))
		if err != nil {
			return err
		}

		if _, err := openpgp.CheckDetachedSignature(entityList, checksums, signature, nil); err == nil {
			matchFound = true
			break
		}
	}

	if !matchFound {
		return fmt.Errorf("no matching key found for signature or signature mismatch")
	}

	return nil
}
