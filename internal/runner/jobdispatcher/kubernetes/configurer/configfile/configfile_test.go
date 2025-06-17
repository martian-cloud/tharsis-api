package configfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	// Create a temporary file to use as a kubeconfig
	tempDir := t.TempDir()
	validConfigPath := filepath.Join(tempDir, "valid-kubeconfig")

	// Create an empty file
	err := os.WriteFile(validConfigPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Define a non-existent path
	nonExistentPath := filepath.Join(tempDir, "non-existent-kubeconfig")

	type args struct {
		configFilePath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Successfully create new ConfigFile with valid path",
			args: args{
				configFilePath: validConfigPath,
			},
			wantErr: false,
		},
		{
			name: "Fail to create new ConfigFile with non-existent path",
			args: args{
				configFilePath: nonExistentPath,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.configFilePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got == nil {
					t.Errorf("New() returned nil, expected a valid ConfigFile instance")
					return
				}

				if got.filePath != tt.args.configFilePath {
					t.Errorf("New().filePath = %v, want %v", got.filePath, tt.args.configFilePath)
				}
			}
		})
	}
}
