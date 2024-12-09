package version

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestNewService(t *testing.T) {
	dbClient := &db.Client{}
	apiVersion := "1.0.0"

	expect := &service{
		dbClient:   dbClient,
		apiVersion: apiVersion,
	}

	assert.Equal(t, expect, NewService(dbClient, apiVersion))
}

func TestGetCurrentVersion(t *testing.T) {
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
				dbClient:   dbClient,
				apiVersion: "1.0.0",
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
					APIVersion:         "1.0.0",
				}
				assert.Equal(t, expectedInfo, actualInfo)
			}
		})
	}
}
