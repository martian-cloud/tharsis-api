package controllers

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// discoveryController implements the http.Handler interface and is used to serve the service discovery documents.
type discoveryController struct {
	logger            logger.Logger
	respWriter        response.Writer
	host              string
	port              string
	transportSecurity string
}

// NewServiceDiscoveryController returns a new discoveryController.
func NewServiceDiscoveryController(
	logger logger.Logger,
	respWriter response.Writer,
	apiURL string,
	externalGRPCPort string,
) (Controller, error) {
	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		logger.Error("error parsing API URL", err)
		return nil, err
	}

	transportSecurity := "plaintext"
	if parsedURL.Scheme == "https" {
		transportSecurity = "tls"
	}

	return &discoveryController{
		logger:            logger,
		respWriter:        respWriter,
		host:              parsedURL.Hostname(),
		port:              externalGRPCPort,
		transportSecurity: transportSecurity,
	}, nil
}

// RegisterRoutes adds the service discovery routes to the router.
func (c *discoveryController) RegisterRoutes(router chi.Router) {
	router.Get("/.well-known/tharsis.json", c.getServiceDiscovery)
}

func (c *discoveryController) getServiceDiscovery(w http.ResponseWriter, r *http.Request) {
	body := map[string]any{
		"grpc": &client.GRPCDiscoveryDocument{
			Host:              c.host,
			TransportSecurity: c.transportSecurity,
			Port:              c.port,
		},
	}

	c.respWriter.RespondWithJSON(r.Context(), w, body, http.StatusOK)
}
