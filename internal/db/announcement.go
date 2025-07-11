package db

//go:generate go tool mockery --name Announcements --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Announcements encapsulates the logic to access announcements from the database
type Announcements interface {
	GetAnnouncementByID(ctx context.Context, id string) (*models.Announcement, error)
	GetAnnouncementByTRN(ctx context.Context, trn string) (*models.Announcement, error)
	GetAnnouncements(ctx context.Context, input *GetAnnouncementsInput) (*AnnouncementsResult, error)
	CreateAnnouncement(ctx context.Context, announcement *models.Announcement) (*models.Announcement, error)
	UpdateAnnouncement(ctx context.Context, announcement *models.Announcement) (*models.Announcement, error)
	DeleteAnnouncement(ctx context.Context, announcement *models.Announcement) error
}

// AnnouncementSortableField represents the fields that announcements can be sorted by
type AnnouncementSortableField string

// AnnouncementSortableField constants
const (
	AnnouncementSortableFieldCreatedAtAsc  AnnouncementSortableField = "CREATED_AT_ASC"
	AnnouncementSortableFieldCreatedAtDesc AnnouncementSortableField = "CREATED_AT_DESC"
	AnnouncementSortableFieldStartTimeAsc  AnnouncementSortableField = "START_TIME_ASC"
	AnnouncementSortableFieldStartTimeDesc AnnouncementSortableField = "START_TIME_DESC"
)

func (as AnnouncementSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch as {
	case AnnouncementSortableFieldCreatedAtAsc, AnnouncementSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "announcements", Col: "created_at"}
	case AnnouncementSortableFieldStartTimeAsc, AnnouncementSortableFieldStartTimeDesc:
		return &pagination.FieldDescriptor{Key: "start_time", Table: "announcements", Col: "start_time"}
	default:
		return nil
	}
}

func (as AnnouncementSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(as), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// AnnouncementFilter contains the supported fields for filtering Announcement resources
type AnnouncementFilter struct {
	Active *bool
}

// GetAnnouncementsInput is the input for listing announcements
type GetAnnouncementsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *AnnouncementSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *AnnouncementFilter
}

// AnnouncementsResult contains the response data and page information
type AnnouncementsResult struct {
	PageInfo      *pagination.PageInfo
	Announcements []models.Announcement
}

type announcements struct {
	dbClient *Client
}

var announcementsFieldList = append(metadataFieldList, "message", "start_time", "end_time", "created_by", "type", "dismissible")

// NewAnnouncements returns an instance of the Announcements interface
func NewAnnouncements(dbClient *Client) Announcements {
	return &announcements{dbClient: dbClient}
}

func (a *announcements) GetAnnouncementByID(ctx context.Context, id string) (*models.Announcement, error) {
	ctx, span := tracer.Start(ctx, "db.GetAnnouncementByID")
	defer span.End()

	return a.getAnnouncement(ctx, goqu.Ex{"announcements.id": id})
}

func (a *announcements) GetAnnouncementByTRN(ctx context.Context, trn string) (*models.Announcement, error) {
	ctx, span := tracer.Start(ctx, "db.GetAnnouncementByTRN")
	defer span.End()

	path, err := types.AnnouncementModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	return a.getAnnouncement(ctx, goqu.Ex{"announcements.id": gid.FromGlobalID(path)})
}

func (a *announcements) GetAnnouncements(ctx context.Context, input *GetAnnouncementsInput) (*AnnouncementsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetAnnouncements")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.Active != nil && *input.Filter.Active {
			currentTime := time.Now().UTC()
			ex = ex.Append(
				goqu.I("announcements.start_time").Lte(currentTime),
				goqu.Or(
					goqu.I("announcements.end_time").IsNull(),
					goqu.I("announcements.end_time").Gte(currentTime),
				),
			)
		}
	}

	query := dialect.From(goqu.T("announcements")).
		Select(a.getSelectFields()...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "announcements", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, a.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	defer rows.Close()

	// Scan rows
	results := []models.Announcement{}
	for rows.Next() {
		item, aErr := scanAnnouncement(rows)
		if aErr != nil {
			return nil, errors.Wrap(aErr, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err = rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	result := AnnouncementsResult{
		PageInfo:      rows.GetPageInfo(),
		Announcements: results,
	}

	return &result, nil
}

func (a *announcements) CreateAnnouncement(ctx context.Context, announcement *models.Announcement) (*models.Announcement, error) {
	ctx, span := tracer.Start(ctx, "db.CreateAnnouncement")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("announcements").
		Prepared(true).
		With("announcements",
			dialect.Insert("announcements").Rows(
				goqu.Record{
					"id":          newResourceID(),
					"version":     initialResourceVersion,
					"created_at":  timestamp,
					"updated_at":  timestamp,
					"message":     announcement.Message,
					"start_time":  announcement.StartTime,
					"end_time":    announcement.EndTime,
					"created_by":  announcement.CreatedBy,
					"type":        announcement.Type,
					"dismissible": announcement.Dismissible,
				}).Returning("*"),
		).Select(a.getSelectFields()...).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdAnnouncement, err := scanAnnouncement(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return createdAnnouncement, nil
}

func (a *announcements) UpdateAnnouncement(ctx context.Context, announcement *models.Announcement) (*models.Announcement, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateAnnouncement")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("announcements").
		Prepared(true).
		With("announcements",
			dialect.Update("announcements").
				Set(goqu.Record{
					"version":     goqu.L("? + ?", goqu.C("version"), 1),
					"updated_at":  timestamp,
					"message":     announcement.Message,
					"start_time":  announcement.StartTime,
					"end_time":    announcement.EndTime,
					"type":        announcement.Type,
					"dismissible": announcement.Dismissible,
				}).Where(goqu.Ex{"id": announcement.Metadata.ID, "version": announcement.Metadata.Version}).
				Returning("*"),
		).Select(a.getSelectFields()...).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedAnnouncement, err := scanAnnouncement(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updatedAnnouncement, nil
}

func (a *announcements) DeleteAnnouncement(ctx context.Context, announcement *models.Announcement) error {
	ctx, span := tracer.Start(ctx, "db.DeleteAnnouncement")
	defer span.End()

	sql, args, err := dialect.From("announcements").
		Prepared(true).
		With("announcements",
			dialect.Delete("announcements").
				Where(goqu.Ex{"id": announcement.Metadata.ID, "version": announcement.Metadata.Version}).
				Returning("*"),
		).Select(a.getSelectFields()...).
		ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	_, err = scanAnnouncement(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (a *announcements) getAnnouncement(ctx context.Context, exp goqu.Ex) (*models.Announcement, error) {
	ctx, span := tracer.Start(ctx, "db.getAnnouncement")
	defer span.End()

	query := dialect.From(goqu.T("announcements")).
		Prepared(true).
		Select(a.getSelectFields()...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	announcement, err := scanAnnouncement(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return announcement, nil
}

func (*announcements) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range announcementsFieldList {
		selectFields = append(selectFields, fmt.Sprintf("announcements.%s", field))
	}

	return selectFields
}

func scanAnnouncement(row scanner) (*models.Announcement, error) {
	announcement := &models.Announcement{}

	fields := []interface{}{
		&announcement.Metadata.ID,
		&announcement.Metadata.CreationTimestamp,
		&announcement.Metadata.LastUpdatedTimestamp,
		&announcement.Metadata.Version,
		&announcement.Message,
		&announcement.StartTime,
		&announcement.EndTime,
		&announcement.CreatedBy,
		&announcement.Type,
		&announcement.Dismissible,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	announcement.Metadata.TRN = types.AnnouncementModelType.BuildTRN(announcement.GetGlobalID())

	return announcement, nil
}
