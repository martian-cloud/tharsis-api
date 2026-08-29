package tfe

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	workspaceservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestStateController_DownloadStateVersionJSON(t *testing.T) {
	const stateVersionID = "b8e2f2b4-1c39-4c8f-9a2f-2c4d5e6f7a80"

	stateJSON := []byte(`{"format_version":"1.0","values":{"root_module":{}}}`)

	tests := []struct {
		name         string
		serviceError error
		expectCode   int
		expectBody   []byte
	}{
		{
			name:       "streams the rendering",
			expectCode: http.StatusOK,
			expectBody: stateJSON,
		},
		{
			// A state version created before renderings existed, or by an executor too old to upload
			// one, has nothing to serve. The service reports that as not found and the handler passes
			// it through rather than returning an empty 200 body.
			name:         "no rendering stored",
			serviceError: errors.New("state version JSON not found", errors.WithErrorCode(errors.ENotFound)),
			expectCode:   http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockWorkspaceService := workspaceservice.NewMockService(t)

			var reader io.ReadCloser
			if test.serviceError == nil {
				reader = io.NopCloser(bytes.NewReader(stateJSON))
			}
			mockWorkspaceService.On("GetStateVersionJSONContent", mock.Anything, stateVersionID).
				Return(reader, test.serviceError)

			testLogger, _ := logger.NewForTest()
			c := &stateController{
				respWriter:       response.NewWriter(testLogger),
				logger:           testLogger,
				workspaceService: mockWorkspaceService,
			}

			rec := httptest.NewRecorder()
			c.DownloadStateVersionJSON(rec, newStateVersionJSONRequest(
				gid.ToGlobalID(types.StateVersionModelType, stateVersionID)))

			assert.Equal(t, test.expectCode, rec.Code)
			if test.expectBody != nil {
				assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
				assert.Equal(t, string(test.expectBody), rec.Body.String())
			}
		})
	}
}

func TestTharsisStateVersionToStateVersion_downloadURLs(t *testing.T) {
	const (
		stateVersionID = "b8e2f2b4-1c39-4c8f-9a2f-2c4d5e6f7a80"
		apiURL         = "https://tharsis.example.com"
		versionedPath  = "/tfe/v2"
	)

	stateVersionGID := gid.ToGlobalID(types.StateVersionModelType, stateVersionID)
	objectStoreKey := "workspaces/ws/state_versions/" + stateVersionID + ".json"

	tests := []struct {
		name               string
		jsonObjectStoreKey *string
		expectJSONURL      string
	}{
		{
			// hosted-json-state-download-url is only meaningful once a rendering is stored, which is
			// also how HCP Terraform behaves: the attribute is empty until it has one.
			name: "no rendering stored leaves the JSON URL empty",
		},
		{
			name:               "rendering stored advertises the JSON URL",
			jsonObjectStoreKey: &objectStoreKey,
			expectJSONURL:      apiURL + versionedPath + "/state-versions/" + stateVersionGID + "/content.json",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sv := &models.StateVersion{
				Metadata:           models.ResourceMetadata{ID: stateVersionID},
				JSONObjectStoreKey: test.jsonObjectStoreKey,
			}

			result := TharsisStateVersionToStateVersion(sv, apiURL, versionedPath)

			assert.Equal(t, apiURL+versionedPath+"/state-versions/"+stateVersionGID+"/content", result.DownloadURL)
			assert.Equal(t, test.expectJSONURL, result.JSONDownloadURL)
			// The upload URL does not depend on a rendering existing: it is where one gets written.
			assert.Equal(t, apiURL+versionedPath+"/state-versions/"+stateVersionGID+"/content.json", result.JSONUploadURL)
		})
	}
}

func TestStateController_UploadStateVersionJSON(t *testing.T) {
	const stateVersionID = "b8e2f2b4-1c39-4c8f-9a2f-2c4d5e6f7a80"

	stateJSON := []byte(`{"format_version":"1.0","values":{"root_module":{}}}`)

	tests := []struct {
		name string
		// gzipBody sends the rendering gzipped, which the job executor may do because a rendering is
		// large and highly compressible.
		gzipBody     bool
		serviceError error
		expectCode   int
	}{
		{
			name:       "stores an uncompressed rendering",
			expectCode: http.StatusOK,
		},
		{
			name:       "stores a gzipped rendering",
			gzipBody:   true,
			expectCode: http.StatusOK,
		},
		{
			name:         "service failure surfaces",
			serviceError: errors.New("no permission", errors.WithErrorCode(errors.EForbidden)),
			expectCode:   http.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockWorkspaceService := workspaceservice.NewMockService(t)

			// Read what the handler passes through so the gzip case proves it decompressed rather than
			// storing the compressed bytes.
			var uploaded []byte
			mockWorkspaceService.On("UploadStateVersionJSON", mock.Anything, stateVersionID, mock.Anything).
				Run(func(args mock.Arguments) {
					read, err := io.ReadAll(args.Get(2).(io.Reader))
					require.NoError(t, err)
					uploaded = read
				}).
				Return(test.serviceError)

			body := stateJSON
			var contentEncoding string
			if test.gzipBody {
				var compressed bytes.Buffer
				writer := gzip.NewWriter(&compressed)
				_, err := writer.Write(stateJSON)
				require.NoError(t, err)
				require.NoError(t, writer.Close())
				body = compressed.Bytes()
				contentEncoding = "gzip"
			}

			testLogger, _ := logger.NewForTest()
			c := &stateController{
				respWriter:       response.NewWriter(testLogger),
				logger:           testLogger,
				workspaceService: mockWorkspaceService,
			}

			rec := httptest.NewRecorder()
			c.UploadStateVersionJSON(rec, newStateVersionJSONUploadRequest(
				gid.ToGlobalID(types.StateVersionModelType, stateVersionID), body, contentEncoding))

			assert.Equal(t, test.expectCode, rec.Code)
			assert.Equal(t, string(stateJSON), string(uploaded))
		})
	}
}

// newStateVersionJSONRequest builds a GET request whose chi route context carries the state version id,
// so the handler can be invoked directly without mounting the router (and its JWT middleware).
func newStateVersionJSONRequest(stateVersionGID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("stateVersionId", stateVersionGID)
	req := httptest.NewRequest(http.MethodGet, "/state-versions/"+stateVersionGID+"/content.json", nil)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// newStateVersionJSONUploadRequest builds the PUT counterpart, optionally marked as gzipped.
func newStateVersionJSONUploadRequest(stateVersionGID string, body []byte, contentEncoding string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("stateVersionId", stateVersionGID)
	req := httptest.NewRequest(http.MethodPut, "/state-versions/"+stateVersionGID+"/content.json", bytes.NewReader(body))
	if contentEncoding != "" {
		req.Header.Set("Content-Encoding", contentEncoding)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
