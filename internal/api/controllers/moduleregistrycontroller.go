package controllers

import (
	"net/http"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-chi/chi/v5"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// RegistryModuleVersion represents a module version
type RegistryModuleVersion struct {
	Version string `json:"version"`
}

// RegistryModuleVersionList contains a list of module versions
type RegistryModuleVersionList struct {
	Versions []RegistryModuleVersion `json:"versions"`
}

// RegistryModuleVersionsResponse is the response for the modules versions endpoint
type RegistryModuleVersionsResponse struct {
	Modules []RegistryModuleVersionList `json:"modules"`
}

type moduleRegistryController struct {
	respWriter                  response.Writer
	jwtAuthMiddleware           middleware.Handler
	logger                      logger.Logger
	moduleRegistryService       moduleregistry.Service
	moduleRegistryMaxUploadSize int
}

// NewModuleRegistryController creates an instance of moduleRegistryController
func NewModuleRegistryController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	moduleRegistryService moduleregistry.Service,
	moduleRegistryMaxUploadSize int,
) Controller {
	return &moduleRegistryController{
		respWriter,
		jwtAuthMiddleware,
		logger,
		moduleRegistryService,
		moduleRegistryMaxUploadSize,
	}
}

// RegisterRoutes adds health routes to the router
func (c *moduleRegistryController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Get("/module-registry/modules/{namespace}/{name}/{system}/versions", c.GetModuleVersions)
	router.Get("/module-registry/modules/{namespace}/{name}/{system}/{version}/download", c.GetModuleVersionPackageURL)

	router.Put("/module-registry/versions/{moduleVersionId}/upload", c.UploadModuleVersionPackage)
}

func (c *moduleRegistryController) UploadModuleVersionPackage(w http.ResponseWriter, r *http.Request) {
	moduleVersionID := gid.FromGlobalID(chi.URLParam(r, "moduleVersionId"))

	moduleVersion, err := c.moduleRegistryService.GetModuleVersionByID(r.Context(), moduleVersionID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	// Limit size of request body
	limitReader := http.MaxBytesReader(w, r.Body, int64(c.moduleRegistryMaxUploadSize))
	defer limitReader.Close()

	if err := c.moduleRegistryService.UploadModuleVersionPackage(r.Context(), moduleVersion, limitReader); err != nil {
		if strings.Contains(err.Error(), "read multipart upload data failed, http: request body too large") {
			c.respWriter.RespondWithError(w, terrors.New("upload failed, module size exceeds maximum size of %d bytes", c.moduleRegistryMaxUploadSize, terrors.WithErrorCode(errors.ETooLarge)))
		} else {
			c.respWriter.RespondWithError(w, err)
		}
		return
	}

	c.respWriter.RespondWithJSONAPI(w, nil, http.StatusOK)
}

func (c *moduleRegistryController) GetModuleVersions(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	moduleName := chi.URLParam(r, "name")
	moduleSystem := chi.URLParam(r, "system")

	module, err := c.moduleRegistryService.GetModuleByAddress(r.Context(), namespace, moduleName, moduleSystem)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	// Only return modules that have an uploaded package
	statusFilter := models.TerraformModuleVersionStatusUploaded
	versionsResponse, err := c.moduleRegistryService.GetModuleVersions(r.Context(), &moduleregistry.GetModuleVersionsInput{
		ModuleID: module.Metadata.ID,
		Status:   &statusFilter,
	})
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	tfeResponse := RegistryModuleVersionsResponse{
		Modules: []RegistryModuleVersionList{
			{Versions: []RegistryModuleVersion{}},
		},
	}

	for _, v := range versionsResponse.ModuleVersions {
		tfeResponse.Modules[0].Versions = append(tfeResponse.Modules[0].Versions, RegistryModuleVersion{
			Version: v.SemanticVersion,
		})
	}

	c.respWriter.RespondWithJSON(w, &tfeResponse, http.StatusOK)
}

func (c *moduleRegistryController) GetModuleVersionPackageURL(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	moduleName := chi.URLParam(r, "name")
	moduleSystem := chi.URLParam(r, "system")
	version := chi.URLParam(r, "version")

	module, err := c.moduleRegistryService.GetModuleByAddress(r.Context(), namespace, moduleName, moduleSystem)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	// Get module version by module ID and version
	statusFilter := models.TerraformModuleVersionStatusUploaded
	versionsResponse, err := c.moduleRegistryService.GetModuleVersions(r.Context(), &moduleregistry.GetModuleVersionsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		ModuleID:        module.Metadata.ID,
		SemanticVersion: &version,
		Status:          &statusFilter,
	})
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	if len(versionsResponse.ModuleVersions) == 0 {
		c.respWriter.RespondWithError(w, terrors.New("module version %s not found", version, terrors.WithErrorCode(errors.ENotFound)))
		return
	}

	moduleVersion := versionsResponse.ModuleVersions[0]

	downloadURL, err := c.moduleRegistryService.GetModuleVersionPackageDownloadURL(r.Context(), &moduleVersion)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	w.Header().Set("Access-Control-Expose-Headers", "X-Terraform-Get")
	w.Header().Set("X-Terraform-Get", downloadURL)
	w.WriteHeader(http.StatusNoContent)
}
