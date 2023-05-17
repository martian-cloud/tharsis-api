package db

//go:generate mockery --name WorkspaceVCSProviderLinks --inpackage --case underscore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// WorkspaceVCSProviderLinks encapsulates the logic to access workspace vcs provider links from the database.
type WorkspaceVCSProviderLinks interface {
	GetLinksByProviderID(ctx context.Context, providerID string) ([]models.WorkspaceVCSProviderLink, error)
	GetLinkByID(ctx context.Context, id string) (*models.WorkspaceVCSProviderLink, error)
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

	sql, args, err := dialect.Insert("workspace_vcs_provider_links").
		Prepared(true).
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
		}).
		Returning(workspaceVCSProviderLinksFieldList...).ToSQL()
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
				return nil, errors.New(errors.EConflict, "workspace is already linked with a vcs provider")
			}

			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_workspace_id":
					tracing.RecordError(span, nil, "workspace does not exist")
					return nil, errors.New(errors.ENotFound, "workspace does not exist")
				case "fk_provider_id":
					tracing.RecordError(span, nil, "vcs provider does not exist")
					return nil, errors.New(errors.ENotFound, "vcs provider does not exist")
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

	sql, args, err := dialect.Update("workspace_vcs_provider_links").
		Prepared(true).
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
		Returning(workspaceVCSProviderLinksFieldList...).ToSQL()

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

	sql, args, err := dialect.Delete("workspace_vcs_provider_links").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      provider.Metadata.ID,
				"version": provider.Metadata.Version,
			},
		).Returning(workspaceVCSProviderLinksFieldList...).ToSQL()

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
		return nil, err
	}

	return link, nil
}

func (wpl *workspaceVCSProviderLinks) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range workspaceVCSProviderLinksFieldList {
		selectFields = append(selectFields, fmt.Sprintf("workspace_vcs_provider_links.%s", field))
	}

	return selectFields
}

func scanLink(row scanner) (*models.WorkspaceVCSProviderLink, error) {
	var webhookID sql.NullString

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
	}

	err := row.Scan(fields...)

	if err != nil {
		return nil, err
	}

	if webhookID.Valid {
		wpl.WebhookID = webhookID.String
	}

	return wpl, nil
}
