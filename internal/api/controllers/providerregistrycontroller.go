package controllers

import (
	"net/http"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-chi/chi/v5"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// RegistryProviderPlatform represents a platform for a provider version
type RegistryProviderPlatform struct {
	OperatingSystem string `json:"os"`
	Arch            string `json:"arch"`
}

// RegistryProviderVersion represents a provider version
type RegistryProviderVersion struct {
	Version   string                     `json:"version"`
	Protocols []string                   `json:"protocols"`
	Platforms []RegistryProviderPlatform `json:"platforms"`
}

// RegistryProviderVersionList contains a list of provider versions
type RegistryProviderVersionList struct {
	Versions []RegistryProviderVersion `json:"versions"`
}

// GPGPublicKey represents a GPG public key used to sign a provider version
type GPGPublicKey struct {
	KeyID          string `json:"key_id"`
	ASCIIArmor     string `json:"ascii_armor"`
	TrustSignature string `json:"trust_signature"`
	Source         string `json:"source"`
	SourceURL      string `json:"source_url"`
}

// SigningKeys contains a list of GPG public keys
type SigningKeys struct {
	GPGPublicKeys []GPGPublicKey `json:"gpg_public_keys"`
}

// RegistryProviderDownloadResponse is the response that adheres to the
// Terraform Provider Registry Protocol
type RegistryProviderDownloadResponse struct {
	SHASumsSignatureURL string      `json:"shasums_signature_url"`
	OperatingSystem     string      `json:"os"`
	Arch                string      `json:"arch"`
	Filename            string      `json:"filename"`
	DownloadURL         string      `json:"download_url"`
	SHASumsURL          string      `json:"shasums_url"`
	SHASum              string      `json:"shasum"`
	Protocols           []string    `json:"protocols"`
	SigningKeys         SigningKeys `json:"signing_keys"`
}

type providerRegistryController struct {
	respWriter              response.Writer
	jwtAuthMiddleware       middleware.Handler
	logger                  logger.Logger
	providerRegistryService providerregistry.Service
}

// NewProviderRegistryController creates an instance of providerRegistryController
func NewProviderRegistryController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	providerRegistryService providerregistry.Service,
) Controller {
	return &providerRegistryController{
		respWriter,
		jwtAuthMiddleware,
		logger,
		providerRegistryService,
	}
}

// RegisterRoutes adds health routes to the router
func (c *providerRegistryController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Get("/provider-registry/providers/{namespace}/{name}/versions", c.GetVersions)
	router.Get("/provider-registry/providers/{namespace}/{name}/{version}/download/{os}/{arch}", c.GetVersion)

	router.Put("/provider-registry/platforms/{platformId}/upload", c.UploadPlatformBinary)
	router.Put("/provider-registry/versions/{providerVersionId}/readme/upload", c.UploadProviderVersionReadme)
	router.Put("/provider-registry/versions/{providerVersionId}/checksums/upload", c.UploadProviderVersionSHASums)
	router.Put("/provider-registry/versions/{providerVersionId}/signature/upload", c.UploadProviderVersionSHASumsSignature)
}

func (c *providerRegistryController) UploadProviderVersionReadme(w http.ResponseWriter, r *http.Request) {
	providerVersionID := gid.FromGlobalID(chi.URLParam(r, "providerVersionId"))

	if err := c.providerRegistryService.UploadProviderVersionReadme(r.Context(), providerVersionID, r.Body); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, nil, http.StatusOK)
}

func (c *providerRegistryController) UploadProviderVersionSHASums(w http.ResponseWriter, r *http.Request) {
	providerVersionID := gid.FromGlobalID(chi.URLParam(r, "providerVersionId"))

	if err := c.providerRegistryService.UploadProviderVersionSHA256Sums(r.Context(), providerVersionID, r.Body); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, nil, http.StatusOK)
}

func (c *providerRegistryController) UploadProviderVersionSHASumsSignature(w http.ResponseWriter, r *http.Request) {
	providerVersionID := gid.FromGlobalID(chi.URLParam(r, "providerVersionId"))

	if err := c.providerRegistryService.UploadProviderVersionSHA256SumsSignature(r.Context(), providerVersionID, r.Body); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, nil, http.StatusOK)
}

func (c *providerRegistryController) UploadPlatformBinary(w http.ResponseWriter, r *http.Request) {
	platformID := gid.FromGlobalID(chi.URLParam(r, "platformId"))

	if err := c.providerRegistryService.UploadProviderPlatformBinary(r.Context(), platformID, r.Body); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, nil, http.StatusOK)
}

func (c *providerRegistryController) GetVersions(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	providerName := chi.URLParam(r, "name")

	provider, err := c.providerRegistryService.GetProviderByAddress(r.Context(), namespace, providerName)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	// Only return providers that have the checksums and signature uploaded
	versionsResponse, err := c.providerRegistryService.GetProviderVersions(r.Context(), &providerregistry.GetProviderVersionsInput{
		ProviderID:               provider.Metadata.ID,
		SHASumsUploaded:          ptr.Bool(true),
		SHASumsSignatureUploaded: ptr.Bool(true),
	})
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	// Only return platforms that have the binary uploaded
	platformsResponse, err := c.providerRegistryService.GetProviderPlatforms(r.Context(), &providerregistry.GetProviderPlatformsInput{
		ProviderID:     &provider.Metadata.ID,
		BinaryUploaded: ptr.Bool(true),
	})
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	platformMap := map[string][]*models.TerraformProviderPlatform{}
	for _, p := range platformsResponse.ProviderPlatforms {
		pCopy := p
		if _, ok := platformMap[p.ProviderVersionID]; !ok {
			platformMap[p.ProviderVersionID] = []*models.TerraformProviderPlatform{}
		}
		platformMap[p.ProviderVersionID] = append(platformMap[p.ProviderVersionID], &pCopy)
	}

	response := RegistryProviderVersionList{Versions: []RegistryProviderVersion{}}

	for _, v := range versionsResponse.ProviderVersions {
		tfeVersion := RegistryProviderVersion{
			Version:   v.SemanticVersion,
			Protocols: v.Protocols,
			Platforms: []RegistryProviderPlatform{},
		}

		platforms, ok := platformMap[v.Metadata.ID]
		if ok {
			for _, p := range platforms {
				tfeVersion.Platforms = append(tfeVersion.Platforms, RegistryProviderPlatform{
					OperatingSystem: p.OperatingSystem,
					Arch:            p.Architecture,
				})
			}
		}

		response.Versions = append(response.Versions, tfeVersion)
	}

	c.respWriter.RespondWithJSON(w, &response, 200)
}

func (c *providerRegistryController) GetVersion(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	providerName := chi.URLParam(r, "name")
	version := chi.URLParam(r, "version")
	os := chi.URLParam(r, "os")
	arch := chi.URLParam(r, "arch")

	provider, err := c.providerRegistryService.GetProviderByAddress(r.Context(), namespace, providerName)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	// Get provider version by provider ID and version
	versionsResponse, err := c.providerRegistryService.GetProviderVersions(r.Context(), &providerregistry.GetProviderVersionsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		ProviderID:               provider.Metadata.ID,
		SemanticVersion:          &version,
		SHASumsUploaded:          ptr.Bool(true),
		SHASumsSignatureUploaded: ptr.Bool(true),
	})
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	if len(versionsResponse.ProviderVersions) == 0 {
		c.respWriter.RespondWithError(w, errors.New("provider version %s not found", version, errors.WithErrorCode(errors.ENotFound)))
		return
	}

	providerVersion := versionsResponse.ProviderVersions[0]

	platformsResponse, err := c.providerRegistryService.GetProviderPlatforms(r.Context(), &providerregistry.GetProviderPlatformsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		ProviderVersionID: &providerVersion.Metadata.ID,
		BinaryUploaded:    ptr.Bool(true),
		OperatingSystem:   &os,
		Architecture:      &arch,
	})
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	if len(platformsResponse.ProviderPlatforms) == 0 {
		c.respWriter.RespondWithError(w, errors.New("provider platform %s_%s not found", os, arch, errors.WithErrorCode(errors.ENotFound)))
		return
	}

	providerPlatform := platformsResponse.ProviderPlatforms[0]

	downloadURLs, err := c.providerRegistryService.GetProviderPlatformDownloadURLs(r.Context(), &providerPlatform)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	downloadResponse := RegistryProviderDownloadResponse{
		Protocols:           providerVersion.Protocols,
		OperatingSystem:     providerPlatform.OperatingSystem,
		Arch:                providerPlatform.Architecture,
		Filename:            providerPlatform.Filename,
		DownloadURL:         downloadURLs.DownloadURL,
		SHASumsURL:          downloadURLs.SHASumsURL,
		SHASumsSignatureURL: downloadURLs.SHASumsSignatureURL,
		SHASum:              providerPlatform.SHASum,
		SigningKeys: SigningKeys{
			GPGPublicKeys: []GPGPublicKey{
				{
					KeyID:      *providerVersion.GetHexGPGKeyID(),
					ASCIIArmor: *providerVersion.GPGASCIIArmor,
				},
			},
		},
	}

	c.respWriter.RespondWithJSON(w, &downloadResponse, 200)
}
