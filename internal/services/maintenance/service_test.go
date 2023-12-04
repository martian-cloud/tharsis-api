package maintenance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestNewService(t *testing.T) {
	logger, _ := logger.NewForTest()
	dbClient := &db.Client{}

	expect := &service{
		logger:   logger,
		dbClient: dbClient,
	}

	assert.Equal(t, expect, NewService(logger, dbClient))
}

func TestGetMaintenanceMode(t *testing.T) {
	sampleMaintenanceMode := &models.MaintenanceMode{
		Message:   "test",
		CreatedBy: "testSubject",
	}

	type testCase struct {
		expectMaintenanceMode *models.MaintenanceMode
		name                  string
		expectErrorCode       errors.CodeType
		withCaller            bool
	}

	tests := []testCase{
		{
			name:                  "get maintenance mode",
			expectMaintenanceMode: sampleMaintenanceMode,
			withCaller:            true,
		},
		{
			name:            "no caller returns error",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "maintenance mode not enabled",
			expectErrorCode: errors.ENotFound,
			withCaller:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockMaintenanceModes := db.NewMockMaintenanceModes(t)
			mockCaller := auth.NewMockCaller(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)

				mockMaintenanceModes.On("GetMaintenanceMode", mock.Anything).Return(test.expectMaintenanceMode, nil)
			}

			dbClient := &db.Client{
				MaintenanceModes: mockMaintenanceModes,
			}

			service := &service{
				dbClient: dbClient,
			}

			maintenanceMode, err := service.GetMaintenanceMode(ctx)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, sampleMaintenanceMode, maintenanceMode)
		})
	}
}

func TestEnableMaintenanceMode(t *testing.T) {
	testSubject := "testSubject"

	sampleMaintenanceMode := &models.MaintenanceMode{
		Message:   "test",
		CreatedBy: testSubject,
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		expectCreated   bool
		isAdmin         bool
	}

	tests := []testCase{
		{
			name:          "admin can enable maintenance mode",
			expectCreated: true,
			isAdmin:       true,
		},
		{
			name:            "non admin caller cannot enable maintenance mode",
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockMaintenanceModes := db.NewMockMaintenanceModes(t)
			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("IsAdmin").Return(test.isAdmin)

			if test.expectCreated {
				mockCaller.On("GetSubject").Return(testSubject)

				mockMaintenanceModes.On("CreateMaintenanceMode", mock.Anything, sampleMaintenanceMode).Return(sampleMaintenanceMode, nil)
			}

			dbClient := &db.Client{
				MaintenanceModes: mockMaintenanceModes,
			}

			logger, _ := logger.NewForTest()

			service := &service{
				logger:   logger,
				dbClient: dbClient,
			}

			maintenanceMode, err := service.EnableMaintenanceMode(auth.WithCaller(ctx, mockCaller), &EnableMaintenanceModeInput{
				Message: sampleMaintenanceMode.Message,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, sampleMaintenanceMode, maintenanceMode)
		})
	}
}

func TestDisableMaintenanceMode(t *testing.T) {
	testSubject := "testSubject"

	sampleMaintenanceMode := &models.MaintenanceMode{
		Message:   "test",
		CreatedBy: testSubject,
	}

	type testCase struct {
		existingMaintenanceMode *models.MaintenanceMode
		name                    string
		expectErrorCode         errors.CodeType
		isAdmin                 bool
	}

	tests := []testCase{
		{
			name:                    "admin can disable maintenance mode",
			isAdmin:                 true,
			existingMaintenanceMode: sampleMaintenanceMode,
		},
		{
			name:            "non admin caller cannot disable maintenance mode",
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "maintenance mode not enabled",
			isAdmin:         true,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockMaintenanceModes := db.NewMockMaintenanceModes(t)
			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("IsAdmin").Return(test.isAdmin)

			if test.isAdmin {
				mockMaintenanceModes.On("GetMaintenanceMode", mock.Anything).Return(test.existingMaintenanceMode, nil)
			}

			if test.existingMaintenanceMode != nil {
				mockCaller.On("GetSubject").Return(testSubject)

				mockMaintenanceModes.On("DeleteMaintenanceMode", mock.Anything, sampleMaintenanceMode).Return(nil)
			}

			dbClient := &db.Client{
				MaintenanceModes: mockMaintenanceModes,
			}

			logger, _ := logger.NewForTest()

			service := &service{
				logger:   logger,
				dbClient: dbClient,
			}

			err := service.DisableMaintenanceMode(auth.WithCaller(ctx, mockCaller))

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}
