package controllers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providermirror"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// AvailableProviderVersionsResponse represents the response for GetAvailableProviderVersions.
type AvailableProviderVersionsResponse struct {
	Versions map[string]struct{} `json:"versions"`
}

// AvailableInstallationPackagesResponse represents the response for GetAvailableInstallationPackages.
type AvailableInstallationPackagesResponse struct {
	Archives map[string]any `json:"archives"`
}

// InstallationPackageResponse represents the response for GetInstallationPackage.
type InstallationPackageResponse struct {
	URL    string   `json:"url"`
	Hashes []string `json:"hashes"`
}

type providerMirrorController struct {
	logger                logger.Logger
	respWriter            response.Writer
	jwtAuthMiddleware     middleware.Handler
	providerMirrorService providermirror.Service
}

// NewProviderMirrorController creates an instance of providerMirrorController.
func NewProviderMirrorController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	providerMirrorService providermirror.Service,
) Controller {
	return &providerMirrorController{
		logger:                logger,
		respWriter:            respWriter,
		jwtAuthMiddleware:     jwtAuthMiddleware,
		providerMirrorService: providerMirrorService,
	}
}

// RegisterRoutes adds health routes to the router
func (c *providerMirrorController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Get("/provider-mirror/providers/{groupName}/{hostname}/{namespace}/{type}/index.json", c.GetAvailableProviderVersions)
	router.Get("/provider-mirror/providers/{groupName}/{hostname}/{namespace}/{type}/{version:.+\\.json}", c.GetAvailableInstallationPackages)
	router.Get("/provider-mirror/providers/{groupName}/{hostname}/{namespace}/{type}/{version}/{os}/{arch}", c.GetInstallationPackage)

	router.Put("/provider-mirror/providers/{versionMirrorId}/{os}/{architecture}/upload", c.UploadInstallationPackage)
}

func (c *providerMirrorController) GetAvailableProviderVersions(w http.ResponseWriter, r *http.Request) {
	input := &providermirror.GetAvailableProviderVersionsInput{
		RegistryHostname:  chi.URLParam(r, "hostname"),
		RegistryNamespace: chi.URLParam(r, "namespace"),
		Type:              chi.URLParam(r, "type"),
		GroupPath:         chi.URLParam(r, "groupName"),
	}

	versions, err := c.providerMirrorService.GetAvailableProviderVersions(r.Context(), input)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, &AvailableProviderVersionsResponse{Versions: versions}, http.StatusOK)
}

func (c *providerMirrorController) GetAvailableInstallationPackages(w http.ResponseWriter, r *http.Request) {
	input := &providermirror.GetAvailableInstallationPackagesInput{
		Type:              chi.URLParam(r, "type"),
		RegistryNamespace: chi.URLParam(r, "namespace"),
		RegistryHostname:  chi.URLParam(r, "hostname"),
		SemanticVersion:   strings.TrimSuffix(chi.URLParam(r, "version"), ".json"), // Remove the .json suffix.
		GroupPath:         chi.URLParam(r, "groupName"),
	}

	packages, err := c.providerMirrorService.GetAvailableInstallationPackages(r.Context(), input)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, &AvailableInstallationPackagesResponse{Archives: packages}, http.StatusOK)
}

func (c *providerMirrorController) GetInstallationPackage(w http.ResponseWriter, r *http.Request) {
	input := &providermirror.GetInstallationPackageInput{
		GroupPath:         chi.URLParam(r, "groupName"),
		RegistryHostname:  chi.URLParam(r, "hostname"),
		RegistryNamespace: chi.URLParam(r, "namespace"),
		Type:              chi.URLParam(r, "type"),
		SemanticVersion:   chi.URLParam(r, "version"),
		OS:                chi.URLParam(r, "os"),
		Arch:              chi.URLParam(r, "arch"),
	}

	pkg, err := c.providerMirrorService.GetInstallationPackage(r.Context(), input)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, &InstallationPackageResponse{URL: pkg.URL, Hashes: pkg.Hashes}, http.StatusOK)
}

func (c *providerMirrorController) UploadInstallationPackage(w http.ResponseWriter, r *http.Request) {
	input := &providermirror.UploadInstallationPackageInput{
		Data:            r.Body,
		VersionMirrorID: gid.FromGlobalID(chi.URLParam(r, "versionMirrorId")),
		OS:              chi.URLParam(r, "os"),
		Architecture:    chi.URLParam(r, "architecture"),
	}

	if err := c.providerMirrorService.UploadInstallationPackage(r.Context(), input); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, nil, http.StatusOK)
}
