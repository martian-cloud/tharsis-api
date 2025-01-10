//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetCurrentMigration(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "get current migration",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			migration, err := testClient.client.SchemaMigrations.GetCurrentMigration(ctx)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NotNil(t, migration)
			assert.NotEmpty(t, migration.Version)
			assert.False(t, migration.Dirty)
		})
	}
}
