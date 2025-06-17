package kubernetes

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_k8sRunner_CreateJob(t *testing.T) {
	type args struct {
		ctx context.Context
		job *v1.Job
	}
	tests := []struct {
		args    args
		k       *k8sRunner
		want    *v1.Job
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.CreateJob(tt.args.ctx, tt.args.job)
			if (err != nil) != tt.wantErr {
				t.Errorf("k8sRunner.CreateJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("k8sRunner.CreateJob() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobDispatcher_DispatchJob(t *testing.T) {
	type args struct {
		ctx   context.Context
		job   *models.Job
		token string
	}
	tests := []struct {
		name    string
		j       *JobDispatcher
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "failed to create job",
			j: &JobDispatcher{
				logger: nil,
				image:  "hello-world",
				apiURL: "http://localhost",
				client: func() client {
					client := &mockClient{}

					client.On("CreateJob", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("failed to launch job")).Once()
					return client
				}(),
			},
			args: args{
				ctx: context.TODO(),
				job: &models.Job{
					Metadata: models.ResourceMetadata{
						ID: "test-job-123",
					},
				},
				token: "myToken",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "create job succeeds",
			j: &JobDispatcher{
				logger: nil,
				image:  "hello-world",
				apiURL: "http://localhost",
				client: func() client {
					client := &mockClient{}

					client.On("CreateJob", mock.Anything, mock.Anything).Return(&v1.Job{
						ObjectMeta: metav1.ObjectMeta{
							UID: "id",
						},
					}, nil).Once()
					return client
				}(),
			},
			args: args{
				ctx: context.TODO(),
				job: &models.Job{
					Metadata: models.ResourceMetadata{
						ID: "test-job-123",
					},
				},
				token: "myToken",
			},
			want:    "id",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.j.DispatchJob(tt.args.ctx, tt.args.job.Metadata.ID, tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("JobDispatcher.DispatchJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("JobDispatcher.DispatchJob() = %v, want %v", got, tt.want)
			}
		})
	}
}
func Test_New(t *testing.T) {
	// Create a temporary kubeconfig file for testing
	tempDir := t.TempDir()
	kubeConfigPath := filepath.Join(tempDir, "kubeconfig")
	err := os.WriteFile(kubeConfigPath, []byte("test kubeconfig content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test kubeconfig file: %v", err)
	}

	// Create a mock token getter function
	tokenGetter := func(_ context.Context) (string, error) {
		return "test-token", nil
	}

	tests := []struct {
		name                string
		pluginData          map[string]string
		discoveryHost       string
		tokenGetter         types.TokenGetterFunc
		wantErr             bool
		expectedErrContains string
	}{
		{
			name: "Missing required field",
			pluginData: map[string]string{
				"api_url": "https://api.example.com",
				// Missing other required fields
			},
			discoveryHost:       "discovery.example.com",
			tokenGetter:         tokenGetter,
			wantErr:             true,
			expectedErrContains: "kubernetes job dispatcher requires plugin data",
		},
		{
			name: "Unsupported auth type",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":        "https://api.example.com",
					"image":          "test-image:latest",
					"memory_request": "128Mi",
					"memory_limit":   "256Mi",
					"auth_type":      "unsupported_auth",
				}
				return data
			}(),
			discoveryHost:       "discovery.example.com",
			tokenGetter:         tokenGetter,
			wantErr:             true,
			expectedErrContains: "kubernetes job dispatcher doesn't support auth_type",
		},
		{
			name: "EKS IAM auth type missing required fields",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":        "https://api.example.com",
					"image":          "test-image:latest",
					"memory_request": "128Mi",
					"memory_limit":   "256Mi",
					"auth_type":      AuthTypeEKSIAM,
				}
				return data
			}(),
			discoveryHost:       "discovery.example.com",
			tokenGetter:         tokenGetter,
			wantErr:             true,
			expectedErrContains: "kubernetes job dispatcher requires plugin data",
		},
		{
			name: "KubeConfig auth type with valid config",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":          "https://api.example.com",
					"image":            "test-image:latest",
					"memory_request":   "128Mi",
					"memory_limit":     "256Mi",
					"auth_type":        AuthTypeKubeConfig,
					"kube_config_path": kubeConfigPath,
				}
				return data
			}(),
			discoveryHost: "discovery.example.com",
			tokenGetter:   tokenGetter,
			wantErr:       false,
		},
		{
			name: "KubeConfig auth type with invalid path",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":          "https://api.example.com",
					"image":            "test-image:latest",
					"memory_request":   "128Mi",
					"memory_limit":     "256Mi",
					"auth_type":        AuthTypeKubeConfig,
					"kube_config_path": "/non/existent/path",
				}
				return data
			}(),
			discoveryHost:       "discovery.example.com",
			tokenGetter:         tokenGetter,
			wantErr:             true,
			expectedErrContains: "failed to configure kube job dispatcher plugin",
		},
		{
			name: "X509Cert auth type missing required fields",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":        "https://api.example.com",
					"image":          "test-image:latest",
					"memory_request": "128Mi",
					"memory_limit":   "256Mi",
					"auth_type":      AuthTypeX509Cert,
				}
				return data
			}(),
			discoveryHost:       "discovery.example.com",
			tokenGetter:         tokenGetter,
			wantErr:             true,
			expectedErrContains: "kubernetes job dispatcher requires plugin data",
		},
		{
			name: "X509Cert auth type with invalid cert data",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":        "https://api.example.com",
					"image":          "test-image:latest",
					"memory_request": "128Mi",
					"memory_limit":   "256Mi",
					"auth_type":      AuthTypeX509Cert,
					"kube_server":    "https://kubernetes.default.svc",
					"client_cert":    "invalid-base64",
					"client_key":     "valid-key",
				}
				return data
			}(),
			discoveryHost:       "discovery.example.com",
			tokenGetter:         tokenGetter,
			wantErr:             true,
			expectedErrContains: "failed to configure kube job dispatcher plugin",
		},
		{
			name: "X509Cert auth type with valid data",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":        "https://api.example.com",
					"image":          "test-image:latest",
					"memory_request": "128Mi",
					"memory_limit":   "256Mi",
					"auth_type":      AuthTypeX509Cert,
					"kube_server":    "https://kubernetes.default.svc",
					"client_cert":    base64.StdEncoding.EncodeToString([]byte("test-cert")),
					"client_key":     base64.StdEncoding.EncodeToString([]byte("test-key")),
				}
				return data
			}(),
			discoveryHost: "discovery.example.com",
			tokenGetter:   tokenGetter,
			wantErr:       false,
		},
		{
			name: "RunnerIDToken auth type missing required fields",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":        "https://api.example.com",
					"image":          "test-image:latest",
					"memory_request": "128Mi",
					"memory_limit":   "256Mi",
					"auth_type":      AuthTypeRunnerIDToken,
				}
				return data
			}(),
			discoveryHost:       "discovery.example.com",
			tokenGetter:         tokenGetter,
			wantErr:             true,
			expectedErrContains: "kubernetes job dispatcher requires plugin data",
		},
		{
			name: "RunnerIDToken auth type with valid data",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":        "https://api.example.com",
					"image":          "test-image:latest",
					"memory_request": "128Mi",
					"memory_limit":   "256Mi",
					"auth_type":      AuthTypeRunnerIDToken,
					"kube_server":    "https://kubernetes.default.svc",
				}
				return data
			}(),
			discoveryHost: "discovery.example.com",
			tokenGetter:   tokenGetter,
			wantErr:       false,
		},
		{
			name: "InCluster auth type",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":        "https://api.example.com",
					"image":          "test-image:latest",
					"memory_request": "128Mi",
					"memory_limit":   "256Mi",
					"auth_type":      AuthTypeInCluster,
				}
				return data
			}(),
			discoveryHost: "discovery.example.com",
			tokenGetter:   tokenGetter,
			wantErr:       false,
		},
		{
			name: "With security context settings",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":                          "https://api.example.com",
					"image":                            "test-image:latest",
					"memory_request":                   "128Mi",
					"memory_limit":                     "256Mi",
					"auth_type":                        AuthTypeInCluster,
					"security_context_run_as_user":     "1000",
					"security_context_run_as_group":    "1000",
					"security_context_run_as_non_root": "true",
				}
				return data
			}(),
			discoveryHost: "discovery.example.com",
			tokenGetter:   tokenGetter,
			wantErr:       false,
		},
		{
			name: "With invalid security context settings",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":                      "https://api.example.com",
					"image":                        "test-image:latest",
					"memory_request":               "128Mi",
					"memory_limit":                 "256Mi",
					"auth_type":                    AuthTypeInCluster,
					"security_context_run_as_user": "not-a-number",
				}
				return data
			}(),
			discoveryHost:       "discovery.example.com",
			tokenGetter:         tokenGetter,
			wantErr:             true,
			expectedErrContains: "failed to parse security_context_run_as_user",
		},
		{
			name: "With extra discovery hosts",
			pluginData: func() map[string]string {
				data := map[string]string{
					"api_url":                       "https://api.example.com",
					"image":                         "test-image:latest",
					"memory_request":                "128Mi",
					"memory_limit":                  "256Mi",
					"auth_type":                     AuthTypeInCluster,
					"extra_service_discovery_hosts": "host1.example.com, host2.example.com",
				}
				return data
			}(),
			discoveryHost: "discovery.example.com",
			tokenGetter:   tokenGetter,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogger, _ := logger.NewForTest()
			dispatcher, err := New(context.Background(), tt.pluginData, tt.discoveryHost, tt.tokenGetter, testLogger)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErrContains != "" {
					assert.Contains(t, err.Error(), tt.expectedErrContains)
				}
				assert.Nil(t, dispatcher)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, dispatcher)

				// Verify basic properties of the dispatcher
				assert.Equal(t, tt.pluginData["api_url"], dispatcher.apiURL)
				assert.Equal(t, tt.pluginData["image"], dispatcher.image)

				// Check if discovery hosts are properly set
				if tt.discoveryHost != "" {
					assert.Contains(t, dispatcher.discoveryProtocolHosts, tt.discoveryHost)
				}

				// Check if extra discovery hosts are properly set
				if extraHosts, ok := tt.pluginData["extra_service_discovery_hosts"]; ok {
					for _, host := range []string{"host1.example.com", "host2.example.com"} {
						if extraHosts != "" {
							assert.Contains(t, dispatcher.discoveryProtocolHosts, host)
						}
					}
				}
			}
		})
	}
}
