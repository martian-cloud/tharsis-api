package db

//go:generate go tool mockery --name TerraformProviderDocItems --inpackage --case underscore

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// providerDocItemInsertChunkSize bounds the number of rows per multi-row INSERT so a provider with many
// doc files is split into chunks that stay under the database's bind-parameter limit.
const providerDocItemInsertChunkSize = 1000

// TerraformProviderDocItems encapsulates the logic to access terraform provider doc items from the database
type TerraformProviderDocItems interface {
	GetProviderDocItems(ctx context.Context, input *GetProviderDocItemsInput) ([]*models.TerraformProviderDocItem, error)
	GetProviderDocItemByFilename(ctx context.Context, input *GetProviderDocItemByFilenameInput) (*models.TerraformProviderDocItem, error)
	CreateProviderDocItems(ctx context.Context, docItems []*models.TerraformProviderDocItem) error
}

// GetProviderDocItemsInput is the input for listing provider doc items.
type GetProviderDocItemsInput struct {
	// Category optionally filters the results to a single documentation category.
	Category *models.TerraformProviderDocItemCategory
	// ProviderVersionID is the provider version whose doc items are returned.
	ProviderVersionID string
}

// GetProviderDocItemByFilenameInput is the input for a point lookup of a single provider doc item.
type GetProviderDocItemByFilenameInput struct {
	// ProviderVersionID is the provider version the doc item belongs to.
	ProviderVersionID string
	// Category is the documentation category the doc item belongs to.
	Category models.TerraformProviderDocItemCategory
	// Filename is the doc item's filename within the category.
	Filename string
}

type terraformProviderDocItems struct {
	dbClient *Client
}

var providerDocItemFieldList = append(metadataFieldList, "provider_version_id", "filename", "category", "subcategory")

// NewTerraformProviderDocItems returns an instance of the TerraformProviderDocItems interface
func NewTerraformProviderDocItems(dbClient *Client) TerraformProviderDocItems {
	return &terraformProviderDocItems{dbClient: dbClient}
}

// GetProviderDocItems returns a provider version's doc items, optionally filtered to a single category.
func (t *terraformProviderDocItems) GetProviderDocItems(ctx context.Context, input *GetProviderDocItemsInput) ([]*models.TerraformProviderDocItem, error) {
	ctx, span := tracer.Start(ctx, "db.GetProviderDocItems")
	defer span.End()

	ex := goqu.Ex{"terraform_provider_doc_items.provider_version_id": input.ProviderVersionID}
	if input.Category != nil {
		ex["terraform_provider_doc_items.category"] = string(*input.Category)
	}

	query := dialect.From(goqu.T("terraform_provider_doc_items")).
		Prepared(true).
		Select(t.getSelectFields()...).
		Where(ex).
		Order(
			goqu.I("terraform_provider_doc_items.category").Asc(),
			goqu.I("terraform_provider_doc_items.filename").Asc(),
		)

	sql, args, err := toSQLWithTag("terraformproviderdocitem.GetProviderDocItems", query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	rows, err := t.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	results := []*models.TerraformProviderDocItem{}
	for rows.Next() {
		docItem, err := scanTerraformProviderDocItem(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}
		results = append(results, docItem)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read rows", errors.WithSpan(span))
	}

	return results, nil
}

// GetProviderDocItemByFilename returns a single doc item via a point lookup on the unique
// (provider_version_id, category, filename) index, or nil if no matching item exists.
func (t *terraformProviderDocItems) GetProviderDocItemByFilename(ctx context.Context, input *GetProviderDocItemByFilenameInput) (*models.TerraformProviderDocItem, error) {
	ctx, span := tracer.Start(ctx, "db.GetProviderDocItemByFilename")
	defer span.End()

	query := dialect.From(goqu.T("terraform_provider_doc_items")).
		Prepared(true).
		Select(t.getSelectFields()...).
		Where(goqu.Ex{
			"terraform_provider_doc_items.provider_version_id": input.ProviderVersionID,
			"terraform_provider_doc_items.category":            input.Category,
			"terraform_provider_doc_items.filename":            input.Filename,
		})

	sql, args, err := toSQLWithTag("terraformproviderdocitem.GetProviderDocItemByFilename", query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	docItem, err := scanTerraformProviderDocItem(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return docItem, nil
}

// CreateProviderDocItems inserts multiple doc items, chunked into multi-row INSERTs.
func (t *terraformProviderDocItems) CreateProviderDocItems(ctx context.Context, docItems []*models.TerraformProviderDocItem) error {
	ctx, span := tracer.Start(ctx, "db.CreateProviderDocItems")
	defer span.End()

	if len(docItems) == 0 {
		return nil
	}

	timestamp := currentTime()

	for start := 0; start < len(docItems); start += providerDocItemInsertChunkSize {
		end := min(start+providerDocItemInsertChunkSize, len(docItems))

		records := make([]goqu.Record, 0, end-start)
		for _, docItem := range docItems[start:end] {
			records = append(records, goqu.Record{
				"id":                  newResourceID(),
				"version":             initialResourceVersion,
				"created_at":          timestamp,
				"updated_at":          timestamp,
				"provider_version_id": docItem.ProviderVersionID,
				"filename":            docItem.Filename,
				"category":            docItem.Category,
				"subcategory":         docItem.Subcategory,
			})
		}

		sql, args, err := toSQLWithTag("terraformproviderdocitem.CreateProviderDocItems", dialect.Insert("terraform_provider_doc_items").
			Prepared(true).
			Rows(records))
		if err != nil {
			return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
		}

		if _, err = t.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
			if pgErr := asPgError(err); pgErr != nil && isUniqueViolation(pgErr) {
				return errors.New("terraform provider doc item already exists", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
			return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
		}
	}

	return nil
}

func (t *terraformProviderDocItems) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range providerDocItemFieldList {
		selectFields = append(selectFields, fmt.Sprintf("terraform_provider_doc_items.%s", field))
	}

	return selectFields
}

func scanTerraformProviderDocItem(row scanner) (*models.TerraformProviderDocItem, error) {
	docItem := &models.TerraformProviderDocItem{}

	fields := []interface{}{
		&docItem.Metadata.ID,
		&docItem.Metadata.CreationTimestamp,
		&docItem.Metadata.LastUpdatedTimestamp,
		&docItem.Metadata.Version,
		&docItem.ProviderVersionID,
		&docItem.Filename,
		&docItem.Category,
		&docItem.Subcategory,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	docItem.Metadata.TRN = trn.TypeTerraformProviderDocItem.Build(
		docItem.ProviderVersionID,
		string(docItem.Category),
		docItem.Filename,
	)

	return docItem, nil
}
