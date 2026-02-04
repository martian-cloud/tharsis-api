package jobexecutor

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/provider"
	sdkTypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

func TestMirrorProxy_url(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	proxy := &mirrorProxy{listener: listener}

	assert.Equal(t, "https://"+listener.Addr().String(), proxy.url())
}

func TestMirrorProxy_caCert(t *testing.T) {
	caCert := []byte("test-ca-cert")
	proxy := &mirrorProxy{caCertPEM: caCert}

	assert.Equal(t, caCert, proxy.caCert())
}

func TestNewProviderMirrorProxy(t *testing.T) {
	mockClient := jobclient.NewMockClient(t)
	mockLogger := joblogger.NewMockLogger(t)
	mockLogger.On("Errorf", mock.Anything, mock.Anything).Maybe()

	proxy, err := newProviderMirrorProxy(
		t.Context(),
		mockClient,
		"test-workspace-id",
		mockLogger,
		map[string]string{"example.com": "token"},
		10*time.Minute,
	)
	require.NoError(t, err)

	mp := proxy.(*mirrorProxy)
	assert.Equal(t, mockClient, mp.client)
	assert.Equal(t, "test-workspace-id", mp.workspaceID)
	assert.Equal(t, mockLogger, mp.log.logger)
	assert.Equal(t, "token", mp.registryTokens["example.com"])
	assert.NotNil(t, mp.server)
	assert.NotNil(t, mp.listener)
	assert.NotNil(t, mp.registryClient)
	assert.NotEmpty(t, mp.caCertPEM)
	assert.Contains(t, proxy.url(), "https://127.0.0.1:")

	proxy.start()

	// Server is running - HTTP request should work (even if route not found)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := client.Get(proxy.url() + "/test")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	proxy.shutdown()

	// Server is stopped - HTTP request should fail with connection refused
	_, err = client.Get(proxy.url() + "/test")
	assert.ErrorContains(t, err, "connection refused")
}

func TestMirrorProxy_shutdown(t *testing.T) {
	t.Run("stops server", func(t *testing.T) {
		mockLogger := joblogger.NewMockLogger(t)

		ts := httptest.NewUnstartedServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
		ts.Start()

		proxy := &mirrorProxy{
			server: ts.Config,
			log:    &batchedLog{logger: mockLogger},
		}

		_, err := http.Get(ts.URL)
		require.NoError(t, err)

		proxy.shutdown()

		_, err = http.Get(ts.URL)
		assert.Error(t, err)
	})

	t.Run("waits for in-flight caching", func(t *testing.T) {
		mockLogger := joblogger.NewMockLogger(t)
		ts := httptest.NewUnstartedServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
		ts.Start()

		proxy := &mirrorProxy{
			server: ts.Config,
			log:    &batchedLog{logger: mockLogger},
		}

		done := make(chan struct{})
		proxy.cacheWg.Add(1)
		go func() {
			defer proxy.cacheWg.Done()
			<-done // Block until test signals
		}()

		shutdownComplete := make(chan struct{})
		go func() {
			proxy.shutdown()
			close(shutdownComplete)
		}()

		// Shutdown should be blocked
		select {
		case <-shutdownComplete:
			t.Fatal("shutdown completed before caching finished")
		case <-time.After(50 * time.Millisecond):
			// Expected - shutdown is waiting
		}

		close(done) // Allow goroutine to finish

		select {
		case <-shutdownComplete:
			// Expected - shutdown completed
		case <-time.After(time.Second):
			t.Fatal("shutdown did not complete after caching finished")
		}
	})
}

func TestGenerateTLSConfig(t *testing.T) {
	tlsConfig, caCert, err := generateTLSConfig(10 * time.Minute)

	require.NoError(t, err)
	assert.NotNil(t, tlsConfig)
	assert.NotEmpty(t, caCert)
	assert.Len(t, tlsConfig.Certificates, 1)
}

func TestMirrorProxy_handleVersions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockResolver := provider.NewMockRegistryProtocol(t)
		mockLogger := joblogger.NewMockLogger(t)
		prov := &provider.Provider{Hostname: "registry.terraform.io", Namespace: "hashicorp", Type: "null"}
		mockResolver.On("ListVersions", mock.Anything, prov).
			Return([]provider.VersionInfo{{Version: "3.2.0"}, {Version: "3.1.0"}}, nil)
		mockLogger.On("Infof", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

		proxy := &mirrorProxy{registryClient: mockResolver, log: &batchedLog{logger: mockLogger}}
		w := httptest.NewRecorder()
		proxy.handleVersions(w, buildRequest("registry.terraform.io", "hashicorp", "null", ""))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"versions":{"3.1.0":{},"3.2.0":{}}}`, w.Body.String())
	})

	t.Run("uses token for matching hostname", func(t *testing.T) {
		mockResolver := provider.NewMockRegistryProtocol(t)
		mockLogger := joblogger.NewMockLogger(t)
		prov := &provider.Provider{Hostname: "other-tharsis.example.com", Namespace: "myorg", Type: "custom"}
		mockResolver.On("ListVersions", mock.Anything, prov, mock.AnythingOfType("provider.RequestOption")).
			Return([]provider.VersionInfo{{Version: "1.0.0"}}, nil)
		mockLogger.On("Infof", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

		proxy := &mirrorProxy{
			registryClient: mockResolver,
			log:            &batchedLog{logger: mockLogger},
			registryTokens: map[string]string{"other-tharsis.example.com": "secret-token"},
		}
		w := httptest.NewRecorder()
		proxy.handleVersions(w, buildRequest("other-tharsis.example.com", "myorg", "custom", ""))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"versions":{"1.0.0":{}}}`, w.Body.String())
	})

	t.Run("invalid provider address", func(t *testing.T) {
		proxy := &mirrorProxy{}
		w := httptest.NewRecorder()
		proxy.handleVersions(w, buildRequest("", "hashicorp", "null", ""))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("falls back to cached versions when upstream fails", func(t *testing.T) {
		mockResolver := provider.NewMockRegistryProtocol(t)
		mockClient := jobclient.NewMockClient(t)
		mockLogger := joblogger.NewMockLogger(t)

		prov := &provider.Provider{Hostname: "registry.terraform.io", Namespace: "hashicorp", Type: "null"}
		mockResolver.On("ListVersions", mock.Anything, prov).Return(nil, assert.AnError)
		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("GetAvailableProviderVersions", mock.Anything, &sdkTypes.GetAvailableProviderVersionsInput{
			GroupPath:         "test-group",
			RegistryHostname:  "registry.terraform.io",
			RegistryNamespace: "hashicorp",
			Type:              "null",
		}).Return(map[string]struct{}{"3.0.0": {}, "2.0.0": {}}, nil)

		proxy := &mirrorProxy{
			registryClient: mockResolver,
			client:         mockClient,
			log:            &batchedLog{logger: mockLogger},
			workspaceID:    "test-workspace-id",
		}
		w := httptest.NewRecorder()
		proxy.handleVersions(w, buildRequest("registry.terraform.io", "hashicorp", "null", ""))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"versions":{"2.0.0":{},"3.0.0":{}}}`, w.Body.String())
	})
}

func TestMirrorProxy_handlePackages(t *testing.T) {
	version := "3.2.0"
	testOS := runtime.GOOS
	testArch := runtime.GOARCH

	t.Run("returns cached platform", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)
		mockLogger := joblogger.NewMockLogger(t)

		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("GetProviderPlatformPackageDownloadURL", mock.Anything, mock.Anything).
			Return(&sdkTypes.ProviderPlatformPackageInfo{URL: "https://mirror/pkg.zip", Hashes: []string{"zh:abc123"}}, nil)

		proxy := &mirrorProxy{client: mockClient, log: &batchedLog{logger: mockLogger}, workspaceID: "test-workspace-id"}
		w := httptest.NewRecorder()
		proxy.handlePackages(w, buildRequest("registry.terraform.io", "hashicorp", "null", version+".json"))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "https://mirror/pkg.zip")
	})

	t.Run("falls back to upstream on forbidden error", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)
		mockResolver := provider.NewMockRegistryProtocol(t)
		mockLogger := joblogger.NewMockLogger(t)

		// Forbidden - workspace migrated to different group
		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("GetProviderPlatformPackageDownloadURL", mock.Anything, mock.Anything).
			Return(nil, &sdkTypes.Error{Code: sdkTypes.ErrForbidden})

		prov := &provider.Provider{Hostname: "registry.terraform.io", Namespace: "hashicorp", Type: "null"}
		mockResolver.On("GetPackageInfo", mock.Anything, prov, version, testOS, testArch).
			Return(&provider.PackageInfo{
				DownloadURL: "https://releases.hashicorp.com/pkg.zip",
				SHASumsURL:  "https://releases.hashicorp.com/shasums",
			}, nil)

		filename := provider.GetPackageName("null", version, testOS, testArch)
		mockResolver.On("GetChecksums", mock.Anything, mock.Anything).
			Return(provider.Checksums{filename: []byte("checksum123456789012345678901234")}, nil)

		mockClient.On("CreateProviderVersionMirror", mock.Anything, mock.Anything).
			Return(&sdkTypes.TerraformProviderVersionMirror{Metadata: sdkTypes.ResourceMetadata{ID: "mirror-id"}}, nil).Maybe()
		mockResolver.On("DownloadPackage", mock.Anything, mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte("data"))), int64(4), nil).Maybe()
		mockClient.On("UploadProviderPlatformPackageToMirror", mock.Anything, mock.Anything).Return(nil).Maybe()
		mockLogger.On("Errorf", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

		proxy := &mirrorProxy{
			client:         mockClient,
			registryClient: mockResolver,
			log:            &batchedLog{logger: mockLogger},
			workspaceID:    "test-workspace-id",
			cancellableCtx: t.Context(),
		}
		w := httptest.NewRecorder()
		proxy.handlePackages(w, buildRequest("registry.terraform.io", "hashicorp", "null", version+".json"))

		proxy.cacheWg.Wait()

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "https://releases.hashicorp.com/pkg.zip")
	})

	t.Run("returns upstream URL when not cached", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)
		mockResolver := provider.NewMockRegistryProtocol(t)
		mockLogger := joblogger.NewMockLogger(t)

		// Not cached - returns not found
		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("GetProviderPlatformPackageDownloadURL", mock.Anything, mock.Anything).
			Return(nil, &sdkTypes.Error{Code: sdkTypes.ErrNotFound})

		prov := &provider.Provider{Hostname: "registry.terraform.io", Namespace: "hashicorp", Type: "null"}
		mockResolver.On("GetPackageInfo", mock.Anything, prov, version, testOS, testArch).
			Return(&provider.PackageInfo{
				DownloadURL: "https://releases.hashicorp.com/pkg.zip",
				SHASumsURL:  "https://releases.hashicorp.com/shasums",
			}, nil)

		filename := provider.GetPackageName("null", version, testOS, testArch)
		mockResolver.On("GetChecksums", mock.Anything, mock.Anything).
			Return(provider.Checksums{filename: []byte("checksum123456789012345678901234")}, nil)

		// Async caching mocks
		mockClient.On("CreateProviderVersionMirror", mock.Anything, mock.Anything).
			Return(&sdkTypes.TerraformProviderVersionMirror{Metadata: sdkTypes.ResourceMetadata{ID: "mirror-id"}}, nil).Maybe()
		mockResolver.On("DownloadPackage", mock.Anything, mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte("data"))), int64(4), nil).Maybe()
		mockClient.On("UploadProviderPlatformPackageToMirror", mock.Anything, mock.Anything).Return(nil).Maybe()
		mockLogger.On("Infof", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
		mockLogger.On("Errorf", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

		proxy := &mirrorProxy{
			client:         mockClient,
			registryClient: mockResolver,
			log:            &batchedLog{logger: mockLogger},
			workspaceID:    "test-workspace-id",
			cancellableCtx: t.Context(),
		}
		w := httptest.NewRecorder()
		proxy.handlePackages(w, buildRequest("registry.terraform.io", "hashicorp", "null", version+".json"))

		// Wait for async goroutine
		proxy.cacheWg.Wait()

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "https://releases.hashicorp.com/pkg.zip")
	})

	t.Run("cancelled context stops async caching", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)
		mockResolver := provider.NewMockRegistryProtocol(t)
		mockLogger := joblogger.NewMockLogger(t)

		// Not cached
		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("GetProviderPlatformPackageDownloadURL", mock.Anything, mock.Anything).
			Return(nil, &sdkTypes.Error{Code: sdkTypes.ErrNotFound})

		prov := &provider.Provider{Hostname: "registry.terraform.io", Namespace: "hashicorp", Type: "null"}
		mockResolver.On("GetPackageInfo", mock.Anything, prov, version, testOS, testArch).
			Return(&provider.PackageInfo{
				DownloadURL: "https://releases.hashicorp.com/pkg.zip",
				SHASumsURL:  "https://releases.hashicorp.com/shasums",
			}, nil)

		filename := provider.GetPackageName("null", version, testOS, testArch)
		mockResolver.On("GetChecksums", mock.Anything, mock.Anything).
			Return(provider.Checksums{filename: []byte("checksum123456789012345678901234")}, nil)

		// Caching should be skipped due to cancelled context
		mockClient.On("CreateProviderVersionMirror", mock.Anything, mock.Anything).
			Return(nil, context.Canceled).Maybe()
		mockLogger.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel before request

		proxy := &mirrorProxy{
			client:         mockClient,
			registryClient: mockResolver,
			log:            &batchedLog{logger: mockLogger},
			workspaceID:    "test-workspace-id",
			cancellableCtx: ctx,
		}
		w := httptest.NewRecorder()
		proxy.handlePackages(w, buildRequest("registry.terraform.io", "hashicorp", "null", version+".json"))

		proxy.cacheWg.Wait()

		// Response should still succeed (upstream URL returned)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not cached falls back to upstream and caches async", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)
		mockResolver := provider.NewMockRegistryProtocol(t)
		mockLogger := joblogger.NewMockLogger(t)

		prov := &provider.Provider{Hostname: "registry.terraform.io", Namespace: "hashicorp", Type: "null"}
		filename := provider.GetPackageName("null", version, testOS, testArch)
		downloadURL := "https://releases.hashicorp.com/pkg.zip"
		shasumURL := "https://releases.hashicorp.com/shasums"
		packageInfo := &provider.PackageInfo{DownloadURL: downloadURL, SHASumsURL: shasumURL}
		groupPath := "test-group"

		// Not cached
		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("GetProviderPlatformPackageDownloadURL", mock.Anything, mock.Anything).
			Return(nil, &sdkTypes.Error{Code: sdkTypes.ErrNotFound})

		mockResolver.On("GetPackageInfo", mock.Anything, prov, version, testOS, testArch).Return(packageInfo, nil)
		mockResolver.On("GetChecksums", mock.Anything, packageInfo).
			Return(provider.Checksums{filename: []byte("checksum123456789012345678901234")}, nil)

		// Async caching
		mockClient.On("CreateProviderVersionMirror", mock.Anything, &sdkTypes.CreateTerraformProviderVersionMirrorInput{
			GroupPath:         groupPath,
			Type:              "null",
			RegistryHostname:  "registry.terraform.io",
			RegistryNamespace: "hashicorp",
			SemanticVersion:   version,
		}).Return(&sdkTypes.TerraformProviderVersionMirror{Metadata: sdkTypes.ResourceMetadata{ID: "mirror-id"}}, nil).Maybe()
		mockResolver.On("DownloadPackage", mock.Anything, downloadURL).
			Return(io.NopCloser(bytes.NewReader([]byte("data"))), int64(4), nil).Maybe()
		mockClient.On("UploadProviderPlatformPackageToMirror", mock.Anything, mock.Anything).Return(nil).Maybe()
		mockLogger.On("Infof", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

		proxy := &mirrorProxy{
			client:         mockClient,
			registryClient: mockResolver,
			log:            &batchedLog{logger: mockLogger},
			workspaceID:    "test-workspace-id",
			cancellableCtx: t.Context(),
		}
		w := httptest.NewRecorder()
		proxy.handlePackages(w, buildRequest("registry.terraform.io", "hashicorp", "null", version+".json"))

		proxy.cacheWg.Wait()

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), downloadURL)
	})

	t.Run("returns hash in response", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)
		mockResolver := provider.NewMockRegistryProtocol(t)
		mockLogger := joblogger.NewMockLogger(t)

		prov := &provider.Provider{Hostname: "registry.terraform.io", Namespace: "hashicorp", Type: "null"}
		filename := provider.GetPackageName("null", version, testOS, testArch)
		downloadURL := "https://releases.hashicorp.com/pkg.zip"
		shasumURL := "https://releases.hashicorp.com/shasums"
		packageInfo := &provider.PackageInfo{DownloadURL: downloadURL, SHASumsURL: shasumURL}
		groupPath := "test-group"
		checksum := make([]byte, 32)
		for i := range checksum {
			checksum[i] = byte(i)
		}

		// Not cached
		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("GetProviderPlatformPackageDownloadURL", mock.Anything, mock.Anything).
			Return(nil, &sdkTypes.Error{Code: sdkTypes.ErrNotFound})

		mockResolver.On("GetPackageInfo", mock.Anything, prov, version, testOS, testArch).Return(packageInfo, nil)
		mockResolver.On("GetChecksums", mock.Anything, packageInfo).
			Return(provider.Checksums{filename: checksum}, nil)

		// Async caching mocks
		mockClient.On("CreateProviderVersionMirror", mock.Anything, &sdkTypes.CreateTerraformProviderVersionMirrorInput{
			GroupPath:         groupPath,
			Type:              "null",
			RegistryHostname:  "registry.terraform.io",
			RegistryNamespace: "hashicorp",
			SemanticVersion:   version,
		}).Return(&sdkTypes.TerraformProviderVersionMirror{Metadata: sdkTypes.ResourceMetadata{ID: "mirror-id"}}, nil).Maybe()
		mockResolver.On("DownloadPackage", mock.Anything, downloadURL).
			Return(io.NopCloser(bytes.NewReader([]byte("data"))), int64(4), nil).Maybe()
		// UploadProviderPlatformPackageToMirror has Reader field that can't be matched exactly
		mockClient.On("UploadProviderPlatformPackageToMirror", mock.Anything, mock.Anything).Return(nil).Maybe()
		mockLogger.On("Infof", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

		proxy := &mirrorProxy{
			client:         mockClient,
			registryClient: mockResolver,
			log:            &batchedLog{logger: mockLogger},
			workspaceID:    "test-workspace-id",
			cancellableCtx: t.Context(),
		}
		w := httptest.NewRecorder()
		proxy.handlePackages(w, buildRequest("registry.terraform.io", "hashicorp", "null", version+".json"))

		proxy.cacheWg.Wait()

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "zh:")
	})

	t.Run("uses token for matching hostname", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)
		mockResolver := provider.NewMockRegistryProtocol(t)
		mockLogger := joblogger.NewMockLogger(t)

		prov := &provider.Provider{Hostname: "tharsis.example.com", Namespace: "myorg", Type: "custom"}
		filename := provider.GetPackageName("custom", version, testOS, testArch)
		downloadURL := "https://tharsis.example.com/pkg.zip"
		shasumURL := "https://tharsis.example.com/shasums"
		packageInfo := &provider.PackageInfo{DownloadURL: downloadURL, SHASumsURL: shasumURL}
		groupPath := "test-group"
		token := "secret-token"

		// Not cached
		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("GetProviderPlatformPackageDownloadURL", mock.Anything, mock.Anything).
			Return(nil, &sdkTypes.Error{Code: sdkTypes.ErrNotFound})

		mockResolver.On("GetPackageInfo", mock.Anything, prov, version, testOS, testArch, mock.AnythingOfType("provider.RequestOption")).
			Return(packageInfo, nil)
		mockResolver.On("GetChecksums", mock.Anything, packageInfo).
			Return(provider.Checksums{filename: []byte("checksum123456789012345678901234")}, nil)

		// Async caching mocks
		mockClient.On("CreateProviderVersionMirror", mock.Anything, &sdkTypes.CreateTerraformProviderVersionMirrorInput{
			GroupPath:         groupPath,
			Type:              "custom",
			RegistryHostname:  "tharsis.example.com",
			RegistryNamespace: "myorg",
			SemanticVersion:   version,
			RegistryToken:     &token,
		}).Return(&sdkTypes.TerraformProviderVersionMirror{Metadata: sdkTypes.ResourceMetadata{ID: "mirror-id"}}, nil).Maybe()
		mockResolver.On("DownloadPackage", mock.Anything, downloadURL).
			Return(io.NopCloser(bytes.NewReader([]byte("data"))), int64(4), nil).Maybe()
		// UploadProviderPlatformPackageToMirror has Reader field that can't be matched exactly
		mockClient.On("UploadProviderPlatformPackageToMirror", mock.Anything, mock.Anything).Return(nil).Maybe()
		mockLogger.On("Infof", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

		proxy := &mirrorProxy{
			client:         mockClient,
			registryClient: mockResolver,
			log:            &batchedLog{logger: mockLogger},
			workspaceID:    "test-workspace-id",
			cancellableCtx: t.Context(),
			registryTokens: map[string]string{"tharsis.example.com": "secret-token"},
		}
		w := httptest.NewRecorder()
		proxy.handlePackages(w, buildRequest("tharsis.example.com", "myorg", "custom", version+".json"))

		proxy.cacheWg.Wait()

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid provider address", func(t *testing.T) {
		proxy := &mirrorProxy{}
		w := httptest.NewRecorder()
		proxy.handlePackages(w, buildRequest("", "hashicorp", "null", "3.2.0.json"))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestMirrorProxy_cacheProviderPackage(t *testing.T) {
	prov := &provider.Provider{Hostname: "registry.terraform.io", Namespace: "hashicorp", Type: "null"}
	version := "3.2.0"
	testOS := "linux"
	testArch := "amd64"
	downloadURL := "https://example.com/pkg.zip"

	t.Run("success", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)
		mockResolver := provider.NewMockRegistryProtocol(t)

		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("CreateProviderVersionMirror", mock.Anything, &sdkTypes.CreateTerraformProviderVersionMirrorInput{
			GroupPath:         "test-group",
			Type:              "null",
			RegistryHostname:  "registry.terraform.io",
			RegistryNamespace: "hashicorp",
			SemanticVersion:   version,
		}).Return(&sdkTypes.TerraformProviderVersionMirror{Metadata: sdkTypes.ResourceMetadata{ID: "mirror-id"}}, nil)
		mockResolver.On("DownloadPackage", mock.Anything, downloadURL).
			Return(io.NopCloser(bytes.NewReader([]byte("data"))), int64(4), nil)
		// UploadProviderPlatformPackageToMirror has Reader field that can't be matched exactly
		mockClient.On("UploadProviderPlatformPackageToMirror", mock.Anything, mock.Anything).Return(nil)

		proxy := &mirrorProxy{client: mockClient, registryClient: mockResolver, workspaceID: "test-workspace-id"}
		err := proxy.cacheProviderPackage(t.Context(), prov, version, testOS, testArch, downloadURL)

		assert.NoError(t, err)
	})

	t.Run("conflict on upload is ignored", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)
		mockResolver := provider.NewMockRegistryProtocol(t)

		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("CreateProviderVersionMirror", mock.Anything, &sdkTypes.CreateTerraformProviderVersionMirrorInput{
			GroupPath:         "test-group",
			Type:              "null",
			RegistryHostname:  "registry.terraform.io",
			RegistryNamespace: "hashicorp",
			SemanticVersion:   version,
		}).Return(&sdkTypes.TerraformProviderVersionMirror{Metadata: sdkTypes.ResourceMetadata{ID: "mirror-id"}}, nil)
		mockResolver.On("DownloadPackage", mock.Anything, downloadURL).
			Return(io.NopCloser(bytes.NewReader([]byte("data"))), int64(4), nil)
		// UploadProviderPlatformPackageToMirror has Reader field that can't be matched exactly
		mockClient.On("UploadProviderPlatformPackageToMirror", mock.Anything, mock.Anything).
			Return(&sdkTypes.Error{Code: sdkTypes.ErrConflict})

		proxy := &mirrorProxy{client: mockClient, registryClient: mockResolver, workspaceID: "test-workspace-id"}
		err := proxy.cacheProviderPackage(t.Context(), prov, version, testOS, testArch, downloadURL)

		assert.NoError(t, err)
	})

	t.Run("conflict on version mirror creation returns early", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)

		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("CreateProviderVersionMirror", mock.Anything, &sdkTypes.CreateTerraformProviderVersionMirrorInput{
			GroupPath:         "test-group",
			Type:              "null",
			RegistryHostname:  "registry.terraform.io",
			RegistryNamespace: "hashicorp",
			SemanticVersion:   version,
		}).Return(nil, &sdkTypes.Error{Code: sdkTypes.ErrConflict})

		proxy := &mirrorProxy{client: mockClient, workspaceID: "test-workspace-id"}
		err := proxy.cacheProviderPackage(t.Context(), prov, version, testOS, testArch, downloadURL)

		assert.NoError(t, err)
	})

	t.Run("empty content length returns error", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)
		mockResolver := provider.NewMockRegistryProtocol(t)

		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("CreateProviderVersionMirror", mock.Anything, mock.Anything).
			Return(&sdkTypes.TerraformProviderVersionMirror{Metadata: sdkTypes.ResourceMetadata{ID: "mirror-id"}}, nil)
		mockResolver.On("DownloadPackage", mock.Anything, downloadURL).
			Return(io.NopCloser(bytes.NewReader(nil)), int64(0), nil)

		proxy := &mirrorProxy{client: mockClient, registryClient: mockResolver, workspaceID: "test-workspace-id"}
		err := proxy.cacheProviderPackage(t.Context(), prov, version, testOS, testArch, downloadURL)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty response from upstream")
	})

	t.Run("cancelled context stops caching", func(t *testing.T) {
		mockClient := jobclient.NewMockClient(t)
		mockResolver := provider.NewMockRegistryProtocol(t)

		mockClient.On("GetWorkspace", mock.Anything, "test-workspace-id").Return(&sdkTypes.Workspace{FullPath: "test-group/workspace"}, nil)
		mockClient.On("CreateProviderVersionMirror", mock.Anything, &sdkTypes.CreateTerraformProviderVersionMirrorInput{
			GroupPath:         "test-group",
			Type:              "null",
			RegistryHostname:  "registry.terraform.io",
			RegistryNamespace: "hashicorp",
			SemanticVersion:   version,
		}).Return(&sdkTypes.TerraformProviderVersionMirror{Metadata: sdkTypes.ResourceMetadata{ID: "mirror-id"}}, nil)
		mockResolver.On("DownloadPackage", mock.Anything, downloadURL).
			Return(nil, int64(0), context.Canceled)

		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		proxy := &mirrorProxy{client: mockClient, registryClient: mockResolver, workspaceID: "test-workspace-id"}
		err := proxy.cacheProviderPackage(ctx, prov, version, testOS, testArch, downloadURL)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to download")
	})
}

func buildRequest(hostname, namespace, providerType, version string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("hostname", hostname)
	rctx.URLParams.Add("namespace", namespace)
	rctx.URLParams.Add("type", providerType)
	if version != "" {
		rctx.URLParams.Add("version", version)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
