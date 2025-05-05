package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

//go:generate go tool mockery --name FederatedRegistries --inpackage --case underscore

// FederatedRegistries encapsulates the logic to access federatedRegistries from the database
type FederatedRegistries interface {
	GetFederatedRegistryByID(ctx context.Context, id string) (*models.FederatedRegistry, error)
	GetFederatedRegistries(ctx context.Context, input *GetFederatedRegistriesInput) (*FederatedRegistriesResult, error)
	CreateFederatedRegistry(ctx context.Context, federatedRegistry *models.FederatedRegistry) (*models.FederatedRegistry, error)
	UpdateFederatedRegistry(ctx context.Context, federatedRegistry *models.FederatedRegistry) (*models.FederatedRegistry, error)
	DeleteFederatedRegistry(ctx context.Context, federatedRegistry *models.FederatedRegistry) error
}

// FederatedRegistrySortableField represents the fields that a federatedRegistry can be sorted by
type FederatedRegistrySortableField string

// FederatedRegistrySortableField constants
const (
	FederatedRegistrySortableFieldUpdatedAtAsc  FederatedRegistrySortableField = "UPDATED_AT_ASC"
	FederatedRegistrySortableFieldUpdatedAtDesc FederatedRegistrySortableField = "UPDATED_AT_DESC"
)

func (js FederatedRegistrySortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch js {
	case FederatedRegistrySortableFieldUpdatedAtAsc, FederatedRegistrySortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "federated_registries", Col: "updated_at"}
	default:
		return nil
	}
}

func (js FederatedRegistrySortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(js), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// FederatedRegistryFilter contains the supported fields for filtering FederatedRegistry resources
type FederatedRegistryFilter struct {
	FederatedRegistryIDs []string
	Hostname             *string
	GroupID              *string
	GroupPaths           []string
}

// GetFederatedRegistriesInput is the input for listing federatedRegistries
type GetFederatedRegistriesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *FederatedRegistrySortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *FederatedRegistryFilter
}

// FederatedRegistriesResult contains the response data and page information
type FederatedRegistriesResult struct {
	PageInfo            *pagination.PageInfo
	FederatedRegistries []*models.FederatedRegistry
}

type federatedRegistries struct {
	dbClient *Client
}

var federatedRegistryFieldList = append(metadataFieldList, "hostname", "group_id", "audience")

// NewFederatedRegistries returns an instance of the FederatedRegistries interface
func NewFederatedRegistries(dbClient *Client) FederatedRegistries {
	return &federatedRegistries{dbClient: dbClient}
}

func (p *federatedRegistries) GetFederatedRegistryByID(ctx context.Context,
	id string,
) (*models.FederatedRegistry, error) {
	ctx, span := tracer.Start(ctx, "db.GetFederatedRegistryByID")
	defer span.End()

	return p.getFederatedRegistry(ctx, goqu.Ex{"federated_registries.id": id})
}

func (p *federatedRegistries) GetFederatedRegistries(ctx context.Context,
	input *GetFederatedRegistriesInput) (*FederatedRegistriesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetFederatedRegistries")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.FederatedRegistryIDs != nil {
			ex = ex.Append(goqu.I("federated_registries.id").In(input.Filter.FederatedRegistryIDs))
		}
		if input.Filter.Hostname != nil {
			ex = ex.Append(goqu.I("federated_registries.hostname").Eq(*input.Filter.Hostname))
		}
		if input.Filter.GroupID != nil {
			ex = ex.Append(goqu.I("federated_registries.group_id").Eq(*input.Filter.GroupID))
		}
		if len(input.Filter.GroupPaths) > 0 {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.GroupPaths))
		}
	}

	query := dialect.From(goqu.T("federated_registries")).
		Select(p.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"federated_registries.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "federated_registries", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, p.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []*models.FederatedRegistry{}
	for rows.Next() {
		item, err := scanFederatedRegistry(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, item)
	}

	if err := rows.Finalize(&results); err != nil {
		tracing.RecordError(span, err, "failed to finalize rows")
		return nil, err
	}

	result := FederatedRegistriesResult{
		PageInfo:            rows.GetPageInfo(),
		FederatedRegistries: results,
	}

	return &result, nil
}

func (p *federatedRegistries) CreateFederatedRegistry(ctx context.Context, input *models.FederatedRegistry) (*models.FederatedRegistry, error) {
	ctx, span := tracer.Start(ctx, "db.CreateFederatedRegistry")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("federated_registries").
		Prepared(true).
		Rows(goqu.Record{
			"id":         newResourceID(),
			"version":    initialResourceVersion,
			"created_at": timestamp,
			"updated_at": timestamp,
			"hostname":   input.Hostname,
			"group_id":   input.GroupID,
			"audience":   input.Audience,
		}).
		Returning(federatedRegistryFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdFederatedRegistry, err := scanFederatedRegistry(p.dbClient.getConnection(ctx).
		QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.Wrap(err,
					"federated registry with registry endpoint %s and group ID %s already exists",
					input.Hostname,
					gid.ToGlobalID(gid.FederatedRegistryType, input.GroupID),
					errors.WithErrorCode(errors.EConflict),
					errors.WithSpan(span))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdFederatedRegistry, nil
}

func (p *federatedRegistries) UpdateFederatedRegistry(ctx context.Context,
	federatedRegistry *models.FederatedRegistry) (*models.FederatedRegistry, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateFederatedRegistry")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("federated_registries").
		Prepared(true).
		Set(
			goqu.Record{
				"version":    goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at": timestamp,
				"hostname":   federatedRegistry.Hostname,
				"audience":   federatedRegistry.Audience,
			},
		).Where(goqu.Ex{"id": federatedRegistry.Metadata.ID, "version": federatedRegistry.Metadata.Version}).
		Returning(federatedRegistryFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedFederatedRegistry, err := scanFederatedRegistry(
		p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, errors.Wrap(pgErr, "invalid ID; %s", pgErr.Message, errors.WithSpan(span), errors.WithErrorCode(errors.EInvalid))
			}

			if isUniqueViolation(pgErr) {
				return nil, errors.New("federated registry with group ID  %s and registry endpoint %s already exists",
					gid.ToGlobalID(gid.FederatedRegistryType, federatedRegistry.GroupID),
					federatedRegistry.Hostname, errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedFederatedRegistry, nil
}

func (p *federatedRegistries) DeleteFederatedRegistry(ctx context.Context, input *models.FederatedRegistry) error {
	ctx, span := tracer.Start(ctx, "db.DeleteFederatedRegistry")
	defer span.End()

	sql, args, err := dialect.Delete("federated_registries").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      input.Metadata.ID,
				"version": input.Metadata.Version,
			},
		).Returning(federatedRegistryFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = scanFederatedRegistry(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return errors.Wrap(pgErr, "invalid ID; %s",
					pgErr.Message, errors.WithSpan(span), errors.WithErrorCode(errors.EInvalid))
			}
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (p *federatedRegistries) getFederatedRegistry(ctx context.Context, exp goqu.Ex) (*models.FederatedRegistry, error) {
	ctx, span := tracer.Start(ctx, "db.getFederatedRegistry")
	defer span.End()

	query := dialect.From(goqu.T("federated_registries")).
		Prepared(true).
		Select(federatedRegistryFieldList...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	federatedRegistry, err := scanFederatedRegistry(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, errors.Wrap(pgErr, "invalid ID; %s", pgErr.Message, errors.WithSpan(span), errors.WithErrorCode(errors.EInvalid))
			}
		}

		return nil, err
	}

	return federatedRegistry, nil
}

func (p *federatedRegistries) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range federatedRegistryFieldList {
		selectFields = append(selectFields, fmt.Sprintf("federated_registries.%s", field))
	}

	return selectFields
}

func scanFederatedRegistry(row scanner) (*models.FederatedRegistry, error) {
	federatedRegistry := &models.FederatedRegistry{}

	fields := []interface{}{
		&federatedRegistry.Metadata.ID,
		&federatedRegistry.Metadata.CreationTimestamp,
		&federatedRegistry.Metadata.LastUpdatedTimestamp,
		&federatedRegistry.Metadata.Version,
		&federatedRegistry.Hostname,
		&federatedRegistry.GroupID,
		&federatedRegistry.Audience,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	return federatedRegistry, nil
}
