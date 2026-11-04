package controllers

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	runservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const testPlanMaxUploadSize = 200

func newRunControllerRouter(svc runservice.Service) chi.Router {
	testLogger, _ := logger.NewForTest()
	router := chi.NewRouter()
	NewRunController(testLogger, response.NewWriter(testLogger), passthroughMiddleware, svc, testPlanMaxUploadSize).RegisterRoutes(router)
	return router
}

func gzipString(t *testing.T, s string) []byte {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err := writer.Write([]byte(s))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buf.Bytes()
}

func TestRunController_UploadPlanData(t *testing.T) {
	// The bodies are rejected before the plan is decoded, so the run service is never called.
	planPath := "/plans/" + models.RunNodeGID("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11") + "/content.json"
	tests := []struct {
		name            string
		body            []byte
		contentEncoding string
		expectStatus    int
	}{
		{
			name:         "a body over the upload limit is too large",
			body:         []byte(`{"plan":"` + strings.Repeat("a", testPlanMaxUploadSize) + `"}`),
			expectStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:            "a gzip body that decompresses past the upload limit is too large",
			body:            gzipString(t, `{"plan":"`+strings.Repeat("a", 4*testPlanMaxUploadSize)+`"}`),
			contentEncoding: "gzip",
			expectStatus:    http.StatusRequestEntityTooLarge,
		},
		{
			name:            "a body that claims gzip but is not gzip is invalid",
			body:            []byte(`{"plan":{}}`),
			contentEncoding: "gzip",
			expectStatus:    http.StatusBadRequest,
		},
		{
			name:         "a body that is not JSON is invalid",
			body:         []byte(`not json`),
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, planPath, bytes.NewReader(test.body))
			if test.contentEncoding != "" {
				req.Header.Set("Content-Encoding", test.contentEncoding)
			}

			rec := httptest.NewRecorder()
			newRunControllerRouter(runservice.NewMockService(t)).ServeHTTP(rec, req)

			assert.Equal(t, test.expectStatus, rec.Code, rec.Body.String())
		})
	}
}

func TestRunController_UploadPlanBinaryOverLimit(t *testing.T) {
	svc := runservice.NewMockService(t)
	svc.On("UploadPlanBinary", mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ string, reader io.Reader) error {
			_, err := io.ReadAll(reader)
			return err
		})

	planPath := "/plans/" + models.RunNodeGID("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11") + "/content"
	body := bytes.Repeat([]byte("x"), testPlanMaxUploadSize+1)

	rec := httptest.NewRecorder()
	newRunControllerRouter(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodPut, planPath, bytes.NewReader(body)))

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
}
