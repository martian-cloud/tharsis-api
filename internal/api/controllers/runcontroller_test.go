package controllers

import (
	"reflect"
	"testing"

	"github.com/aws/smithy-go/ptr"
	gotfe "github.com/hashicorp/go-tfe"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
)

func Test_parseRunVariables(t *testing.T) {
	type args struct {
		req  gotfe.RunCreateOptions
		body []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []run.Variable
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
			want:    []run.Variable{},
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
			want: []run.Variable{
				{
					Key:      "foo",
					Value:    ptr.String("\"bar\""),
					Category: models.TerraformVariableCategory,
					Hcl:      true,
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
			want: []run.Variable{
				{
					Key:      "foo",
					Value:    ptr.String("\"bar\""),
					Category: models.TerraformVariableCategory,
					Hcl:      true,
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
