package azurefederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/azurefederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const (
	// Port range for temporary web server.
	tempServerMinPort = 21000
	tempServerMaxPort = 21200
)

// Logger used to provider error information
type Logger interface {
	Errorf(format string, a ...interface{})
}

type tokenResponse struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// Authenticator supports AWS OIDC Federation
type Authenticator struct {
	server    *http.Server
	jobLogger Logger
}

// New creates a new instance of Authenticator
func New(jobLogger Logger) *Authenticator {
	return &Authenticator{jobLogger: jobLogger}
}

// Close cleans up any open resources
func (a *Authenticator) Close(ctx context.Context) error {
	if a.server != nil {
		// Immediately terminate the web server.
		if err := a.terminateWebServer(ctx, a.server); err != nil {
			return fmt.Errorf("failed to terminate local web server for azure OIDC callback %v", err)
		}
	}
	return nil
}

// Authenticate configures the environment with the identity information used by the AWS terraform provider
func (a *Authenticator) Authenticate(ctx context.Context, managedIdentity *types.ManagedIdentity, creds []byte) (map[string]string, error) {
	decodedData, err := base64.StdEncoding.DecodeString(string(managedIdentity.Data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode managed identity payload %v", err)
	}

	federatedData := azurefederated.Data{}
	if err = json.Unmarshal(decodedData, &federatedData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal managed identity payload %v", err)
	}

	// Open the net.listener and build the callback URL.
	netListener, callbackURL, err := openNetListener()
	if err != nil {
		return nil, err
	}

	// Launch local server that is used for OIDC federation.

	// TODO: This can be removed once the following issue has been
	// resolved: https://github.com/hashicorp/terraform-provider-azurerm/issues/16900
	a.server = a.launchWebServer(netListener)

	return map[string]string{
		"ARM_TENANT_ID":          federatedData.TenantID,
		"ARM_CLIENT_ID":          federatedData.ClientID,
		"ARM_USE_OIDC":           "true",
		"ARM_OIDC_REQUEST_TOKEN": string(creds),
		"ARM_OIDC_REQUEST_URL":   callbackURL,
	}, nil
}

func openNetListener() (net.Listener, string, error) {
	for port := tempServerMinPort; port < tempServerMaxPort; port++ {
		listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			// At this point the listener has been opened
			callbackURL := fmt.Sprintf("http://localhost:%d", port)
			return listener, callbackURL, nil
		}
	}

	// Return error if no port was available
	return nil, "", fmt.Errorf("no port could be opened for the temporary azure web server that is used for OIDC Federation")
}

func (a *Authenticator) launchWebServer(netListener net.Listener) *http.Server {
	// Create server
	httpServer := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			token := ""
			bearer := req.Header.Get("Authorization")
			if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
				token = bearer[7:]
			}

			if token == "" {
				a.jobLogger.Errorf("Local server for Azure OIDC federation received an empty token")
				resp.WriteHeader(http.StatusUnauthorized)
				return
			}

			jsonResp, err := json.Marshal(&tokenResponse{Count: 1, Value: token})
			if err != nil {
				a.jobLogger.Errorf("Failed to marshal token response: %v", err)
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Return an HTTP response.
			resp.Header().Add("Content-Type", "application/json")
			resp.WriteHeader(http.StatusOK)
			_, _ = resp.Write(jsonResp)
		}),
	}

	// Launch server
	go func() {
		err := httpServer.Serve(netListener)
		if err != nil && err != http.ErrServerClosed {
			a.jobLogger.Errorf("Local server for azure OIDC federation failed unexpectedly: %v", err)
		}
	}()

	return httpServer
}

func (a *Authenticator) terminateWebServer(ctx context.Context, server *http.Server) error {
	// Gracefully shutdown server
	err := server.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}
