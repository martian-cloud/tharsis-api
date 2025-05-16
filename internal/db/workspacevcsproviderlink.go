package db

//go:generate go tool mockery --name WorkspaceVCSProviderLinks --inpackage --case underscore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// WorkspaceVCSProviderLinks encapsulates the logic to access workspace vcs provider links from the database.
type WorkspaceVCSProviderLinks interface {
	GetLinksByProviderID(ctx context.Context, providerID string) ([]models.WorkspaceVCSProviderLink, error)
	GetLinkByID(ctx context.Context, id string) (*models.WorkspaceVCSProviderLink, error)
	GetLinkByTRN(ctx context.Context, trn string) (*models.WorkspaceVCSProviderLink, error)
	GetLinkByWorkspaceID(ctx context.Context, workspaceID string) (*models.WorkspaceVCSProviderLink, error)
	CreateLink(ctx context.Context, link *models.WorkspaceVCSProviderLink) (*models.WorkspaceVCSProviderLink, error)
	UpdateLink(ctx context.Context, link *models.WorkspaceVCSProviderLink) (*models.WorkspaceVCSProviderLink, error)
	DeleteLink(ctx context.Context, provider *models.WorkspaceVCSProviderLink) error
}

type workspaceVCSProviderLinks struct {
	dbClient *Client
}

var workspaceVCSProviderLinksFieldList = append(
	metadataFieldList,
	"created_by",
	"workspace_id",
	"provider_id",
	"token_nonce",
	"repository_path",
	"auto_speculative_plan",
	"webhook_id",
	"module_directory",
	"branch",
	"tag_regex",
	"glob_patterns",
	"webhook_disabled",
)

// NewWorkspaceVCSProviderLinks returns an instance of the VCSProviderLinks interface.
func NewWorkspaceVCSProviderLinks(dbClient *Client) WorkspaceVCSProviderLinks {
	return &workspaceVCSProviderLinks{dbClient: dbClient}
}

func (wpl *workspaceVCSProviderLinks) GetLinksByProviderID(ctx context.Context, providerID string) ([]models.WorkspaceVCSProviderLink, error) {
	ctx, span := tracer.Start(ctx, "db.GetLinksByProviderID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("workspace_vcs_provider_links").
		Prepared(true).
		Select(wpl.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspace_vcs_provider_links.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(goqu.Ex{"workspace_vcs_provider_links.provider_id": providerID}).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	rows, err := wpl.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.WorkspaceVCSProviderLink{}
	for rows.Next() {
		item, err := scanLink(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, *item)
	}

	return results, nil
}

func (wpl *workspaceVCSProviderLinks) GetLinkByID(ctx context.Context, id string) (*models.WorkspaceVCSProviderLink, error) {
	ctx, span := tracer.Start(ctx, "db.GetLinkByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return wpl.getLink(ctx, goqu.Ex{"workspace_vcs_provider_links.id": id})
}

func (wpl *workspaceVCSProviderLinks) GetLinkByTRN(ctx context.Context, trn string) (*models.WorkspaceVCSProviderLink, error) {
	ctx, span := tracer.Start(ctx, "db.GetLinkByTRN")
	defer span.End()

	path, err := types.WorkspaceVCSProviderLinkModelType.ResourcePathFromTRN(trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to parse TRN")
		return nil, err
	}

	lastSlashIndex := strings.LastIndex(path, "/")
	if lastSlashIndex == -1 {
		return nil, errors.New("a workspace vcs provider link TRN must have the workspace path, and link GID separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return wpl.getLink(ctx, goqu.Ex{
		"workspace_vcs_provider_links.id": gid.FromGlobalID(path[lastSlashIndex+1:]),
		"namespaces.path":                 path[:lastSlashIndex],
	})
}

func (wpl *workspaceVCSProviderLinks) GetLinkByWorkspaceID(ctx context.Context, workspaceID string) (*models.WorkspaceVCSProviderLink, error) {
	ctx, span := tracer.Start(ctx, "db.GetLinkByWorkspaceID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return wpl.getLink(ctx, goqu.Ex{"workspace_vcs_provider_links.workspace_id": workspaceID})
}

func (wpl *workspaceVCSProviderLinks) CreateLink(ctx context.Context, link *models.WorkspaceVCSProviderLink) (*models.WorkspaceVCSProviderLink, error) {
	ctx, span := tracer.Start(ctx, "db.CreateLink")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	globPatternsJSON, err := json.Marshal(link.GlobPatterns)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal link glob patterns")
		return nil, err
	}

	sql, args, err := dialect.From("workspace_vcs_provider_links").
		Prepared(true).
		With("workspace_vcs_provider_links",
			dialect.Insert("workspace_vcs_provider_links").
				Rows(goqu.Record{
					"id":                    newResourceID(),
					"version":               initialResourceVersion,
					"created_at":            timestamp,
					"updated_at":            timestamp,
					"created_by":            link.CreatedBy,
					"workspace_id":          link.WorkspaceID,
					"provider_id":           link.ProviderID,
					"token_nonce":           link.TokenNonce,
					"repository_path":       link.RepositoryPath,
					"auto_speculative_plan": link.AutoSpeculativePlan,
					"webhook_id":            nullableString(link.WebhookID),
					"module_directory":      link.ModuleDirectory,
					"branch":                link.Branch,
					"tag_regex":             link.TagRegex,
					"glob_patterns":         globPatternsJSON,
					"webhook_disabled":      link.WebhookDisabled,
				}).Returning("*"),
		).Select(wpl.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("workspace_vcs_provider_links.workspace_id").Eq(goqu.I("namespaces.workspace_id")))).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdLink, err := scanLink(wpl.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil,
					"workspace is already linked with a vcs provider")
				return nil, errors.New("workspace is already linked with a vcs provider", errors.WithErrorCode(errors.EConflict))
			}

			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_workspace_id":
					tracing.RecordError(span, nil, "workspace does not exist")
					return nil, errors.New("workspace does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_provider_id":
					tracing.RecordError(span, nil, "vcs provider does not exist")
					return nil, errors.New("vcs provider does not exist", errors.WithErrorCode(errors.ENotFound))
				}
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdLink, nil
}

func (wpl *workspaceVCSProviderLinks) UpdateLink(ctx context.Context, link *models.WorkspaceVCSProviderLink) (*models.WorkspaceVCSProviderLink, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateLink")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	globPatternsJSON, err := json.Marshal(link.GlobPatterns)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal link glob patterns")
		return nil, err
	}

	sql, args, err := dialect.From("workspace_vcs_provider_links").
		Prepared(true).
		With("workspace_vcs_provider_links",
			dialect.Update("workspace_vcs_provider_links").
				Set(
					goqu.Record{
						"version":               goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":            timestamp,
						"auto_speculative_plan": link.AutoSpeculativePlan,
						"module_directory":      link.ModuleDirectory,
						"webhook_id":            nullableString(link.WebhookID),
						"branch":                link.Branch,
						"tag_regex":             link.TagRegex,
						"glob_patterns":         globPatternsJSON,
						"webhook_disabled":      link.WebhookDisabled,
					},
				).Where(goqu.Ex{"id": link.Metadata.ID, "version": link.Metadata.Version}).
				Returning("*"),
		).Select(wpl.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("workspace_vcs_provider_links.workspace_id").Eq(goqu.I("namespaces.workspace_id")))).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedLink, err := scanLink(wpl.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedLink, nil
}

func (wpl *workspaceVCSProviderLinks) DeleteLink(ctx context.Context, provider *models.WorkspaceVCSProviderLink) error {
	ctx, span := tracer.Start(ctx, "db.DeleteLink")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("workspace_vcs_provider_links").
		Prepared(true).
		With("workspace_vcs_provider_links",
			dialect.Delete("workspace_vcs_provider_links").
				Where(
					goqu.Ex{
						"id":      provider.Metadata.ID,
						"version": provider.Metadata.Version,
					},
				).Returning("*"),
		).Select(wpl.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("workspace_vcs_provider_links.workspace_id").Eq(goqu.I("namespaces.workspace_id")))).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanLink(wpl.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (wpl *workspaceVCSProviderLinks) getLink(ctx context.Context, exp goqu.Ex) (*models.WorkspaceVCSProviderLink, error) {
	query := dialect.From(goqu.T("workspace_vcs_provider_links")).
		Prepared(true).
		Select(wpl.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("workspace_vcs_provider_links.workspace_id").Eq(goqu.I("namespaces.workspace_id")))).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	link, err := scanLink(wpl.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		return nil, err
	}

	return link, nil
}

func (wpl *workspaceVCSProviderLinks) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range workspaceVCSProviderLinksFieldList {
		selectFields = append(selectFields, fmt.Sprintf("workspace_vcs_provider_links.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanLink(row scanner) (*models.WorkspaceVCSProviderLink, error) {
	var webhookID sql.NullString
	var workspacePath string

	wpl := &models.WorkspaceVCSProviderLink{}

	fields := []interface{}{
		&wpl.Metadata.ID,
		&wpl.Metadata.CreationTimestamp,
		&wpl.Metadata.LastUpdatedTimestamp,
		&wpl.Metadata.Version,
		&wpl.CreatedBy,
		&wpl.WorkspaceID,
		&wpl.ProviderID,
		&wpl.TokenNonce,
		&wpl.RepositoryPath,
		&wpl.AutoSpeculativePlan,
		&webhookID,
		&wpl.ModuleDirectory,
		&wpl.Branch,
		&wpl.TagRegex,
		&wpl.GlobPatterns,
		&wpl.WebhookDisabled,
		&workspacePath,
	}

	err := row.Scan(fields...)

	if err != nil {
		return nil, err
	}

	if webhookID.Valid {
		wpl.WebhookID = webhookID.String
	}

	wpl.Metadata.TRN = types.WorkspaceVCSProviderLinkModelType.BuildTRN(workspacePath, wpl.GetGlobalID())

	return wpl, nil
}
