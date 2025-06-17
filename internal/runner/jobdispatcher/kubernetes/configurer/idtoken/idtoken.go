// Package idtoken provides a Kubernetes configurer that uses the runner's ID token for authentication.
package idtoken

import (
	"context"
	"encoding/base64"
	"fmt"

	"k8s.io/client-go/rest"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
)

var _ configurer.Configurer = (*IDToken)(nil)

// IDToken provides a Kubernetes configurer that uses the runner's ID token for authentication.
type IDToken struct {
	host        string
	tokenGetter types.TokenGetterFunc
	caData      []byte
}

// New creates a new IDToken configurer with the provided host, CA data, and token getter function.
func New(host string, ca string, tokenGetter types.TokenGetterFunc) (*IDToken, error) {
	var caData []byte
	if ca != "" {
		data, err := base64.StdEncoding.DecodeString(ca)
		if err != nil {
			return nil, fmt.Errorf("failed to decode CA certificate for host %s: %w", host, err)
		}
		caData = data
	}
	return &IDToken{
		host:        host,
		tokenGetter: tokenGetter,
		caData:      caData,
	}, nil
}

// GetConfig returns a Kubernetes rest.Config using the runner's ID token for authentication.
func (i *IDToken) GetConfig(ctx context.Context) (*rest.Config, error) {
	token, err := i.tokenGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get runner ID token: %w", err)
	}

	return &rest.Config{
		Host:        i.host,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: i.caData,
		},
	}, nil
}
