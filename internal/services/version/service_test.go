package version

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestNewService(t *testing.T) {
	dbClient := &db.Client{}
	apiVersion := "1.0.0"
	buildTimestampStr := "2025-06-02T13:00:00Z"

	buildTimestamp, err := time.Parse(time.RFC3339, buildTimestampStr)
	require.NoError(t, err)

	expect := &service{
		dbClient:       dbClient,
		version:        apiVersion,
		buildTimestamp: buildTimestamp,
	}

	actualService, err := NewService(dbClient, apiVersion, buildTimestampStr)
	require.NoError(t, err)
	assert.Equal(t, expect, actualService)
}

func TestGetCurrentVersion(t *testing.T) {
	buildTimestamp := time.Now().UTC()

	testCases := []struct {
		dbError         error
		dbMigration     *db.SchemaMigration
		name            string
		expectErrorCode errors.CodeType
		withCaller      bool
	}{
		{
			name:        "successfully retrieve version info",
			withCaller:  true,
			dbMigration: &db.SchemaMigration{Version: 1, Dirty: false},
		},
		{
			name:            "without caller",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "db error",
			withCaller:      true,
			dbError:         errors.New("db error"),
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockMigrations := db.NewMockSchemaMigrations(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)
			}

			if test.dbMigration != nil || test.dbError != nil {
				mockMigrations.On("GetCurrentMigration", mock.Anything).Return(test.dbMigration, test.dbError)
			}

			dbClient := &db.Client{
				SchemaMigrations: mockMigrations,
			}

			service := &service{
				dbClient:       dbClient,
				version:        "1.0.0",
				buildTimestamp: buildTimestamp,
			}

			actualInfo, err := service.GetCurrentVersion(ctx)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if test.dbMigration != nil {
				expectedInfo := &Info{
					DBMigrationVersion: strconv.Itoa(test.dbMigration.Version),
					DBMigrationDirty:   test.dbMigration.Dirty,
					Version:            "1.0.0",
					BuildTimestamp:     buildTimestamp,
				}
				assert.Equal(t, expectedInfo, actualInfo)
			}
		})
	}
}
