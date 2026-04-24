package db

//go:generate go tool mockery --name AgentSessionMessages --inpackage --case underscore

import (
	"context"
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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// AgentSessionMessages encapsulates the logic to access agent session messages from the database
type AgentSessionMessages interface {
	GetAgentSessionMessageByID(ctx context.Context, sessionID string, id string) (*models.AgentSessionMessage, error)
	GetAgentSessionMessages(ctx context.Context, input *GetAgentSessionMessagesInput) (*AgentSessionMessagesResult, error)
	CreateAgentSessionMessage(ctx context.Context, msg *models.AgentSessionMessage) (*models.AgentSessionMessage, error)
}

// AgentSessionMessageSortableField represents the fields that a message can be sorted by
type AgentSessionMessageSortableField string

// AgentSessionMessageSortableField constants
const (
	AgentSessionMessageSortableFieldCreatedAtAsc  AgentSessionMessageSortableField = "CREATED_AT_ASC"
	AgentSessionMessageSortableFieldCreatedAtDesc AgentSessionMessageSortableField = "CREATED_AT_DESC"
)

func (sf AgentSessionMessageSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case AgentSessionMessageSortableFieldCreatedAtAsc, AgentSessionMessageSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "agent_session_messages", Col: "created_at"}
	default:
		return nil
	}
}

func (sf AgentSessionMessageSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// AgentSessionMessageFilter contains the supported fields for filtering agent session messages.
type AgentSessionMessageFilter struct {
	SessionID *string
	RunID     *string
}

// GetAgentSessionMessagesInput is the input for listing agent session messages.
type GetAgentSessionMessagesInput struct {
	Sort              *AgentSessionMessageSortableField
	PaginationOptions *pagination.Options
	Filter            *AgentSessionMessageFilter
}

// AgentSessionMessagesResult contains the response data and page information.
type AgentSessionMessagesResult struct {
	PageInfo             *pagination.PageInfo
	AgentSessionMessages []models.AgentSessionMessage
}

type agentSessionMessages struct {
	dbClient *Client
}

var agentSessionMessageFieldList = append(metadataFieldList, "session_id", "run_id", "parent_id", "role", "content")

// NewAgentSessionMessages returns an instance of the AgentSessionMessages interface
func NewAgentSessionMessages(dbClient *Client) AgentSessionMessages {
	return &agentSessionMessages{dbClient: dbClient}
}

func (a *agentSessionMessages) GetAgentSessionMessageByID(ctx context.Context, sessionID string, id string) (*models.AgentSessionMessage, error) {
	ctx, span := tracer.Start(ctx, "db.GetAgentSessionMessageByID")
	defer span.End()

	sql, args, err := dialect.From(goqu.T("agent_session_messages")).
		Prepared(true).
		Select(a.getSelectFields()...).
		Where(goqu.Ex{
			"agent_session_messages.id":         id,
			"agent_session_messages.session_id": sessionID,
		}).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	msg, err := scanAgentSessionMessage(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return msg, nil
}

func (a *agentSessionMessages) GetAgentSessionMessages(ctx context.Context, input *GetAgentSessionMessagesInput) (*AgentSessionMessagesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetAgentSessionMessages")
	defer span.End()

	ex := goqu.Ex{}
	if input.Filter != nil {
		if input.Filter.SessionID != nil {
			ex["agent_session_messages.session_id"] = *input.Filter.SessionID
		}
		if input.Filter.RunID != nil {
			ex["agent_session_messages.run_id"] = *input.Filter.RunID
		}
	}

	query := dialect.From("agent_session_messages").
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
		&pagination.FieldDescriptor{Key: "id", Table: "agent_session_messages", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, a.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	defer rows.Close()

	results := []models.AgentSessionMessage{}
	for rows.Next() {
		item, err := scanAgentSessionMessage(rows)
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

	return &AgentSessionMessagesResult{
		PageInfo:             rows.GetPageInfo(),
		AgentSessionMessages: results,
	}, nil
}

func (a *agentSessionMessages) CreateAgentSessionMessage(ctx context.Context, msg *models.AgentSessionMessage) (*models.AgentSessionMessage, error) {
	ctx, span := tracer.Start(ctx, "db.CreateAgentSessionMessage")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("agent_session_messages").Prepared(true).Rows(
		goqu.Record{
			"id":         newResourceID(),
			"version":    initialResourceVersion,
			"created_at": timestamp,
			"updated_at": timestamp,
			"session_id": msg.SessionID,
			"run_id":     msg.RunID,
			"parent_id":  msg.ParentID,
			"role":       msg.Role,
			"content":    nullableRawJSON(msg.Content),
		},
	).Returning(agentSessionMessageFieldList...).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	created, err := scanAgentSessionMessage(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("invalid session ID", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
			}
			if isInvalidIDViolation(pgErr) {
				return nil, errors.Wrap(pgErr, "invalid ID; %s", pgErr.Message, errors.WithSpan(span), errors.WithErrorCode(errors.EInvalid))
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return created, nil
}

func (*agentSessionMessages) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range agentSessionMessageFieldList {
		selectFields = append(selectFields, fmt.Sprintf("agent_session_messages.%s", field))
	}
	return selectFields
}

func scanAgentSessionMessage(row scanner) (*models.AgentSessionMessage, error) {
	msg := &models.AgentSessionMessage{}

	fields := []interface{}{
		&msg.Metadata.ID,
		&msg.Metadata.CreationTimestamp,
		&msg.Metadata.LastUpdatedTimestamp,
		&msg.Metadata.Version,
		&msg.SessionID,
		&msg.RunID,
		&msg.ParentID,
		&msg.Role,
		&msg.Content,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	msg.Metadata.TRN = types.AgentSessionMessageModelType.BuildTRN(
		gid.ToGlobalID(types.AgentSessionModelType, msg.SessionID),
		gid.ToGlobalID(types.AgentSessionRunModelType, msg.RunID),
		msg.GetGlobalID(),
	)

	return msg, nil
}

func nullableRawJSON(data json.RawMessage) interface{} {
	if data == nil {
		return nil
	}
	return []byte(data)
}
