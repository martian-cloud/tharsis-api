package jobexecutor

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/ansi"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/provider"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// Provider Mirror Proxy
//
// A local HTTPS proxy implementing Terraform's provider network mirror protocol. Intercepts
// provider download requests to provide caching with upstream fallback.
//
// Behavior:
//   - Cached: Returns presigned URLs to packages stored in Tharsis
//   - Uncached: Returns upstream URL with hash, caches asynchronously in background
//   - Offline: Falls back to cached versions when upstream registries unavailable
//
// Design decisions:
//   - Async caching: Terraform's HTTP timeout (~30s) is too short for large providers (AWS ~400MB)
//   - Hash passthrough: Returns zh: hash from upstream so Terraform verifies authenticity
//   - Self-signed TLS: Generates ephemeral cert; CA injected via SSL_CERT_FILE
//   - Graceful shutdown: Waits for in-flight caching goroutines before stopping

const (
	// localhostIP is the IP address the proxy listens on.
	localhostIP = "127.0.0.1"
	// proxyListenAddr is the address the proxy listens on with a random port.
	proxyListenAddr = localhostIP + ":0"
	// contentTypeJSON is the content type for JSON responses.
	contentTypeJSON = "application/json"
	// providerFetchTimeout is the timeout for fetching providers from upstream registries.
	providerFetchTimeout = 10 * time.Minute
	// shutdownTimeout is the timeout for graceful server shutdown.
	shutdownTimeout = 5 * time.Second
	// tlsCertAcceptableClockSkew is the buffer added to certificate validity to handle clock skew.
	tlsCertAcceptableClockSkew = 5 * time.Second
)

type logLevel int

const (
	logLevelInfo logLevel = iota
	logLevelError
)

type logEntry struct {
	message string
	level   logLevel
}

// batchedLog batches log messages to be flushed later.
type batchedLog struct {
	logger  joblogger.Logger
	entries []logEntry
	mu      sync.Mutex
}

func (l *batchedLog) printf(format string, args ...any) {
	l.mu.Lock()
	l.entries = append(l.entries, logEntry{message: fmt.Sprintf(format, args...), level: logLevelInfo})
	l.mu.Unlock()
}

func (l *batchedLog) errorf(format string, args ...any) {
	l.mu.Lock()
	l.entries = append(l.entries, logEntry{message: fmt.Sprintf(format, args...), level: logLevelError})
	l.mu.Unlock()
}

func (l *batchedLog) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) == 0 {
		return
	}

	l.logger.Printf("\n%sProvider caching summary...%s", ansi.Bold, ansi.Reset)

	for _, e := range l.entries {
		switch e.level {
		case logLevelError:
			l.logger.Errorf("%s", e.message)
		default:
			l.logger.Printf("%s", e.message)
		}
	}

	l.logger.Printf("%sProvider caching has been completed%s\n", ansi.Bold, ansi.Reset)
}

// packagesResponse is the response for the packages endpoint.
type packagesResponse struct {
	Archives map[string]packageArchive `json:"archives"`
}

// packageArchive represents a single platform archive.
type packageArchive struct {
	URL    string   `json:"url"`
	Hashes []string `json:"hashes,omitempty"`
}

// providerMirrorProxy is a local HTTPS server that proxies provider mirror requests.
type providerMirrorProxy interface {
	start()
	shutdown()
	url() string
	caCert() []byte
}

type mirrorProxy struct {
	server         *http.Server
	client         jobclient.Client
	listener       net.Listener
	log            *batchedLog
	registryClient provider.RegistryProtocol
	workspaceID    string
	caCertPEM      []byte
	registryTokens map[string]string // hostname -> token
	cacheWg        sync.WaitGroup
	cancellableCtx context.Context
}

func newProviderMirrorProxy(
	cancellableCtx context.Context,
	client jobclient.Client,
	workspaceID string,
	jobLogger joblogger.Logger,
	registryTokens map[string]string,
	maxJobDuration time.Duration,
) (providerMirrorProxy, error) {
	tlsConfig, caCert, err := generateTLSConfig(maxJobDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TLS config: %w", err)
	}

	listener, err := tls.Listen("tcp", proxyListenAddr, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS listener: %w", err)
	}

	proxy := &mirrorProxy{
		client:         client,
		listener:       listener,
		log:            &batchedLog{logger: jobLogger},
		registryClient: provider.NewRegistryClient(&http.Client{Timeout: providerFetchTimeout}),
		workspaceID:    workspaceID,
		caCertPEM:      caCert,
		registryTokens: registryTokens,
		cancellableCtx: cancellableCtx,
	}

	mux := chi.NewRouter()
	mux.Get("/{hostname}/{namespace}/{type}/index.json", proxy.handleVersions)
	mux.Get("/{hostname}/{namespace}/{type}/{version:.+\\.json}", proxy.handlePackages)
	proxy.server = &http.Server{Handler: mux}

	return proxy, nil
}

func (p *mirrorProxy) start() {
	go func() {
		if err := p.server.Serve(p.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			p.log.errorf("Mirror proxy server error: %v", err)
		}
	}()
}

func (p *mirrorProxy) shutdown() {
	// Wait for async caching goroutines to complete
	p.cacheWg.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := p.server.Shutdown(ctx); err != nil {
		p.log.errorf("Failed to shutdown provider mirror proxy: %v", err)
	}

	// Flush batched log messages
	p.log.flush()
}

func (p *mirrorProxy) url() string {
	return "https://" + p.listener.Addr().String()
}

func (p *mirrorProxy) caCert() []byte {
	return p.caCertPEM
}

func (p *mirrorProxy) handleVersions(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "hostname")
	namespace := chi.URLParam(r, "namespace")
	providerType := chi.URLParam(r, "type")

	prov, err := provider.NewProvider(hostname, namespace, providerType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var opts []provider.RequestOption
	if token, ok := p.registryTokens[hostname]; ok {
		opts = append(opts, provider.WithToken(token))
	}

	versionInfos, listErr := p.registryClient.ListVersions(r.Context(), prov, opts...)
	if listErr == nil {
		p.writeVersionsResponse(w, versionInfos)
		return
	}

	p.log.errorf("Failed to list upstream versions for %s, falling back to cached versions: %v", prov, listErr)

	rootGroupPath, err := p.getRootGroupPath(r.Context())
	if err != nil {
		http.Error(w, err.Error(), httpStatusFromError(err))
		return
	}

	// Fallback to cached versions if upstream fails
	versions, err := p.client.GetAvailableProviderVersions(r.Context(), &types.GetAvailableProviderVersionsInput{
		GroupPath:         rootGroupPath,
		RegistryHostname:  hostname,
		RegistryNamespace: namespace,
		Type:              providerType,
	})
	if err != nil {
		http.Error(w, err.Error(), httpStatusFromError(err))
		return
	}

	if len(versions) == 0 {
		http.Error(w, "no versions found in provider mirror", http.StatusNotFound)
		return
	}

	p.log.printf("- Used cached versions for %s (upstream unavailable)", prov)

	cachedVersions := make([]provider.VersionInfo, 0, len(versions))
	for v := range versions {
		cachedVersions = append(cachedVersions, provider.VersionInfo{Version: v})
	}

	p.writeVersionsResponse(w, cachedVersions)
}

func (p *mirrorProxy) handlePackages(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "hostname")
	namespace := chi.URLParam(r, "namespace")
	providerType := chi.URLParam(r, "type")
	version := strings.TrimSuffix(chi.URLParam(r, "version"), ".json")
	os := runtime.GOOS
	arch := runtime.GOARCH

	prov, err := provider.NewProvider(hostname, namespace, providerType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rootGroupPath, err := p.getRootGroupPath(r.Context())
	if err != nil {
		http.Error(w, err.Error(), httpStatusFromError(err))
		return
	}

	// Try to get cached package directly - returns not found or forbidden if not cached/accessible
	packageInfo, err := p.client.GetProviderPlatformPackageDownloadURL(r.Context(), &types.GetProviderPlatformPackageDownloadURLInput{
		GroupPath:         rootGroupPath,
		RegistryHostname:  hostname,
		RegistryNamespace: namespace,
		Type:              providerType,
		Version:           version,
		OS:                os,
		Arch:              arch,
	})
	if err != nil && !tharsis.IsNotFoundError(err) && httpStatusFromError(err) != http.StatusForbidden {
		http.Error(w, err.Error(), httpStatusFromError(err))
		return
	}

	if packageInfo != nil {
		p.log.printf("- Used cached %s v%s", prov, version)
		p.writePackageResponse(w, os, arch, packageInfo.URL, packageInfo.Hashes)
		return
	}

	// Not cached - get upstream info and return with hashes for verification
	var opts []provider.RequestOption
	if token, ok := p.registryTokens[hostname]; ok {
		opts = append(opts, provider.WithToken(token))
	}

	upstreamPackageInfo, err := p.registryClient.GetPackageInfo(r.Context(), prov, version, os, arch, opts...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Fetch checksums for verification
	checksums, err := p.registryClient.GetChecksums(r.Context(), upstreamPackageInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get hash for this platform's package
	hash, ok := checksums.GetZipHash(provider.GetPackageName(prov.Type, version, os, arch))
	if !ok {
		http.Error(w, "checksum not found", http.StatusNotFound)
		return
	}

	// Cache asynchronously since Terraform's HTTP client timeout is too short for large providers.
	p.cacheWg.Add(1)
	go func() {
		defer p.cacheWg.Done()

		if err := p.cacheProviderPackage(p.cancellableCtx, prov, version, os, arch, upstreamPackageInfo.DownloadURL); err != nil {
			p.log.errorf("Failed to cache %s v%s: %v", prov, version, err)
			return
		}

		p.log.printf("- Cached %s v%s", prov, version)
	}()

	p.writePackageResponse(w, os, arch, upstreamPackageInfo.DownloadURL, []string{hash})
}

func (p *mirrorProxy) cacheProviderPackage(ctx context.Context, prov *provider.Provider, version, os, arch, downloadURL string) error {
	// Re-query workspace to get current root group path (may have changed if workspace was migrated)
	groupPath, err := p.getRootGroupPath(ctx)
	if err != nil {
		return fmt.Errorf("failed to get root group path: %w", err)
	}

	var token *string
	if t, ok := p.registryTokens[prov.Hostname]; ok {
		token = &t
	}

	versionMirror, err := p.client.CreateProviderVersionMirror(ctx, &types.CreateTerraformProviderVersionMirrorInput{
		GroupPath:         groupPath,
		Type:              prov.Type,
		RegistryNamespace: prov.Namespace,
		RegistryHostname:  prov.Hostname,
		SemanticVersion:   version,
		RegistryToken:     token,
	})
	if err != nil {
		if tharsis.IsConflictError(err) {
			// Another job already created it, nothing to do
			return nil
		}
		return fmt.Errorf("failed to create version mirror: %w", err)
	}

	body, contentLength, err := p.registryClient.DownloadPackage(ctx, downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer body.Close()

	if contentLength == 0 {
		return fmt.Errorf("empty response from upstream")
	}

	if err := p.client.UploadProviderPlatformPackageToMirror(ctx, &types.UploadProviderPlatformPackageToMirrorInput{
		Reader:          body,
		VersionMirrorID: versionMirror.Metadata.ID,
		OS:              os,
		Arch:            arch,
	}); err != nil && !tharsis.IsConflictError(err) {
		return fmt.Errorf("failed to upload: %w", err)
	}

	return nil
}

func (p *mirrorProxy) writeVersionsResponse(w http.ResponseWriter, versionInfos []provider.VersionInfo) {
	versions := make(map[string]struct{}, len(versionInfos))
	for _, v := range versionInfos {
		versions[v.Version] = struct{}{}
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]any{"versions": versions}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (p *mirrorProxy) writePackageResponse(w http.ResponseWriter, os, arch, url string, hashes []string) {
	w.Header().Set("Content-Type", contentTypeJSON)
	platformKey := os + "_" + arch
	if err := json.NewEncoder(w).Encode(packagesResponse{
		Archives: map[string]packageArchive{
			platformKey: {URL: url, Hashes: hashes},
		},
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// getRootGroupPath fetches the workspace and returns its root group path.
func (p *mirrorProxy) getRootGroupPath(ctx context.Context) (string, error) {
	ws, err := p.client.GetWorkspace(ctx, p.workspaceID)
	if err != nil {
		return "", err
	}

	return strings.Split(ws.FullPath, "/")[0], nil
}

func generateTLSConfig(validity time.Duration) (*tls.Config, []byte, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-tlsCertAcceptableClockSkew),
		NotAfter:     time.Now().Add(validity + tlsCertAcceptableClockSkew),
		KeyUsage:     x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP(localhostIP)},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
		Leaf:        cert,
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}, certDER, nil
}

// httpStatusFromError returns an appropriate HTTP status code for the given error.
func httpStatusFromError(err error) int {
	var tErr *types.Error
	if errors.As(err, &tErr) {
		switch tErr.Code {
		case types.ErrNotFound:
			return http.StatusNotFound
		case types.ErrForbidden:
			return http.StatusForbidden
		case types.ErrUnauthorized:
			return http.StatusUnauthorized
		case types.ErrBadRequest:
			return http.StatusBadRequest
		case types.ErrConflict, types.ErrOptimisticLock:
			return http.StatusConflict
		case types.ErrTooManyRequests:
			return http.StatusTooManyRequests
		case types.ErrTooLarge:
			return http.StatusRequestEntityTooLarge
		case types.ErrServiceUnavailable:
			return http.StatusServiceUnavailable
		}
	}
	return http.StatusInternalServerError
}
