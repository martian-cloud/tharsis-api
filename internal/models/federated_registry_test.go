package models

import (
	"strings"
	"testing"
)

func TestFederatedRegistryValidate(t *testing.T) {
	testCases := []struct {
		name      string
		registry  FederatedRegistry
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid hostname and audience",
			registry: FederatedRegistry{
				Hostname: "registry.example.com",
				Audience: "tharsis-registry",
			},
			expectErr: false,
		},
		{
			name: "valid IP address as hostname",
			registry: FederatedRegistry{
				Hostname: "192.168.1.1",
				Audience: "tharsis-registry",
			},
			expectErr: false,
		},
		{
			name: "localhost as hostname",
			registry: FederatedRegistry{
				Hostname: "localhost",
				Audience: "tharsis-registry",
			},
			expectErr: false,
		},
		{
			name: "port number only as hostname",
			registry: FederatedRegistry{
				Hostname: "8000",
				Audience: "tharsis-registry",
			},
			expectErr: false,
		},
		{
			name: "hostname with port",
			registry: FederatedRegistry{
				Hostname: "example.com:8000",
				Audience: "tharsis-registry",
			},
			expectErr: false,
		},
		{
			name: "localhost with port",
			registry: FederatedRegistry{
				Hostname: "localhost:8080",
				Audience: "tharsis-registry",
			},
			expectErr: false,
		},
		{
			name: "empty hostname",
			registry: FederatedRegistry{
				Hostname: "",
				Audience: "tharsis-registry",
			},
			expectErr: true,
			errMsg:    "hostname cannot be empty",
		},
		{
			name: "hostname with non-numeric port",
			registry: FederatedRegistry{
				Hostname: "example.com:port",
				Audience: "tharsis-registry",
			},
			expectErr: true,
			errMsg:    "invalid hostname format",
		},
		{
			name: "empty audience",
			registry: FederatedRegistry{
				Hostname: "registry.example.com",
				Audience: "",
			},
			expectErr: true,
			errMsg:    "audience cannot be empty",
		},
		{
			name: "audience exceeding max length",
			registry: FederatedRegistry{
				Hostname: "registry.example.com",
				Audience: "this-is-a-very-long-audience-that-exceeds-the-maximum-length-of-sixty-four-characters-limit",
			},
			expectErr: true,
			errMsg:    "audience cannot exceed 64 characters",
		},
		{
			name: "short audience now allowed",
			registry: FederatedRegistry{
				Hostname: "registry.example.com",
				Audience: "abc",
			},
			expectErr: false,
		},
		{
			name: "generic audience now allowed",
			registry: FederatedRegistry{
				Hostname: "registry.example.com",
				Audience: "test",
			},
			expectErr: false,
		},
		{
			name: "audience with special characters now allowed",
			registry: FederatedRegistry{
				Hostname: "registry.example.com",
				Audience: "invalid*audience",
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.registry.Validate()

			if tc.expectErr {
				if err == nil {
					t.Fatalf("Expected error but got nil")
				}
				if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Fatalf("Expected error to contain %q but got %q", tc.errMsg, err.Error())
				}
			} else if err != nil {
				t.Fatalf("Expected no error but got %v", err)
			}
		})
	}
}
