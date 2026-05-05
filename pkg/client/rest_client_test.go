package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/provider"
)

type mockTokenResolver struct {
	token string
	err   error
}

func (m *mockTokenResolver) Token(_ context.Context) (string, error) {
	return m.token, m.err
}

func (m *mockTokenResolver) Close() error {
	return nil
}

func TestNewRESTClient(t *testing.T) {
	client, err := NewRESTClient(&RESTClientConfig{Endpoint: "https://example.com", TokenResolver: &mockTokenResolver{token: "test"}, UserAgent: ptr.String("test")})
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestUploadConfigurationVersion(t *testing.T) {
	type testCase struct {
		name          string
		setupMock     func(*testing.T, *provider.MockServiceDiscoverer, *httptest.Server)
		setupDir      func(*testing.T) string
		setupServer   func(*testing.T) *httptest.Server
		expectError   bool
		errorContains string
	}

	testCases := []testCase{
		{
			name: "successful upload",
			setupMock: func(_ *testing.T, mockDiscoverer *provider.MockServiceDiscoverer, server *httptest.Server) {
				baseURL, _ := url.Parse(server.URL)
				mockDiscoverer.On("DiscoverTFEServices", mock.Anything, baseURL.String()).Return(&provider.TFEServices{
					Services: map[provider.ServiceID]*url.URL{
						provider.TFEServiceID: baseURL.JoinPath("/v2/tfe"),
					},
				}, nil)
			},
			setupDir: func(t *testing.T) string {
				configDir := filepath.Join(t.TempDir(), "config")
				require.NoError(t, os.Mkdir(configDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(configDir, "main.tf"), []byte("resource \"test\" {}"), 0600))
				return configDir
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPut, r.Method)
					assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
					body, _ := io.ReadAll(r.Body)
					assert.Equal(t, r.ContentLength, int64(len(body)))
					w.WriteHeader(http.StatusOK)
				}))
			},
		},
		{
			name: "service discovery fails",
			setupMock: func(_ *testing.T, mockDiscoverer *provider.MockServiceDiscoverer, server *httptest.Server) {
				baseURL, _ := url.Parse(server.URL)
				mockDiscoverer.On("DiscoverTFEServices", mock.Anything, baseURL.String()).Return(
					(*provider.TFEServices)(nil),
					assert.AnError,
				)
			},
			setupDir: func(t *testing.T) string {
				configDir := filepath.Join(t.TempDir(), "config")
				require.NoError(t, os.Mkdir(configDir, 0755))
				return configDir
			},
			setupServer: func(_ *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError:   true,
			errorContains: "failed to discover tfe v2 service",
		},
		{
			name: "directory does not exist",
			setupMock: func(_ *testing.T, mockDiscoverer *provider.MockServiceDiscoverer, server *httptest.Server) {
				baseURL, _ := url.Parse(server.URL)
				mockDiscoverer.On("DiscoverTFEServices", mock.Anything, baseURL.String()).Return(&provider.TFEServices{
					Services: map[provider.ServiceID]*url.URL{
						provider.TFEServiceID: baseURL.JoinPath("/v2/tfe"),
					},
				}, nil)
			},
			setupDir: func(_ *testing.T) string {
				return "/nonexistent/path"
			},
			setupServer: func(_ *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError:   true,
			errorContains: "failed to create slug",
		},
		{
			name: "upload fails with non-200 status",
			setupMock: func(_ *testing.T, mockDiscoverer *provider.MockServiceDiscoverer, server *httptest.Server) {
				baseURL, _ := url.Parse(server.URL)
				mockDiscoverer.On("DiscoverTFEServices", mock.Anything, baseURL.String()).Return(&provider.TFEServices{
					Services: map[provider.ServiceID]*url.URL{
						provider.TFEServiceID: baseURL.JoinPath("/v2/tfe"),
					},
				}, nil)
			},
			setupDir: func(t *testing.T) string {
				configDir := filepath.Join(t.TempDir(), "config")
				require.NoError(t, os.Mkdir(configDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(configDir, "main.tf"), []byte("resource \"test\" {}"), 0600))
				return configDir
			},
			setupServer: func(_ *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			expectError:   true,
			errorContains: "upload failed with status code",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := tc.setupServer(t)
			defer server.Close()

			baseURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			mockDiscoverer := provider.NewMockServiceDiscoverer(t)
			tc.setupMock(t, mockDiscoverer, server)

			client := &restClient{
				baseURL:           baseURL,
				tokenResolver:     &mockTokenResolver{token: "test-token"},
				httpClient:        http.DefaultClient,
				serviceDiscoverer: mockDiscoverer,
			}

			err = client.UploadConfigurationVersion(t.Context(), &UploadConfigurationVersionInput{
				WorkspaceID:     "ws-123",
				ConfigVersionID: "cv-456",
				DirectoryPath:   tc.setupDir(t),
			})

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestDownloadConfigurationVersion(t *testing.T) {
	type testCase struct {
		name         string
		setupMock    func(*testing.T, *provider.MockServiceDiscoverer, *httptest.Server)
		setupServer  func() *httptest.Server
		expectError  bool
		expectOutput string
	}

	testCases := []testCase{
		{
			name: "successful download",
			setupMock: func(_ *testing.T, mockDiscoverer *provider.MockServiceDiscoverer, server *httptest.Server) {
				baseURL, _ := url.Parse(server.URL)
				mockDiscoverer.On("DiscoverTFEServices", mock.Anything, baseURL.String()).Return(&provider.TFEServices{
					Services: map[provider.ServiceID]*url.URL{
						provider.TFEServiceID: baseURL.JoinPath("/v2/tfe"),
					},
				}, nil)
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, "/v2/tfe/configuration-versions/cv-123/content", r.URL.Path)
					assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte("test content"))
					require.NoError(t, err)
				}))
			},
			expectOutput: "test content",
		},
		{
			name: "download fails",
			setupMock: func(_ *testing.T, mockDiscoverer *provider.MockServiceDiscoverer, server *httptest.Server) {
				baseURL, _ := url.Parse(server.URL)
				mockDiscoverer.On("DiscoverTFEServices", mock.Anything, baseURL.String()).Return(&provider.TFEServices{
					Services: map[provider.ServiceID]*url.URL{
						provider.TFEServiceID: baseURL.JoinPath("/v2/tfe"),
					},
				}, nil)
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := tc.setupServer()
			defer server.Close()

			baseURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			mockDiscoverer := provider.NewMockServiceDiscoverer(t)
			tc.setupMock(t, mockDiscoverer, server)

			client := &restClient{
				baseURL:           baseURL,
				tokenResolver:     &mockTokenResolver{token: "test-token"},
				httpClient:        http.DefaultClient,
				serviceDiscoverer: mockDiscoverer,
			}

			var buf bytes.Buffer
			err = client.DownloadConfigurationVersion(t.Context(), &DownloadConfigurationVersionInput{
				ConfigVersionID: "cv-123",
				Writer:          &buf,
			})

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectOutput, buf.String())
		})
	}
}

func TestUploadModuleVersion(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "module.tar.gz")
	require.NoError(t, os.WriteFile(testFile, []byte("module content"), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/module-registry/versions/mv-123/upload", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, "module content", string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewRESTClient(&RESTClientConfig{Endpoint: server.URL, TokenResolver: &mockTokenResolver{token: "test-token"}, UserAgent: ptr.String("test")})
	require.NoError(t, err)

	err = client.UploadModuleVersion(t.Context(), &UploadModuleVersionInput{
		ModuleVersionID: "mv-123",
		PackagePath:     testFile,
	})
	require.NoError(t, err)
}

func TestUploadProviderPlatformPackageToMirror(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/provider-mirror/providers/vm-123/linux/amd64/upload", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, "package data", string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewRESTClient(&RESTClientConfig{Endpoint: server.URL, TokenResolver: &mockTokenResolver{token: "test-token"}, UserAgent: ptr.String("test")})
	require.NoError(t, err)

	err = client.UploadProviderPlatformPackageToMirror(t.Context(), &UploadProviderPlatformPackageToMirrorInput{
		VersionMirrorID: "vm-123",
		OS:              "linux",
		Arch:            "amd64",
		Reader:          bytes.NewReader([]byte("package data")),
	})
	require.NoError(t, err)
}

func TestUploadProviderReadme(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "readme.md")
	require.NoError(t, os.WriteFile(testFile, []byte("readme content"), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/provider-registry/versions/pv-123/readme/upload", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewRESTClient(&RESTClientConfig{Endpoint: server.URL, TokenResolver: &mockTokenResolver{token: "test-token"}, UserAgent: ptr.String("test")})
	require.NoError(t, err)

	err = client.UploadProviderReadme(t.Context(), &UploadProviderReadmeInput{
		ProviderVersionID: "pv-123",
		ReadmePath:        testFile,
	})
	require.NoError(t, err)
}

func TestUploadProviderChecksums(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "checksums.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("checksums"), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/provider-registry/versions/pv-123/checksums/upload", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewRESTClient(&RESTClientConfig{Endpoint: server.URL, TokenResolver: &mockTokenResolver{token: "test-token"}, UserAgent: ptr.String("test")})
	require.NoError(t, err)

	err = client.UploadProviderChecksums(t.Context(), &UploadProviderChecksumsInput{
		ProviderVersionID: "pv-123",
		ChecksumsPath:     testFile,
	})
	require.NoError(t, err)
}

func TestUploadProviderChecksumSignature(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "signature.sig")
	require.NoError(t, os.WriteFile(testFile, []byte("signature"), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/provider-registry/versions/pv-123/signature/upload", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewRESTClient(&RESTClientConfig{Endpoint: server.URL, TokenResolver: &mockTokenResolver{token: "test-token"}, UserAgent: ptr.String("test")})
	require.NoError(t, err)

	err = client.UploadProviderChecksumSignature(t.Context(), &UploadProviderChecksumSignatureInput{
		ProviderVersionID: "pv-123",
		SignaturePath:     testFile,
	})
	require.NoError(t, err)
}

func TestUploadProviderPlatformBinary(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "provider.zip")
	require.NoError(t, os.WriteFile(testFile, []byte("binary"), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/provider-registry/platforms/pp-123/upload", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewRESTClient(&RESTClientConfig{Endpoint: server.URL, TokenResolver: &mockTokenResolver{token: "test-token"}, UserAgent: ptr.String("test")})
	require.NoError(t, err)

	err = client.UploadProviderPlatformBinary(t.Context(), &UploadProviderPlatformBinaryInput{
		PlatformID: "pp-123",
		BinaryPath: testFile,
	})
	require.NoError(t, err)
}

func TestUploadPlanCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/plans/plan-123/content", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, "plan cache data", string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewRESTClient(&RESTClientConfig{Endpoint: server.URL, TokenResolver: &mockTokenResolver{token: "test-token"}, UserAgent: ptr.String("test")})
	require.NoError(t, err)

	err = client.UploadPlanCache(t.Context(), &UploadPlanCacheInput{
		PlanID: "plan-123",
		Reader: bytes.NewReader([]byte("plan cache data")),
	})
	require.NoError(t, err)
}

func TestUploadPlanData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/plans/plan-123/content.json", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewRESTClient(&RESTClientConfig{Endpoint: server.URL, TokenResolver: &mockTokenResolver{token: "test-token"}, UserAgent: ptr.String("test")})
	require.NoError(t, err)

	err = client.UploadPlanData(t.Context(), &UploadPlanDataInput{
		PlanID: "plan-123",
		Reader: bytes.NewReader([]byte(`{"plan": "data"}`)),
	})
	require.NoError(t, err)
}

func TestDownloadStateVersion(t *testing.T) {
	type testCase struct {
		name         string
		setupMock    func(*provider.MockServiceDiscoverer, *httptest.Server)
		setupServer  func() *httptest.Server
		expectError  bool
		expectOutput string
	}

	testCases := []testCase{
		{
			name: "successful download",
			setupMock: func(mockDiscoverer *provider.MockServiceDiscoverer, server *httptest.Server) {
				baseURL, _ := url.Parse(server.URL)
				mockDiscoverer.On("DiscoverTFEServices", mock.Anything, baseURL.String()).Return(&provider.TFEServices{
					Services: map[provider.ServiceID]*url.URL{
						provider.TFEServiceID: baseURL.JoinPath("/v2/tfe"),
					},
				}, nil)
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, "/v2/tfe/state-versions/sv-123/content", r.URL.Path)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("state content"))
				}))
			},
			expectOutput: "state content",
		},
		{
			name: "download fails",
			setupMock: func(mockDiscoverer *provider.MockServiceDiscoverer, server *httptest.Server) {
				baseURL, _ := url.Parse(server.URL)
				mockDiscoverer.On("DiscoverTFEServices", mock.Anything, baseURL.String()).Return(&provider.TFEServices{
					Services: map[provider.ServiceID]*url.URL{
						provider.TFEServiceID: baseURL.JoinPath("/v2/tfe"),
					},
				}, nil)
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := tc.setupServer()
			defer server.Close()

			baseURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			mockDiscoverer := provider.NewMockServiceDiscoverer(t)
			tc.setupMock(mockDiscoverer, server)

			client := &restClient{
				baseURL:           baseURL,
				tokenResolver:     &mockTokenResolver{token: "test-token"},
				httpClient:        http.DefaultClient,
				serviceDiscoverer: mockDiscoverer,
			}

			var buf bytes.Buffer
			err = client.DownloadStateVersion(t.Context(), &DownloadStateVersionInput{
				StateVersionID: "sv-123",
				Writer:         &buf,
			})

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectOutput, buf.String())
		})
	}
}

func TestDownloadPlanCache(t *testing.T) {
	type testCase struct {
		name         string
		setupMock    func(*provider.MockServiceDiscoverer, *httptest.Server)
		setupServer  func() *httptest.Server
		expectError  bool
		expectOutput string
	}

	testCases := []testCase{
		{
			name: "successful download",
			setupMock: func(mockDiscoverer *provider.MockServiceDiscoverer, server *httptest.Server) {
				baseURL, _ := url.Parse(server.URL)
				mockDiscoverer.On("DiscoverTFEServices", mock.Anything, baseURL.String()).Return(&provider.TFEServices{
					Services: map[provider.ServiceID]*url.URL{
						provider.TFEServiceID: baseURL.JoinPath("/v2/tfe"),
					},
				}, nil)
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, "/v2/tfe/plans/plan-123/content", r.URL.Path)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("plan cache content"))
				}))
			},
			expectOutput: "plan cache content",
		},
		{
			name: "download fails",
			setupMock: func(mockDiscoverer *provider.MockServiceDiscoverer, server *httptest.Server) {
				baseURL, _ := url.Parse(server.URL)
				mockDiscoverer.On("DiscoverTFEServices", mock.Anything, baseURL.String()).Return(&provider.TFEServices{
					Services: map[provider.ServiceID]*url.URL{
						provider.TFEServiceID: baseURL.JoinPath("/v2/tfe"),
					},
				}, nil)
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := tc.setupServer()
			defer server.Close()

			baseURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			mockDiscoverer := provider.NewMockServiceDiscoverer(t)
			tc.setupMock(mockDiscoverer, server)

			client := &restClient{
				baseURL:           baseURL,
				tokenResolver:     &mockTokenResolver{token: "test-token"},
				httpClient:        http.DefaultClient,
				serviceDiscoverer: mockDiscoverer,
			}

			var buf bytes.Buffer
			err = client.DownloadPlanCache(t.Context(), &DownloadPlanCacheInput{
				PlanID: "plan-123",
				Writer: &buf,
			})

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectOutput, buf.String())
		})
	}
}
