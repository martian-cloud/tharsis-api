package kubernetes

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	// ownerLabelKey identifies pods created by this dispatcher.
	ownerLabelKey = "app.kubernetes.io/managed-by"
	// ownerLabelValue is the value of ownerLabelKey on job pods.
	ownerLabelValue = "tharsis-runner"
	// jobIDLabelKey carries the Tharsis job ID, whose raw-URL base64 encoding is always a valid label value.
	jobIDLabelKey = "tharsis-job-id"
	// podNamePrefix prefixes every job pod name; with the decoded UUID the name is 48 chars, within DNS limits.
	podNamePrefix = "tharsis-job-"
)

var pluginDataRequiredFields = []string{"endpoint", "auth_type", "image", "memory_request", "memory_limit"}

// config holds the parsed plugin data for the kubernetes job dispatcher.
type config struct {
	securityContext        *corev1.SecurityContext
	nodeSelector           map[string]string
	extraAnnotations       map[string]string
	labels                 map[string]string
	limits                 *types.ResourceLimits
	activeDeadlineSeconds  *int64
	image                  string
	apiEndpoint            string
	namespace              string
	memoryRequest          resource.Quantity
	memoryLimit            resource.Quantity
	cpuRequest             resource.Quantity
	cpuLimit               resource.Quantity
	hostAliases            []corev1.HostAlias
	discoveryProtocolHosts []string
}

// buildPod constructs the job pod spec, deriving a deterministic name from the job's decoded UUID.
func (c *config) buildPod(jobID, token string) (*corev1.Pod, error) {
	name, err := podName(jobID)
	if err != nil {
		return nil, err
	}

	annotations := map[string]string{}
	maps.Copy(annotations, c.extraAnnotations)

	labels := map[string]string{}
	maps.Copy(labels, c.labels)
	// Force the reaper's labels last so operator-supplied pod_labels can't shadow them.
	labels[ownerLabelKey] = ownerLabelValue
	labels[jobIDLabelKey] = jobID

	env := []corev1.EnvVar{
		{Name: "JOB_ID", Value: jobID},
		{Name: "JOB_TOKEN", Value: token},
		{Name: "ENDPOINT", Value: c.apiEndpoint},
		{Name: "DISCOVERY_PROTOCOL_HOSTS", Value: strings.Join(c.discoveryProtocolHosts, ",")},
	}

	limitEnv := c.limits.AsEnvVars()
	for _, name := range slices.Sorted(maps.Keys(limitEnv)) {
		env = append(env, corev1.EnvVar{Name: name, Value: limitEnv[name]})
	}

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: new(false),
			// Disable service link env injection so the API's service address isn't leaked into user Terraform.
			EnableServiceLinks:    new(false),
			NodeSelector:          c.nodeSelector,
			HostAliases:           c.hostAliases,
			ActiveDeadlineSeconds: c.activeDeadlineSeconds,
			Containers: []corev1.Container{
				{
					Name:            "main",
					Image:           c.image,
					SecurityContext: c.securityContext,
					Env:             env,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceMemory: c.memoryRequest,
							corev1.ResourceCPU:    c.cpuRequest,
						},
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: c.memoryLimit,
							corev1.ResourceCPU:    c.cpuLimit,
						},
					},
				},
			},
			RestartPolicy:                 corev1.RestartPolicyNever,
			TerminationGracePeriodSeconds: new(int64(time.Hour / time.Second)),
		},
	}

	return pod, nil
}

// parseConfig validates and parses the plugin data into a config, excluding the auth configurer.
func parseConfig(pluginData map[string]string, discoveryProtocolHost string, logger logger.Logger) (*config, error) {
	if err := types.MigrateDeprecatedPluginDataFields(pluginData, logger); err != nil {
		return nil, err
	}

	for _, field := range pluginDataRequiredFields {
		if _, ok := pluginData[field]; !ok {
			return nil, fmt.Errorf("kubernetes job dispatcher requires plugin data '%s' field", field)
		}
	}

	securityContext, err := parseSecurityContext(pluginData)
	if err != nil {
		return nil, err
	}

	memoryRequest, err := resource.ParseQuantity(pluginData["memory_request"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory request for runner jobs: %v", err)
	}

	limits, err := types.LoadResourceLimits(pluginData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource limits for runner jobs: %v", err)
	}

	if limits.MemoryBytes == 0 {
		return nil, fmt.Errorf("memory_limit must be a non-zero value for runner jobs")
	}

	memoryLimit := *resource.NewQuantity(int64(limits.MemoryBytes), resource.BinarySI)

	cpuRequest, err := optionalQuantity(pluginData, "cpu_request")
	if err != nil {
		return nil, err
	}

	cpuLimit, err := optionalQuantity(pluginData, "cpu_limit")
	if err != nil {
		return nil, err
	}

	nodeSelector, err := parseNodeSelector(pluginData["node_selector"])
	if err != nil {
		return nil, err
	}

	extraAnnotations, err := parseJSONMap(pluginData["pod_annotations"], "pod annotations")
	if err != nil {
		return nil, err
	}

	labels, err := parseJSONMap(pluginData["pod_labels"], "pod labels")
	if err != nil {
		return nil, err
	}

	activeDeadlineSeconds, err := optionalPositiveInt64(pluginData, "pod_active_deadline_seconds")
	if err != nil {
		return nil, err
	}

	namespace := "default"
	if ns, ok := pluginData["namespace"]; ok && ns != "" {
		namespace = ns
	}

	return &config{
		image:                  pluginData["image"],
		apiEndpoint:            pluginData["endpoint"],
		namespace:              namespace,
		discoveryProtocolHosts: parseDiscoveryHosts(discoveryProtocolHost, pluginData["extra_service_discovery_hosts"]),
		memoryRequest:          memoryRequest,
		memoryLimit:            memoryLimit,
		cpuRequest:             cpuRequest,
		cpuLimit:               cpuLimit,
		securityContext:        securityContext,
		nodeSelector:           nodeSelector,
		hostAliases:            parseHostAliases(pluginData["host_aliases"]),
		extraAnnotations:       extraAnnotations,
		labels:                 labels,
		limits:                 limits,
		activeDeadlineSeconds:  activeDeadlineSeconds,
	}, nil
}

func parseSecurityContext(pluginData map[string]string) (*corev1.SecurityContext, error) {
	runAsUser, err := optionalInt64(pluginData, "security_context_run_as_user")
	if err != nil {
		return nil, err
	}

	runAsGroup, err := optionalInt64(pluginData, "security_context_run_as_group")
	if err != nil {
		return nil, err
	}

	var runAsNonRoot *bool
	if v, ok := pluginData["security_context_run_as_non_root"]; ok {
		parsed, parseErr := strconv.ParseBool(v)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse security_context_run_as_non_root for runner jobs: %v", parseErr)
		}

		runAsNonRoot = &parsed
	}

	return &corev1.SecurityContext{
		Privileged:               new(false),
		AllowPrivilegeEscalation: new(false),
		RunAsUser:                runAsUser,
		RunAsGroup:               runAsGroup,
		RunAsNonRoot:             runAsNonRoot,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"NET_RAW"},
		},
	}, nil
}

func parseNodeSelector(value string) (map[string]string, error) {
	if value == "" {
		return nil, nil
	}

	nodeSelector := map[string]string{}
	for pair := range strings.SplitSeq(value, ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid node selector format: %q, expected format: key1=value1,key2=value2", pair)
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" || val == "" {
			return nil, fmt.Errorf("invalid node selector format: %q, key and value cannot be empty", pair)
		}

		nodeSelector[key] = val
	}

	return nodeSelector, nil
}

func parseHostAliases(value string) []corev1.HostAlias {
	if value == "" {
		return nil
	}

	var hostAliases []corev1.HostAlias
	for _, hostEntry := range strings.Split(value, ",") {
		parts := strings.SplitN(strings.TrimSpace(hostEntry), ":", 2)
		if len(parts) == 2 {
			hostAliases = append(hostAliases, corev1.HostAlias{
				IP:        strings.TrimSpace(parts[1]),
				Hostnames: []string{strings.TrimSpace(parts[0])},
			})
		}
	}

	return hostAliases
}

func parseDiscoveryHosts(primary, extra string) []string {
	hosts := []string{}
	if primary != "" {
		hosts = append(hosts, primary)
	}

	if extra != "" {
		for _, host := range strings.Split(extra, ",") {
			hosts = append(hosts, strings.TrimSpace(host))
		}
	}

	return hosts
}

func parseJSONMap(value, label string) (map[string]string, error) {
	result := map[string]string{}
	if value == "" {
		return result, nil
	}

	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, fmt.Errorf("%s options is invalid: %w", label, err)
	}

	return result, nil
}

func optionalInt64(pluginData map[string]string, key string) (*int64, error) {
	v, ok := pluginData[key]
	if !ok || v == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s for runner jobs: %v", key, err)
	}

	return &parsed, nil
}

func optionalPositiveInt64(pluginData map[string]string, key string) (*int64, error) {
	v, ok := pluginData[key]
	if !ok || v == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s for runner jobs: %v", key, err)
	}

	if parsed <= 0 {
		return nil, fmt.Errorf("%s must be a positive value for runner jobs", key)
	}

	return &parsed, nil
}

func optionalQuantity(pluginData map[string]string, key string) (resource.Quantity, error) {
	v, ok := pluginData[key]
	if !ok || v == "" {
		return resource.Quantity{}, nil
	}

	q, err := resource.ParseQuantity(v)
	if err != nil {
		return resource.Quantity{}, fmt.Errorf("failed to parse %s for runner jobs: %v", key, err)
	}

	return q, nil
}

// podName derives a deterministic, DNS-safe pod name from the job's global ID by decoding it to its UUID.
func podName(jobID string) (string, error) {
	parsed, err := gid.ParseGlobalID(jobID)
	if err != nil {
		return "", fmt.Errorf("failed to derive pod name from job ID %q: %w", jobID, err)
	}

	return podNamePrefix + parsed.ID, nil
}
