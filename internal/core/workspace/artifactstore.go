// Package workspace provides core workspace functionality, including the artifact
// store for workspace and run artifacts (configuration versions, state versions,
// plan caches, and run variables).
package workspace

//go:generate go tool mockery --name ArtifactStore --inpackage --case underscore

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// ArtifactStore interface encapsulates the logic for saving workspace artifacts
type ArtifactStore interface {
	DownloadConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion, writer io.WriterAt) error
	UploadConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion, body io.Reader) (db.RetainObjectRefFunc, string, error)
	GetConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion) (io.ReadCloser, int64, error)
	DownloadStateVersion(ctx context.Context, stateVersion *models.StateVersion, writer io.WriterAt) error
	GetStateVersion(ctx context.Context, stateVersion *models.StateVersion) (io.ReadCloser, error)
	UploadStateVersion(ctx context.Context, stateVersion *models.StateVersion, body io.Reader) (db.RetainObjectRefFunc, string, error)
	DownloadPlanCache(ctx context.Context, run *models.Run, writer io.WriterAt) error
	UploadPlanCache(ctx context.Context, run *models.Run, body io.Reader) (db.RetainObjectRefFunc, string, error)
	UploadPlanJSON(ctx context.Context, run *models.Run, body io.Reader) (db.RetainObjectRefFunc, string, error)
	UploadPlanDiff(ctx context.Context, run *models.Run, body io.Reader) (db.RetainObjectRefFunc, string, error)
	GetPlanCache(ctx context.Context, run *models.Run) (io.ReadCloser, error)
	GetPlanJSON(ctx context.Context, run *models.Run) (io.ReadCloser, error)
	GetPlanDiff(ctx context.Context, run *models.Run) (io.ReadCloser, error)
	UploadRunVariables(ctx context.Context, run *models.Run, body io.Reader) (db.RetainObjectRefFunc, string, error)
	GetRunVariables(ctx context.Context, run *models.Run) (io.ReadCloser, error)
}

type artifactStore struct {
	objectStore     objectstore.ObjectStore
	objectStoreRefs db.ObjectStoreRefs
}

// NewArtifactStore creates an instance of the ArtifactStore interface
func NewArtifactStore(objectStore objectstore.ObjectStore, objectStoreRefs db.ObjectStoreRefs) ArtifactStore {
	return &artifactStore{objectStore: objectStore, objectStoreRefs: objectStoreRefs}
}

func (a *artifactStore) UploadConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion, body io.Reader) (db.RetainObjectRefFunc, string, error) {
	key := configurationVersionObjectKey(configurationVersion.WorkspaceID, uuid.New().String())
	if err := a.upload(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return a.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerConfigurationVersion, ownerID)
	}, key, nil
}

func (a *artifactStore) DownloadConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion, writer io.WriterAt) error {
	return a.download(ctx, configurationVersion.ObjectStoreKey, writer)
}

func (a *artifactStore) UploadStateVersion(ctx context.Context, stateVersion *models.StateVersion, body io.Reader) (db.RetainObjectRefFunc, string, error) {
	key := stateVersionObjectKey(stateVersion.WorkspaceID, uuid.New().String())
	if err := a.upload(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return a.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerStateVersion, ownerID)
	}, key, nil
}

func (a *artifactStore) DownloadStateVersion(ctx context.Context, stateVersion *models.StateVersion, writer io.WriterAt) error {
	return a.download(ctx, stateVersion.ObjectStoreKey, writer)
}

func (a *artifactStore) GetConfigurationVersion(ctx context.Context, configurationVersion *models.ConfigurationVersion) (io.ReadCloser, int64, error) {
	result, err := a.getObjectStream(ctx, configurationVersion.ObjectStoreKey)
	if err != nil {
		return nil, 0, err
	}
	return result.Body, result.ContentLength, nil
}

func (a *artifactStore) GetStateVersion(ctx context.Context, stateVersion *models.StateVersion) (io.ReadCloser, error) {
	result, err := a.getObjectStream(ctx, stateVersion.ObjectStoreKey)
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func (a *artifactStore) UploadPlanJSON(ctx context.Context, run *models.Run, body io.Reader) (db.RetainObjectRefFunc, string, error) {
	key := planJSONObjectKey(run.WorkspaceID, run.Metadata.ID, uuid.New().String())
	if err := a.upload(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return a.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerRun, ownerID)
	}, key, nil
}

func (a *artifactStore) GetPlanJSON(ctx context.Context, run *models.Run) (io.ReadCloser, error) {
	result, err := a.getObjectStream(ctx, ptr.ToString(run.Plan.JSONObjectStoreKey))
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func (a *artifactStore) UploadPlanDiff(ctx context.Context, run *models.Run, body io.Reader) (db.RetainObjectRefFunc, string, error) {
	key := planDiffObjectKey(run.WorkspaceID, run.Metadata.ID, uuid.New().String())
	if err := a.upload(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return a.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerRun, ownerID)
	}, key, nil
}

func (a *artifactStore) GetPlanDiff(ctx context.Context, run *models.Run) (io.ReadCloser, error) {
	result, err := a.getObjectStream(ctx, ptr.ToString(run.Plan.DiffObjectStoreKey))
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func (a *artifactStore) UploadPlanCache(ctx context.Context, run *models.Run, body io.Reader) (db.RetainObjectRefFunc, string, error) {
	key := planCacheObjectKey(run.WorkspaceID, run.Metadata.ID, run.Plan.GetID())
	if err := a.upload(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return a.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerRun, ownerID)
	}, key, nil
}

func (a *artifactStore) DownloadPlanCache(ctx context.Context, run *models.Run, writer io.WriterAt) error {
	return a.download(ctx, ptr.ToString(run.Plan.CacheObjectStoreKey), writer)
}

func (a *artifactStore) GetPlanCache(ctx context.Context, run *models.Run) (io.ReadCloser, error) {
	result, err := a.getObjectStream(ctx, ptr.ToString(run.Plan.CacheObjectStoreKey))
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func (a *artifactStore) UploadRunVariables(ctx context.Context, run *models.Run, body io.Reader) (db.RetainObjectRefFunc, string, error) {
	// Reuse the run's existing key on re-upload; mint a UUID key on first upload (the run ID isn't
	// known yet at creation, which is why the key isn't derived from it).
	var key string
	if run.VariablesObjectStoreKey != nil {
		key = *run.VariablesObjectStoreKey
	} else {
		key = runVariablesObjectKey(run.WorkspaceID, uuid.New().String())
	}

	if err := a.upload(ctx, key, body); err != nil {
		return nil, "", err
	}

	return func(ctx context.Context, ownerID string) error {
		return a.objectStoreRefs.LinkRef(ctx, key, db.ObjectStoreRefOwnerRun, ownerID)
	}, key, nil
}

func (a *artifactStore) GetRunVariables(ctx context.Context, run *models.Run) (io.ReadCloser, error) {
	result, err := a.getObjectStream(ctx, ptr.ToString(run.VariablesObjectStoreKey))
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func (a *artifactStore) upload(ctx context.Context, key string, body io.Reader) error {
	return a.objectStore.UploadObject(ctx, key, body)
}

func (a *artifactStore) download(ctx context.Context, key string, writer io.WriterAt) error {
	return a.objectStore.DownloadObject(ctx, key, writer, nil)
}

func (a *artifactStore) getObjectStream(ctx context.Context, key string) (*objectstore.GetObjectStreamOutput, error) {
	return a.objectStore.GetObjectStream(ctx, key, nil)
}

func configurationVersionObjectKey(workspaceID, id string) string {
	return fmt.Sprintf("workspaces/%s/configuration_versions/%s", workspaceID, id)
}

func stateVersionObjectKey(workspaceID, id string) string {
	return fmt.Sprintf("workspaces/%s/state_versions/%s", workspaceID, id)
}

func planJSONObjectKey(workspaceID, runID, id string) string {
	return fmt.Sprintf("workspaces/%s/runs/%s/plan/%s.json", workspaceID, runID, id)
}

func planDiffObjectKey(workspaceID, runID, id string) string {
	return fmt.Sprintf("workspaces/%s/runs/%s/plan/%s.diff", workspaceID, runID, id)
}

func planCacheObjectKey(workspaceID, runID, planID string) string {
	return fmt.Sprintf("workspaces/%s/runs/%s/plan/%s", workspaceID, runID, planID)
}

func runVariablesObjectKey(workspaceID, id string) string {
	return fmt.Sprintf("workspaces/%s/run_variables/%s.json", workspaceID, id)
}
