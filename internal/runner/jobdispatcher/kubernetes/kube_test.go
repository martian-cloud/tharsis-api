package kubernetes

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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

func TestNew(t *testing.T) {
	type args struct {
		ctx        context.Context
		pluginData map[string]string
		logger     logger.Logger
	}
	tests := []struct {
		args    args
		want    func(*testing.T, *JobDispatcher)
		name    string
		wantErr bool
	}{
		{
			name: "Missing Require Plugin Data fails",
			args: args{
				ctx:        context.TODO(),
				pluginData: map[string]string{},
				logger:     nil,
			},
			want: func(t *testing.T, jd *JobDispatcher) {
				if jd != nil {
					t.Errorf("New() = %v, want %v", jd, nil)
				}
			},
			wantErr: true,
		},
		{
			name: "Invalid auth_type fails",
			args: args{
				ctx: context.TODO(),
				pluginData: map[string]string{
					"api_url":        "http://localhost",
					"auth_type":      "service-account",
					"image":          "hello-world",
					"memory_request": "256Mi",
					"memory_limit":   "512Mi",
				},
				logger: nil,
			},
			want: func(t *testing.T, jd *JobDispatcher) {
				if jd != nil {
					t.Errorf("New() = %v, want %v", jd, nil)
				}
			},
			wantErr: true,
		},
		{
			name: "Missing Require Plugin Data for eks_iam auth_type fails",
			args: args{
				ctx: context.TODO(),
				pluginData: map[string]string{
					"api_url":        "http://localhost",
					"auth_type":      "eks_iam",
					"image":          "hello-world",
					"memory_request": "256Mi",
					"memory_limit":   "512Mi",
				},
				logger: nil,
			},
			want: func(t *testing.T, jd *JobDispatcher) {
				if jd != nil {
					t.Errorf("New() = %v, want %v", jd, nil)
				}
			},
			wantErr: true,
		},
		// // This can't be tested currently as there isn't a clear way
		// // to inject the eks configurer client
		// {
		// 	name: "Invalid value for requested memory fails",
		// 	args: args{
		// 		ctx: context.TODO(),
		// 		pluginData: map[string]string{
		// 			"api_url":        "http://localhost",
		// 			"auth_type":      "eks_iam",
		// 			"image":          "hello-world",
		// 			"memory_request": "Two Hundred and Fifty-Six mebibytes",
		// 			"memory_limit":   "512Mi",
		// 		},
		// 		logger: nil,
		// 	},
		// 	want: func(t *testing.T, jd *JobDispatcher) {
		// 		if jd != nil {
		// 			t.Errorf("New() = %v, want %v", jd, nil)
		// 		}
		// 	},
		// 	wantErr: true,
		// },
		// {
		// 	name: "Invalid value for memory limit fails",
		// 	args: args{
		// 		ctx: context.TODO(),
		// 		pluginData: map[string]string{
		// 			"api_url":        "http://localhost",
		// 			"auth_type":      "eks_iam",
		// 			"image":          "hello-world",
		// 			"memory_request": "256Mi",
		// 			"memory_limit":   "Five Hundred and Twelve mebibytes",
		// 		},
		// 		logger: nil,
		// 	},
		// 	want: func(t *testing.T, jd *JobDispatcher) {
		// 		if jd != nil {
		// 			t.Errorf("New() = %v, want %v", jd, nil)
		// 		}
		// 	},
		// 	wantErr: true,
		// },
		// {
		// 	name: "Default namespace is returned when no provided",
		// 	args: args{
		// 		ctx: context.TODO(),
		// 		pluginData: map[string]string{
		// 			"api_url":        "http://localhost",
		// 			"auth_type":      "eks_iam",
		// 			"region":         "us-east-2",
		// 			"eks_cluster":    "test",
		// 			"image":          "hello-world",
		// 			"memory_request": "256Mi",
		// 			"memory_limit":   "512Mi",
		// 		},
		// 		logger: nil,
		// 	},
		// 	want: func(t *testing.T, jd *JobDispatcher) {
		// 		want := "default"

		// 		runner, ok := jd.client.(*k8sRunner)
		// 		if !ok {
		// 			t.Errorf("client returned wasn't a k8sRunner")
		// 		}
		// 		if runner.namespace != want {
		// 			t.Errorf("New() = %v, want %v", runner.namespace, want)
		// 		}
		// 	},
		// 	wantErr: false,
		// },
		// {
		// 	name: "Provided namespace doesn't default",
		// 	args: args{
		// 		ctx: context.TODO(),
		// 		pluginData: map[string]string{
		// 			"api_url":        "http://localhost",
		// 			"auth_type":      "eks_iam",
		// 			"region":         "us-east-2",
		// 			"eks_cluster":    "test",
		// 			"image":          "hello-world",
		// 			"namespace":      "runner-ns",
		// 			"memory_request": "256Mi",
		// 			"memory_limit":   "512Mi",
		// 		},
		// 		logger: nil,
		// 	},
		// 	want: func(t *testing.T, jd *JobDispatcher) {
		// 		want := "runner-ns"

		// 		runner, ok := jd.client.(*k8sRunner)
		// 		if !ok {
		// 			t.Errorf("client returned wasn't a k8sRunner")
		// 		}
		// 		if runner.namespace != want {
		// 			t.Errorf("New() = %v, want %v", runner.namespace, want)
		// 		}
		// 	},
		// 	wantErr: false,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.ctx, tt.args.pluginData, "http://localhost", tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			tt.want(t, got)
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
						ID: "id",
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
						ID: "id",
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
