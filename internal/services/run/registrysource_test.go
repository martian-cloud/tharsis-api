package run

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
)

// TestResolveModuleVersion tests the various cases for the ResolveModuleVersion function.
//
// The GetModuleRegistryEndpointForHost function is mocked out via a separate function and the private new method.
// The plumbing for that is modeled after the unit test for the publicKeyGetter function in the awskms module.
//
// The other HTTP accesses by the underlying Terraform functions are handled by an embedded HTTP server.
func TestResolveModuleVersionRemote(t *testing.T) {
	apiMapKey := "modules.v1"
	apiMapVal := "/api/v4/packages/terraform/modules/v1/" // starts and ends with slashes

	// Second-level structure and highest version.
	highestVersionReturn := "2.1.0"
	versionStructReturn := `
	{
		"modules": [
				{
						"versions": [
								{
										"version": "2.1.0",
										"submodules": [],
										"root": {
												"dependencies": [],
												"providers": [
														{
																"name": "aws",
																"version": ""
														}
												]
										}
								},
								{
										"version": "0.0.3",
										"submodules": [],
										"root": {
												"dependencies": [],
												"providers": [
														{
																"name": "aws",
																"version": ""
														}
												]
										}
								},
								{
										"version": "0.0.2",
										"submodules": [],
										"root": {
												"dependencies": [],
												"providers": [
														{
																"name": "aws",
																"version": ""
														}
												]
										}
								},
								{
										"version": "0.0.1",
										"submodules": [],
										"root": {
												"dependencies": [],
												"providers": [
														{
																"name": "aws",
																"version": ""
														}
												]
										}
								}
						]
				}
		]
	}
	`

	// Fake API "well-known" map.
	apiMap := map[string]string{apiMapKey: apiMapVal}
	apiMapJSON, err := json.Marshal(apiMap)
	assert.Nil(t, err)

	// Interior path to a fake module.
	fakeModulePath := "somegroup/hypothetical-test-module-001/aws" // no slashes on ends

	// Fake access token to test that it gets through to the server.
	fakeAccessToken := fmt.Sprintf("fake-access-token-%d", time.Now().Unix())

	// Launch the test server to listen for requests to 127.0.0.1.
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/terraform.json", func(w http.ResponseWriter, r *http.Request) {
		_ = r
		_, _ = w.Write(apiMapJSON)
	})
	nextLevelPath := apiMapVal + fakeModulePath + "/versions"
	mux.HandleFunc(nextLevelPath, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Header.Get("Authorization"), "Bearer "+fakeAccessToken)
		_, _ = w.Write([]byte(versionStructReturn))
	})
	s := httptest.NewTLSServer(mux)
	defer s.Close()
	assert.True(t, strings.HasPrefix(s.URL, "https://127.0.0.1"))
	serverURL, err := url.Parse(s.URL)
	assert.Nil(t, err)
	serverHostPort := serverURL.Host
	properClient := s.Client()

	// Test cases:
	tests := []struct {
		expectError   error
		origVersion   *string
		expectVersion *string
		name          string
		origSource    string
	}{
		{
			name:          "local module, absolute path, returned as non-registry, go-getter filesystem path",
			origSource:    "/home/someuser/stuff/modules/some-module",
			expectVersion: nil,
			expectError:   nil,
		},
		{
			name:          "local module, relative path, not supported",
			origSource:    "../modules/some-other-module",
			expectVersion: nil,
			expectError:   fmt.Errorf("local modules are not supported"),
		},
		{
			name:          "registry module on 127.0.0.1 (using 'localhost' won't work because it does have dot)",
			origSource:    serverHostPort + "/" + fakeModulePath,
			expectVersion: &highestVersionReturn,
			expectError:   nil,
		},
		{
			name:          "exact version registry module on 127.0.0.1 (using 'localhost' won't work because it does have dot)",
			origSource:    serverHostPort + "/" + fakeModulePath,
			origVersion:   ptr.String("0.0.3"),
			expectVersion: ptr.String("0.0.3"),
			expectError:   nil,
		},
		{
			name:          "remote source (non-registry, already go-getter style)",
			origSource:    "http://somewhere.somecompany.com/somedir/someotherdir/anotherdir/some-module.tgz",
			expectVersion: nil,
			expectError:   nil,
		},
	}

	vars := []Variable{
		{
			Key:   "TF_TOKEN_" + strings.ReplaceAll(serverHostPort, ".", "_"),
			Value: &fakeAccessToken,
		},
	}

	// Run the test cases.
	for _, test := range tests {
		ctx := context.Background()

		mockModuleService := moduleregistry.NewMockService(t)

		// Resolve the module version.
		gotVersion, err := NewModuleResolver(mockModuleService, properClient, logger.New(), "http://testserver").
			ResolveModuleVersion(ctx, test.origSource, test.origVersion, vars)

		// Compare vs. expected results.
		assert.Equal(t, test.expectError, err)
		assert.Equal(t, (test.expectVersion == nil), (gotVersion == nil))
		if (test.expectVersion != nil) && (gotVersion != nil) {
			assert.Equal(t, *test.expectVersion, *gotVersion)
		}
	}
}

func TestResolveModuleVersionLocal(t *testing.T) {
	apiMapKey := "modules.v1"
	apiMapVal := "/api/v4/packages/terraform/modules/v1/" // starts and ends with slashes

	// Fake API "well-known" map.
	apiMap := map[string]string{apiMapKey: apiMapVal}
	apiMapJSON, err := json.Marshal(apiMap)
	assert.Nil(t, err)

	// Launch the test server to listen for requests to 127.0.0.1.
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/terraform.json", func(w http.ResponseWriter, r *http.Request) {
		_ = r
		_, _ = w.Write(apiMapJSON)
	})

	s := httptest.NewTLSServer(mux)
	defer s.Close()

	serverURL, err := url.Parse(s.URL)
	require.Nil(t, err)

	properClient := s.Client()

	// Test cases:
	tests := []struct {
		version         *string
		name            string
		moduleNamespace string
		moduleName      string
		moduleSystem    string
		expectVersion   string
	}{
		{
			name:            "get latest version for local module",
			moduleNamespace: "ns1",
			moduleName:      "m1",
			moduleSystem:    "s1",
			expectVersion:   "1.0.0",
		},
		{
			name:            "get specific version for local module",
			moduleNamespace: "ns1",
			moduleName:      "m1",
			moduleSystem:    "s1",
			version:         ptr.String("0.0.1"),
			expectVersion:   "0.0.1",
		},
	}

	// Run the test cases.
	for _, test := range tests {
		ctx := context.Background()

		mockModuleService := moduleregistry.NewMockService(t)
		mockModuleService.On("GetModuleByAddress", mock.Anything, test.moduleNamespace, test.moduleName, test.moduleSystem).Return(&models.TerraformModule{
			Metadata: models.ResourceMetadata{ID: "123"},
		}, nil)

		statusFilter := models.TerraformModuleVersionStatusUploaded
		mockModuleService.On("GetModuleVersions", mock.Anything, &moduleregistry.GetModuleVersionsInput{
			ModuleID: "123",
			Status:   &statusFilter,
		}).Return(&db.ModuleVersionsResult{
			ModuleVersions: []models.TerraformModuleVersion{
				{Metadata: models.ResourceMetadata{ID: "mv1"}, SemanticVersion: "0.0.1"},
				{Metadata: models.ResourceMetadata{ID: "mv1"}, SemanticVersion: "1.0.0"},
			},
		}, nil)

		// Resolve the module version.
		gotVersion, err := NewModuleResolver(mockModuleService, properClient, logger.New(), s.URL).
			ResolveModuleVersion(ctx, fmt.Sprintf("%s/%s/%s/%s", serverURL.Host, test.moduleNamespace, test.moduleName, test.moduleSystem), test.version, []Variable{})

		require.Nil(t, err)

		require.NotNil(t, gotVersion)
		assert.Equal(t, test.expectVersion, *gotVersion)
	}
}

// TestGetLatestMatchingVersion tests the getLatestMatchingVersion function
// with minimal overhead.
func TestGetLatestMatchingVersion(t *testing.T) {

	versions := map[string]bool{
		"0.0.1": true,
		"0.0.2": true,
		"0.0.3": true,
		"2.1.0": true,
	}

	// Test cases:
	tests := []struct {
		expectError error
		constraints *string
		name        string
		expected    string
	}{
		{
			name:        "invalid range string",
			constraints: ptr.String(""),
			expected:    "",
			expectError: fmt.Errorf("failed to parse wanted version range string: Malformed constraint: "),
		},
		{
			name:        "no constraint, return latest of all",
			constraints: nil,
			expected:    "2.1.0",
		},
		{
			name:        "exact match",
			constraints: ptr.String("0.0.2"),
			expected:    "0.0.2",
		},
		{
			name:        "exact match but does not exist",
			constraints: ptr.String("1.2.1"),
			expected:    "",
			expectError: fmt.Errorf("no matching version found"),
		},
		{
			name:        "less than",
			constraints: ptr.String("< 1.0"),
			expected:    "0.0.3",
		},
		{
			name:        "less than or equal, 0.0.1",
			constraints: ptr.String("<= 0.0.1"),
			expected:    "0.0.1",
		},
		{
			name:        "less than or equal, 0.0.2",
			constraints: ptr.String("<= 0.0.2"),
			expected:    "0.0.2",
		},
		{
			name:        "less than or equal, 0.0.3",
			constraints: ptr.String("<= 0.0.3"),
			expected:    "0.0.3",
		},
		{
			name:        "greater than",
			constraints: ptr.String("> 1.0"),
			expected:    "2.1.0",
		},
		{
			name:        "between exclusive",
			constraints: ptr.String("> 0.0.1 , < 0.0.3"),
			expected:    "0.0.2",
		},
		{
			name:        "between inclusive",
			constraints: ptr.String(">= 0.0.1 , <= 0.0.3"),
			expected:    "0.0.3",
		},
		{
			name:        "contradictory",
			constraints: ptr.String("< 0.0.1 , > 0.0.3"),
			expected:    "nil",
			expectError: fmt.Errorf("no matching version found"),
		},
	}

	for _, test := range tests {
		got, err := getLatestMatchingVersion(versions, test.constraints)
		assert.Equal(t, test.expectError, err)
		if (err == nil) && (test.expectError == nil) {
			// Don't report noise if there is or should have been an error.
			assert.Equal(t, test.expected, got)
		}
	}
}

// The End.
