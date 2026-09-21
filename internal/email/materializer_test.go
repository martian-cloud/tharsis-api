package email

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

type materializerMocks struct {
	teamMembers  *db.MockTeamMembers
	users        *db.MockUsers
	suppressions *db.MockEmailSuppressions
	outbox       *db.MockEmailOutboxItems
	recipients   *db.MockEmailRecipients
	transactions *db.MockTransactions
}

func newMaterializerMocks(t *testing.T) *materializerMocks {
	return &materializerMocks{
		teamMembers:  db.NewMockTeamMembers(t),
		users:        db.NewMockUsers(t),
		suppressions: db.NewMockEmailSuppressions(t),
		outbox:       db.NewMockEmailOutboxItems(t),
		recipients:   db.NewMockEmailRecipients(t),
		transactions: db.NewMockTransactions(t),
	}
}

func (m *materializerMocks) newMaterializer(_ *testing.T) *Materializer {
	testLogger, _ := logger.NewForTest()
	return &Materializer{
		dbClient: &db.Client{
			TeamMembers:       m.teamMembers,
			Users:             m.users,
			EmailSuppressions: m.suppressions,
			EmailOutboxItems:  m.outbox,
			EmailRecipients:   m.recipients,
			Transactions:      m.transactions,
		},
		logger: testLogger,
	}
}

func emptySuppressions() *db.EmailSuppressionsResult {
	return &db.EmailSuppressionsResult{PageInfo: &pagination.PageInfo{HasNextPage: false}}
}

func TestMaterializerMaterializeBatch(t *testing.T) {
	t.Run("commits without work when no items are claimed", func(t *testing.T) {
		mocks := newMaterializerMocks(t)
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)
		mocks.outbox.On("ClaimOutboxItems", mock.Anything, mock.Anything).Return(nil, nil)

		m := mocks.newMaterializer(t)
		_, mErr := m.materializeBatch(t.Context())
		assert.NoError(t, mErr)
	})

	t.Run("materializes an explicit user selection and marks it ready", func(t *testing.T) {
		mocks := newMaterializerMocks(t)
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)

		item := &models.EmailOutboxItem{Status: models.EmailOutboxItemStatusPreparing, RecipientUserIDs: []string{"u1"}}
		item.Metadata.ID = "outbox-1"
		mocks.outbox.On("ClaimOutboxItems", mock.Anything, mock.Anything).Return([]*models.EmailOutboxItem{item}, nil)
		mocks.users.On("GetUsers", mock.Anything, mock.Anything).
			Return(&db.UsersResult{Users: []models.User{{Metadata: models.ResourceMetadata{ID: "u1"}, Email: "a@x.com"}}}, nil)
		mocks.suppressions.On("GetSuppressions", mock.Anything, mock.Anything).Return(emptySuppressions(), nil)
		mocks.recipients.On("CreateRecipients", mock.Anything, mock.MatchedBy(func(rs []*models.EmailRecipient) bool {
			return len(rs) == 1 && rs[0].Address == "a@x.com" && rs[0].EmailOutboxItemID == "outbox-1"
		})).Return(nil)
		mocks.outbox.On("UpdateOutboxItem", mock.Anything, mock.MatchedBy(func(o *models.EmailOutboxItem) bool {
			return o.Metadata.ID == "outbox-1" && o.Status == models.EmailOutboxItemStatusReady
		})).Return(item, nil)

		m := mocks.newMaterializer(t)
		_, mErr := m.materializeBatch(t.Context())
		assert.NoError(t, mErr)
	})

	t.Run("send-to-all queries every active user", func(t *testing.T) {
		mocks := newMaterializerMocks(t)
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)

		item := &models.EmailOutboxItem{Status: models.EmailOutboxItemStatusPreparing, SendToAllUsers: true}
		item.Metadata.ID = "outbox-1"
		mocks.outbox.On("ClaimOutboxItems", mock.Anything, mock.Anything).Return([]*models.EmailOutboxItem{item}, nil)
		// GetAllActiveUserIDs paginates users, then emailsForUserIDs looks them up.
		mocks.users.On("GetUsers", mock.Anything, mock.Anything).
			Return(&db.UsersResult{Users: []models.User{{Email: "a@x.com"}}, PageInfo: &pagination.PageInfo{HasNextPage: false}}, nil)
		mocks.suppressions.On("GetSuppressions", mock.Anything, mock.Anything).Return(emptySuppressions(), nil)
		mocks.recipients.On("CreateRecipients", mock.Anything, mock.Anything).Return(nil)
		mocks.outbox.On("UpdateOutboxItem", mock.Anything, mock.MatchedBy(func(o *models.EmailOutboxItem) bool {
			return o.Metadata.ID == "outbox-1" && o.Status == models.EmailOutboxItemStatusReady
		})).Return(item, nil)

		m := mocks.newMaterializer(t)
		_, mErr := m.materializeBatch(t.Context())
		assert.NoError(t, mErr)
	})

	t.Run("marks the item failed when resolution errors", func(t *testing.T) {
		mocks := newMaterializerMocks(t)
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)

		item := &models.EmailOutboxItem{Status: models.EmailOutboxItemStatusPreparing, RecipientUserIDs: []string{"u1"}}
		item.Metadata.ID = "outbox-1"
		mocks.outbox.On("ClaimOutboxItems", mock.Anything, mock.Anything).Return([]*models.EmailOutboxItem{item}, nil)
		mocks.users.On("GetUsers", mock.Anything, mock.Anything).Return(nil, errors.New("db down"))
		mocks.outbox.On("UpdateOutboxItem", mock.Anything, mock.MatchedBy(func(o *models.EmailOutboxItem) bool {
			return o.Metadata.ID == "outbox-1" && o.Status == models.EmailOutboxItemStatusFailed
		})).Return(item, nil)

		m := mocks.newMaterializer(t)
		_, mErr := m.materializeBatch(t.Context())
		assert.NoError(t, mErr)
	})

	t.Run("all recipients suppressed completes the item without creating rows", func(t *testing.T) {
		mocks := newMaterializerMocks(t)
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)

		item := &models.EmailOutboxItem{Status: models.EmailOutboxItemStatusPreparing, RecipientUserIDs: []string{"u1"}}
		item.Metadata.ID = "outbox-1"
		mocks.outbox.On("ClaimOutboxItems", mock.Anything, mock.Anything).Return([]*models.EmailOutboxItem{item}, nil)
		mocks.users.On("GetUsers", mock.Anything, mock.Anything).
			Return(&db.UsersResult{Users: []models.User{{Metadata: models.ResourceMetadata{ID: "u1"}, Email: "drop@x.com"}}}, nil)
		mocks.suppressions.On("GetSuppressions", mock.Anything, mock.Anything).
			Return(&db.EmailSuppressionsResult{
				Suppressions: []*models.EmailSuppression{{Address: "drop@x.com"}},
				PageInfo:     &pagination.PageInfo{HasNextPage: false},
			}, nil)
		// No CreateRecipients call; a zero-recipient item completes immediately rather than wedging in ready.
		mocks.outbox.On("UpdateOutboxItem", mock.Anything, mock.MatchedBy(func(o *models.EmailOutboxItem) bool {
			return o.Metadata.ID == "outbox-1" && o.Status == models.EmailOutboxItemStatusCompleted
		})).Return(item, nil)

		m := mocks.newMaterializer(t)
		_, mErr := m.materializeBatch(t.Context())
		assert.NoError(t, mErr)
	})
}

func TestMaterializerResolveRecipientEmails(t *testing.T) {
	t.Run("teams expand to members and dedupe against direct user IDs", func(t *testing.T) {
		mocks := newMaterializerMocks(t)
		mocks.teamMembers.On("GetTeamMembers", mock.Anything, mock.Anything).
			Return(&db.TeamMembersResult{TeamMembers: []models.TeamMember{{UserID: "u1"}, {UserID: "u2"}}}, nil)
		mocks.users.On("GetUsers", mock.Anything, mock.MatchedBy(func(in *db.GetUsersInput) bool {
			return in.Filter.Active && len(in.Filter.UserIDs) == 2
		})).Return(&db.UsersResult{Users: []models.User{
			{Metadata: models.ResourceMetadata{ID: "u1"}, Email: "a@x.com"},
			{Metadata: models.ResourceMetadata{ID: "u2"}, Email: "b@x.com"},
		}}, nil)
		mocks.suppressions.On("GetSuppressions", mock.Anything, mock.Anything).Return(emptySuppressions(), nil)

		m := mocks.newMaterializer(t)
		addrs, err := m.resolveRecipientEmails(t.Context(), &models.EmailOutboxItem{RecipientUserIDs: []string{"u1"}, RecipientTeamIDs: []string{"t1"}}, newRecipientCache())
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"a@x.com", "b@x.com"}, addrs)
	})

	t.Run("suppression matches case-insensitively against a differently-cased user email", func(t *testing.T) {
		mocks := newMaterializerMocks(t)
		mocks.users.On("GetUsers", mock.Anything, mock.Anything).
			Return(&db.UsersResult{Users: []models.User{{Metadata: models.ResourceMetadata{ID: "u1"}, Email: "Foo@Example.com"}}}, nil)
		// The suppression is stored lowercase; the mixed-case user email must still be dropped.
		mocks.suppressions.On("GetSuppressions", mock.Anything, mock.Anything).
			Return(&db.EmailSuppressionsResult{
				Suppressions: []*models.EmailSuppression{{Address: "foo@example.com"}},
				PageInfo:     &pagination.PageInfo{HasNextPage: false},
			}, nil)

		m := mocks.newMaterializer(t)
		addrs, err := m.resolveRecipientEmails(t.Context(), &models.EmailOutboxItem{RecipientUserIDs: []string{"u1"}}, newRecipientCache())
		require.NoError(t, err)
		assert.Empty(t, addrs)
	})
}
