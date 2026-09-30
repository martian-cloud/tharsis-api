package kubernetes

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestJobDispatcher_DispatchJob(t *testing.T) {
	jobID := testJobGID(t)
	podName := "tharsis-job-" + testJobUUID

	tests := []struct {
		name       string
		setupMocks func(*mockClient)
		jobID      string
		token      string
		want       string
		wantErr    bool
	}{
		{
			name: "failed to create pod",
			setupMocks: func(c *mockClient) {
				c.On("CreatePod", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("failed to launch job")).Once()
			},
			jobID:   jobID,
			token:   "myToken",
			want:    "",
			wantErr: true,
		},
		{
			name:       "invalid job ID fails before create",
			setupMocks: func(_ *mockClient) {},
			jobID:      "not-a-global-id",
			token:      "myToken",
			want:       "",
			wantErr:    true,
		},
		{
			name: "create pod succeeds returns pod name",
			setupMocks: func(c *mockClient) {
				c.On("CreatePod", mock.Anything, mock.MatchedBy(func(pod *corev1.Pod) bool {
					return pod.Labels[ownerLabelKey] == ownerLabelValue &&
						pod.Labels[jobIDLabelKey] == jobID &&
						pod.Name == podName &&
						pod.Spec.RestartPolicy == corev1.RestartPolicyNever &&
						pod.Spec.EnableServiceLinks != nil && !*pod.Spec.EnableServiceLinks
				})).Return(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: podName},
				}, nil).Once()
			},
			jobID:   jobID,
			token:   "myToken",
			want:    podName,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newMockClient(t)
			tt.setupMocks(c)

			j := &JobDispatcher{
				config: &config{
					image:       "hello-world",
					apiEndpoint: "http://localhost",
					limits:      &types.ResourceLimits{},
				},
				client: c,
			}

			got, err := j.DispatchJob(t.Context(), tt.jobID, tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("JobDispatcher.DispatchJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got[podNameKey])
		})
	}
}

func Test_New(t *testing.T) {
	tempDir := t.TempDir()
	kubeConfigPath := filepath.Join(tempDir, "kubeconfig")
	require.NoError(t, os.WriteFile(kubeConfigPath, []byte("test kubeconfig content"), 0644))

	tokenGetter := func(_ context.Context) (string, error) {
		return "test-token", nil
	}

	withData := func(extra map[string]string) map[string]string {
		data := baseInClusterData()
		for k, v := range extra {
			data[k] = v
		}

		return data
	}

	tests := []struct {
		pluginData          map[string]string
		name                string
		expectedErrContains string
		wantErr             bool
	}{
		{
			name:                "unsupported auth type",
			pluginData:          withData(map[string]string{"auth_type": "unsupported_auth"}),
			wantErr:             true,
			expectedErrContains: "kubernetes job dispatcher doesn't support auth_type",
		},
		{
			name:                "EKS IAM auth type missing required fields",
			pluginData:          withData(map[string]string{"auth_type": AuthTypeEKSIAM}),
			wantErr:             true,
			expectedErrContains: "kubernetes job dispatcher requires plugin data",
		},
		{
			name: "KubeConfig auth type with valid config",
			pluginData: withData(map[string]string{
				"auth_type":        AuthTypeKubeConfig,
				"kube_config_path": kubeConfigPath,
			}),
			wantErr: false,
		},
		{
			name: "KubeConfig auth type with invalid path",
			pluginData: withData(map[string]string{
				"auth_type":        AuthTypeKubeConfig,
				"kube_config_path": "/non/existent/path",
			}),
			wantErr:             true,
			expectedErrContains: "failed to configure kube job dispatcher plugin",
		},
		{
			name: "X509Cert auth type with valid data",
			pluginData: withData(map[string]string{
				"auth_type":   AuthTypeX509Cert,
				"kube_server": "https://kubernetes.default.svc",
				"client_cert": base64.StdEncoding.EncodeToString([]byte("test-cert")),
				"client_key":  base64.StdEncoding.EncodeToString([]byte("test-key")),
			}),
			wantErr: false,
		},
		{
			name: "RunnerIDToken auth type with valid data",
			pluginData: withData(map[string]string{
				"auth_type":   AuthTypeRunnerIDToken,
				"kube_server": "https://kubernetes.default.svc",
			}),
			wantErr: false,
		},
		{
			name:       "InCluster auth type",
			pluginData: baseInClusterData(),
			wantErr:    false,
		},
		{
			name:                "empty memory_limit is rejected",
			pluginData:          withData(map[string]string{"memory_limit": ""}),
			wantErr:             true,
			expectedErrContains: "memory_limit must be a non-zero value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogger, _ := logger.NewForTest()
			dispatcher, err := New(t.Context(), tt.pluginData, "discovery.example.com", types.TokenGetterFunc(tokenGetter), testLogger)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErrContains != "" {
					assert.Contains(t, err.Error(), tt.expectedErrContains)
				}

				assert.Nil(t, dispatcher)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, dispatcher)
			assert.Equal(t, tt.pluginData["endpoint"], dispatcher.config.apiEndpoint)
			assert.Equal(t, tt.pluginData["image"], dispatcher.config.image)
			assert.Contains(t, dispatcher.config.discoveryProtocolHosts, "discovery.example.com")
		})
	}
}

func TestJobDispatcher_CleanupJob(t *testing.T) {
	const podName = "tharsis-job-abc"

	tests := []struct {
		name       string
		data       map[string]string
		setupMocks func(*mockClient)
		wantErr    bool
	}{
		{
			name: "deletes the pod by resource name",
			data: map[string]string{podNameKey: podName},
			setupMocks: func(c *mockClient) {
				c.On("DeletePod", mock.Anything, podName).Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "not found is treated as already cleaned up",
			data: map[string]string{podNameKey: podName},
			setupMocks: func(c *mockClient) {
				c.On("DeletePod", mock.Anything, podName).Return(apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, podName)).Once()
			},
			wantErr: false,
		},
		{
			name: "delete error is returned",
			data: map[string]string{podNameKey: podName},
			setupMocks: func(c *mockClient) {
				c.On("DeletePod", mock.Anything, podName).Return(fmt.Errorf("boom")).Once()
			},
			wantErr: true,
		},
		{
			name:       "missing pod name skips the delete call",
			data:       map[string]string{},
			setupMocks: func(*mockClient) {},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newMockClient(t)
			tt.setupMocks(c)

			j := &JobDispatcher{config: &config{}, client: c, logger: logger.New()}

			err := j.CleanupJob(t.Context(), "job-gid", tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
