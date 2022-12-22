package docker

//go:generate mockery --name client --inpackage --case underscore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
)

var pluginDataRequiredFields = []string{"host", "image", "api_url"}

type client interface {
	ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error)
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error
}

// JobDispatcher uses the local docker api to dispatch jobs
type JobDispatcher struct {
	logger                logger.Logger
	client                client
	image                 string
	bindPath              string
	registryUsername      string
	registryPassword      string
	apiURL                string
	discoveryProtocolHost string
	localImage            bool
}

// New creates a JobDispatcher
func New(pluginData map[string]string, discoveryProtocolHost string, logger logger.Logger) (*JobDispatcher, error) {
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

	client, err := dockerclient.NewClientWithOpts(dockerclient.WithHost(pluginData["host"]), dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("job dispatcher failed to initialize docker cli: %v", err)
	}

	return &JobDispatcher{
		image:                 pluginData["image"],
		bindPath:              pluginData["bind_path"],
		apiURL:                pluginData["api_url"],
		discoveryProtocolHost: discoveryProtocolHost,
		registryUsername:      pluginData["registry_username"],
		registryPassword:      pluginData["registry_password"],
		localImage:            localImage,
		client:                client,
		logger:                logger,
	}, nil
}

// DispatchJob will start a docker container to execute the job
func (j *JobDispatcher) DispatchJob(ctx context.Context, jobID string, token string) (string, error) {
	if !j.localImage {
		authStr, err := j.getRegistryAuth()
		if err != nil {
			return "", err
		}

		out, err := j.client.ImagePull(ctx, j.image, types.ImagePullOptions{
			RegistryAuth: authStr,
		})
		if err != nil {
			return "", err
		}
		_, _ = io.Copy(os.Stdout, out)
	}

	hostConfig := &container.HostConfig{}

	if j.bindPath != "" {
		hostConfig.Binds = []string{j.bindPath}
	}

	resp, err := j.client.ContainerCreate(ctx, &container.Config{
		Image: j.image,
		Env: []string{
			fmt.Sprintf("API_URL=%s", j.apiURL),
			fmt.Sprintf("JOB_ID=%s", jobID),
			fmt.Sprintf("JOB_TOKEN=%s", token),
			fmt.Sprintf("DISCOVERY_PROTOCOL_HOST=%s", j.discoveryProtocolHost),
		},
	}, hostConfig, nil, nil, "")
	if err != nil {
		return "", err
	}

	if err := j.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (j *JobDispatcher) getRegistryAuth() (string, error) {
	if j.registryUsername != "" && j.registryPassword != "" {
		authConfig := types.AuthConfig{
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
