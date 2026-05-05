// Package types defines types used by the job dispatchers
package types

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// TokenGetterFunc is a function type that retrieves a runner ID token for authentication.
type TokenGetterFunc func(ctx context.Context) (string, error)

// MigrateDeprecatedPluginDataFields migrates deprecated plugin data field names.
func MigrateDeprecatedPluginDataFields(pluginData map[string]string, logger logger.Logger) error {
	if pluginData["endpoint"] != "" && pluginData["api_url"] != "" {
		return fmt.Errorf("plugin data fields 'endpoint' and 'api_url' cannot both be set")
	}

	if pluginData["api_url"] != "" {
		logger.Warnf("plugin data field 'api_url' is deprecated, use 'endpoint' instead")
		pluginData["endpoint"] = pluginData["api_url"]
	}

	return nil
}
