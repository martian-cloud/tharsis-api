// Package configurer package
package configurer

//go:generate go tool mockery --name Configurer --inpackage --case underscore

import (
	"context"

	"k8s.io/client-go/rest"
)

// Configurer is an interface for returning a kubernetes rest config for the kubernetes go-client
type Configurer interface {
	GetConfig(context.Context) (*rest.Config, error)
}
