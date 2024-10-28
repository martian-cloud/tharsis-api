// Package workspace package
package workspace

//go:generate mockery --name ArtifactStore --inpackage --case underscore

import (
	"context"
	"fmt"
	"io"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// ArtifactStore interface encapsulates the logic for saving workspace artifacts
type ArtifactStore interface {
	DownloadConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion, writer io.WriterAt) error
	UploadConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion, body io.Reader) error
	GetConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion) (io.ReadCloser, error)
	DownloadStateVersion(ctx context.Context, stateVersion *models.StateVersion, writer io.WriterAt) error
	GetStateVersion(ctx context.Context, stateVersion *models.StateVersion) (io.ReadCloser, error)
	UploadStateVersion(ctx context.Context, stateVersion *models.StateVersion, body io.Reader) error
	DownloadPlanCache(ctx context.Context, run *models.Run, writer io.WriterAt) error
	UploadPlanCache(ctx context.Context, run *models.Run, body io.Reader) error
	UploadPlanJSON(ctx context.Context, run *models.Run, body io.Reader) error
	UploadPlanDiff(ctx context.Context, run *models.Run, body io.Reader) error
	GetPlanCache(ctx context.Context, run *models.Run) (io.ReadCloser, error)
	GetPlanJSON(ctx context.Context, run *models.Run) (io.ReadCloser, error)
	GetPlanDiff(ctx context.Context, run *models.Run) (io.ReadCloser, error)
	UploadRunVariables(ctx context.Context, run *models.Run, body io.Reader) error
	GetRunVariables(ctx context.Context, run *models.Run) (io.ReadCloser, error)
}

type artifactStore struct {
	objectStore objectstore.ObjectStore
}

// NewArtifactStore creates an instance of the ArtifactStore interface
func NewArtifactStore(objectStore objectstore.ObjectStore) ArtifactStore {
	return &artifactStore{objectStore: objectStore}
}

func (a *artifactStore) UploadConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion, body io.Reader) error {
	return a.upload(
		ctx,
		getConfigurationVersionObjectKey(configurationVersion),
		body,
	)
}

func (a *artifactStore) DownloadConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion, writer io.WriterAt) error {
	return a.download(
		ctx,
		getConfigurationVersionObjectKey(configurationVersion),
		writer,
	)
}

func (a *artifactStore) UploadStateVersion(ctx context.Context, stateVersion *models.StateVersion, body io.Reader) error {
	return a.upload(
		ctx,
		getStateVersionObjectKey(stateVersion),
		body,
	)
}

func (a *artifactStore) DownloadStateVersion(ctx context.Context, stateVersion *models.StateVersion, writer io.WriterAt) error {
	return a.download(
		ctx,
		getStateVersionObjectKey(stateVersion),
		writer,
	)
}

func (a *artifactStore) GetConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion) (io.ReadCloser, error) {
	return a.objectStore.GetObjectStream(
		ctx,
		getConfigurationVersionObjectKey(configurationVersion),
		nil,
	)
}

func (a *artifactStore) GetStateVersion(ctx context.Context, stateVersion *models.StateVersion) (io.ReadCloser, error) {
	return a.objectStore.GetObjectStream(
		ctx,
		getStateVersionObjectKey(stateVersion),
		nil,
	)
}

func (a *artifactStore) UploadPlanJSON(ctx context.Context, run *models.Run, body io.Reader) error {
	return a.upload(
		ctx,
		getPlanJSONObjectKey(run),
		body,
	)
}

func (a *artifactStore) GetPlanJSON(ctx context.Context, run *models.Run) (io.ReadCloser, error) {
	return a.objectStore.GetObjectStream(
		ctx,
		getPlanJSONObjectKey(run),
		nil,
	)
}

func (a *artifactStore) UploadPlanDiff(ctx context.Context, run *models.Run, body io.Reader) error {
	return a.upload(
		ctx,
		getPlanDiffObjectKey(run),
		body,
	)
}

func (a *artifactStore) GetPlanDiff(ctx context.Context, run *models.Run) (io.ReadCloser, error) {
	return a.objectStore.GetObjectStream(
		ctx,
		getPlanDiffObjectKey(run),
		nil,
	)
}

func (a *artifactStore) UploadPlanCache(ctx context.Context, run *models.Run, body io.Reader) error {
	return a.upload(
		ctx,
		getPlanCacheObjectKey(run),
		body,
	)
}

func (a *artifactStore) DownloadPlanCache(ctx context.Context, run *models.Run, writer io.WriterAt) error {
	return a.download(
		ctx,
		getPlanCacheObjectKey(run),
		writer,
	)
}

func (a *artifactStore) GetPlanCache(ctx context.Context, run *models.Run) (io.ReadCloser, error) {
	return a.objectStore.GetObjectStream(
		ctx,
		getPlanCacheObjectKey(run),
		nil,
	)
}

func (a *artifactStore) UploadRunVariables(ctx context.Context, run *models.Run, body io.Reader) error {
	return a.upload(
		ctx,
		getRunVariablesObjectKey(run),
		body,
	)
}

func (a *artifactStore) GetRunVariables(ctx context.Context, run *models.Run) (io.ReadCloser, error) {
	return a.objectStore.GetObjectStream(
		ctx,
		getRunVariablesObjectKey(run),
		nil,
	)
}

func (a *artifactStore) upload(ctx context.Context, key string, body io.Reader) error {
	return a.objectStore.UploadObject(ctx, key, body)
}

func (a *artifactStore) download(ctx context.Context, key string, writer io.WriterAt) error {
	return a.objectStore.DownloadObject(
		ctx,
		key,
		writer,
		nil,
	)
}

func getRunVariablesObjectKey(run *models.Run) string {
	return fmt.Sprintf("workspaces/%s/runs/%s/variables.json", run.WorkspaceID, run.Metadata.ID)
}

func getPlanCacheObjectKey(run *models.Run) string {
	return fmt.Sprintf("workspaces/%s/runs/%s/plan/%s", run.WorkspaceID, run.Metadata.ID, run.PlanID)
}

func getPlanJSONObjectKey(run *models.Run) string {
	return fmt.Sprintf("workspaces/%s/runs/%s/plan/%s.json", run.WorkspaceID, run.Metadata.ID, run.PlanID)
}

func getPlanDiffObjectKey(run *models.Run) string {
	return fmt.Sprintf("workspaces/%s/runs/%s/plan/diff_%s.json", run.WorkspaceID, run.Metadata.ID, run.PlanID)
}

func getConfigurationVersionObjectKey(configurationVersion *models.ConfigurationVersion) string {
	return fmt.Sprintf("workspaces/%s/configuration_versions/%s.tar.gz", configurationVersion.WorkspaceID, configurationVersion.Metadata.ID)
}

func getStateVersionObjectKey(stateVersion *models.StateVersion) string {
	return fmt.Sprintf("workspaces/%s/state_versions/%s.json", stateVersion.WorkspaceID, stateVersion.Metadata.ID)
}
