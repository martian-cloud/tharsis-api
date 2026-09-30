// Package ecs package
package ecs

//go:generate go tool mockery --name client --inpackage --case underscore

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	dispatchertypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// taskArnKey is the dispatcher-data key under which the task ARN is stored.
const taskArnKey = "taskArn"

var pluginDataRequiredFields = []string{"endpoint", "region", "task_definition", "cluster", "subnets", "launch_type"}

type client interface {
	RunTask(ctx context.Context, params *ecs.RunTaskInput, optFns ...func(*ecs.Options)) (*ecs.RunTaskOutput, error)
}

// JobDispatcher uses the AWS ECS client to dispatch jobs
type JobDispatcher struct {
	logger                 logger.Logger
	client                 client
	taskDefinition         string
	cluster                string
	launchType             types.LaunchType
	apiEndpoint            string
	discoveryProtocolHosts []string
	limits                 *dispatchertypes.ResourceLimits
	subnets                []string
}

// New creates a JobDispatcher
func New(ctx context.Context, pluginData map[string]string, discoveryProtocolHost string, logger logger.Logger) (*JobDispatcher, error) {
	if err := dispatchertypes.MigrateDeprecatedPluginDataFields(pluginData, logger); err != nil {
		return nil, err
	}

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

	discoveryProtocolHosts := []string{}

	if discoveryProtocolHost != "" {
		discoveryProtocolHosts = append(discoveryProtocolHosts, discoveryProtocolHost)
	}

	if extraDiscoveryHostsStr, ok := pluginData["extra_service_discovery_hosts"]; ok {
		for _, host := range strings.Split(extraDiscoveryHostsStr, ",") {
			discoveryProtocolHosts = append(discoveryProtocolHosts, strings.TrimSpace(host))
		}
	}

	limits, err := dispatchertypes.LoadResourceLimits(pluginData)
	if err != nil {
		return nil, err
	}

	client := ecs.NewFromConfig(awsCfg)

	return &JobDispatcher{
		logger:                 logger,
		taskDefinition:         pluginData["task_definition"],
		cluster:                pluginData["cluster"],
		launchType:             launchType,
		subnets:                strings.Split(pluginData["subnets"], ","),
		apiEndpoint:            pluginData["endpoint"],
		discoveryProtocolHosts: discoveryProtocolHosts,
		limits:                 limits,
		client:                 client,
	}, nil
}

// DispatchJob will start an ECS task to execute the job
func (j *JobDispatcher) DispatchJob(ctx context.Context, jobID string, token string) (map[string]string, error) {
	environment := []types.KeyValuePair{
		{Name: new("JOB_ID"), Value: &jobID},
		{Name: new("JOB_TOKEN"), Value: &token},
		{Name: new("ENDPOINT"), Value: &j.apiEndpoint},
		{Name: new("DISCOVERY_PROTOCOL_HOSTS"), Value: new(strings.Join(j.discoveryProtocolHosts, ","))},
	}

	limitEnv := j.limits.AsEnvVars()
	for _, name := range slices.Sorted(maps.Keys(limitEnv)) {
		environment = append(environment, types.KeyValuePair{Name: new(name), Value: new(limitEnv[name])})
	}

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
					Name:        new("main"),
					Environment: environment,
				},
			},
		},
	}
	output, err := j.client.RunTask(ctx, &input)
	if err != nil {
		return nil, fmt.Errorf("ECS Job Dispatcher failed to run task for job %s: %v", jobID, err)
	}

	if len(output.Failures) > 0 {
		errors := []string{}
		if output.Failures[0].Reason != nil {
			errors = append(errors, *output.Failures[0].Reason)
		}
		if output.Failures[0].Detail != nil {
			errors = append(errors, *output.Failures[0].Detail)
		}
		return nil, fmt.Errorf("failed to run task: %s", strings.Join(errors, "; "))
	}

	if len(output.Tasks) == 0 {
		return nil, fmt.Errorf("no ECS tasks were created")
	}

	return map[string]string{taskArnKey: *output.Tasks[0].TaskArn}, nil
}

// CleanupJob is a no-op; ECS tasks stop and are reclaimed by ECS on their own.
func (*JobDispatcher) CleanupJob(_ context.Context, _ string, _ map[string]string) error {
	return nil
}

// Limits returns the resource limits jobs run under.
func (j *JobDispatcher) Limits() *dispatchertypes.ResourceLimits {
	return j.limits
}
