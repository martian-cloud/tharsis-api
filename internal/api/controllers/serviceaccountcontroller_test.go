package controllers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// The requests are all rejected before a token is created, so the controller gets a service whose every
// method panics: the service account service has no mockery mock.
type noServiceAccountService struct {
	serviceaccount.Service
}

func newServiceAccountControllerRouter() chi.Router {
	testLogger, _ := logger.NewForTest()
	router := chi.NewRouter()
	NewServiceAccountController(testLogger, response.NewWriter(testLogger), noServiceAccountService{}).RegisterRoutes(router)
	return router
}

func TestServiceAccountController_LoginInvalidBody(t *testing.T) {
	rec := httptest.NewRecorder()
	newServiceAccountControllerRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/serviceaccounts/login", strings.NewReader("not json")))

	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestServiceAccountController_ClientCredentialsTokenIgnoresQueryString(t *testing.T) {
	// Credentials in the query string end up in access logs, so only the form body is read (RFC 6749 section 3.2).
	query := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"client"},
		"client_secret": {"secret"},
	}
	req := httptest.NewRequest(http.MethodPost, "/serviceaccounts/token?"+query.Encode(), strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	newServiceAccountControllerRouter().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}
