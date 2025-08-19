package gpgkey

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

type mockDBClient struct {
	*db.Client
	MockTransactions   *db.MockTransactions
	MockGroups         *db.MockGroups
	MockResourceLimits *db.MockResourceLimits
	MockActivityEvents *db.MockActivityEvents
	MockGPGKeys        *db.MockGPGKeys
}

func buildDBClientWithMocks(t *testing.T) *mockDBClient {
	mockTransactions := db.MockTransactions{}
	mockTransactions.Test(t)

	mockGroups := db.MockGroups{}
	mockGroups.Test(t)

	mockResourceLimits := db.MockResourceLimits{}
	mockResourceLimits.Test(t)

	mockActivityEvents := db.MockActivityEvents{}
	mockActivityEvents.Test(t)

	mockGPGKeys := db.MockGPGKeys{}
	mockGPGKeys.Test(t)

	return &mockDBClient{
		Client: &db.Client{
			Transactions:   &mockTransactions,
			Groups:         &mockGroups,
			ResourceLimits: &mockResourceLimits,
			ActivityEvents: &mockActivityEvents,
			GPGKeys:        &mockGPGKeys,
		},
		MockTransactions:   &mockTransactions,
		MockGroups:         &mockGroups,
		MockResourceLimits: &mockResourceLimits,
		MockActivityEvents: &mockActivityEvents,
		MockGPGKeys:        &mockGPGKeys,
	}
}

// TODO: Add the rest of the test cases needed to fully test this function.
// At present, it only tests the limit on number of GPG keys per group.
func TestCreateGPGKey(t *testing.T) {

	group1Name := "group-1-name"
	group1 := &models.Group{
		Metadata: models.ResourceMetadata{
			ID: "group-1-id", // okay that this is not a valid UUID
		},
		Name:        group1Name,
		Description: "group 1 description",
		ParentID:    "", // simulate a root group
		FullPath:    group1Name,
	}

	// a made-up GPG key for testing and its resulting fields
	armor := `-----BEGIN PGP PUBLIC KEY BLOCK-----

mQGNBGPipfcBDADTuYQcZy637SMaQuYTKBOLsYAtQWrQcuQggf/bjECDP3zkemON
cr6CNtyudOEd9fzLtbzEDZ3sG6zokQyPxbfKlbowuKVvxP0fQ0evTyoxic0Dm1Th
lDRW1BmEGNSO7qKISwqftghLFwYZkO/l6cu1suhhjXWNYgQXZLaewx+iazQZEVFK
0Bp2Q6Vp61OXpviOOdPQXE0mQAWSIV3YO/j1GBUUZIhTX6N0y+Z78tK4vqSkoFr2
tbnbJlstj4Gy1ElanHVYQhCLk3zlmU+GCIMkqrT9WZW1LWzCW/muUb+7kk+AKI/r
xoMm1Ln4e9t7ed4sy9x7Dkn4buwhtEEaciXBB07SeKvnQtov8GN35sH86+U3poAQ
9W8BUFYBPuud/Pvx996q+H5FlH3YCDq+wwRdYJwK59yr4Auq8+sThjDSp7oIQsvb
d0UyHaKn4zijDJQedE3Gi49pLEPc+BPpysNeAXhHj5E/8xWIoCgbW+LJTELkQd0m
Uyk/NBifKl/yVCEAEQEAAbQ4Si4gUmFuZG9tIFBlcnNvbiBJSUkgPGoucmFuZG9t
LnBlcnNvbi4zQGludmFsaWQuZXhhbXBsZT6JAdQEEwEKAD4WIQTEj38ZsU5ZQz35
QCNB9Xq2dB+S8QUCY+Kl9wIbAwUJA8JnAAULCQgHAgYVCgkICwIEFgIDAQIeAQIX
gAAKCRBB9Xq2dB+S8VqrDACgNqecLXdkc/bmvpEWJdg7Rg0OC8cbguDZvIqpwr2x
dqZjXu2NUaHirXfVmGsHVcDnRPfIs+2dj7Lq2SeJRN7qnMbqG6OTBi3m+EVYFiY4
j/dBPzDBcferVk+tFLypWoF9gTB2jAT0TNuaxiKbT25sBbTJrR44M8tldizM1bAX
Dtp27K9/9oFtK5lqHpih9fxEaXbiTOKPKUlGdzcPt7KTV6w1BjK8ZT62bZlWXOvU
8oZBhy3jkLLNL17138nACCzJ5NdtnxmKKr4BASB3Midp5iWovKXFLwcM8aekL/vx
IdekmtiPmDlmIc68s63X2GcyqLfLAQBcwJIlcYCFlR3GNWbNyl+WZra6uDqShZ3T
A02d8Slvmp5Q0xOLCttxHYm1g2aTwCsqsh6lDTltrt+USBUFhd11/AKQg4AiP2eQ
dzMmLlsKHSEPF5r8N2NWLXfbD2uKKmTTNYj8/vFluTXLYuDqAlwrEATp4p2kV7WV
MhIP6dr2IiWxxEJzyZbr88m5AY0EY+Kl9wEMAJddzP9wM5tIoDJoyod/9l5IvFgk
smh4tVDRUVGZ9WKt/BNtPUYrxP3Z97yfF9MUdM3PVgkMGZdTYgtVRK1wXHxUEvgP
NPzQXjUIWVPum66amZqXUEZnIOx9w9deNIXQLCKYCUvBTThSvVOJHHa1F55gkuzl
5Xja0QIs7rmWEdMgGFsDIkweIMYnXgMm0fd18LZqAFduBe/qVOLtQJaXoUlp8gfw
ensQlbw17c37HOtaoxLG3B5CK2ZvF0mkrHGB58LOoj4FRWOe4w8EbxgzHxzGeKLg
nbGCW3h6h6S3w4gAvqAlfmEr1zP2tujnKuHcLb4vmNyTCQVzrzRpUP39LE6LL4kV
rNnzpakRjRREgSmjbiSc3+27USs0zIk6yTgFjAKahowyUfwMVYYssFG5qYf5a2kj
WrPRRjI5fhE+DgmNITeI96y7iF3NY1o98PeU+pf9TiU8aLW/9G2TLpnEv96QeIlL
cq5YK7JuTKbflZQpytkXUOGf18YYswrGoPdXOwARAQABiQG8BBgBCgAmFiEExI9/
GbFOWUM9+UAjQfV6tnQfkvEFAmPipfcCGwwFCQPCZwAACgkQQfV6tnQfkvGnbAwA
uMZ4ThOXOA17iyBgKQ4tj0TGTqErKb0dxuuvf0g+ozRfFdnhr+UiuD2QtgNcYNNm
U+qLAt96sPCN+nit2/coE0P+YI24iTC8AYJXXSgP/ZnyjkkbKNQEBRm/hdocejzB
5BM3ztV1VriQqIQEqp4HTzcOTXiEhZ8jZW0mrBTlHenYMe/83zoBuABQGMnuy/JJ
pSgJ+XQ6uBnGa7b/35nHUfoIhC2GNQ8/uI/VBy1vhnEBFubROVMyss9IpTheDHOC
oYbE8Lq9J8Giu8mqyF4ifzXl9A2lowPFDg6Ey9Yms+wnVWUD2uMdQ00PIMB0HUFo
alugyNSEqc6GP9rOUkR4TUwNmeV1OJCJtX6sdb+WY2ZczoiT7SYVBkqS6xEujeRy
DGGMeh5+2/26EiP2nBcIJqTCqZi+yq/5k7QKNtNYNdb/u1WvtseDsfOgekZSwOoN
lNBLBcAMCdEMd4qgt0YvzKzE3GbQoiAkBKJ2qoqun2MXM60324j01B/x/r3E+p15
=HJT6
-----END PGP PUBLIC KEY BLOCK-----
`
	gpgKeyID := uint64(0x41F57AB6741F92F1)
	fingerprint := "C48F7F19B14E59433DF9402341F57AB6741F92F1"

	positiveGPGKey := models.GPGKey{
		Metadata: models.ResourceMetadata{
			ID: "gpg-key-id-1", // okay that this is not a valid UUID
		},
		GroupID:     "group-id-1",
		ASCIIArmor:  armor,
		Fingerprint: fingerprint,
		GPGKeyID:    gpgKeyID,
	}

	type testCase struct {
		expectErrorCode errors.CodeType
		input           *models.GPGKey
		expectOutput    *models.GPGKey
		name            string
		limit           int
		keyCount        int32
	}

	testCases := []testCase{

		{
			name:            "positive",
			input:           &positiveGPGKey,
			limit:           5,
			keyCount:        5,
			expectOutput:    &positiveGPGKey,
			expectErrorCode: "",
		},

		{
			name:            "negative, limit exceeded",
			input:           &positiveGPGKey,
			limit:           5,
			keyCount:        6,
			expectOutput:    nil,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient := buildDBClientWithMocks(t)
			limiter := limits.NewLimitChecker(dbClient.Client)

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)

			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)

			mockAuthorizer.On("RequireAccess", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			dbClient.MockGroups.On("GetGroupByID", mock.Anything, mock.Anything).Return(group1, nil)

			dbClient.MockGPGKeys.On("CreateGPGKey", mock.Anything, mock.Anything).
				Return(
					func(_ context.Context, _ *models.GPGKey) *models.GPGKey {
						return test.expectOutput
					},
					nil,
				)

			dbClient.MockGPGKeys.On("GetGPGKeys", mock.Anything, mock.Anything).
				Return(
					func(ctx context.Context, input *db.GetGPGKeysInput) *db.GPGKeysResult {
						_ = ctx
						_ = input

						return &db.GPGKeysResult{
							PageInfo: &pagination.PageInfo{
								TotalCount: test.keyCount,
							},
						}
					},
					func(_ context.Context, _ *db.GetGPGKeysInput) error {
						return nil
					},
				)

			dbClient.MockResourceLimits.On("GetResourceLimit", mock.Anything, string(limits.ResourceLimitGPGKeysPerGroup)).
				Return(&models.ResourceLimit{
					Value: test.limit,
				}, nil)

			mockActivityEvents := activityevent.NewMockService(t)
			if test.expectErrorCode == "" {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			testCaller := auth.NewUserCaller(
				&models.User{
					Metadata: models.ResourceMetadata{
						ID: "user-caller-id",
					},
					Admin:    false,
					Username: "user1",
				},
				&mockAuthorizer,
				dbClient.Client,
				mockMaintenanceMonitor,
				nil,
			)

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient.Client, limiter, mockActivityEvents)

			// Call the service function.
			toCreate := CreateGPGKeyInput{
				GroupID:    test.input.GroupID,
				ASCIIArmor: test.input.ASCIIArmor,
			}

			actualOutput, actualError := service.CreateGPGKey(auth.WithCaller(ctx, testCaller), &toCreate)

			assert.Equal(t, test.expectErrorCode, errors.ErrorCode(actualError))
			assert.Equal(t, test.expectOutput, actualOutput)
		})
	}
}

func TestGPGKeyByTRN(t *testing.T) {
	sampleGPGKey := &models.GPGKey{
		Metadata: models.ResourceMetadata{
			ID:  "gpg-key-id-1",
			TRN: types.GPGKeyModelType.BuildTRN("my-group/123341"),
		},
		GroupID: "group-1",
	}

	type testCase struct {
		name            string
		authError       error
		gpgkey          *models.GPGKey
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:   "successfully get gpg key by trn",
			gpgkey: sampleGPGKey,
		},
		{
			name:            "gpg key not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject is not authorized to view gpg key",
			gpgkey:          sampleGPGKey,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockGPGKeys := db.NewMockGPGKeys(t)

			mockGPGKeys.On("GetGPGKeyByTRN", mock.Anything, sampleGPGKey.Metadata.TRN).Return(test.gpgkey, nil)

			if test.gpgkey != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.GPGKeyModelType, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				GPGKeys: mockGPGKeys,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualGPGKey, err := service.GetGPGKeyByTRN(auth.WithCaller(ctx, mockCaller), sampleGPGKey.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.gpgkey, actualGPGKey)
		})
	}
}
