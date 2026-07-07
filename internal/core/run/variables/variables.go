// Package variables provides the run variable type and the logic for building and
// reading a run's variables (merging workspace-inherited variables, resolving
// secrets, and reading the variables stored for an existing run).
package variables

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/module"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/secret"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// Variable represents a run variable
type Variable struct {
	VersionID          *string                 `json:"version_id"`
	Value              *string                 `json:"value"`
	NamespacePath      *string                 `json:"namespacePath"`
	Key                string                  `json:"key"`
	Category           models.VariableCategory `json:"category"`
	Sensitive          bool                    `json:"sensitive"`
	IncludedInTFConfig *bool                   `json:"includedInTFConfig"`
}

// Builder builds and reads run variables.
type Builder struct {
	dbClient      *db.Client
	secretManager secret.Manager
	artifactStore workspace.ArtifactStore
}

// NewBuilder creates a new variable Builder.
func NewBuilder(dbClient *db.Client, secretManager secret.Manager, artifactStore workspace.ArtifactStore) *Builder {
	return &Builder{dbClient: dbClient, secretManager: secretManager, artifactStore: artifactStore}
}

// Build merges the run-provided variables with the workspace's inherited
// variables (run-provided take precedence) and resolves sensitive values.
func (b *Builder) Build(ctx context.Context, workspaceID string, runVariables []Variable) ([]Variable, error) {
	variableMap := map[string]Variable{}

	buildMapKey := func(key string, category string) string {
		return fmt.Sprintf("%s::%s", key, category)
	}

	// Add run variables first since they have the highest precedence
	for _, v := range runVariables {
		variableMap[buildMapKey(v.Key, string(v.Category))] = Variable{
			Key:       v.Key,
			Value:     v.Value,
			Category:  v.Category,
			Sensitive: false,
		}
	}

	ws, err := b.dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	if ws == nil {
		return nil, errors.New("workspace with id %s not found", workspaceID, errors.WithErrorCode(errors.ENotFound))
	}

	// Use a descending sort so the variables from the closest ancestor will take precedence
	sortBy := db.VariableSortableFieldNamespacePathDesc
	result, err := b.dbClient.Variables.GetVariables(ctx, &db.GetVariablesInput{
		Filter: &db.VariableFilter{
			NamespacePaths: ws.ExpandPath(),
		},
		Sort: &sortBy,
	})
	if err != nil {
		return nil, err
	}

	for _, v := range result.Variables {
		v := v

		keyAndCategory := buildMapKey(v.Key, string(v.Category))
		if _, ok := variableMap[keyAndCategory]; !ok {
			value := v.Value
			// Get secret value if variable is sensitive
			if v.Sensitive {
				// Use secret manager to get the secret value
				secretValue, err := b.secretManager.Get(ctx, v.Key, v.SecretData)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get secret value for variable %q when saving run variables", v.Key)
				}
				value = &secretValue
			}

			variableMap[keyAndCategory] = Variable{
				Key:           v.Key,
				Value:         value,
				Category:      v.Category,
				NamespacePath: &v.NamespacePath,
				Sensitive:     v.Sensitive,
				VersionID:     &v.LatestVersionID,
			}
		}
	}

	variables := []Variable{}
	for _, v := range variableMap {
		variables = append(variables, v)
	}

	return variables, nil
}

// Get reads the variables stored for an existing run, optionally resolving
// sensitive values.
func (b *Builder) Get(ctx context.Context, run *models.Run, includeSensitiveValues bool) ([]Variable, error) {
	result, err := b.artifactStore.GetRunVariables(ctx, run)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run variables from object store")
	}
	defer result.Close()

	var variables []Variable
	if err := json.NewDecoder(result).Decode(&variables); err != nil {
		return nil, err
	}

	if includeSensitiveValues {
		runID := run.Metadata.ID

		// Extract variable version IDs for sensitive variables
		variableVersionIDs := make([]string, 0, len(variables))
		for _, v := range variables {
			if v.Sensitive {
				if v.VersionID == nil {
					return nil, errors.New("variable version ID is missing for sensitive variable %q in run %q", v.Key, runID)
				}
				variableVersionIDs = append(variableVersionIDs, *v.VersionID)
			}
		}

		if len(variableVersionIDs) > 0 {
			variableVersionsResp, err := b.dbClient.VariableVersions.GetVariableVersions(ctx, &db.GetVariableVersionsInput{
				Filter: &db.VariableVersionFilter{
					VariableVersionIDs: variableVersionIDs,
				},
			})
			if err != nil {
				return nil, errors.Wrap(err, "failed to query for variable versions associated with run %q", runID)
			}

			// Ensure that we received all the requested variable versions
			if len(variableVersionsResp.VariableVersions) != len(variableVersionIDs) {
				return nil, errors.New("some of the requested variable versions are missing for run %q", runID, errors.WithErrorCode(errors.ENotFound))
			}

			// Build map of secret values
			secretValues := make(map[string]string, len(variableVersionsResp.VariableVersions))
			for _, v := range variableVersionsResp.VariableVersions {
				value, err := b.secretManager.Get(ctx, v.Key, v.SecretData)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get secret value for variable version with ID %q", v.Metadata.ID)
				}
				secretValues[v.Metadata.ID] = value
			}

			// Populate sensitive variable values
			for i, v := range variables {
				if v.Sensitive {
					if value, ok := secretValues[*v.VersionID]; ok {
						variables[i].Value = &value
					} else {
						return nil, errors.New("failed to populate secret value for variable version %q because secret value was not found", *v.VersionID)
					}
				}
			}
		}
	}

	return variables, nil
}

// ModuleRegistryToken returns a token getter that resolves module-registry tokens
// from the run's environment variables.
func ModuleRegistryToken(envVars []Variable) registry.TokenGetterFunc {
	return func(_ context.Context, hostname string) (string, error) {
		seeking, err := module.BuildTokenEnvVar(hostname)
		if err == nil {
			for _, variable := range envVars {
				if variable.Key == seeking {
					return *variable.Value, nil
				}
			}
		}
		return "", nil
	}
}
