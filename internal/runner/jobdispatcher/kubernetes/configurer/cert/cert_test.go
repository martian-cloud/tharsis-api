package cert

import (
	"context"
	"encoding/base64"
	"reflect"
	"testing"

	"k8s.io/client-go/rest"
)

func TestNew(t *testing.T) {
	type args struct {
		host string
		cert string
		key  string
		ca   string
	}
	tests := []struct {
		name    string
		args    args
		want    *Cert
		wantErr bool
	}{
		{
			name: "Successfully create new Cert with empty CA",
			args: args{
				host: "https://kubernetes.default.svc",
				cert: base64.StdEncoding.EncodeToString([]byte("test-cert-data")),
				key:  base64.StdEncoding.EncodeToString([]byte("test-key-data")),
				ca:   "",
			},
			want: &Cert{
				host:     "https://kubernetes.default.svc",
				certData: []byte("test-cert-data"),
				keyData:  []byte("test-key-data"),
				caData:   nil,
			},
			wantErr: false,
		},
		{
			name: "Successfully create new Cert with valid CA",
			args: args{
				host: "https://kubernetes.default.svc",
				cert: base64.StdEncoding.EncodeToString([]byte("test-cert-data")),
				key:  base64.StdEncoding.EncodeToString([]byte("test-key-data")),
				ca:   base64.StdEncoding.EncodeToString([]byte("test-ca-data")),
			},
			want: &Cert{
				host:     "https://kubernetes.default.svc",
				certData: []byte("test-cert-data"),
				keyData:  []byte("test-key-data"),
				caData:   []byte("test-ca-data"),
			},
			wantErr: false,
		},
		{
			name: "Fail to create new Cert with invalid cert",
			args: args{
				host: "https://kubernetes.default.svc",
				cert: "invalid-base64",
				key:  base64.StdEncoding.EncodeToString([]byte("test-key-data")),
				ca:   "",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Fail to create new Cert with invalid key",
			args: args{
				host: "https://kubernetes.default.svc",
				cert: base64.StdEncoding.EncodeToString([]byte("test-cert-data")),
				key:  "invalid-base64",
				ca:   "",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Fail to create new Cert with invalid CA",
			args: args{
				host: "https://kubernetes.default.svc",
				cert: base64.StdEncoding.EncodeToString([]byte("test-cert-data")),
				key:  base64.StdEncoding.EncodeToString([]byte("test-key-data")),
				ca:   "invalid-base64",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.host, tt.args.cert, tt.args.key, tt.args.ca)
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

func TestCert_GetConfig(t *testing.T) {
	type fields struct {
		host     string
		certData []byte
		keyData  []byte
		caData   []byte
	}
	tests := []struct {
		name    string
		fields  fields
		want    *rest.Config
		wantErr bool
	}{
		{
			name: "Successfully get config with all fields",
			fields: fields{
				host:     "https://kubernetes.default.svc",
				certData: []byte("test-cert-data"),
				keyData:  []byte("test-key-data"),
				caData:   []byte("test-ca-data"),
			},
			want: &rest.Config{
				Host: "https://kubernetes.default.svc",
				TLSClientConfig: rest.TLSClientConfig{
					CertData: []byte("test-cert-data"),
					KeyData:  []byte("test-key-data"),
					CAData:   []byte("test-ca-data"),
				},
			},
			wantErr: false,
		},
		{
			name: "Successfully get config without CA data",
			fields: fields{
				host:     "https://kubernetes.default.svc",
				certData: []byte("test-cert-data"),
				keyData:  []byte("test-key-data"),
				caData:   nil,
			},
			want: &rest.Config{
				Host: "https://kubernetes.default.svc",
				TLSClientConfig: rest.TLSClientConfig{
					CertData: []byte("test-cert-data"),
					KeyData:  []byte("test-key-data"),
					CAData:   nil,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cert{
				host:     tt.fields.host,
				certData: tt.fields.certData,
				keyData:  tt.fields.keyData,
				caData:   tt.fields.caData,
			}
			got, err := c.GetConfig(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Cert.GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cert.GetConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
