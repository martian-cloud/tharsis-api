package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// unsignedProviderVersionStub serves a provider version with no signing key. The provider registry service
// has no mockery mock, so it embeds the interface: a call to any other method panics.
type unsignedProviderVersionStub struct {
	providerregistry.Service
}

func (unsignedProviderVersionStub) GetProviderByAddress(context.Context, string, string) (*models.TerraformProvider, error) {
	return &models.TerraformProvider{Metadata: models.ResourceMetadata{ID: "provider-1"}}, nil
}

func (unsignedProviderVersionStub) GetProviderVersions(context.Context, *providerregistry.GetProviderVersionsInput) (*db.ProviderVersionsResult, error) {
	return &db.ProviderVersionsResult{ProviderVersions: []models.TerraformProviderVersion{
		{Metadata: models.ResourceMetadata{ID: "version-1"}, SemanticVersion: "1.0.0"},
	}}, nil
}

func (unsignedProviderVersionStub) GetProviderPlatforms(context.Context, *providerregistry.GetProviderPlatformsInput) (*db.ProviderPlatformsResult, error) {
	return &db.ProviderPlatformsResult{ProviderPlatforms: []models.TerraformProviderPlatform{
		{Metadata: models.ResourceMetadata{ID: "platform-1"}, OperatingSystem: "linux", Architecture: "amd64"},
	}}, nil
}

func TestProviderRegistryController_GetVersionWithoutSigningKey(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	router := chi.NewRouter()
	NewProviderRegistryController(testLogger, response.NewWriter(testLogger), passthroughMiddleware, unsignedProviderVersionStub{}).RegisterRoutes(router)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/provider-registry/providers/ns/aws/1.0.0/download/linux/amd64", nil))

	// A version without a signing key used to panic building the download response.
	assert.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
}

// passthroughMiddleware stands in for the JWT middleware so a controller's routes can be mounted.
func passthroughMiddleware(next http.Handler) http.Handler { return next }
