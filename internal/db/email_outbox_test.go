//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// getValue implements the sortableField interface for EmailOutboxItemSortableField.
func (sf EmailOutboxItemSortableField) getValue() string {
	return string(sf)
}

func createTestOutboxItem(ctx context.Context, t *testing.T, testClient *testClient, outbox *models.EmailOutboxItem) *models.EmailOutboxItem {
	if outbox.EmailType == "" {
		outbox.EmailType = builder.FailedRunEmailType
	}
	if outbox.Subject == "" {
		outbox.Subject = "test subject"
	}
	if outbox.PayloadObjectStoreKey == nil {
		outbox.PayloadObjectStoreKey = new("emails/test/payload.json")
	}
	if outbox.Status == "" {
		outbox.Status = models.EmailOutboxItemStatusReady
	}

	created, err := testClient.client.EmailOutboxItems.CreateOutboxItem(ctx, outbox)
	require.NoError(t, err)
	return created
}

func TestGetOutboxItemByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	outbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{})

	type testCase struct {
		expectErrorCode  errors.CodeType
		name             string
		id               string
		expectOutboxItem bool
	}

	testCases := []testCase{
		{
			name:             "get resource by id",
			id:               outbox.Metadata.ID,
			expectOutboxItem: true,
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
			got, err := testClient.client.EmailOutboxItems.GetOutboxItemByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectOutboxItem {
				require.NotNil(t, got)
				assert.Equal(t, test.id, got.Metadata.ID)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestGetOutboxItemByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	outbox := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{})

	type testCase struct {
		expectErrorCode  errors.CodeType
		name             string
		trn              string
		expectOutboxItem bool
	}

	testCases := []testCase{
		{
			name:             "get resource by TRN",
			trn:              outbox.Metadata.TRN,
			expectOutboxItem: true,
		},
		{
			name: "resource with TRN not found",
			trn:  trn.TypeEmailOutboxItem.Build(nonExistentGlobalID),
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			got, err := testClient.client.EmailOutboxItems.GetOutboxItemByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectOutboxItem {
				require.NotNil(t, got)
				assert.Equal(t, test.trn, got.Metadata.TRN)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestCreateOutboxItem(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		outbox          *models.EmailOutboxItem
	}

	testCases := []testCase{
		{
			name: "successfully create ephemeral outbox",
			outbox: &models.EmailOutboxItem{
				EmailType:             builder.FailedRunEmailType,
				Subject:               "ephemeral subject",
				PayloadObjectStoreKey: new("emails/abc/payload.json"),
				Ephemeral:             true,
			},
		},
		{
			name: "successfully create retained outbox",
			outbox: &models.EmailOutboxItem{
				EmailType:             builder.ServiceAccountSecretExpirationEmailType,
				Subject:               "retained subject",
				PayloadObjectStoreKey: new("emails/def/payload.json"),
				Ephemeral:             false,
			},
		},
		{
			name: "successfully create outbox with an inline payload",
			outbox: &models.EmailOutboxItem{
				EmailType: builder.FailedRunEmailType,
				Subject:   "inline subject",
				Payload:   []byte{0x01, 0x02, 0x03},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			created, err := testClient.client.EmailOutboxItems.CreateOutboxItem(ctx, test.outbox)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, created)
			assert.Equal(t, test.outbox.EmailType, created.EmailType)
			assert.Equal(t, test.outbox.Subject, created.Subject)
			assert.Equal(t, test.outbox.Ephemeral, created.Ephemeral)
			assert.Equal(t, test.outbox.PayloadObjectStoreKey, created.PayloadObjectStoreKey)
			if len(test.outbox.Payload) > 0 {
				assert.Equal(t, test.outbox.Payload, created.Payload)
			} else {
				assert.Empty(t, created.Payload)
			}
			assert.NotEmpty(t, created.Metadata.ID)
			assert.NotEmpty(t, created.Metadata.TRN)
		})
	}
}

func TestDeleteOutboxItems(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	o1 := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{Subject: "del-1"})
	o2 := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{Subject: "del-2"})

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		ids             []string
		expectDeleted   []string
	}

	testCases := []testCase{
		{
			name: "empty ids is a no-op",
			ids:  nil,
		},
		{
			name:          "successfully delete the given outboxes",
			ids:           []string{o1.Metadata.ID, o2.Metadata.ID},
			expectDeleted: []string{o1.Metadata.ID, o2.Metadata.ID},
		},
		{
			name:            "deleting a missing id returns an optimistic lock error",
			ids:             []string{nonExistentID},
			expectErrorCode: errors.EOptimisticLock,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.EmailOutboxItems.DeleteOutboxItems(ctx, test.ids)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			for _, id := range test.expectDeleted {
				got, gErr := testClient.client.EmailOutboxItems.GetOutboxItemByID(ctx, id)
				require.NoError(t, gErr)
				assert.Nil(t, got)
			}
		})
	}
}

func TestGetOutboxItems(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_ = createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{Subject: "ephemeral-one", Ephemeral: true})
	_ = createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{Subject: "retained-two", Ephemeral: false})

	type testCase struct {
		filter            *EmailOutboxItemFilter
		name              string
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name:              "return all outboxes",
			expectResultCount: 2,
		},
		{
			name:              "filter ephemeral only",
			filter:            &EmailOutboxItemFilter{Ephemeral: ptr.Bool(true)},
			expectResultCount: 1,
		},
		{
			name:              "filter retained only",
			filter:            &EmailOutboxItemFilter{Ephemeral: ptr.Bool(false)},
			expectResultCount: 1,
		},
		{
			name:              "subject search",
			filter:            &EmailOutboxItemFilter{SubjectSearch: ptr.String("ephemeral-")},
			expectResultCount: 1,
		},
		{
			name:              "subject search with no match",
			filter:            &EmailOutboxItemFilter{SubjectSearch: ptr.String("nomatch")},
			expectResultCount: 0,
		},
		{
			name:              "empty subject search is ignored and returns all",
			filter:            &EmailOutboxItemFilter{SubjectSearch: ptr.String("")},
			expectResultCount: 2,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.EmailOutboxItems.GetOutboxItems(ctx, &GetEmailOutboxItemsInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectResultCount, len(result.OutboxItems))
		})
	}
}

func TestClaimOutboxItems(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Completed ephemeral outbox -> claimable by the cleanup poller (status=completed, ephemeral=true).
	completedEphemeral := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{
		Subject:   "completed-ephemeral",
		Status:    models.EmailOutboxItemStatusCompleted,
		Ephemeral: true,
	})

	// Preparing outbox -> claimable by the materializer (status=preparing).
	preparing := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{
		Subject:   "preparing",
		Status:    models.EmailOutboxItemStatusPreparing,
		Ephemeral: true,
	})

	// Completed retained (non-ephemeral) outbox -> not claimable by the cleanup poller.
	completedRetained := createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{
		Subject:   "completed-retained",
		Status:    models.EmailOutboxItemStatusCompleted,
		Ephemeral: false,
	})

	type testCase struct {
		name       string
		input      *ClaimOutboxItemsInput
		wantIDs    []string
		wantNoneOf []string
	}

	testCases := []testCase{
		{
			name:       "no filter claims nothing",
			input:      &ClaimOutboxItemsInput{Limit: 10},
			wantNoneOf: []string{completedEphemeral.Metadata.ID, preparing.Metadata.ID, completedRetained.Metadata.ID},
		},
		{
			name:       "zero limit claims nothing",
			input:      &ClaimOutboxItemsInput{Status: new(models.EmailOutboxItemStatusPreparing), Limit: 0},
			wantNoneOf: []string{preparing.Metadata.ID},
		},
		{
			name:       "completed ephemeral is claimable; preparing and retained are not",
			input:      &ClaimOutboxItemsInput{Status: new(models.EmailOutboxItemStatusCompleted), Ephemeral: new(true), Limit: 10},
			wantIDs:    []string{completedEphemeral.Metadata.ID},
			wantNoneOf: []string{preparing.Metadata.ID, completedRetained.Metadata.ID},
		},
		{
			name:       "preparing is claimable by status alone",
			input:      &ClaimOutboxItemsInput{Status: new(models.EmailOutboxItemStatusPreparing), Limit: 10},
			wantIDs:    []string{preparing.Metadata.ID},
			wantNoneOf: []string{completedEphemeral.Metadata.ID, completedRetained.Metadata.ID},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// Run each claim in its own transaction so the FOR UPDATE lock releases between cases.
			txCtx, err := testClient.client.Transactions.BeginTx(ctx)
			require.NoError(t, err)
			defer func() { _ = testClient.client.Transactions.RollbackTx(txCtx) }()

			claimed, err := testClient.client.EmailOutboxItems.ClaimOutboxItems(txCtx, test.input)
			require.NoError(t, err)

			ids := map[string]struct{}{}
			for _, o := range claimed {
				ids[o.Metadata.ID] = struct{}{}
			}

			for _, want := range test.wantIDs {
				_, ok := ids[want]
				assert.True(t, ok, "expected %s to be claimed", want)
			}
			for _, notWant := range test.wantNoneOf {
				_, ok := ids[notWant]
				assert.False(t, ok, "did not expect %s to be claimed", notWant)
			}
		})
	}
}

func TestClaimOutboxItemsRespectsLimit(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Three preparing outboxes; a limit of 2 must return at most 2.
	for i := range 3 {
		createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{
			Subject: fmt.Sprintf("preparing-%d", i),
			Status:  models.EmailOutboxItemStatusPreparing,
		})
	}

	txCtx, err := testClient.client.Transactions.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = testClient.client.Transactions.RollbackTx(txCtx) }()

	claimed, err := testClient.client.EmailOutboxItems.ClaimOutboxItems(txCtx, &ClaimOutboxItemsInput{
		Status: new(models.EmailOutboxItemStatusPreparing),
		Limit:  2,
	})
	require.NoError(t, err)
	assert.Len(t, claimed, 2)
}

func TestGetOutboxItemsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	resourceCount := 10
	for i := range resourceCount {
		createTestOutboxItem(ctx, t, testClient, &models.EmailOutboxItem{Subject: fmt.Sprintf("outbox-%d", i)})
	}

	sortableFields := []sortableField{
		EmailOutboxItemSortableFieldCreatedAtAsc,
		EmailOutboxItemSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := EmailOutboxItemSortableField(sortByField.getValue())

		result, err := testClient.client.EmailOutboxItems.GetOutboxItems(ctx, &GetEmailOutboxItemsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.OutboxItems {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
