package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestNewStreamableHTTPHandler(t *testing.T) {
	testLogger, _ := logger.NewForTest()

	type testCase struct {
		name      string
		opts      *ServerOptions
		expectErr bool
	}

	tests := []testCase{
		{
			name: "with enabled toolsets",
			opts: &ServerOptions{
				ServicesCatalog: &services.Catalog{},
				Version:         "1.0.0",
				Config:          &config.MCPServerConfig{EnabledToolsets: "workspace,run"},
				Logger:          testLogger,
			},
		},
		{
			name: "with enabled tools",
			opts: &ServerOptions{
				ServicesCatalog: &services.Catalog{},
				Version:         "1.0.0",
				Config:          &config.MCPServerConfig{EnabledTools: "get_workspace,get_run"},
				Logger:          testLogger,
			},
		},
		{
			name: "with read_only and enabled toolsets",
			opts: &ServerOptions{
				ServicesCatalog: &services.Catalog{},
				Version:         "1.0.0",
				Config:          &config.MCPServerConfig{ReadOnly: true, EnabledToolsets: "workspace"},
				Logger:          testLogger,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewStreamableHTTPHandler(tt.opts)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, handler)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestNewServer(t *testing.T) {
	testLogger, _ := logger.NewForTest()

	type testCase struct {
		name      string
		opts      *ServerOptions
		expectErr bool
	}

	tests := []testCase{
		{
			name: "full config",
			opts: &ServerOptions{
				ServicesCatalog: &services.Catalog{},
				Version:         "2.0.0",
				Config:          &config.MCPServerConfig{ReadOnly: true, EnabledToolsets: "workspace"},
				Logger:          testLogger,
			},
		},
		{
			name: "missing version",
			opts: &ServerOptions{
				ServicesCatalog: &services.Catalog{},
				Config:          &config.MCPServerConfig{EnabledToolsets: "workspace"},
				Logger:          testLogger,
			},
			expectErr: true,
		},
		{
			name: "read_only false",
			opts: &ServerOptions{
				ServicesCatalog: &services.Catalog{},
				Version:         "1.0.0",
				Config:          &config.MCPServerConfig{ReadOnly: false, EnabledToolsets: "workspace"},
				Logger:          testLogger,
			},
		},
		{
			name: "defaults to all toolsets when not specified",
			opts: &ServerOptions{
				ServicesCatalog: &services.Catalog{},
				Version:         "1.0.0",
				Config:          &config.MCPServerConfig{},
				Logger:          testLogger,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := newServer(tt.opts)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, server)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, server)
			}
		})
	}
}
