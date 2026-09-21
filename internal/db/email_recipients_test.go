//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// retryableDeliveryStatuses mirrors EmailDeliveryStatus.IsRetryableStatus for claim-query tests.
var retryableDeliveryStatuses = []models.EmailDeliveryStatus{
	models.EmailDeliveryPending,
	models.EmailDeliverySoftBounced,
	models.EmailDeliveryDelayed,
	models.EmailDeliveryAccepted,
}

// getValue implements the sortableField interface for EmailRecipientSortableField.
func (sf EmailRecipientSortableField) getValue() string {
	return string(sf)
}

// createTestRecipient creates one recipient under outbox and returns the persisted row.
func createTestRecipient(ctx context.Context, t *testing.T, testClient *testClient, recipient *models.EmailRecipient) *models.EmailRecipient {
	if recipient.DeliveryStatus == "" {
		recipient.DeliveryStatus = models.EmailDeliveryPending
	}
	if recipient.AvailableAt.IsZero() {
		recipient.AvailableAt = time.Now().UTC()
	}

	require.NoError(t, testClient.client.EmailRecipients.CreateRecipients(ctx, []*models.EmailRecipient{recipient}))

	result, err := testClient.client.EmailRecipients.GetRecipients(ctx, &GetEmailRecipientsInput{
		Filter: &EmailRecipientFilter{EmailOutboxItemID: ptr.String(recipient.EmailOutboxItemID), Search: ptr.String(recipient.Address)},
	})
	require.NoError(t, err)
	require.Len(t, result.Recipients, 1)

	created := result.Recipients[0]
	return &created
}

func TestGetRecipientByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	outbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{})
	recipient := createTestRecipient(ctx, t, testClient, &models.EmailRecipient{EmailOutboxItemID: outbox.Metadata.ID, Address: "byid@x.com"})

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectRecipient bool
	}

	testCases := []testCase{
		{
			name:            "get resource by id",
			id:              recipient.Metadata.ID,
			expectRecipient: true,
		},
		{
			name: "resource with id not found",
			id:   nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			got, err := testClient.client.EmailRecipients.GetRecipientByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRecipient {
				require.NotNil(t, got)
				assert.Equal(t, test.id, got.Metadata.ID)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestGetRecipientByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	outbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{})
	recipient := createTestRecipient(ctx, t, testClient, &models.EmailRecipient{EmailOutboxItemID: outbox.Metadata.ID, Address: "bytrn@x.com"})

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectRecipient bool
	}

	testCases := []testCase{
		{
			name:            "get resource by TRN",
			trn:             recipient.Metadata.TRN,
			expectRecipient: true,
		},
		{
			name: "resource with TRN not found",
			trn:  trn.TypeEmailRecipient.Build(nonExistentGlobalID),
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			got, err := testClient.client.EmailRecipients.GetRecipientByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRecipient {
				require.NotNil(t, got)
				assert.Equal(t, test.trn, got.Metadata.TRN)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestCreateRecipients(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	outbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{})

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		recipients      []*models.EmailRecipient
		expectCount     int
	}

	testCases := []testCase{
		{
			name:       "empty slice is a no-op",
			recipients: nil,
		},
		{
			name: "successfully create recipients for an existing outbox",
			recipients: []*models.EmailRecipient{
				{EmailOutboxItemID: outbox.Metadata.ID, Address: "a@x.com", DeliveryStatus: models.EmailDeliveryPending, AvailableAt: time.Now().UTC()},
				{EmailOutboxItemID: outbox.Metadata.ID, Address: "b@x.com", DeliveryStatus: models.EmailDeliveryPending, AvailableAt: time.Now().UTC()},
			},
			expectCount: 2,
		},
		{
			name: "unknown outbox violates the foreign key",
			recipients: []*models.EmailRecipient{
				{EmailOutboxItemID: nonExistentID, Address: "c@x.com", DeliveryStatus: models.EmailDeliveryPending, AvailableAt: time.Now().UTC()},
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.EmailRecipients.CreateRecipients(ctx, test.recipients)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectCount > 0 {
				result, gErr := testClient.client.EmailRecipients.GetRecipients(ctx, &GetEmailRecipientsInput{
					Filter: &EmailRecipientFilter{EmailOutboxItemID: ptr.String(outbox.Metadata.ID)},
				})
				require.NoError(t, gErr)
				assert.Equal(t, test.expectCount, len(result.Recipients))
			}
		})
	}
}

func TestUpdateRecipient(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	outbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{})
	recipient := createTestRecipient(ctx, t, testClient, &models.EmailRecipient{EmailOutboxItemID: outbox.Metadata.ID, Address: "update@x.com"})

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		status          models.EmailDeliveryStatus
	}

	testCases := []testCase{
		{
			name:    "successfully update mutable fields",
			version: recipient.Metadata.Version,
			status:  models.EmailDeliveryCompleted,
		},
		{
			name:            "update fails because resource version doesn't match",
			version:         -1,
			status:          models.EmailDeliveryFailed,
			expectErrorCode: errors.EOptimisticLock,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			toUpdate := *recipient
			toUpdate.Metadata.Version = test.version
			toUpdate.DeliveryStatus = test.status

			updated, err := testClient.client.EmailRecipients.UpdateRecipient(ctx, &toUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, updated)
			assert.Equal(t, test.status, updated.DeliveryStatus)
			assert.Equal(t, test.version+1, updated.Metadata.Version)
		})
	}
}

func TestGetRecipients(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	outbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{})
	otherOutboxItem := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{Subject: "other"})

	require.NoError(t, testClient.client.EmailRecipients.CreateRecipients(ctx, []*models.EmailRecipient{
		{EmailOutboxItemID: outbox.Metadata.ID, Address: "pending@x.com", DeliveryStatus: models.EmailDeliveryPending, AvailableAt: time.Now().UTC()},
		{EmailOutboxItemID: outbox.Metadata.ID, Address: "completed@x.com", DeliveryStatus: models.EmailDeliveryCompleted, AvailableAt: time.Now().UTC()},
		{EmailOutboxItemID: otherOutboxItem.Metadata.ID, Address: "elsewhere@x.com", DeliveryStatus: models.EmailDeliveryPending, AvailableAt: time.Now().UTC()},
	}))

	opened := createTestRecipient(ctx, t, testClient, &models.EmailRecipient{EmailOutboxItemID: outbox.Metadata.ID, Address: "opened@x.com"})
	openedNow := time.Now().UTC()
	opened.OpenedAt = &openedNow
	_, err := testClient.client.EmailRecipients.UpdateRecipient(ctx, opened)
	require.NoError(t, err)

	bounced := createTestRecipient(ctx, t, testClient, &models.EmailRecipient{EmailOutboxItemID: outbox.Metadata.ID, Address: "bounced@x.com"})
	bounced.DeliveryStatus = models.EmailDeliveryHardBounced
	_, err = testClient.client.EmailRecipients.UpdateRecipient(ctx, bounced)
	require.NoError(t, err)

	type testCase struct {
		filter            *EmailRecipientFilter
		name              string
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name:              "no filter returns all recipients",
			expectResultCount: 5,
		},
		{
			name:              "filter by outbox",
			filter:            &EmailRecipientFilter{EmailOutboxItemID: ptr.String(outbox.Metadata.ID)},
			expectResultCount: 4,
		},
		{
			name:              "filter by delivery status",
			filter:            &EmailRecipientFilter{EmailOutboxItemID: ptr.String(outbox.Metadata.ID), DeliveryStatuses: []models.EmailDeliveryStatus{models.EmailDeliveryCompleted}},
			expectResultCount: 1,
		},
		{
			name:              "filter by multiple delivery statuses",
			filter:            &EmailRecipientFilter{EmailOutboxItemID: ptr.String(outbox.Metadata.ID), DeliveryStatuses: []models.EmailDeliveryStatus{models.EmailDeliveryPending, models.EmailDeliveryCompleted}},
			expectResultCount: 3,
		},
		{
			name:              "search by address substring",
			filter:            &EmailRecipientFilter{EmailOutboxItemID: ptr.String(outbox.Metadata.ID), Search: ptr.String("pending@")},
			expectResultCount: 1,
		},
		{
			name:              "empty search is ignored",
			filter:            &EmailRecipientFilter{EmailOutboxItemID: ptr.String(outbox.Metadata.ID), Search: ptr.String("")},
			expectResultCount: 4,
		},
		{
			name:              "filter by has opened",
			filter:            &EmailRecipientFilter{EmailOutboxItemID: ptr.String(outbox.Metadata.ID), HasOpened: ptr.Bool(true)},
			expectResultCount: 1,
		},
		{
			name:              "filter by has not opened",
			filter:            &EmailRecipientFilter{EmailOutboxItemID: ptr.String(outbox.Metadata.ID), HasOpened: ptr.Bool(false)},
			expectResultCount: 3,
		},
		{
			name:              "filter by has issues",
			filter:            &EmailRecipientFilter{EmailOutboxItemID: ptr.String(outbox.Metadata.ID), HasIssues: ptr.Bool(true)},
			expectResultCount: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.EmailRecipients.GetRecipients(ctx, &GetEmailRecipientsInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectResultCount, len(result.Recipients))
		})
	}
}

func TestClaimRecipients(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	outbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{})

	past := time.Now().UTC().Add(-time.Hour)
	future := time.Now().UTC().Add(time.Hour)

	require.NoError(t, testClient.client.EmailRecipients.CreateRecipients(ctx, []*models.EmailRecipient{
		{EmailOutboxItemID: outbox.Metadata.ID, Address: "due-pending@x.com", DeliveryStatus: models.EmailDeliveryPending, AvailableAt: past},
		{EmailOutboxItemID: outbox.Metadata.ID, Address: "due-softbounce@x.com", DeliveryStatus: models.EmailDeliverySoftBounced, AvailableAt: past},
		{EmailOutboxItemID: outbox.Metadata.ID, Address: "future@x.com", DeliveryStatus: models.EmailDeliveryPending, AvailableAt: future},
		{EmailOutboxItemID: outbox.Metadata.ID, Address: "accepted-in-window@x.com", DeliveryStatus: models.EmailDeliveryAccepted, AvailableAt: future},
	}))

	t.Run("empty statuses claims nothing", func(t *testing.T) {
		claimed, err := testClient.client.EmailRecipients.ClaimRecipients(ctx, &ClaimRecipientsInput{Limit: 10})
		require.NoError(t, err)
		assert.Empty(t, claimed)
	})

	t.Run("zero limit claims nothing", func(t *testing.T) {
		claimed, err := testClient.client.EmailRecipients.ClaimRecipients(ctx, &ClaimRecipientsInput{
			Statuses: retryableDeliveryStatuses,
			Limit:    0,
		})
		require.NoError(t, err)
		assert.Empty(t, claimed)
	})

	t.Run("respects the limit", func(t *testing.T) {
		claimed, err := testClient.client.EmailRecipients.ClaimRecipients(ctx, &ClaimRecipientsInput{
			Statuses: retryableDeliveryStatuses,
			Limit:    1,
		})
		require.NoError(t, err)
		assert.Len(t, claimed, 1)
		// Claiming advances the attempt count and records the attempt time.
		assert.Equal(t, 1, claimed[0].AttemptCount)
		require.NotNil(t, claimed[0].LastAttemptAt)
	})

	t.Run("claims the remaining due retryable recipient, skipping those not yet due", func(t *testing.T) {
		// "respects the limit" already claimed one due retryable recipient (pushing its available_at into the
		// future), leaving one due retryable recipient; future@ and accepted-in-window@ are not yet due.
		claimed, err := testClient.client.EmailRecipients.ClaimRecipients(ctx, &ClaimRecipientsInput{
			Statuses: retryableDeliveryStatuses,
			Limit:    10,
		})
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		assert.Contains(t, []string{"due-pending@x.com", "due-softbounce@x.com"}, claimed[0].Address)
	})

	t.Run("claims an accepted recipient once its feedback window has elapsed", func(t *testing.T) {
		// available_at in the past simulates the feedback window having elapsed with no delivery feedback.
		require.NoError(t, testClient.client.EmailRecipients.CreateRecipients(ctx, []*models.EmailRecipient{
			{EmailOutboxItemID: outbox.Metadata.ID, Address: "accepted-elapsed@x.com", DeliveryStatus: models.EmailDeliveryAccepted, AvailableAt: past},
		}))

		claimed, err := testClient.client.EmailRecipients.ClaimRecipients(ctx, &ClaimRecipientsInput{
			Statuses: retryableDeliveryStatuses,
			Limit:    10,
		})
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		assert.Equal(t, "accepted-elapsed@x.com", claimed[0].Address)
	})
}

func TestGetRecipientStats(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	outbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{})
	otherOutbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{Subject: "other"})

	// setStatus updates a freshly created recipient to a terminal/feedback state.
	setStatus := func(address string, status models.EmailDeliveryStatus) {
		r := createTestRecipient(ctx, t, testClient, &models.EmailRecipient{EmailOutboxItemID: outbox.Metadata.ID, Address: address})
		r.DeliveryStatus = status
		_, err := testClient.client.EmailRecipients.UpdateRecipient(ctx, r)
		require.NoError(t, err)
	}

	// setTimestamps updates a recipient's opened/clicked/complained fields, leaving status delivered.
	setTimestamps := func(address string, opened, clicked, complained bool) {
		r := createTestRecipient(ctx, t, testClient, &models.EmailRecipient{EmailOutboxItemID: outbox.Metadata.ID, Address: address, DeliveryStatus: models.EmailDeliveryCompleted})
		now := time.Now().UTC()
		if opened {
			r.OpenedAt = &now
		}
		if clicked {
			r.ClickedAt = &now
		}
		if complained {
			r.ComplainedAt = &now
		}
		_, err := testClient.client.EmailRecipients.UpdateRecipient(ctx, r)
		require.NoError(t, err)
	}

	// Delivered bucket: 3 completed (one opened, one opened+clicked, one complained).
	setTimestamps("delivered-plain@x.com", false, false, false)
	setTimestamps("delivered-opened@x.com", true, false, false)
	setTimestamps("delivered-opened-clicked@x.com", true, true, false)
	// Delivered but complained -> counts toward issues via complained_at, not the status set.
	setTimestamps("delivered-complained@x.com", false, false, true)

	// Issue statuses.
	setStatus("hard@x.com", models.EmailDeliveryHardBounced)
	setStatus("soft@x.com", models.EmailDeliverySoftBounced)
	setStatus("failed@x.com", models.EmailDeliveryFailed)
	setStatus("abandoned@x.com", models.EmailDeliveryAbandoned)

	// Non-terminal, contributes to no derived bucket.
	setStatus("pending@x.com", models.EmailDeliveryPending)

	// A recipient on a different outbox must never be counted.
	createTestRecipient(ctx, t, testClient, &models.EmailRecipient{EmailOutboxItemID: otherOutbox.Metadata.ID, Address: "elsewhere@x.com", DeliveryStatus: models.EmailDeliveryCompleted})

	t.Run("empty outbox returns zeroed stats", func(t *testing.T) {
		emptyOutbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{Subject: "empty"})
		stats, err := testClient.client.EmailRecipients.GetRecipientStats(ctx, emptyOutbox.Metadata.ID)
		require.NoError(t, err)
		assert.Equal(t, &EmailRecipientStatsResult{}, stats)
	})

	t.Run("aggregates counts scoped to the outbox", func(t *testing.T) {
		stats, err := testClient.client.EmailRecipients.GetRecipientStats(ctx, outbox.Metadata.ID)
		require.NoError(t, err)

		// 4 completed + 4 issue statuses + 1 pending = 9.
		assert.Equal(t, 9, stats.Total)
		// completed status: 4 delivered rows.
		assert.Equal(t, 4, stats.Delivered)
		assert.Equal(t, 2, stats.Opened)
		assert.Equal(t, 1, stats.Clicked)
		// hard_bounced + soft_bounced + failed + abandoned + the complained recipient = 5.
		assert.Equal(t, 5, stats.Issues)
	})
}

func TestGetRecipientsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	outbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{})

	resourceCount := 10
	recipients := make([]*models.EmailRecipient, resourceCount)
	for i := range resourceCount {
		recipients[i] = &models.EmailRecipient{
			EmailOutboxItemID: outbox.Metadata.ID,
			Address:           fmt.Sprintf("recipient-%d@x.com", i),
			DeliveryStatus:    models.EmailDeliveryPending,
			AvailableAt:       time.Now().UTC(),
		}
	}
	require.NoError(t, testClient.client.EmailRecipients.CreateRecipients(ctx, recipients))

	sortableFields := []sortableField{
		EmailRecipientSortableFieldCreatedAtAsc,
		EmailRecipientSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := EmailRecipientSortableField(sortByField.getValue())

		result, err := testClient.client.EmailRecipients.GetRecipients(ctx, &GetEmailRecipientsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Recipients {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
