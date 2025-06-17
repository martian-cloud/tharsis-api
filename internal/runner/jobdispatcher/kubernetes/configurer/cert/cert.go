// Package cert provides a Kubernetes configurer that uses TLS certificates for authentication.
package cert

import (
	"context"
	"encoding/base64"
	"fmt"

	"k8s.io/client-go/rest"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer"
)

var _ configurer.Configurer = (*Cert)(nil)

// Cert provides a Kubernetes configurer that uses TLS certificates for authentication.
type Cert struct {
	host     string
	certData []byte
	keyData  []byte
	caData   []byte
}

// New creates a new Cert configurer with the provided host, certificate, key, and CA data.
func New(host string, cert string, key string, ca string) (*Cert, error) {
	certData, err := base64.StdEncoding.DecodeString(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to decode certificate for host %s: %w", host, err)
	}
	keyData, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key for host %s: %w", host, err)
	}
	var caData []byte
	if ca != "" {
		data, err := base64.StdEncoding.DecodeString(ca)
		if err != nil {
			return nil, fmt.Errorf("failed to decode CA certificate for host %s: %w", host, err)
		}
		caData = data
	}
	return &Cert{
		host:     host,
		certData: certData,
		keyData:  keyData,
		caData:   caData,
	}, nil
}

// GetConfig returns a Kubernetes rest.Config using the provided TLS certificate and key for authentication.
func (c *Cert) GetConfig(_ context.Context) (*rest.Config, error) {
	return &rest.Config{
		Host: c.host,
		TLSClientConfig: rest.TLSClientConfig{
			CertData: c.certData,
			KeyData:  c.keyData,
			CAData:   c.caData,
		},
	}, nil
}
