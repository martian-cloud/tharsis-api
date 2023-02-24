package module

import "testing"

func TestBuildTokenEnvVar(t *testing.T) {
	type args struct {
		host string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Invalid port should result in an error",
			args: args{
				host: "example.com:invalid",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Empty string should result in an error",
			args: args{
				host: "",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Full URI should result in an error",
			args: args{
				host: "https//example.com:1234",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Double periods should result in an error",
			args: args{
				host: "example..com",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Simple host without unicode should convert without ASCII Compatible Encoding prefix",
			args: args{
				host: "an-example.com",
			},
			want:    "TF_TOKEN_an__example_com",
			wantErr: false,
		},
		{
			name: "Unicode host should convert",
			args: args{
				host: "例えば.com",
			},
			want:    "TF_TOKEN_xn____r8j3dr99h_com",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildTokenEnvVar(tt.args.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildTokenEnvVar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BuildTokenEnvVar() = %v, want %v", got, tt.want)
			}
		})
	}
}
