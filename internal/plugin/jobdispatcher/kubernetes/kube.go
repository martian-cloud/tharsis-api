package kubernetes

//go:generate mockery --name client --inpackage --case underscore

import (
	"context"
	"fmt"

	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/jobdispatcher/kubernetes/configurer"
	ekscfg "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/jobdispatcher/kubernetes/configurer/eks"
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
	logger                logger.Logger
	client                client
	image                 string
	apiURL                string
	discoveryProtocolHost string
	memoryRequest         resource.Quantity
	memoryLimit           resource.Quantity
}

// New creates a JobDispatcher
func New(ctx context.Context, pluginData map[string]string, logger logger.Logger) (*JobDispatcher, error) {
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

	var namespace = "default"
	if ns, ok := pluginData["namespace"]; ok {
		namespace = ns
	}

	memoryRequest, err := resource.ParseQuantity(pluginData["memory_request"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory request for runner jobs: %v", err)
	}

	memoryLimit, err := resource.ParseQuantity(pluginData["memory_limit"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory limit for runner jobs: %v", err)
	}

	return &JobDispatcher{
		logger:                logger,
		image:                 pluginData["image"],
		apiURL:                pluginData["api_url"],
		discoveryProtocolHost: pluginData["discovery_protocol_host"],
		memoryRequest:         memoryRequest,
		memoryLimit:           memoryLimit,
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
			GenerateName: "tharsis-job-executor",
		},
		Spec: v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"cluster-autoscaler.kubernetes.io/safe-to-evict": "false",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "main",
							Image: j.image,
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
									Name:  "DISCOVERY_PROTOCOL_HOST",
									Value: j.discoveryProtocolHost,
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
