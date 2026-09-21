package plunk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestPlunkSendMail(t *testing.T) {
	testLogger, _ := logger.NewForTest()

	t.Run("posts the payload and succeeds on 200", func(t *testing.T) {
		var (
			gotPath string
			gotAuth string
			gotType string
			gotBody sendEmailPayload
		)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotAuth = r.Header.Get("Authorization")
			gotType = r.Header.Get("Content-Type")
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		p := NewProvider(testLogger, server.URL, "secret-key")
		err := p.SendMail(t.Context(), "user@x.com", "subject", "body", "corr-1")
		require.NoError(t, err)

		assert.Equal(t, "/v1/send", gotPath)
		assert.Equal(t, "Bearer secret-key", gotAuth)
		assert.Equal(t, "application/json", gotType)
		assert.Equal(t, []string{"user@x.com"}, gotBody.To)
		assert.Equal(t, "subject", gotBody.Subject)
		assert.Equal(t, "body", gotBody.Body)
	})

	t.Run("non-200 status is an error including the response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("invalid recipient"))
		}))
		defer server.Close()

		p := NewProvider(testLogger, server.URL, "secret-key")
		err := p.SendMail(t.Context(), "user@x.com", "subject", "body", "corr-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid recipient")
	})

	t.Run("transport error is returned", func(t *testing.T) {
		// Point at a closed server so the request fails at the transport layer.
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
		endpoint := server.URL
		server.Close()

		p := NewProvider(testLogger, endpoint, "secret-key")
		err := p.SendMail(t.Context(), "user@x.com", "subject", "body", "corr-1")
		assert.Error(t, err)
	})
}
