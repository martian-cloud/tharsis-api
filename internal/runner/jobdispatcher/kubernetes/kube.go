// Package kubernetes package
package kubernetes

//go:generate go tool mockery --name client --inpackage --case underscore

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/smithy-go/ptr"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer"
	ekscfg "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer/eks"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var (
	pluginDataRequiredFields        = []string{"api_url", "auth_type", "image", "memory_request", "memory_limit"}
	requireEKSIAMAuthFields         = []string{"region", "eks_cluster"}
	_                        client = (*k8sRunner)(nil)
)

type client interface {
	CreateJob(context.Context, *v1.Job) (*v1.Job, error)
}

type k8sRunner struct {
	logger     logger.Logger
	configurer configurer.Configurer
	namespace  string
}

// CreateJob get a kubernetes config, sets up the client and creates the batch job.
func (k *k8sRunner) CreateJob(ctx context.Context, job *v1.Job) (*v1.Job, error) {
	config, err := k.configurer.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return cs.BatchV1().Jobs(k.namespace).Create(ctx, job, metav1.CreateOptions{})
}

// JobDispatcher uses a kubernetes client to dispatch jobs
type JobDispatcher struct {
	logger                 logger.Logger
	client                 client
	image                  string
	apiURL                 string
	discoveryProtocolHosts []string
	memoryRequest          resource.Quantity
	memoryLimit            resource.Quantity
	securityContext        *corev1.SecurityContext
}

// New creates a JobDispatcher
func New(ctx context.Context, pluginData map[string]string, discoveryProtocolHost string, logger logger.Logger) (*JobDispatcher, error) {
	for _, field := range pluginDataRequiredFields {
		if _, ok := pluginData[field]; !ok {
			return nil, fmt.Errorf("kubernetes job dispatcher requires plugin data '%s' field", field)
		}
	}

	var (
		c   configurer.Configurer
		err error
	)
	switch pluginData["auth_type"] {
	case "eks_iam":
		for _, field := range requireEKSIAMAuthFields {
			if _, ok := pluginData[field]; !ok {
				return nil, fmt.Errorf("auth_type 'eks_iam' requires plugin data '%s' field", field)
			}
		}

		c, err = ekscfg.New(ctx, pluginData["region"], pluginData["eks_cluster"])
		if err != nil {
			return nil, fmt.Errorf("failed to configure EKS IAM plugin: %v", err)
		}
	default:
		return nil, fmt.Errorf("kubernetes job dispatcher doesn't support auth_type '%s'", pluginData["auth_type"])
	}

	namespace := "default"
	if ns, ok := pluginData["namespace"]; ok {
		namespace = ns
	}

	var runAsUser *int64
	if runAsUserStr, ok := pluginData["security_context_run_as_user"]; ok {
		val, err := strconv.ParseInt(runAsUserStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse security_context_run_as_user for runner jobs: %v", err)
		}
		runAsUser = &val
	}

	var runAsGroup *int64
	if runAsGroupStr, ok := pluginData["security_context_run_as_group"]; ok {
		val, err := strconv.ParseInt(runAsGroupStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse security_context_run_as_group for runner jobs: %v", err)
		}
		runAsGroup = &val
	}

	var runAsNonRoot *bool
	if runAsNonRootStr, ok := pluginData["security_context_run_as_non_root"]; ok {
		val, err := strconv.ParseBool(runAsNonRootStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse security_context_run_as_non_root for runner jobs: %v", err)
		}
		runAsNonRoot = &val
	}

	memoryRequest, err := resource.ParseQuantity(pluginData["memory_request"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory request for runner jobs: %v", err)
	}

	memoryLimit, err := resource.ParseQuantity(pluginData["memory_limit"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory limit for runner jobs: %v", err)
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

	return &JobDispatcher{
		logger:                 logger,
		image:                  pluginData["image"],
		apiURL:                 pluginData["api_url"],
		discoveryProtocolHosts: discoveryProtocolHosts,
		memoryRequest:          memoryRequest,
		memoryLimit:            memoryLimit,
		securityContext: &corev1.SecurityContext{
			Privileged:               ptr.Bool(false),
			AllowPrivilegeEscalation: ptr.Bool(false),
			RunAsUser:                runAsUser,
			RunAsGroup:               runAsGroup,
			RunAsNonRoot:             runAsNonRoot,
			// TODO: Add host users option when user namespace feature is generally available
			//HostUsers:                    ptr.Bool(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"NET_RAW"},
			},
		},
		client: &k8sRunner{
			logger:     logger,
			namespace:  namespace,
			configurer: c,
		},
	}, nil
}

// DispatchJob will start a kubernetes batch job to execute the job
func (j *JobDispatcher) DispatchJob(ctx context.Context, jobID string, token string) (string, error) {
	// Disable retries
	backoffLimit := int32(0)
	// Remove once completed
	ttlSecondsAfterFinished := int32(0)

	k8sJob := &v1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "tharsis-job-" + strings.ToLower(jobID[:8]),
		},
		Spec: v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"cluster-autoscaler.kubernetes.io/safe-to-evict": "false",
					},
					Annotations: map[string]string{
						"job.tharsis.io/id": jobID,
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: ptr.Bool(false),
					Containers: []corev1.Container{
						{
							Name:            "main",
							Image:           j.image,
							SecurityContext: j.securityContext,
							Env: []corev1.EnvVar{
								{
									Name:  "JOB_ID",
									Value: jobID,
								},
								{
									Name:  "JOB_TOKEN",
									Value: token,
								},
								{
									Name:  "API_URL",
									Value: j.apiURL,
								},
								{
									Name:  "DISCOVERY_PROTOCOL_HOSTS",
									Value: strings.Join(j.discoveryProtocolHosts, ","),
								},
								{
									Name:  "MEMORY_LIMIT",
									Value: j.memoryLimit.String(),
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: j.memoryRequest,
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: j.memoryLimit,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
		},
	}

	result, err := j.client.CreateJob(ctx, k8sJob)
	if err != nil {
		return "", fmt.Errorf("kubernetes job dispatcher failed to run for job %s: %v", jobID, err)
	}

	return string(result.UID), nil
}
