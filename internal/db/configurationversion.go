package db

//go:generate go tool mockery --name ConfigurationVersions --inpackage --case underscore

import (
	"context"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// ConfigurationVersionSortableField represents the fields that a list of configuration versions can be sorted by
type ConfigurationVersionSortableField string

// ConfigurationVersionSortableField constants
const (
	ConfigurationVersionSortableFieldUpdatedAtAsc  ConfigurationVersionSortableField = "UPDATED_AT_ASC"
	ConfigurationVersionSortableFieldUpdatedAtDesc ConfigurationVersionSortableField = "UPDATED_AT_DESC"
)

func (sf ConfigurationVersionSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case ConfigurationVersionSortableFieldUpdatedAtAsc, ConfigurationVersionSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "configuration_versions", Col: "updated_at"}
	default:
		return nil
	}
}

func (sf ConfigurationVersionSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// ConfigurationVersionFilter contains the supported fields for filtering ConfigurationVersion resources
type ConfigurationVersionFilter struct {
	TimeRangeStart          *time.Time
	WorkspaceID             *string
	ConfigurationVersionIDs []string
}

// GetConfigurationVersionsInput is the input for listing configuration versions
type GetConfigurationVersionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *ConfigurationVersionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *ConfigurationVersionFilter
}

// ConfigurationVersionsResult contains the response data and page information
type ConfigurationVersionsResult struct {
	PageInfo              *pagination.PageInfo
	ConfigurationVersions []models.ConfigurationVersion
}

// ConfigurationVersions encapsulates the logic to access configuration version from the database
type ConfigurationVersions interface {
	GetConfigurationVersions(ctx context.Context, input *GetConfigurationVersionsInput) (*ConfigurationVersionsResult, error)
	// GetConfigurationVersion returns a configuration version
	GetConfigurationVersion(ctx context.Context, id string) (*models.ConfigurationVersion, error)
	// CreateConfigurationVersion creates a new configuration version
	CreateConfigurationVersion(ctx context.Context, configurationVersion models.ConfigurationVersion) (*models.ConfigurationVersion, error)
	// UpdateConfigurationVersion updates a configuration version in the database
	UpdateConfigurationVersion(ctx context.Context, configurationVersion models.ConfigurationVersion) (*models.ConfigurationVersion, error)
}

type configurationVersions struct {
	dbClient *Client
}

var configurationVersionFieldList = append(
	metadataFieldList,
	"status",
	"speculative",
	"workspace_id",
	"created_by",
	"vcs_event_id",
)

// NewConfigurationVersions returns an instance of the ConfigurationVersions interface
func NewConfigurationVersions(dbClient *Client) ConfigurationVersions {
	return &configurationVersions{dbClient: dbClient}
}

func (c *configurationVersions) GetConfigurationVersions(ctx context.Context, input *GetConfigurationVersionsInput) (*ConfigurationVersionsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetConfigurationVersions")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.ConfigurationVersionIDs != nil {
			ex = ex.Append(goqu.I("configuration_versions.id").In(input.Filter.ConfigurationVersionIDs))
		}

		if input.Filter.WorkspaceID != nil {
			ex = ex.Append(goqu.I("configuration_versions.workspace_id").Eq(*input.Filter.WorkspaceID))
		}

		if input.Filter.TimeRangeStart != nil {
			// Must use UTC here otherwise, queries will return unexpected results.
			ex = ex.Append(goqu.I("configuration_versions.created_at").Gte(input.Filter.TimeRangeStart.UTC()))
		}
	}

	query := dialect.From("configuration_versions").
		Select(configurationVersionFieldList...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "configuration_versions", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, c.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.ConfigurationVersion{}
	for rows.Next() {
		item, err := scanConfigurationVersion(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		tracing.RecordError(span, err, "failed to finalize rows")
		return nil, err
	}

	result := ConfigurationVersionsResult{
		PageInfo:              rows.GetPageInfo(),
		ConfigurationVersions: results,
	}

	return &result, nil
}

func (c *configurationVersions) GetConfigurationVersion(ctx context.Context, id string) (*models.ConfigurationVersion, error) {
	ctx, span := tracer.Start(ctx, "db.GetConfigurationVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("configuration_versions").
		Prepared(true).
		Select(configurationVersionFieldList...).
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	configurationVersion, err := scanConfigurationVersion(c.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return configurationVersion, nil
}

func (c *configurationVersions) CreateConfigurationVersion(ctx context.Context, configurationVersion models.ConfigurationVersion) (*models.ConfigurationVersion, error) {
	ctx, span := tracer.Start(ctx, "db.CreateConfigurationVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("configuration_versions").
		Prepared(true).
		Rows(goqu.Record{
			"id":           newResourceID(),
			"version":      initialResourceVersion,
			"created_at":   timestamp,
			"updated_at":   timestamp,
			"status":       configurationVersion.Status,
			"speculative":  configurationVersion.Speculative,
			"workspace_id": configurationVersion.WorkspaceID,
			"created_by":   configurationVersion.CreatedBy,
			"vcs_event_id": configurationVersion.VCSEventID,
		}).
		Returning(configurationVersionFieldList...).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdConfigurationVersion, err := scanConfigurationVersion(c.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		c.dbClient.logger.Error(err)
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return createdConfigurationVersion, nil
}

func (c *configurationVersions) UpdateConfigurationVersion(ctx context.Context, configurationVersion models.ConfigurationVersion) (*models.ConfigurationVersion, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateConfigurationVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("configuration_versions").
		Prepared(true).
		Set(
			goqu.Record{
				"version":      goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":   timestamp,
				"status":       configurationVersion.Status,
				"speculative":  configurationVersion.Speculative,
				"workspace_id": configurationVersion.WorkspaceID,
			},
		).Where(goqu.Ex{"id": configurationVersion.Metadata.ID, "version": configurationVersion.Metadata.Version}).Returning(configurationVersionFieldList...).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedConfigurationVersion, err := scanConfigurationVersion(c.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		c.dbClient.logger.Error(err)
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return updatedConfigurationVersion, nil
}

func scanConfigurationVersion(row scanner) (*models.ConfigurationVersion, error) {
	configurationVersion := &models.ConfigurationVersion{}

	err := row.Scan(
		&configurationVersion.Metadata.ID,
		&configurationVersion.Metadata.CreationTimestamp,
		&configurationVersion.Metadata.LastUpdatedTimestamp,
		&configurationVersion.Metadata.Version,
		&configurationVersion.Status,
		&configurationVersion.Speculative,
		&configurationVersion.WorkspaceID,
		&configurationVersion.CreatedBy,
		&configurationVersion.VCSEventID,
	)
	if err != nil {
		return nil, err
	}

	return configurationVersion, nil
}
