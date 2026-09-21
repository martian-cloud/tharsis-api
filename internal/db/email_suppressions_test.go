//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// getValue implements the sortableField interface for EmailSuppressionSortableField.
func (sf EmailSuppressionSortableField) getValue() string {
	return string(sf)
}

func TestGetSuppressionByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	suppression, err := testClient.client.EmailSuppressions.CreateSuppression(ctx, &models.EmailSuppression{
		Address: "byid@example.com",
		Cause:   models.EmailSuppressionCauseHardBounce,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode   errors.CodeType
		name              string
		id                string
		expectSuppression bool
	}

	testCases := []testCase{
		{
			name:              "get resource by id",
			id:                suppression.Metadata.ID,
			expectSuppression: true,
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
			got, err := testClient.client.EmailSuppressions.GetSuppressionByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectSuppression {
				require.NotNil(t, got)
				assert.Equal(t, test.id, got.Metadata.ID)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestGetSuppressionByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	suppression, err := testClient.client.EmailSuppressions.CreateSuppression(ctx, &models.EmailSuppression{
		Address: "bytrn@example.com",
		Cause:   models.EmailSuppressionCauseHardBounce,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode   errors.CodeType
		name              string
		trn               string
		expectSuppression bool
	}

	testCases := []testCase{
		{
			name:              "get resource by TRN",
			trn:               suppression.Metadata.TRN,
			expectSuppression: true,
		},
		{
			// The suppression TRN path is the address itself.
			name: "resource with TRN not found",
			trn:  trn.TypeEmailSuppression.Build("missing@example.com"),
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			got, err := testClient.client.EmailSuppressions.GetSuppressionByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectSuppression {
				require.NotNil(t, got)
				assert.Equal(t, test.trn, got.Metadata.TRN)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestCreateSuppression(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		address         string
		wantAddress     string
		cause           models.EmailSuppressionCause
	}

	testCases := []testCase{
		{
			name:        "successfully create hard bounce suppression, lower-casing the address",
			address:     "Bounce@Example.com",
			wantAddress: "bounce@example.com",
			cause:       models.EmailSuppressionCauseHardBounce,
		},
		{
			name:        "successfully create complaint suppression",
			address:     "complaint@example.com",
			wantAddress: "complaint@example.com",
			cause:       models.EmailSuppressionCauseComplaint,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			created, err := testClient.client.EmailSuppressions.CreateSuppression(ctx, &models.EmailSuppression{
				Address: test.address,
				Cause:   test.cause,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, created)
			assert.Equal(t, test.wantAddress, created.Address)
			assert.Equal(t, test.cause, created.Cause)
			assert.NotEmpty(t, created.Metadata.ID)
			assert.NotEmpty(t, created.Metadata.TRN)
		})
	}
}

func TestCreateSuppressionDuplicate(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, err := testClient.client.EmailSuppressions.CreateSuppression(ctx, &models.EmailSuppression{
		Address: "dup@example.com",
		Cause:   models.EmailSuppressionCauseHardBounce,
	})
	require.NoError(t, err)

	// A second suppression for the same address (case-insensitive) conflicts on the unique index.
	_, err = testClient.client.EmailSuppressions.CreateSuppression(ctx, &models.EmailSuppression{
		Address: "DUP@example.com",
		Cause:   models.EmailSuppressionCauseComplaint,
	})
	assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
}

func TestDeleteSuppression(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	suppression, err := testClient.client.EmailSuppressions.CreateSuppression(ctx, &models.EmailSuppression{
		Address: "delete@example.com",
		Cause:   models.EmailSuppressionCauseHardBounce,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		entry           *models.EmailSuppression
	}

	testCases := []testCase{
		{
			name: "delete will fail because resource version doesn't match",
			entry: &models.EmailSuppression{
				Metadata: models.ResourceMetadata{ID: suppression.Metadata.ID, Version: -1},
			},
			expectErrorCode: errors.EOptimisticLock,
		},
		{
			name:  "successfully delete resource",
			entry: suppression,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.EmailSuppressions.DeleteSuppression(ctx, test.entry)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			got, gErr := testClient.client.EmailSuppressions.GetSuppressionByID(ctx, test.entry.Metadata.ID)
			require.NoError(t, gErr)
			assert.Nil(t, got)
		})
	}
}

func TestGetSuppressions(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	addresses := []string{"a@example.com", "b@example.com", "c@example.com"}
	for _, addr := range addresses {
		_, err := testClient.client.EmailSuppressions.CreateSuppression(ctx, &models.EmailSuppression{
			Address: addr,
			Cause:   models.EmailSuppressionCauseHardBounce,
		})
		require.NoError(t, err)
	}

	type testCase struct {
		filter            *EmailSuppressionFilter
		name              string
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name:              "return all suppressions",
			expectResultCount: len(addresses),
		},
		{
			name:              "filter by addresses",
			filter:            &EmailSuppressionFilter{Addresses: []string{"a@example.com", "b@example.com"}},
			expectResultCount: 2,
		},
		{
			name:              "filter by addresses is case-insensitive",
			filter:            &EmailSuppressionFilter{Addresses: []string{"A@Example.com"}},
			expectResultCount: 1,
		},
		{
			name:              "filter by addresses with no match",
			filter:            &EmailSuppressionFilter{Addresses: []string{"nobody@example.com"}},
			expectResultCount: 0,
		},
		{
			name:              "search by substring",
			filter:            &EmailSuppressionFilter{Search: ptr.String("c@")},
			expectResultCount: 1,
		},
		{
			name:              "search with no match",
			filter:            &EmailSuppressionFilter{Search: ptr.String("nomatch")},
			expectResultCount: 0,
		},
		{
			name:              "empty search is ignored and returns all",
			filter:            &EmailSuppressionFilter{Search: ptr.String("")},
			expectResultCount: len(addresses),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.EmailSuppressions.GetSuppressions(ctx, &GetEmailSuppressionsInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectResultCount, len(result.Suppressions))
		})
	}
}

func TestGetSuppressionsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	resourceCount := 10
	for i := range resourceCount {
		_, err := testClient.client.EmailSuppressions.CreateSuppression(ctx, &models.EmailSuppression{
			Address: fmt.Sprintf("suppress-%d@example.com", i),
			Cause:   models.EmailSuppressionCauseHardBounce,
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		EmailSuppressionSortableFieldCreatedAtAsc,
		EmailSuppressionSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := EmailSuppressionSortableField(sortByField.getValue())

		result, err := testClient.client.EmailSuppressions.GetSuppressions(ctx, &GetEmailSuppressionsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Suppressions {
			resources = append(resources, resource)
		}

		return result.PageInfo, resources, nil
	})
}
