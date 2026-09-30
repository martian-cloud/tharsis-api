// Package types defines types used by the job dispatchers
package types

import (
	"context"
	"fmt"
	"strconv"

	humanize "github.com/dustin/go-humanize"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// TokenGetterFunc is a function type that retrieves a runner ID token for authentication.
type TokenGetterFunc func(ctx context.Context) (string, error)

// Resource limit plugin data keys. A zero or absent value disables the corresponding warning.
const (
	pluginDataMemoryLimit         = "memory_limit"
	pluginDataDiskReadLimit       = "disk_read_limit"
	pluginDataDiskWriteLimit      = "disk_write_limit"
	pluginDataNetworkReceiveLimit = "network_received_limit"
	pluginDataNetworkSendLimit    = "network_sent_limit"
)

// ResourceLimits holds the parsed per-job resource limits. A zero value means the limit is unset.
type ResourceLimits struct {
	MemoryBytes          uint64
	DiskReadBytes        uint64
	DiskWriteBytes       uint64
	NetworkReceivedBytes uint64
	NetworkSentBytes     uint64
}

// goMemLimitFraction is the fraction of the hard memory limit used as the GOMEMLIMIT soft limit for
// the job executor's Go runtime. Below 1.0 it leaves headroom so the runtime GCs aggressively before
// the cgroup limit is hit, reducing the chance the kernel OOM-kills the job.
const goMemLimitFraction = 0.9

// AsEnvVars renders the limits as the env vars the job executor reads.
func (l *ResourceLimits) AsEnvVars() map[string]string {
	env := map[string]string{}

	if l == nil {
		return env
	}

	byteLimits := map[string]uint64{
		"MEMORY_LIMIT":           l.MemoryBytes,
		"DISK_READ_LIMIT":        l.DiskReadBytes,
		"DISK_WRITE_LIMIT":       l.DiskWriteBytes,
		"NETWORK_RECEIVED_LIMIT": l.NetworkReceivedBytes,
		"NETWORK_SENT_LIMIT":     l.NetworkSentBytes,
	}

	for name, v := range byteLimits {
		if v == 0 {
			continue
		}

		env[name] = strconv.FormatUint(v, 10)
	}

	// GOMEMLIMIT makes the executor's Go runtime GC aggressively as its heap nears the limit, so it
	// stays under the cgroup ceiling. Only set when a memory limit is configured.
	if l.MemoryBytes > 0 {
		env["GOMEMLIMIT"] = fmt.Sprintf("%dB", uint64(float64(l.MemoryBytes)*goMemLimitFraction))
	}

	return env
}

// LoadResourceLimits parses the resource limit plugin data fields. Byte-valued limits accept
// humanized sizes (e.g. "512Mi", "1Gi"). Absent fields stay zero, disabling that limit.
func LoadResourceLimits(pluginData map[string]string) (*ResourceLimits, error) {
	limits := &ResourceLimits{}

	fields := []struct {
		key string
		dst *uint64
	}{
		{pluginDataMemoryLimit, &limits.MemoryBytes},
		{pluginDataDiskReadLimit, &limits.DiskReadBytes},
		{pluginDataDiskWriteLimit, &limits.DiskWriteBytes},
		{pluginDataNetworkReceiveLimit, &limits.NetworkReceivedBytes},
		{pluginDataNetworkSendLimit, &limits.NetworkSentBytes},
	}

	for _, f := range fields {
		v, ok := pluginData[f.key]
		if !ok || v == "" {
			continue
		}

		n, err := humanize.ParseBytes(v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse job dispatcher '%s' config: %w", f.key, err)
		}

		*f.dst = n
	}

	return limits, nil
}

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
