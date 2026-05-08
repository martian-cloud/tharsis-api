package jobexecutor

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveStateFiles(t *testing.T) {
	tests := []struct {
		name          string
		existingFiles []string
		expectError   bool
	}{
		{
			name:          "removes both state files when they exist",
			existingFiles: []string{"terraform.tfstate", "terraform.tfstate.backup"},
		},
		{
			name:          "removes only primary state file when backup does not exist",
			existingFiles: []string{"terraform.tfstate"},
		},
		{
			name:          "removes only backup state file when primary does not exist",
			existingFiles: []string{"terraform.tfstate.backup"},
		},
		{
			name:          "succeeds when no state files exist",
			existingFiles: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspaceDir := t.TempDir()

			// Create the existing files
			for _, f := range test.existingFiles {
				err := os.WriteFile(filepath.Join(workspaceDir, f), []byte("{}"), 0o600)
				require.NoError(t, err)
			}

			tw := &terraformWorkspace{workspaceDir: workspaceDir}

			err := tw.removeStateFiles()

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify state files are gone
				_, err = os.Stat(filepath.Join(workspaceDir, "terraform.tfstate"))
				assert.True(t, os.IsNotExist(err))
				_, err = os.Stat(filepath.Join(workspaceDir, "terraform.tfstate.backup"))
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

func TestRemoveStateFiles_FailsWhenCannotDelete(t *testing.T) {
	// Skip when running as root since root can delete files regardless of directory permissions
	if u, err := user.Current(); err == nil && u.Uid == "0" {
		t.Skip("skipping test when running as root")
	}

	workspaceDir := t.TempDir()

	// Create state file
	stateFile := filepath.Join(workspaceDir, "terraform.tfstate")
	require.NoError(t, os.WriteFile(stateFile, []byte("{}"), 0o600))

	// Make directory read-only so file can't be deleted
	require.NoError(t, os.Chmod(workspaceDir, 0o555))
	defer os.Chmod(workspaceDir, 0o755) // restore so cleanup works

	tw := &terraformWorkspace{workspaceDir: workspaceDir}
	err := tw.removeStateFiles()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove state file")
}
