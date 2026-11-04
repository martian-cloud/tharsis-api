package models

import (
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestWorkspaceVCSProviderLink_ValidateModuleDirectory(t *testing.T) {
	tests := []struct {
		moduleDirectory string
		valid           bool
	}{
		{moduleDirectory: "", valid: true},
		{moduleDirectory: ".", valid: true},
		{moduleDirectory: "modules/network", valid: true},
		{moduleDirectory: "modules/../network", valid: true},
		{moduleDirectory: "..", valid: false},
		{moduleDirectory: "../../etc", valid: false},
		{moduleDirectory: "modules/../../etc", valid: false},
		{moduleDirectory: "/etc", valid: false},
	}

	for _, test := range tests {
		t.Run(test.moduleDirectory, func(t *testing.T) {
			link := &WorkspaceVCSProviderLink{RepositoryPath: "owner/repo", ModuleDirectory: ptr.String(test.moduleDirectory)}
			err := link.Validate()
			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
			}
		})
	}
}

func TestWorkspaceVCSProviderLink_ValidateRepositoryPath(t *testing.T) {
	tests := []struct {
		repositoryPath string
		valid          bool
	}{
		{repositoryPath: "owner/repo", valid: true},
		{repositoryPath: "group/subgroup/project", valid: true},
		{repositoryPath: "owner/my-repo.v2_x", valid: true},
		{repositoryPath: "owner/repo..", valid: true},
		{repositoryPath: "", valid: false},
		{repositoryPath: ".", valid: false},
		{repositoryPath: "..", valid: false},
		{repositoryPath: "../orgs/acme", valid: false},
		{repositoryPath: "owner/./repo", valid: false},
		{repositoryPath: "owner/repo/..", valid: false},
		{repositoryPath: "/owner/repo", valid: false},
		{repositoryPath: "owner/repo/", valid: false},
		{repositoryPath: "owner//repo", valid: false},
		{repositoryPath: "owner/repo?x=1", valid: false},
		{repositoryPath: "owner/repo#x", valid: false},
		{repositoryPath: "owner/%2e%2e", valid: false},
		{repositoryPath: `owner\..\repo`, valid: false},
		{repositoryPath: "owner/my repo", valid: false},
		{repositoryPath: "owner/repo\n", valid: false},
		{repositoryPath: "owner/repo\x00", valid: false},
	}

	for _, test := range tests {
		t.Run(test.repositoryPath, func(t *testing.T) {
			link := &WorkspaceVCSProviderLink{RepositoryPath: test.repositoryPath}
			err := link.Validate()
			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
			}
		})
	}
}

func TestWorkspaceVCSProviderLink_ValidateGlobPatterns(t *testing.T) {
	tests := []struct {
		pattern string
		valid   bool
	}{
		{pattern: "**", valid: true},
		{pattern: "/*.tf", valid: true},
		{pattern: "modules/**/*.tf", valid: true},
		{pattern: "a..b/*.tf", valid: true},
		{pattern: "[", valid: false},
		{pattern: "..", valid: false},
		{pattern: "../../../../**", valid: false},
		{pattern: "modules/../../**", valid: false},
		{pattern: "/../**", valid: false},
	}

	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			link := &WorkspaceVCSProviderLink{RepositoryPath: "owner/repo", GlobPatterns: []string{test.pattern}}
			err := link.Validate()
			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
			}
		})
	}
}

func TestNormalizeRepositoryPath(t *testing.T) {
	testCases := []struct {
		name        string
		path        string
		expected    string
		expectError bool
	}{
		{name: "valid path is unchanged", path: "group/subgroup/project", expected: "group/subgroup/project"},
		{name: "leading slash is dropped", path: "/group/project", expected: "group/project"},
		{name: "empty segments are dropped", path: "group//project/", expected: "group/project"},
		{name: "empty path is invalid", path: "", expectError: true},
		{name: "only slashes is invalid", path: "//", expectError: true},
		{name: "dot-dot segment is invalid", path: "group/../project", expectError: true},
		{name: "query character is invalid", path: "group/project?x=y", expectError: true},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actual, err := NormalizeRepositoryPath(test.path)
			if test.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}
