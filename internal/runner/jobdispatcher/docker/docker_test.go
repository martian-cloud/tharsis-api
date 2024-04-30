package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestNew(t *testing.T) {
	pluginData := map[string]string{
		"api_url": "testUrl",
		"host":    "http://localhost",
		"image":   "testImage",
	}
	dispatcher, err := New(pluginData, "http://localhost", logger.New())
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	assert.Equal(t, "testUrl", dispatcher.apiURL)
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			apiURL := "https://test"
			discoveryProtocolHost := "test.com"
			token := "token1"
			image := "testimage"
			memoryLimit := uint64(0)

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
					fmt.Sprintf("API_URL=%s", apiURL),
					fmt.Sprintf("JOB_ID=%s", test.jobID),
					fmt.Sprintf("JOB_TOKEN=%s", token),
					fmt.Sprintf("DISCOVERY_PROTOCOL_HOST=%s", discoveryProtocolHost),
					fmt.Sprintf("MEMORY_LIMIT=%d", memoryLimit),
				},
			}, hostConfig, mock.Anything, mock.Anything, "").Return(test.retOutput, test.containerCreateRetErr)

			client.On("ContainerStart", ctx, test.retOutput.ID, dockercontainer.StartOptions{}).Return(test.containerStartRetErr)

			dispatcher := JobDispatcher{
				logger:                logger.New(),
				image:                 image,
				bindPath:              test.bindPath,
				localImage:            test.localImage,
				registryUsername:      test.username,
				registryPassword:      test.password,
				apiURL:                apiURL,
				discoveryProtocolHost: discoveryProtocolHost,
				client:                &client,
			}

			taskID, err := dispatcher.DispatchJob(ctx, test.jobID, token)
			if test.expectErrorMsg != "" {
				assert.EqualError(t, err, test.expectErrorMsg)
			} else {
				assert.Nil(t, err, "Unexpected error occurred %v", err)
			}

			assert.Equal(t, test.expectTaskID, taskID)
		})
	}
}
