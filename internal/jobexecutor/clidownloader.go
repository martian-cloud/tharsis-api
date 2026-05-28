package jobexecutor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
)

const (
	// hashicorpReleasesBaseURL is used to download checksums for a Terraform CLI binary.
	hashicorpReleasesBaseURL = "https://releases.hashicorp.com"

	// hashicorpWellKnownGPGKeyURL is the official endpoint for HashiCorp's GPG signing keys.
	hashicorpWellKnownGPGKeyURL = "https://www.hashicorp.com/.well-known/pgp-key.txt"

	// maxSigningKeysResponseSize caps the well-known endpoint response to prevent excessive memory usage.
	maxSigningKeysResponseSize = 1024 * 1024 // 1 MB
)

type cliDownloader struct {
	httpClient *http.Client
	client     jobclient.Client
}

func newCLIDownloader(
	httpClient *http.Client,
	client jobclient.Client,
) *cliDownloader {
	return &cliDownloader{
		httpClient: httpClient,
		client:     client,
	}
}

// Download downloads a Terraform CLI binary, unzips it,
// and stores in "terraform_cli" subdirectory where the job executor lives.
func (c *cliDownloader) Download(ctx context.Context, terraformVersion string) (string, error) {
	// Actual name of the zip file. Ex: terraform_1.2.2_linux_amd64.zip
	zipFilename := strings.Join([]string{
		"terraform",
		terraformVersion,
		runtime.GOOS,
		runtime.GOARCH,
	}, "_") + ".zip"

	// Build checksum map for this particular Terraform version.
	checksumMap, err := c.downloadTerraformCLIChecksums(terraformVersion)
	if err != nil {
		return "", err
	}

	checksum, ok := checksumMap[zipFilename]
	if !ok {
		return "", fmt.Errorf("no checksum found for file %s", zipFilename)
	}

	// Get the download URL.
	downloadURL, err := c.client.CreateTerraformCLIDownloadURL(ctx, terraformVersion, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}

	response, err := c.httpClient.Get(downloadURL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download Terraform CLI binary response status: %s", response.Status)
	}

	// Verify the mime type.
	mimeType := response.Header.Get("content-type")
	if !contentTypeIsZip(mimeType) {
		return "", fmt.Errorf("unexpected mime type: expected %v, got %s", zipMimeTypes, mimeType)
	}

	var (
		packageReader io.Reader    // For saving to actual file.
		buffer        bytes.Buffer // For computing checksum.
	)
	reader := io.TeeReader(response.Body, &buffer)
	packageReader = &buffer

	// Calculate and verify the checksum.
	if err = compareChecksum(reader, checksum, response.ContentLength); err != nil {
		return "", fmt.Errorf("failed to verify Terraform CLI binary checksum: %v", err)
	}

	finalDirectory, err := os.MkdirTemp("", "terraform_cli")
	if err != nil {
		return "", fmt.Errorf("failed to make temp directory to unzip Terraform CLI: %v", err)
	}

	// Create, unzip and get the executable's full path.
	if err := unzip(packageReader, finalDirectory, zipFilename); err != nil {
		return "", fmt.Errorf("failed to unzip Terraform CLI: %v", err)
	}

	// Get the full path to the binary so we can modify its permissions.
	execPath := getBinaryPath(finalDirectory)

	// Make the binary an executable. Gives full permissions to owner.
	if err := os.Chmod(execPath, 0700); err != nil { // nosemgrep: gosec.G304-1, gosec.G302-1
		return "", err
	}

	return execPath, nil
}

func (c *cliDownloader) downloadTerraformCLIChecksums(version string) (map[string][]byte, error) {

	// Build the download URLs.
	checksumURL, signatureURL := getChecksumURLs(version)

	// Download checksum signatureResponse.
	signatureResponse, err := c.httpClient.Get(signatureURL)
	if err != nil {
		return nil, err
	}
	defer signatureResponse.Body.Close()

	if signatureResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download Terraform CLI release sums signature response status: %s", signatureResponse.Status)
	}

	// Download checksumsResponse.
	checksumsResponse, err := c.httpClient.Get(checksumURL)
	if err != nil {
		return nil, err
	}
	defer checksumsResponse.Body.Close()

	if checksumsResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download Terraform CLI release sums response status: %s", checksumsResponse.Status)
	}

	// Fetch signing keys from the well-known endpoint.
	keyRing, err := c.fetchSigningKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch signing keys for the Terraform CLI: %w", err)
	}

	var (
		checksumReader io.Reader    // For building checksum map.
		buffer         bytes.Buffer // For validating checksum signature.
	)
	sumsReader := io.TeeReader(checksumsResponse.Body, &buffer)
	checksumReader = &buffer

	// Verify the signature on the checksum file.
	if err = verifySumsSignature(sumsReader, signatureResponse.Body, keyRing); err != nil {
		return nil, fmt.Errorf("failed to verify Terraform CLI checksums signature: %v", err)
	}

	checksumMap, err := fileMapFromChecksums(checksumReader)
	if err != nil {
		return nil, fmt.Errorf("failed to build Terraform CLI checksum map: %v", err)
	}

	// If the map is empty, then the download failed.
	if len(checksumMap) == 0 {
		return nil, fmt.Errorf("no Terraform CLI checksums found in response")
	}

	return checksumMap, nil
}

// fetchSigningKeys fetches GPG signing keys from HashiCorp's well-known endpoint.
// It handles responses containing either a single armored block (possibly with
// multiple entities) or multiple concatenated armored blocks.
func (c *cliDownloader) fetchSigningKeys() (openpgp.EntityList, error) {
	resp, err := c.httpClient.Get(hashicorpWellKnownGPGKeyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to request signing keys from the well-known endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch signing keys from the well-known endpoint: unexpected status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSigningKeysResponseSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read signing keys response body: %w", err)
	}

	// Parse all armored PGP blocks in the body. openpgp.ReadArmoredKeyRing only
	// reads the first block, so we split on block boundaries and decode each
	// individually to handle multiple concatenated armored blocks.
	var keyRing openpgp.EntityList
	var parseErrs []error
	const blockStart = "-----BEGIN PGP PUBLIC KEY BLOCK-----"
	blocks := splitArmoredBlocks(string(body), blockStart)
	for _, blockText := range blocks {
		block, err := armor.Decode(strings.NewReader(blockText))
		if err != nil {
			parseErrs = append(parseErrs, fmt.Errorf("armor decode failed: %w", err))
			continue
		}
		entities, err := openpgp.ReadKeyRing(block.Body)
		if err != nil {
			parseErrs = append(parseErrs, fmt.Errorf("read key ring failed: %w", err))
			continue
		}
		keyRing = append(keyRing, entities...)
	}

	if len(keyRing) == 0 {
		if len(parseErrs) > 0 {
			return nil, fmt.Errorf("no valid signing keys returned: %w", errors.Join(parseErrs...))
		}
		return nil, fmt.Errorf("no valid signing keys returned")
	}

	// Check if all keys are expired.
	now := time.Now()
	allExpired := true
	for _, entity := range keyRing {
		if entity.PrimaryKey != nil {
			identity := entity.PrimaryIdentity()
			if identity == nil || identity.SelfSignature == nil {
				continue
			}
			lifetime := identity.SelfSignature.KeyLifetimeSecs
			if lifetime == nil || entity.PrimaryKey.CreationTime.Add(time.Duration(*lifetime)*time.Second).After(now) {
				allExpired = false
				break
			}
		}
	}
	if allExpired {
		return nil, fmt.Errorf("signing key expired")
	}

	return keyRing, nil
}

// getChecksumURLs returns the URLs to download checksum file and signature.
func getChecksumURLs(version string) (string, string) {
	// URLs to retrieve SHA256 checksums.
	checksumFilename := strings.Join([]string{"terraform", version, "SHA256SUMS"}, "_")
	checksumURL := fmt.Sprintf(
		"%s/terraform/%s/%s",
		hashicorpReleasesBaseURL,
		url.PathEscape(version),
		url.PathEscape(checksumFilename),
	)

	// Use the generic .sig file (not key-ID-specific).
	signaturesFilename := checksumFilename + ".sig"
	signatureURL := fmt.Sprintf(
		"%s/terraform/%s/%s",
		hashicorpReleasesBaseURL,
		url.PathEscape(version),
		url.PathEscape(signaturesFilename),
	)

	return checksumURL, signatureURL
}

// getBinaryPath returns the Terraform CLI executable name.
func getBinaryPath(finalDirectory string) string {
	binaryName := "terraform"

	// Windows has a .exe appended to binary name.
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	return filepath.Join(finalDirectory, binaryName)
}

// splitArmoredBlocks splits a string containing one or more concatenated
// PGP armored blocks into individual block strings. Each returned string
// begins with the blockStart marker and ends at the start of the next
// block (or end of input).
func splitArmoredBlocks(body, blockStart string) []string {
	var blocks []string
	remaining := body
	for {
		idx := strings.Index(remaining, blockStart)
		if idx < 0 {
			break
		}
		remaining = remaining[idx:]
		next := strings.Index(remaining[len(blockStart):], blockStart)
		if next < 0 {
			// Last block — take the rest.
			blocks = append(blocks, remaining)
			break
		}
		blocks = append(blocks, remaining[:len(blockStart)+next])
		remaining = remaining[len(blockStart)+next:]
	}
	return blocks
}
