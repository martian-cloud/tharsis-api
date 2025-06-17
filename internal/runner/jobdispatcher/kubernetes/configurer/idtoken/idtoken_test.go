package idtoken

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"testing"

	"k8s.io/client-go/rest"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
)

func TestNew(t *testing.T) {
	type args struct {
		host        string
		ca          string
		tokenGetter types.TokenGetterFunc
	}
	tests := []struct {
		name    string
		args    args
		want    *IDToken
		wantErr bool
	}{
		{
			name: "Successfully create new IDToken with empty CA",
			args: args{
				host:        "https://kubernetes.default.svc",
				ca:          "",
				tokenGetter: func(_ context.Context) (string, error) { return "token", nil },
			},
			want: &IDToken{
				host:        "https://kubernetes.default.svc",
				tokenGetter: nil, // We can't compare functions directly, will check separately
				caData:      nil,
			},
			wantErr: false,
		},
		{
			name: "Successfully create new IDToken with valid CA",
			args: args{
				host:        "https://kubernetes.default.svc",
				ca:          base64.StdEncoding.EncodeToString([]byte("test-ca-data")),
				tokenGetter: func(_ context.Context) (string, error) { return "token", nil },
			},
			want: &IDToken{
				host:        "https://kubernetes.default.svc",
				tokenGetter: nil, // We can't compare functions directly, will check separately
				caData:      []byte("test-ca-data"),
			},
			wantErr: false,
		},
		{
			name: "Fail to create new IDToken with invalid CA",
			args: args{
				host:        "https://kubernetes.default.svc",
				ca:          "invalid-base64",
				tokenGetter: func(_ context.Context) (string, error) { return "token", nil },
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.host, tt.args.ca, tt.args.tokenGetter)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Check struct fields except for the function
			if got.host != tt.want.host {
				t.Errorf("New().host = %v, want %v", got.host, tt.want.host)
			}
			if !reflect.DeepEqual(got.caData, tt.want.caData) {
				t.Errorf("New().caData = %v, want %v", got.caData, tt.want.caData)
			}

			// Verify the function works as expected
			if got.tokenGetter == nil {
				t.Errorf("New().tokenGetter is nil")
			}
		})
	}
}

func TestIDToken_GetConfig(t *testing.T) {
	type fields struct {
		host        string
		tokenGetter types.TokenGetterFunc
		caData      []byte
	}
	tests := []struct {
		name    string
		fields  fields
		want    *rest.Config
		wantErr bool
	}{
		{
			name: "Successfully get config",
			fields: fields{
				host: "https://kubernetes.default.svc",
				tokenGetter: func(_ context.Context) (string, error) {
					return "test-token", nil
				},
				caData: []byte("test-ca-data"),
			},
			want: &rest.Config{
				Host:        "https://kubernetes.default.svc",
				BearerToken: "test-token",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte("test-ca-data"),
				},
			},
			wantErr: false,
		},
		{
			name: "Fail to get token",
			fields: fields{
				host: "https://kubernetes.default.svc",
				tokenGetter: func(_ context.Context) (string, error) {
					return "", fmt.Errorf("failed to get token")
				},
				caData: []byte("test-ca-data"),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &IDToken{
				host:        tt.fields.host,
				tokenGetter: tt.fields.tokenGetter,
				caData:      tt.fields.caData,
			}
			got, err := j.GetConfig(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("IDToken.GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IDToken.GetConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
