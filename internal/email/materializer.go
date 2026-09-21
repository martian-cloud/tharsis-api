package email

import (
	"context"
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

var outboxItemsMaterialized = metric.NewCounter("email_outbox_items_materialized_total", "Number of email outbox items whose recipients were materialized.")

const (
	// Fallback poll interval; events wake the worker promptly, so this can stay long to spare an idle system.
	minMaterializeInterval = 10 * time.Minute
	maxMaterializeInterval = 15 * time.Minute
	// materializeBatchSize is how many pending outbox items one pass claims.
	materializeBatchSize = 20
	// activeUsersPageSize is how many active users are fetched per page for a send-to-all broadcast.
	activeUsersPageSize = 100
	// suppressionPageSize is how many suppression entries are fetched per page when filtering recipients.
	suppressionPageSize = 100
	// suppressionQueryChunkSize bounds how many addresses a single suppression query filters on, keeping the bind-parameter count well under Postgres's 65535 cap.
	suppressionQueryChunkSize = 1000
	// paginatedQueryPageDelay is a short pause between suppression pages so a large broadcast doesn't hammer the database.
	paginatedQueryPageDelay = 100 * time.Millisecond
)

// recipientCache memoizes resolution within a single batch pass. userEmails maps a user ID to its active
// email (empty string means looked up and found inactive/absent), so a user resolved for one item is O(1)
// for the next. The send-to-all address set is identical for every send-to-all item, so it is memoized too.
type recipientCache struct {
	userEmails         map[string]string
	sendToAllAddresses []string
	sendToAllResolved  bool
}

func newRecipientCache() *recipientCache {
	return &recipientCache{userEmails: map[string]string{}}
}

// Materializer is the background worker that resolves and stores an outbox item's recipient rows,
// then flips the item's recipient status from pending to ready (or failed).
type Materializer struct {
	dbClient *db.Client
	logger   logger.Logger
}

// NewMaterializer creates the recipient materialization worker.
func NewMaterializer(dbClient *db.Client, logger logger.Logger) *Materializer {
	return &Materializer{dbClient: dbClient, logger: logger}
}

// materializeBatch claims a batch of preparing items, resolves each one's recipients outside any
// transaction (the slow, paginated part), then writes each item's recipients and status in its own short
// transaction. Resolution is lock-free so a large broadcast never holds a row lock open across DB paging.
func (m *Materializer) materializeBatch(ctx context.Context) (bool, error) {
	items, err := m.claimBatch(ctx)
	if err != nil {
		return false, err
	}

	// Cache shared resolution across the batch so a burst of send-to-all items resolves the user set once.
	cache := newRecipientCache()
	for _, item := range items {
		if err := m.materializeItem(ctx, item, cache); err != nil {
			return false, err
		}
	}

	outboxItemsMaterialized.Add(float64(len(items)))

	// A full batch means more preparing items may remain; ask to re-run instead of waiting for the poll.
	return len(items) == materializeBatchSize, nil
}

// claimBatch locks and returns a batch of preparing items in a short transaction.
func (m *Materializer) claimBatch(ctx context.Context) ([]*models.EmailOutboxItem, error) {
	txContext, err := m.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, te.Wrap(err, "failed to begin materializer claim transaction")
	}

	defer func() {
		if txErr := m.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			m.logger.WithContextFields(ctx).Errorf("failed to roll back materializer claim transaction: %v", txErr)
		}
	}()

	items, err := m.dbClient.EmailOutboxItems.ClaimOutboxItems(txContext, &db.ClaimOutboxItemsInput{
		Status: new(models.EmailOutboxItemStatusPreparing),
		Limit:  materializeBatchSize,
	})
	if err != nil {
		return nil, te.Wrap(err, "failed to claim outbox items for materialization")
	}

	if err := m.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, te.Wrap(err, "failed to commit materializer claim transaction")
	}

	return items, nil
}

// materializeItem resolves an item's recipients outside any transaction (the slow, paginated part), then
// persists them and flips the item's status in one short transaction.
func (m *Materializer) materializeItem(ctx context.Context, item *models.EmailOutboxItem, cache *recipientCache) error {
	addresses, resolveErr := m.resolveRecipientEmails(ctx, item, cache)
	if resolveErr != nil {
		// Resolution runs before any write transaction and won't succeed on a later pass, so fail the item rather than looping.
		m.logger.WithContextFields(ctx).Errorw("failing email outbox item after resolution error",
			"outboxItemID", item.Metadata.ID,
			"error", resolveErr,
		)
	}

	sendAt := time.Now().UTC()
	if item.SendAt != nil {
		sendAt = *item.SendAt
	}

	recipients := make([]*models.EmailRecipient, len(addresses))
	for i, address := range addresses {
		recipients[i] = &models.EmailRecipient{
			EmailOutboxItemID: item.Metadata.ID,
			Address:           address,
			DeliveryStatus:    models.EmailDeliveryPending,
			AvailableAt:       sendAt,
		}
	}

	txContext, err := m.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return te.Wrap(err, "failed to begin materializer write transaction")
	}

	defer func() {
		if txErr := m.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			m.logger.WithContextFields(ctx).Errorf("failed to roll back materializer write transaction: %v", txErr)
		}
	}()

	switch {
	case resolveErr != nil:
		item.Status = models.EmailOutboxItemStatusFailed
	case len(recipients) == 0:
		// No recipients means nothing will complete this item off a recipient outcome, so complete it here or it leaks uncleaned.
		item.Status = models.EmailOutboxItemStatusCompleted
	default:
		item.Status = models.EmailOutboxItemStatusReady

		if err := m.dbClient.EmailRecipients.CreateRecipients(txContext, recipients); err != nil {
			return te.Wrap(err, "failed to create email recipients")
		}
	}

	// An OLE means another instance already materialized this item, which is the intended outcome.
	if _, err := m.dbClient.EmailOutboxItems.UpdateOutboxItem(txContext, item); err != nil {
		if te.ErrorCode(err) == te.EOptimisticLock {
			return nil
		}

		return te.Wrap(err, "failed to update outbox item status")
	}

	return m.dbClient.Transactions.CommitTx(txContext)
}

// resolveRecipientEmails turns the item's recipient spec into a deduplicated, suppression-filtered address
// list, excluding inactive users. It runs outside any transaction and uses the batch cache.
func (m *Materializer) resolveRecipientEmails(ctx context.Context, item *models.EmailOutboxItem, cache *recipientCache) ([]string, error) {
	if item.SendToAllUsers && cache.sendToAllResolved {
		return cache.sendToAllAddresses, nil
	}

	var userIDs []string
	if !item.SendToAllUsers {
		idSet := make(map[string]struct{}, len(item.RecipientUserIDs))
		for _, id := range item.RecipientUserIDs {
			idSet[id] = struct{}{}
		}

		if len(item.RecipientTeamIDs) > 0 {
			members, err := m.dbClient.TeamMembers.GetTeamMembers(ctx, &db.GetTeamMembersInput{
				Filter: &db.TeamMemberFilter{TeamIDs: item.RecipientTeamIDs},
			})
			if err != nil {
				return nil, te.Wrap(err, "failed to get team members")
			}

			for _, member := range members.TeamMembers {
				idSet[member.UserID] = struct{}{}
			}
		}

		if len(idSet) == 0 {
			return nil, nil
		}

		userIDs = make([]string, 0, len(idSet))
		for id := range idSet {
			userIDs = append(userIDs, id)
		}
	}

	addresses, err := m.getActiveUserEmails(ctx, cache, userIDs...)
	if err != nil {
		return nil, err
	}

	var filtered []string
	if len(addresses) > 0 {
		filtered, err = m.filterSuppressed(ctx, addresses)
		if err != nil {
			return nil, err
		}
	}

	if item.SendToAllUsers {
		cache.sendToAllAddresses = filtered
		cache.sendToAllResolved = true
	}

	return filtered, nil
}

// getActiveUserEmails returns active-user emails, using the cache for userID->email lookups. With no
// userIDs it pages every active user (a send-to-all broadcast), seeding the cache as it goes; otherwise it
// serves cached IDs directly and queries only the uncached ones in a single request.
func (m *Materializer) getActiveUserEmails(ctx context.Context, cache *recipientCache, userIDs ...string) ([]string, error) {
	if len(userIDs) == 0 {
		return m.pageAllActiveUsers(ctx, cache)
	}

	var uncached []string
	for _, id := range userIDs {
		if _, ok := cache.userEmails[id]; !ok {
			uncached = append(uncached, id)
		}
	}

	if len(uncached) > 0 {
		result, err := m.dbClient.Users.GetUsers(ctx, &db.GetUsersInput{
			Filter: &db.UserFilter{UserIDs: uncached, Active: true},
		})
		if err != nil {
			return nil, te.Wrap(err, "failed to get users")
		}

		found := make(map[string]string, len(result.Users))
		for i := range result.Users {
			found[result.Users[i].Metadata.ID] = result.Users[i].Email
		}

		// Cache every requested ID, including misses (inactive/absent) as "" so they aren't re-queried.
		for _, id := range uncached {
			cache.userEmails[id] = found[id]
		}
	}

	emails := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		if email := cache.userEmails[id]; email != "" {
			emails = append(emails, email)
		}
	}

	return emails, nil
}

// pageAllActiveUsers pages through every active user, returning their emails and seeding the cache.
func (m *Materializer) pageAllActiveUsers(ctx context.Context, cache *recipientCache) ([]string, error) {
	var (
		emails []string
		cursor *string
	)

	pageSize := int32(activeUsersPageSize)
	for {
		result, err := m.dbClient.Users.GetUsers(ctx, &db.GetUsersInput{
			Filter:            &db.UserFilter{Active: true},
			PaginationOptions: &pagination.Options{First: &pageSize, After: cursor},
		})
		if err != nil {
			return nil, te.Wrap(err, "failed to get users")
		}

		for i := range result.Users {
			emails = append(emails, result.Users[i].Email)
			cache.userEmails[result.Users[i].Metadata.ID] = result.Users[i].Email
		}

		if !result.PageInfo.HasNextPage {
			break
		}

		cursor, err = result.PageInfo.Cursor(&result.Users[len(result.Users)-1])
		if err != nil {
			return nil, te.Wrap(err, "failed to advance users cursor")
		}

		select {
		case <-time.After(paginatedQueryPageDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return emails, nil
}

// filterSuppressed returns the candidate addresses not on the suppression list; suppressions are stored lowercase, so candidates are lowercased before comparison.
func (m *Materializer) filterSuppressed(ctx context.Context, addresses []string) ([]string, error) {
	suppressed := map[string]struct{}{}

	// Chunk the address filter so a send-to-all broadcast stays well under Postgres's 65535 bind-parameter cap.
	for start := 0; start < len(addresses); start += suppressionQueryChunkSize {
		end := min(start+suppressionQueryChunkSize, len(addresses))

		if err := m.collectSuppressed(ctx, addresses[start:end], suppressed); err != nil {
			return nil, err
		}
	}

	filtered := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if _, ok := suppressed[strings.ToLower(address)]; !ok {
			filtered = append(filtered, address)
		}
	}

	return filtered, nil
}

// collectSuppressed pages the suppressions matching one address chunk into the lowercase suppressed set.
func (m *Materializer) collectSuppressed(ctx context.Context, chunk []string, suppressed map[string]struct{}) error {
	pageSize := int32(suppressionPageSize)
	var cursor *string
	for {
		result, err := m.dbClient.EmailSuppressions.GetSuppressions(ctx, &db.GetEmailSuppressionsInput{
			Filter:            &db.EmailSuppressionFilter{Addresses: chunk},
			PaginationOptions: &pagination.Options{First: &pageSize, After: cursor},
		})
		if err != nil {
			return te.Wrap(err, "failed to get suppressions")
		}

		for i := range result.Suppressions {
			suppressed[strings.ToLower(result.Suppressions[i].Address)] = struct{}{}
		}

		if !result.PageInfo.HasNextPage {
			break
		}

		cursor, err = result.PageInfo.Cursor(result.Suppressions[len(result.Suppressions)-1])
		if err != nil {
			return te.Wrap(err, "failed to advance suppressions cursor")
		}

		select {
		case <-time.After(paginatedQueryPageDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
