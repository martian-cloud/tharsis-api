// Package ecs package
package ecs

//go:generate mockery --name client --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/smithy-go/ptr"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
)

var pluginDataRequiredFields = []string{"api_url", "region", "task_definition", "cluster", "subnets", "launch_type"}

type client interface {
	RunTask(ctx context.Context, params *ecs.RunTaskInput, optFns ...func(*ecs.Options)) (*ecs.RunTaskOutput, error)
}

// JobDispatcher uses the AWS ECS client to dispatch jobs
type JobDispatcher struct {
	logger                logger.Logger
	client                client
	taskDefinition        string
	cluster               string
	launchType            types.LaunchType
	apiURL                string
	discoveryProtocolHost string
	subnets               []string
}

// New creates a JobDispatcher
func New(ctx context.Context, pluginData map[string]string, discoveryProtocolHost string, logger logger.Logger) (*JobDispatcher, error) {
	for _, field := range pluginDataRequiredFields {
		if _, ok := pluginData[field]; !ok {
			return nil, fmt.Errorf("ECS job dispatcher requires plugin data '%s' field", field)
		}
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(pluginData["region"]))
	if err != nil {
		return nil, err
	}

	var launchType types.LaunchType
	switch pluginData["launch_type"] {
	case "ec2":
		launchType = types.LaunchTypeEc2
	case "fargate":
		launchType = types.LaunchTypeFargate
	default:
		return nil, fmt.Errorf("ECS job dispatcher requires a launch type of ec2 or fargate")
	}

	client := ecs.NewFromConfig(awsCfg)

	return &JobDispatcher{
		logger:                logger,
		taskDefinition:        pluginData["task_definition"],
		cluster:               pluginData["cluster"],
		launchType:            launchType,
		subnets:               strings.Split(pluginData["subnets"], ","),
		apiURL:                pluginData["api_url"],
		discoveryProtocolHost: discoveryProtocolHost,
		client:                client,
	}, nil
}

// DispatchJob will start an ECS task to execute the job
func (j *JobDispatcher) DispatchJob(ctx context.Context, jobID string, token string) (string, error) {
	input := ecs.RunTaskInput{
		TaskDefinition: &j.taskDefinition,
		LaunchType:     j.launchType,
		Cluster:        &j.cluster,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				AssignPublicIp: types.AssignPublicIpDisabled,
				Subnets:        j.subnets,
			},
		},
		Overrides: &types.TaskOverride{
			ContainerOverrides: []types.ContainerOverride{
				{
					Name: ptr.String("main"),
					Environment: []types.KeyValuePair{
						{Name: ptr.String("JOB_ID"), Value: &jobID},
						{Name: ptr.String("JOB_TOKEN"), Value: &token},
						{Name: ptr.String("API_URL"), Value: &j.apiURL},
						{Name: ptr.String("DISCOVERY_PROTOCOL_HOST"), Value: &j.discoveryProtocolHost},
					},
				},
			},
		},
	}
	output, err := j.client.RunTask(ctx, &input)
	if err != nil {
		return "", fmt.Errorf("ECS Job Dispatcher failed to run task for job %s: %v", jobID, err)
	}

	if len(output.Failures) > 0 {
		errors := []string{}
		if output.Failures[0].Reason != nil {
			errors = append(errors, *output.Failures[0].Reason)
		}
		if output.Failures[0].Detail != nil {
			errors = append(errors, *output.Failures[0].Detail)
		}
		return "", fmt.Errorf("failed to run task: %s", strings.Join(errors, "; "))
	}

	if len(output.Tasks) == 0 {
		return "", fmt.Errorf("no ECS tasks were created")
	}

	return *output.Tasks[0].TaskArn, nil
}
