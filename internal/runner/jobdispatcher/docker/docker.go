// Package docker package
package docker

//go:generate go tool mockery --name client --inpackage --case underscore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	dockerclient "github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var _ jobdispatcher.JobDispatcher = (*JobDispatcher)(nil)

var pluginDataRequiredFields = []string{"host", "image", "endpoint"}

// containerIDKey is the dispatcher-data key under which the container ID is stored for cleanup.
const containerIDKey = "containerID"

type client interface {
	ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
}

// JobDispatcher uses the local docker api to dispatch jobs
type JobDispatcher struct {
	logger                 logger.Logger
	client                 client
	image                  string
	bindPath               string
	registryUsername       string
	registryPassword       string
	apiEndpoint            string
	discoveryProtocolHosts []string
	extraHosts             []string
	limits                 *types.ResourceLimits
	localImage             bool
}

// New creates a JobDispatcher
func New(pluginData map[string]string, discoveryProtocolHost string, logger logger.Logger) (*JobDispatcher, error) {
	if err := types.MigrateDeprecatedPluginDataFields(pluginData, logger); err != nil {
		return nil, err
	}

	for _, field := range pluginDataRequiredFields {
		if _, ok := pluginData[field]; !ok {
			return nil, fmt.Errorf("docker job dispatcher requires plugin data '%s' field", field)
		}
	}

	var localImage bool
	if _, ok := pluginData["local_image"]; ok {
		var err error
		localImage, err = strconv.ParseBool(pluginData["local_image"])
		if err != nil {
			return nil, fmt.Errorf("failed to parse job dispatcher 'local_image' config: %v", err)
		}
	}

	extraHosts := []string{}
	if _, ok := pluginData["extra_hosts"]; ok {
		extraHosts = append(extraHosts, strings.Split(pluginData["extra_hosts"], ",")...)
	}

	limits, err := types.LoadResourceLimits(pluginData)
	if err != nil {
		return nil, err
	}

	discoveryProtocolHosts := []string{}

	if discoveryProtocolHost != "" {
		discoveryProtocolHosts = append(discoveryProtocolHosts, discoveryProtocolHost)
	}

	if extraDiscoveryHostsStr, ok := pluginData["extra_service_discovery_hosts"]; ok {
		for _, host := range strings.Split(extraDiscoveryHostsStr, ",") {
			discoveryProtocolHosts = append(discoveryProtocolHosts, strings.TrimSpace(host))
		}
	}

	client, err := dockerclient.NewClientWithOpts(dockerclient.WithHost(pluginData["host"]), dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("job dispatcher failed to initialize docker cli: %v", err)
	}

	return &JobDispatcher{
		image:                  pluginData["image"],
		bindPath:               pluginData["bind_path"],
		apiEndpoint:            pluginData["endpoint"],
		discoveryProtocolHosts: discoveryProtocolHosts,
		registryUsername:       pluginData["registry_username"],
		registryPassword:       pluginData["registry_password"],
		extraHosts:             extraHosts,
		limits:                 limits,
		localImage:             localImage,
		client:                 client,
		logger:                 logger,
	}, nil
}

// DispatchJob will start a docker container to execute the job
func (j *JobDispatcher) DispatchJob(ctx context.Context, jobID string, token string) (map[string]string, error) {
	if !j.localImage {
		authStr, err := j.getRegistryAuth()
		if err != nil {
			return nil, err
		}

		out, err := j.client.ImagePull(ctx, j.image, image.PullOptions{
			RegistryAuth: authStr,
		})
		if err != nil {
			return nil, err
		}
		_, _ = io.Copy(os.Stdout, out)
	}

	hostConfig := &container.HostConfig{}

	if len(j.extraHosts) > 0 {
		hostConfig.ExtraHosts = j.extraHosts
	}

	if j.bindPath != "" {
		hostConfig.Binds = []string{j.bindPath}
	}

	if j.limits != nil && j.limits.MemoryBytes != 0 {
		hostConfig.Memory = int64(j.limits.MemoryBytes)
		hostConfig.MemorySwap = int64(j.limits.MemoryBytes)
	}

	env := []string{
		fmt.Sprintf("ENDPOINT=%s", j.apiEndpoint),
		fmt.Sprintf("JOB_ID=%s", jobID),
		fmt.Sprintf("JOB_TOKEN=%s", token),
		fmt.Sprintf("DISCOVERY_PROTOCOL_HOSTS=%s", strings.Join(j.discoveryProtocolHosts, ",")),
	}

	limitEnv := j.limits.AsEnvVars()
	for _, name := range slices.Sorted(maps.Keys(limitEnv)) {
		env = append(env, fmt.Sprintf("%s=%s", name, limitEnv[name]))
	}

	resp, err := j.client.ContainerCreate(ctx, &container.Config{
		Image: j.image,
		Env:   env,
	}, hostConfig, nil, nil, "")
	if err != nil {
		return nil, err
	}

	if err := j.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, err
	}

	return map[string]string{containerIDKey: resp.ID}, nil
}

func (j *JobDispatcher) getRegistryAuth() (string, error) {
	if j.registryUsername != "" && j.registryPassword != "" {
		authConfig := registry.AuthConfig{
			Username: j.registryUsername,
			Password: j.registryPassword,
		}

		encodedAuth, err := json.Marshal(authConfig)
		if err != nil {
			return "", fmt.Errorf("error when encoding registry authConfig: %v", err)
		}

		return base64.URLEncoding.EncodeToString(encodedAuth), nil
	}
	return "", nil
}

// CleanupJob removes the job's container, whose ID DispatchJob returned as the runtime resource name.
func (j *JobDispatcher) CleanupJob(ctx context.Context, jobID string, dispatcherData map[string]string) error {
	containerID := dispatcherData[containerIDKey]
	if containerID == "" {
		// Nothing to remove without a container ID; skip the API call rather than remove an empty ID.
		j.logger.Warnf("docker job dispatcher skipping cleanup for job %s: dispatcher data has no container ID", jobID)
		return nil
	}

	// Force removal so a container that is still running (e.g. a canceled job) is torn down too.
	if err := j.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		if cerrdefs.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("docker job dispatcher failed to remove container %s for job %s: %v", containerID, jobID, err)
	}

	return nil
}

// Limits returns the resource limits jobs run under.
func (j *JobDispatcher) Limits() *types.ResourceLimits {
	return j.limits
}
