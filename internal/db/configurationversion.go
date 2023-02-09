package db

//go:generate mockery --name ConfigurationVersions --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// ConfigurationVersionSortableField represents the fields that a list of configuration versions can be sorted by
type ConfigurationVersionSortableField string

// ConfigurationVersionSortableField constants
const (
	ConfigurationVersionSortableFieldUpdatedAtAsc  ConfigurationVersionSortableField = "UPDATED_AT_ASC"
	ConfigurationVersionSortableFieldUpdatedAtDesc ConfigurationVersionSortableField = "UPDATED_AT_DESC"
)

func (sf ConfigurationVersionSortableField) getFieldDescriptor() *fieldDescriptor {
	switch sf {
	case ConfigurationVersionSortableFieldUpdatedAtAsc, ConfigurationVersionSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "configuration_versions", col: "updated_at"}
	default:
		return nil
	}
}

func (sf ConfigurationVersionSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return DescSort
	}
	return AscSort
}

// ConfigurationVersionFilter contains the supported fields for filtering ConfigurationVersion resources
type ConfigurationVersionFilter struct {
	ConfigurationVersionIDs []string
}

// GetConfigurationVersionsInput is the input for listing configuration versions
type GetConfigurationVersionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *ConfigurationVersionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *ConfigurationVersionFilter
}

// ConfigurationVersionsResult contains the response data and page information
type ConfigurationVersionsResult struct {
	PageInfo              *PageInfo
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
	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.ConfigurationVersionIDs != nil {
			ex["configuration_versions.id"] = input.Filter.ConfigurationVersionIDs
		}
	}

	query := dialect.From("configuration_versions").
		Select(configurationVersionFieldList...).
		Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "configuration_versions", col: "id"},
		sortBy,
		sortDirection,
		configurationVersionFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, c.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.ConfigurationVersion{}
	for rows.Next() {
		item, err := scanConfigurationVersion(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := ConfigurationVersionsResult{
		PageInfo:              rows.getPageInfo(),
		ConfigurationVersions: results,
	}

	return &result, nil
}

func (c *configurationVersions) GetConfigurationVersion(ctx context.Context, id string) (*models.ConfigurationVersion, error) {

	sql, args, err := dialect.From("configuration_versions").
		Prepared(true).
		Select(configurationVersionFieldList...).
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return nil, err
	}

	configurationVersion, err := scanConfigurationVersion(c.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		return nil, err
	}
	return configurationVersion, nil
}

func (c *configurationVersions) CreateConfigurationVersion(ctx context.Context, configurationVersion models.ConfigurationVersion) (*models.ConfigurationVersion, error) {
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
		return nil, err
	}

	createdConfigurationVersion, err := scanConfigurationVersion(c.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		c.dbClient.logger.Error(err)
		return nil, err
	}
	return createdConfigurationVersion, nil
}

func (c *configurationVersions) UpdateConfigurationVersion(ctx context.Context, configurationVersion models.ConfigurationVersion) (*models.ConfigurationVersion, error) {
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
		return nil, err
	}

	updatedConfigurationVersion, err := scanConfigurationVersion(c.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		c.dbClient.logger.Error(err)
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

func configurationVersionFieldResolver(key string, model interface{}) (string, error) {
	configurationVersion, ok := model.(*models.ConfigurationVersion)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected configurationVersion type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &configurationVersion.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}
