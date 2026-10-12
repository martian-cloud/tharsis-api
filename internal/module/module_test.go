package module

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTokenEnvVar(t *testing.T) {
	type args struct {
		host string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Invalid port should result in an error",
			args: args{
				host: "example.com:invalid",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Empty string should result in an error",
			args: args{
				host: "",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Full URI should result in an error",
			args: args{
				host: "https//example.com:1234",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Double periods should result in an error",
			args: args{
				host: "example..com",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Simple host without unicode should convert without ASCII Compatible Encoding prefix",
			args: args{
				host: "an-example.com",
			},
			want:    "TF_TOKEN_an__example_com",
			wantErr: false,
		},
		{
			name: "Unicode host should convert",
			args: args{
				host: "例えば.com",
			},
			want:    "TF_TOKEN_xn____r8j3dr99h_com",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildTokenEnvVar(tt.args.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildTokenEnvVar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BuildTokenEnvVar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetModuleRegistryEndpointForHost(t *testing.T) {
	var body string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(body))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	t.Run("relative modules.v1 path resolves against the host", func(t *testing.T) {
		body = `{"modules.v1":"/api/modules/v1"}`
		got, err := GetModuleRegistryEndpointForHost(server.Client(), serverURL.Host)
		require.NoError(t, err)
		assert.Equal(t, "https://"+serverURL.Host+"/api/modules/v1/", got.String())
	})

	t.Run("an invalid document is not echoed in the error", func(t *testing.T) {
		// The host is user-supplied, so echoing the body would let a caller read whatever it returned.
		body = "<html>internal-secret</html>"
		_, err := GetModuleRegistryEndpointForHost(server.Client(), serverURL.Host)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "internal-secret")
	})
}

func TestReadAllLimited(t *testing.T) {
	t.Run("a body at the limit is read in full", func(t *testing.T) {
		body, err := ReadAllLimited(strings.NewReader("12345"), 5)
		require.NoError(t, err)
		assert.Equal(t, "12345", string(body))
	})

	t.Run("a body over the limit is an error, not a truncated read", func(t *testing.T) {
		_, err := ReadAllLimited(strings.NewReader("123456"), 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds the maximum size of 5 bytes")
	})
}
