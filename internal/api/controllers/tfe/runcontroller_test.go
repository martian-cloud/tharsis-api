package tfe

import (
	"reflect"
	"testing"

	"github.com/aws/smithy-go/ptr"
	gotfe "github.com/hashicorp/go-tfe"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func Test_parseRunVariables(t *testing.T) {
	type args struct {
		req  gotfe.RunCreateOptions
		body []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []runvariables.Variable
		wantErr bool
	}{
		{
			name: "No Variables should be fine",
			args: args{
				req: gotfe.RunCreateOptions{
					Variables: []*gotfe.RunVariable{},
				},
				body: []byte{},
			},
			want:    []runvariables.Variable{},
			wantErr: false,
		},
		{
			name: "Proper API Variables are supported",
			args: args{
				req: gotfe.RunCreateOptions{
					Variables: []*gotfe.RunVariable{
						{
							Key:   "foo",
							Value: "\"bar\"",
						},
					},
				},
				body: []byte{},
			},
			want: []runvariables.Variable{
				{
					Key:      "foo",
					Value:    ptr.String("\"bar\""),
					Category: models.TerraformVariableCategory,
				},
			},
			wantErr: false,
		},
		{
			name: "Broken Terraform API Variables are supported",
			args: args{
				req: gotfe.RunCreateOptions{
					Variables: []*gotfe.RunVariable{
						nil,
						{
							Key:   "",
							Value: "",
						},
					},
				},
				body: []byte(`{"data":{"attributes":{"variables":[{"Key":"foo","Value":"\"bar\""}]}}}`),
			},
			want: []runvariables.Variable{
				{
					Key:      "foo",
					Value:    ptr.String("\"bar\""),
					Category: models.TerraformVariableCategory,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid JSON should result in an error",
			args: args{
				req: gotfe.RunCreateOptions{
					Variables: []*gotfe.RunVariable{
						nil,
						{
							Key:   "",
							Value: "",
						},
					},
				},
				body: []byte(`{"data":{"attributes":{"variables":[{"Key":"foo","Value":"\"bar\""}]}}`),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRunVariables(tt.args.req, tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRunVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRunVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toTFEApplyStatus(t *testing.T) {
	tests := []struct {
		name   string
		status models.ApplyStatus
		want   gotfe.ApplyStatus
	}{
		{
			name:   "skipped maps to created since the CLI has no skipped status",
			status: models.ApplySkipped,
			want:   gotfe.ApplyStatus(models.ApplyCreated),
		},
		{
			name:   "created passes through",
			status: models.ApplyCreated,
			want:   gotfe.ApplyStatus(models.ApplyCreated),
		},
		{
			name:   "pending passes through",
			status: models.ApplyPending,
			want:   gotfe.ApplyStatus(models.ApplyPending),
		},
		{
			name:   "queued passes through",
			status: models.ApplyQueued,
			want:   gotfe.ApplyStatus(models.ApplyQueued),
		},
		{
			name:   "running passes through",
			status: models.ApplyRunning,
			want:   gotfe.ApplyStatus(models.ApplyRunning),
		},
		{
			name:   "finished passes through",
			status: models.ApplyFinished,
			want:   gotfe.ApplyStatus(models.ApplyFinished),
		},
		{
			name:   "errored passes through",
			status: models.ApplyErrored,
			want:   gotfe.ApplyStatus(models.ApplyErrored),
		},
		{
			name:   "canceled passes through",
			status: models.ApplyCanceled,
			want:   gotfe.ApplyStatus(models.ApplyCanceled),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toTFEApplyStatus(tt.status); got != tt.want {
				t.Errorf("toTFEApplyStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
