package moduleregistry

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
)

// Output represents a Terraform configuration output
type Output struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Sensitive   bool   `json:"sensitive,omitempty"`
}

// Variable represents a Terraform conifguration input variable
type Variable struct {
	Default     *string `json:"default"`
	Name        string  `json:"name"`
	Type        string  `json:"type,omitempty"`
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required"`
	Sensitive   bool    `json:"sensitive,omitempty"`
}

// ProviderRef identifies a provider by name and an optional alias
type ProviderRef struct {
	Name  string `json:"name"`
	Alias string `json:"alias,omitempty"`
}

// ProviderRequirement represents a provider that is required by a configuration
type ProviderRequirement struct {
	Source               string        `json:"source,omitempty"`
	VersionConstraints   []string      `json:"version_constraints,omitempty"`
	ConfigurationAliases []ProviderRef `json:"aliases,omitempty"`
}

// Resource represents a resource created by a configuration
type Resource struct {
	Mode string `json:"mode"`
	Type string `json:"type"`
	Name string `json:"name"`

	Provider ProviderRef `json:"provider"`
}

// ModuleCall represents a call to another module
type ModuleCall struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	Version string `json:"version,omitempty"`
}

// ModuleConfigurationDetails includes the metadata for a parsed module configuration
type ModuleConfigurationDetails struct {
	RequiredProviders []*ProviderRequirement `json:"required_providers"`
	ProviderConfigs   []*ProviderRef         `json:"provider_configs,omitempty"`
	ManagedResources  []*Resource            `json:"managed_resources"`
	Variables         []*Variable            `json:"variables"`
	Outputs           []*Output              `json:"outputs"`
	DataResources     []*Resource            `json:"data_resources"`
	ModuleCalls       []*ModuleCall          `json:"module_calls"`
	Readme            string                 `json:"readme"`
	Path              string                 `json:"path"`
	RequiredCore      []string               `json:"required_core,omitempty"`
}

// ParseModuleResponse contains the configuration details for the root module, submodules, and example modules
type ParseModuleResponse struct {
	Root        *ModuleConfigurationDetails
	Submodules  []ModuleConfigurationDetails
	Examples    []ModuleConfigurationDetails
	Diagnostics tfconfig.Diagnostics
}

func parseModule(moduleDir string) (*ParseModuleResponse, error) {
	diagnostics := tfconfig.Diagnostics{}

	rootModuleMetadata, diag, err := loadModuleMetadata(moduleDir, "root")
	if err != nil {
		return nil, err
	}

	diagnostics = append(diagnostics, diag...)

	submodules, diag, err := loadMetadataForModules(moduleDir, "modules")
	if err != nil {
		return nil, err
	}

	diagnostics = append(diagnostics, diag...)

	examples, diag, err := loadMetadataForModules(moduleDir, "examples")
	if err != nil {
		return nil, err
	}

	diagnostics = append(diagnostics, diag...)

	return &ParseModuleResponse{
		Root:        rootModuleMetadata,
		Submodules:  submodules,
		Examples:    examples,
		Diagnostics: diagnostics,
	}, nil
}

func loadMetadataForModules(moduleDir string, subPath string) ([]ModuleConfigurationDetails, tfconfig.Diagnostics, error) {
	modules := []ModuleConfigurationDetails{}
	diagnostics := tfconfig.Diagnostics{}

	fullDirPath := filepath.Join(moduleDir, subPath)
	if _, err := os.Stat(fullDirPath); !os.IsNotExist(err) {
		entries, err := os.ReadDir(fullDirPath)
		if err != nil {
			return nil, nil, err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				metadata, diag, err := loadModuleMetadata(filepath.Join(moduleDir, subPath, entry.Name()), filepath.Join(subPath, entry.Name()))
				if err != nil {
					return nil, nil, err
				}
				modules = append(modules, *metadata)
				diagnostics = append(diagnostics, diag...)
			}
		}
	}

	return modules, diagnostics, nil
}

func loadModuleMetadata(dir string, relativePath string) (*ModuleConfigurationDetails, tfconfig.Diagnostics, error) {
	module, diagnostics := tfconfig.LoadModule(dir)

	meta := ModuleConfigurationDetails{
		Path:              relativePath,
		RequiredCore:      module.RequiredCore,
		Variables:         []*Variable{},
		Outputs:           []*Output{},
		RequiredProviders: []*ProviderRequirement{},
		ProviderConfigs:   []*ProviderRef{},
		ManagedResources:  []*Resource{},
		DataResources:     []*Resource{},
		ModuleCalls:       []*ModuleCall{},
	}

	for _, v := range module.Variables {
		var defaultVal *string
		if v.Default != nil {
			buf, err := json.Marshal(v.Default)
			if err != nil {
				return nil, nil, err
			}
			defaultVal = ptr.String(string(buf))
		}
		meta.Variables = append(meta.Variables, &Variable{
			Name:        v.Name,
			Description: v.Description,
			Type:        v.Type,
			Default:     defaultVal,
			Required:    v.Required,
			Sensitive:   v.Sensitive,
		})
	}

	for _, o := range module.Outputs {
		meta.Outputs = append(meta.Outputs, &Output{
			Name:        o.Name,
			Description: o.Description,
			Sensitive:   o.Sensitive,
		})
	}

	for _, p := range module.RequiredProviders {
		req := &ProviderRequirement{
			Source:               p.Source,
			VersionConstraints:   p.VersionConstraints,
			ConfigurationAliases: []ProviderRef{},
		}
		for _, ca := range p.ConfigurationAliases {
			req.ConfigurationAliases = append(req.ConfigurationAliases, ProviderRef{
				Name:  ca.Name,
				Alias: ca.Alias,
			})
		}
		meta.RequiredProviders = append(meta.RequiredProviders, req)
	}

	for _, c := range module.ProviderConfigs {
		meta.ProviderConfigs = append(meta.ProviderConfigs, &ProviderRef{
			Name:  c.Name,
			Alias: c.Alias,
		})
	}

	for _, r := range module.ManagedResources {
		meta.ManagedResources = append(meta.ManagedResources, &Resource{
			Type: r.Type,
			Name: r.Name,
			Mode: r.Mode.String(),
			Provider: ProviderRef{
				Name:  r.Provider.Name,
				Alias: r.Provider.Alias,
			},
		})
	}

	for _, d := range module.DataResources {
		meta.DataResources = append(meta.DataResources, &Resource{
			Type: d.Type,
			Name: d.Name,
			Mode: d.Mode.String(),
			Provider: ProviderRef{
				Name:  d.Provider.Name,
				Alias: d.Provider.Alias,
			},
		})
	}

	for _, mc := range module.ModuleCalls {
		meta.ModuleCalls = append(meta.ModuleCalls, &ModuleCall{
			Name:    mc.Name,
			Source:  mc.Source,
			Version: mc.Version,
		})
	}

	// Check for README or README.md
	matches, err := filepath.Glob(filepath.Join(dir, "README*"))
	if err != nil {
		return nil, nil, err
	}

	if len(matches) > 0 {
		readmeFile, err := os.Open(matches[0])
		if err != nil {
			return nil, nil, err
		}
		defer readmeFile.Close()

		readmeBuf, err := io.ReadAll(readmeFile)
		if err != nil {
			return nil, nil, err
		}

		meta.Readme = string(readmeBuf)
	}

	return &meta, diagnostics, nil
}
