package kubernetes

//go:generate go tool mockery --name client --inpackage --case underscore

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer"
)

var _ client = (*k8sRunner)(nil)

type client interface {
	CreatePod(context.Context, *corev1.Pod) (*corev1.Pod, error)
	DeletePod(ctx context.Context, name string) error
}

type k8sRunner struct {
	configurer configurer.Configurer
	namespace  string
}

// CreatePod creates the job pod in the configured namespace.
func (k *k8sRunner) CreatePod(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	cs, err := k.clientset(ctx)
	if err != nil {
		return nil, err
	}

	return cs.CoreV1().Pods(k.namespace).Create(ctx, pod, metav1.CreateOptions{})
}

// DeletePod deletes a pod by name in the configured namespace.
func (k *k8sRunner) DeletePod(ctx context.Context, name string) error {
	cs, err := k.clientset(ctx)
	if err != nil {
		return err
	}

	return cs.CoreV1().Pods(k.namespace).Delete(ctx, name, metav1.DeleteOptions{
		// Skip the pod's hour-long grace period; a container still running at this point was force canceled.
		GracePeriodSeconds: new(int64(0)),
	})
}

// clientset builds a Kubernetes clientset from the configurer. The config can change over
// time (e.g. rotating EKS tokens), so it is fetched on each call rather than cached.
func (k *k8sRunner) clientset(ctx context.Context) (*kubernetes.Clientset, error) {
	config, err := k.configurer.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}
