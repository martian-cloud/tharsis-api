package ecs

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
)

func TestNew(t *testing.T) {
	pluginData := map[string]string{
		"api_url":         "testUrl",
		"region":          "testRegion",
		"task_definition": "testTaskDef",
		"cluster":         "testCluster",
		"subnets":         "test1,test2",
		"launch_type":     "fargate",
	}
	dispatcher, err := New(context.Background(), pluginData, "http://localhost", logger.New())
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	assert.Equal(t, "testUrl", dispatcher.apiURL)
	assert.Equal(t, "testTaskDef", dispatcher.taskDefinition)
	assert.Equal(t, "testCluster", dispatcher.cluster)
	assert.Equal(t, []string{"test1", "test2"}, dispatcher.subnets)
	assert.Equal(t, types.LaunchTypeFargate, dispatcher.launchType)
}

func TestNewInvalidLaunchType(t *testing.T) {
	pluginData := map[string]string{
		"api_url":         "testUrl",
		"region":          "testRegion",
		"task_definition": "testTaskDef",
		"cluster":         "testCluster",
		"subnets":         "test1,test2",
		"launch_type":     "invalid",
	}
	_, err := New(context.Background(), pluginData, "http://localhost", logger.New())
	assert.EqualError(t, err, "ECS job dispatcher requires a launch type of ec2 or fargate")
}

func TestDispatchJob(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		jobID          string
		retOutput      *ecs.RunTaskOutput
		retErr         error
		expectTaskID   string
		expectErrorMsg string
	}{
		{
			name:  "successful task launch",
			jobID: "job1",
			retOutput: &ecs.RunTaskOutput{
				Tasks: []types.Task{{TaskArn: ptr.String("test123")}},
			},
			expectTaskID: "test123",
		},
		{
			name:  "missing task ID",
			jobID: "job1",
			retOutput: &ecs.RunTaskOutput{
				Tasks: []types.Task{},
			},
			expectErrorMsg: "no ECS tasks were created",
		},
		{
			name:  "return failure reason",
			jobID: "job1",
			retOutput: &ecs.RunTaskOutput{
				Failures: []types.Failure{{Reason: ptr.String("service limit reached")}},
			},
			expectErrorMsg: "failed to run task: service limit reached",
		},
		{
			name:  "return failure details and reason",
			jobID: "job1",
			retOutput: &ecs.RunTaskOutput{
				Failures: []types.Failure{{
					Reason: ptr.String("service limit"),
					Detail: ptr.String("limit of 500 tasks reached"),
				}},
			},
			expectErrorMsg: "failed to run task: service limit; limit of 500 tasks reached",
		},
		{
			name:           "return error",
			jobID:          "job1",
			retErr:         fmt.Errorf("Failed to launch task"),
			expectErrorMsg: "ECS Job Dispatcher failed to run task for job job1: Failed to launch task",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			taskDefinition := "taskDef1"
			cluster := "cluster1"
			launchType := types.LaunchTypeFargate
			subnets := []string{"subnet1"}
			apiURL := "https://test"
			discoveryProtocolHost := "test.com"
			token := "token1"

			client := mockClient{}
			client.Test(t)

			client.On("RunTask", ctx, &ecs.RunTaskInput{
				TaskDefinition: &taskDefinition,
				LaunchType:     launchType,
				Cluster:        &cluster,
				NetworkConfiguration: &types.NetworkConfiguration{
					AwsvpcConfiguration: &types.AwsVpcConfiguration{
						AssignPublicIp: types.AssignPublicIpDisabled,
						Subnets:        subnets,
					},
				},
				Overrides: &types.TaskOverride{
					ContainerOverrides: []types.ContainerOverride{
						{
							Name: ptr.String("main"),
							Environment: []types.KeyValuePair{
								{Name: ptr.String("JOB_ID"), Value: &test.jobID},
								{Name: ptr.String("JOB_TOKEN"), Value: &token},
								{Name: ptr.String("API_URL"), Value: &apiURL},
								{Name: ptr.String("DISCOVERY_PROTOCOL_HOST"), Value: &discoveryProtocolHost},
							},
						},
					},
				},
			}).Return(test.retOutput, test.retErr)

			dispatcher := JobDispatcher{
				logger:                logger.New(),
				taskDefinition:        taskDefinition,
				cluster:               cluster,
				launchType:            launchType,
				subnets:               subnets,
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
