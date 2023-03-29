package eks

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/smithy-go/ptr"
	mock "github.com/stretchr/testify/mock"
	"k8s.io/client-go/rest"
)

func TestNew(t *testing.T) {
	type args struct {
		ctx         context.Context
		region      string
		clusterName string
	}
	tests := []struct {
		want    *IAMConfig
		args    args
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.ctx, tt.args.region, tt.args.clusterName)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getClusterInfo(t *testing.T) {
	type args struct {
		ctx       context.Context
		eksClient eksClient
		name      string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   []byte
		wantErr bool
	}{
		{
			name: "Failing to describe EKS Cluster results in an error",
			args: args{
				ctx: context.TODO(),
				eksClient: func() eksClient {
					client := &mockEksClient{}

					client.On("DescribeCluster", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("access denied")).Once()

					return client
				}(),
				name: "my-cluster",
			},
			want:    "",
			want1:   nil,
			wantErr: true,
		},
		{
			name: "EKS Cluster missing CA data results in an error",
			args: args{
				ctx: context.TODO(),
				eksClient: func() eksClient {
					client := &mockEksClient{}

					client.On("DescribeCluster", mock.Anything, mock.Anything).Return(&eks.DescribeClusterOutput{
						Cluster: &types.Cluster{
							Endpoint: ptr.String("https://my-cluster"),
						},
					}, nil).Once()

					return client
				}(),
				name: "my-cluster",
			},
			want:    "",
			want1:   nil,
			wantErr: true,
		},
		{
			name: "EKS Cluster bad CA data results in an error",
			args: args{
				ctx: context.TODO(),
				eksClient: func() eksClient {
					client := &mockEksClient{}

					client.On("DescribeCluster", mock.Anything, mock.Anything).Return(&eks.DescribeClusterOutput{
						Cluster: &types.Cluster{
							Endpoint: ptr.String("https://my-cluster"),
							CertificateAuthority: &types.Certificate{
								Data: ptr.String("test="),
							},
						},
					}, nil).Once()

					return client
				}(),
				name: "my-cluster",
			},
			want:    "",
			want1:   nil,
			wantErr: true,
		},
		{
			name: "EKS Cluster bad CA data results in an error",
			args: args{
				ctx: context.TODO(),
				eksClient: func() eksClient {
					client := &mockEksClient{}

					client.On("DescribeCluster", mock.Anything, mock.Anything).Return(&eks.DescribeClusterOutput{
						Cluster: &types.Cluster{
							Endpoint: ptr.String("https://my-cluster"),
							CertificateAuthority: &types.Certificate{
								Data: ptr.String("test="),
							},
						},
					}, nil).Once()

					return client
				}(),
				name: "my-cluster",
			},
			want:    "",
			want1:   nil,
			wantErr: true,
		},
		{
			name: "Successfully generate configuration",
			args: args{
				ctx: context.TODO(),
				eksClient: func() eksClient {
					client := &mockEksClient{}

					client.On("DescribeCluster", mock.Anything, mock.Anything).Return(&eks.DescribeClusterOutput{
						Cluster: &types.Cluster{
							Endpoint: ptr.String("https://my-cluster"),
							CertificateAuthority: &types.Certificate{
								Data: ptr.String(base64.StdEncoding.EncodeToString([]byte("ca-cert"))),
							},
						},
					}, nil).Once()

					return client
				}(),
				name: "my-cluster",
			},
			want:    "https://my-cluster",
			want1:   []byte("ca-cert"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := getClusterInfo(tt.args.ctx, tt.args.eksClient, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("getClusterInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getClusterInfo() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("getClusterInfo() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestEKSIAMConfig_GetConfig(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		args    args
		c       *IAMConfig
		want    *rest.Config
		name    string
		wantErr bool
	}{
		{
			name: "Return error when getToken fails",
			c: &IAMConfig{
				clusterName: "my-cluster",
				presigner: func() presigner {
					p := &mockPresigner{}

					p.On("PresignGetCallerIdentity", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("bad request")).Once()

					return p
				}(),
				clusterEndpoint: "https://my-cluster",
				clusterCAData:   []byte("ca-cert"),
				token:           "",
				tokenExpiresAt:  time.Now(),
			},
			args: args{
				ctx: context.TODO(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "When Token hasn't be generated, generate one",
			c: &IAMConfig{
				clusterName: "my-cluster",
				presigner: func() presigner {
					p := &mockPresigner{}

					p.On("PresignGetCallerIdentity", mock.Anything, mock.Anything, mock.Anything).Return(&v4.PresignedHTTPRequest{
						URL: "https://my-cluster-signed-url",
					}, nil).Once()

					return p
				}(),
				clusterEndpoint: "https://my-cluster",
				clusterCAData:   []byte("ca-cert"),
				token:           "",
				tokenExpiresAt:  time.Now(),
			},
			args: args{
				ctx: context.TODO(),
			},
			want: &rest.Config{
				Host: "https://my-cluster",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte("ca-cert"),
				},
				BearerToken: fmt.Sprintf("%s%s", tokenPrefix, strings.TrimRight(base64.URLEncoding.EncodeToString([]byte("https://my-cluster-signed-url")), "=")),
			},
			wantErr: false,
		},
		{
			name: "When Token has expired, generate new one",
			c: &IAMConfig{
				clusterName: "my-cluster",
				presigner: func() presigner {
					p := &mockPresigner{}

					p.On("PresignGetCallerIdentity", mock.Anything, mock.Anything, mock.Anything).Return(&v4.PresignedHTTPRequest{
						URL: "https://my-cluster-signed-url?new",
					}, nil).Once()

					return p
				}(),
				clusterEndpoint: "https://my-cluster",
				clusterCAData:   []byte("ca-cert"),
				token:           fmt.Sprintf("%s%s", tokenPrefix, strings.TrimRight(base64.URLEncoding.EncodeToString([]byte("https://my-cluster-signed-url")), "=")),
				tokenExpiresAt:  time.Now().Add(-1 * time.Minute),
			},
			args: args{
				ctx: context.TODO(),
			},
			want: &rest.Config{
				Host: "https://my-cluster",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte("ca-cert"),
				},
				BearerToken: fmt.Sprintf("%s%s", tokenPrefix, strings.TrimRight(base64.URLEncoding.EncodeToString([]byte("https://my-cluster-signed-url?new")), "=")),
			},
			wantErr: false,
		},
		{
			name: "When Token has not expired, use it",
			c: &IAMConfig{
				clusterName: "my-cluster",
				presigner: func() presigner {
					p := &mockPresigner{}

					p.On("PresignGetCallerIdentity", mock.Anything, mock.Anything, mock.Anything).Return(&v4.PresignedHTTPRequest{
						URL: "https://my-cluster-signed-url?new",
					}, nil).Once()

					return p
				}(),
				clusterEndpoint: "https://my-cluster",
				clusterCAData:   []byte("ca-cert"),
				token:           fmt.Sprintf("%s%s", tokenPrefix, strings.TrimRight(base64.URLEncoding.EncodeToString([]byte("https://my-cluster-signed-url")), "=")),
				tokenExpiresAt:  time.Now().Add(5 * time.Minute),
			},
			args: args{
				ctx: context.TODO(),
			},
			want: &rest.Config{
				Host: "https://my-cluster",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte("ca-cert"),
				},
				BearerToken: fmt.Sprintf("%s%s", tokenPrefix, strings.TrimRight(base64.URLEncoding.EncodeToString([]byte("https://my-cluster-signed-url")), "=")),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.GetConfig(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("EKSIAMConfig.GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EKSIAMConfig.GetConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEKSIAMConfig_getToken(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		c       *IAMConfig
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Failure to presign results in an error",
			c: &IAMConfig{
				presigner: func() presigner {
					p := &mockPresigner{}

					p.On("PresignGetCallerIdentity", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("bad request")).Once()

					return p
				}(),
				clusterName: "my-cluster",
			},
			args: args{
				ctx: context.TODO(),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Successful token generation",
			c: &IAMConfig{
				presigner: func() presigner {
					p := &mockPresigner{}

					p.On("PresignGetCallerIdentity", mock.Anything, mock.Anything, mock.Anything).Return(&v4.PresignedHTTPRequest{
						URL: "https://my-cluster-signed-url",
					}, nil).Once()

					return p
				}(),
				clusterName: "my-cluster",
			},
			args: args{
				ctx: context.TODO(),
			},
			want:    fmt.Sprintf("%s%s", tokenPrefix, strings.TrimRight(base64.URLEncoding.EncodeToString([]byte("https://my-cluster-signed-url")), "=")),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.getToken(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("EKSIAMConfig.getToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EKSIAMConfig.getToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
