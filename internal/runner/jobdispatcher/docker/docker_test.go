package docker

import (
	"fmt"
	"io"
	"strings"
	"testing"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestNew(t *testing.T) {
	pluginData := map[string]string{
		"endpoint": "testUrl",
		"host":     "http://localhost",
		"image":    "testImage",
	}
	dispatcher, err := New(pluginData, "http://localhost", logger.New())
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	assert.Equal(t, "testUrl", dispatcher.apiEndpoint)
	assert.Equal(t, "testImage", dispatcher.image)
	assert.False(t, dispatcher.localImage)
	assert.NotNil(t, dispatcher.client)
}

func TestDispatchJob(t *testing.T) {
	// Test cases
	tests := []struct {
		containerCreateRetErr error
		containerStartRetErr  error
		name                  string
		jobID                 string
		bindPath              string
		username              string
		password              string
		expectTaskID          string
		expectErrorMsg        string
		expectAuthStr         string
		retOutput             dockercontainer.CreateResponse
		localImage            bool
	}{
		{
			name:       "local image with bind path",
			jobID:      "job1",
			localImage: true,
			bindPath:   "/test",
			retOutput: dockercontainer.CreateResponse{
				ID: "123",
			},
			expectTaskID: "123",
		},
		{
			name:       "remote image no auth",
			jobID:      "job1",
			localImage: false,
			retOutput: dockercontainer.CreateResponse{
				ID: "123",
			},
			expectTaskID: "123",
		},
		{
			name:       "remote image with auth",
			jobID:      "job1",
			localImage: false,
			username:   "admin",
			password:   "secret",
			retOutput: dockercontainer.CreateResponse{
				ID: "123",
			},
			expectTaskID:  "123",
			expectAuthStr: "eyJ1c2VybmFtZSI6ImFkbWluIiwicGFzc3dvcmQiOiJzZWNyZXQifQ==",
		},
		{
			name:                  "container create error",
			jobID:                 "job1",
			containerCreateRetErr: fmt.Errorf("Failed to build container"),
			expectErrorMsg:        "Failed to build container",
		},
		{
			name:  "container start error",
			jobID: "job1",
			retOutput: dockercontainer.CreateResponse{
				ID: "123",
			},
			containerStartRetErr: fmt.Errorf("Failed to start container"),
			expectErrorMsg:       "Failed to start container",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			apiURL := "https://test"
			discoveryProtocolHost := "test.com"
			token := "token1"
			image := "testimage"

			client := mockClient{}
			client.Test(t)

			if !test.localImage {
				client.On("ImagePull", ctx, image, dockerimage.PullOptions{
					RegistryAuth: test.expectAuthStr,
				}).Return(io.NopCloser(strings.NewReader("")), nil)
			}

			hostConfig := &dockercontainer.HostConfig{}

			if test.bindPath != "" {
				hostConfig.Binds = []string{test.bindPath}
			}

			client.On("ContainerCreate", ctx, &dockercontainer.Config{
				Image: image,
				Env: []string{
					fmt.Sprintf("ENDPOINT=%s", apiURL),
					fmt.Sprintf("JOB_ID=%s", test.jobID),
					fmt.Sprintf("JOB_TOKEN=%s", token),
					fmt.Sprintf("DISCOVERY_PROTOCOL_HOSTS=%s", discoveryProtocolHost),
				},
			}, hostConfig, mock.Anything, mock.Anything, "").Return(test.retOutput, test.containerCreateRetErr)

			client.On("ContainerStart", ctx, test.retOutput.ID, dockercontainer.StartOptions{}).Return(test.containerStartRetErr)

			dispatcher := JobDispatcher{
				logger:                 logger.New(),
				image:                  image,
				bindPath:               test.bindPath,
				localImage:             test.localImage,
				registryUsername:       test.username,
				registryPassword:       test.password,
				apiEndpoint:            apiURL,
				discoveryProtocolHosts: []string{discoveryProtocolHost},
				client:                 &client,
			}

			dispatcherData, err := dispatcher.DispatchJob(ctx, test.jobID, token)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else {
				assert.Nil(t, err, "Unexpected error occurred %v", err)
				assert.Equal(t, test.expectTaskID, dispatcherData[containerIDKey])
			}
		})
	}
}

func TestDispatchJob_memoryLimit(t *testing.T) {
	ctx := t.Context()

	apiURL := "https://test"
	discoveryProtocolHost := "test.com"
	token := "token1"
	image := "testimage"

	client := mockClient{}
	client.Test(t)

	client.On("ContainerCreate", ctx, mock.MatchedBy(func(cfg *dockercontainer.Config) bool {
		for _, e := range cfg.Env {
			if e == "MEMORY_LIMIT=268435456" {
				return true
			}
		}

		return false
	}), mock.MatchedBy(func(hc *dockercontainer.HostConfig) bool {
		return hc.Memory == 268435456 && hc.MemorySwap == 268435456
	}), mock.Anything, mock.Anything, "").Return(dockercontainer.CreateResponse{ID: "123"}, nil)

	client.On("ContainerStart", ctx, "123", dockercontainer.StartOptions{}).Return(nil)

	limits, err := types.LoadResourceLimits(map[string]string{"memory_limit": "256Mi"})
	assert.NoError(t, err)

	dispatcher := JobDispatcher{
		logger:                 logger.New(),
		image:                  image,
		localImage:             true,
		apiEndpoint:            apiURL,
		discoveryProtocolHosts: []string{discoveryProtocolHost},
		limits:                 limits,
		client:                 &client,
	}

	dispatcherData, err := dispatcher.DispatchJob(ctx, "job1", token)
	assert.NoError(t, err)
	assert.Equal(t, "123", dispatcherData[containerIDKey])
}

func TestJobDispatcher_CleanupJob(t *testing.T) {
	const containerID = "container-123"

	tests := []struct {
		name       string
		data       map[string]string
		setupMocks func(*mockClient)
		wantErr    bool
	}{
		{
			name: "removes the container by resource name",
			data: map[string]string{containerIDKey: containerID},
			setupMocks: func(c *mockClient) {
				c.On("ContainerRemove", mock.Anything, containerID, dockercontainer.RemoveOptions{Force: true}).Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "not found is treated as already cleaned up",
			data: map[string]string{containerIDKey: containerID},
			setupMocks: func(c *mockClient) {
				c.On("ContainerRemove", mock.Anything, containerID, dockercontainer.RemoveOptions{Force: true}).
					Return(fmt.Errorf("no such container: %w", cerrdefs.ErrNotFound)).Once()
			},
			wantErr: false,
		},
		{
			name: "remove error is returned",
			data: map[string]string{containerIDKey: containerID},
			setupMocks: func(c *mockClient) {
				c.On("ContainerRemove", mock.Anything, containerID, dockercontainer.RemoveOptions{Force: true}).Return(fmt.Errorf("boom")).Once()
			},
			wantErr: true,
		},
		{
			name:       "missing container ID skips the remove call",
			data:       map[string]string{},
			setupMocks: func(*mockClient) {},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newMockClient(t)
			tt.setupMocks(c)

			j := &JobDispatcher{client: c, logger: logger.New()}

			err := j.CleanupJob(t.Context(), "job-gid", tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
