package moduleregistry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSuccess(t *testing.T) {
	moduleDir, err := os.MkdirTemp("", "module-parse-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(moduleDir)

	contents := `
	  provider "aws" {
		region = "us-east-2"
	  }
	`

	if err = os.WriteFile(filepath.Join(moduleDir, "main.tf"), []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}

	response, err := parseModule(moduleDir)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, len(response.Diagnostics))
	assert.Equal(t, "root", response.Root.Path)
	assert.Equal(t, 1, len(response.Root.ProviderConfigs))
	assert.Equal(t, "aws", response.Root.ProviderConfigs[0].Name)
}

func TestParseWithError(t *testing.T) {
	moduleDir, err := os.MkdirTemp("", "module-parse-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(moduleDir)

	contents := `
	  provider "aws" {
		region =
	  }
	`

	if err = os.WriteFile(filepath.Join(moduleDir, "main.tf"), []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}

	response, err := parseModule(moduleDir)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(response.Diagnostics))
	assert.Equal(t, "Invalid expression", response.Diagnostics[0].Summary)
}
