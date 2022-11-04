package serviceaccount

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jws"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/jwsprovider"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
)

type keyPair struct {
	priv jwk.Key
	pub  jwk.Key
}

func TestLogin(t *testing.T) {
	validKeyPair := createKeyPair(t)
	invalidKeyPair := createKeyPair(t)

	keyID := validKeyPair.pub.KeyID()
	serviceAccountID := "d4a94ff5-154e-4758-8039-55e2147fa154"
	issuer := "https://test.tharsis"
	sub := "testSubject1"

	basicPolicy := []models.OIDCTrustPolicy{
		{
			Issuer: issuer,
			BoundClaims: map[string]string{
				"sub": sub,
				"aud": "tharsis",
			},
		},
	}

	// Test cases
	tests := []struct {
		expectErr      error
		name           string
		serviceAccount string
		policy         []models.OIDCTrustPolicy
		token          []byte
	}{
		{
			name:           "login with service account resource path",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
		},
		{
			name:           "login with service account ID",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
		},
		{
			name:           "subject claim doesn't match",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "invalidsubject", time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			expectErr:      errors.New("Failed to verify token \"sub\" not satisfied: values do not match"),
		},
		{
			name:           "expired token",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "invalidsubject", time.Now().Add(-time.Minute)),
			policy:         basicPolicy,
			expectErr:      errors.New("Failed to verify token exp not satisfied"),
		},
		{
			name:           "no matching trust policy",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      "https://notavalidissuer",
					BoundClaims: map[string]string{},
				},
			},
			expectErr: failedLoginError,
		},
		{
			name:           "empty trust policy",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         []models.OIDCTrustPolicy{},
			expectErr:      failedLoginError,
		},
		{
			name:           "invalid token",
			serviceAccount: "groupA/serviceAccount1",
			token:          []byte("invalidtoken"),
			policy:         basicPolicy,
			expectErr:      errors.New("Failed to decode token failed to parse token: invalid character 'i' looking for beginning of value"),
		},
		{
			name:           "missing issuer",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, "", sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			expectErr:      errors.New("JWT is missing issuer claim"),
		},
		{
			name:           "invalid token signature",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, invalidKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			expectErr:      failedLoginError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sa := models.ServiceAccount{
				Metadata:          models.ResourceMetadata{ID: serviceAccountID},
				Name:              "serviceAccount1",
				ResourcePath:      "groupA/serviceAccount1",
				OIDCTrustPolicies: test.policy,
			}

			mockServiceAccounts := db.MockServiceAccounts{}
			mockServiceAccounts.Test(t)

			mockServiceAccounts.On("GetServiceAccountByPath", mock.Anything, test.serviceAccount).Return(&sa, nil)
			mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, test.serviceAccount).Return(&sa, nil)

			mockJWSProvider := jwsprovider.MockJWSProvider{}
			mockJWSProvider.Test(t)

			mockJWSProvider.On("Sign", ctx, mock.MatchedBy(func(payload []byte) bool {
				parsedToken, err := jwt.Parse(payload)
				if err != nil {
					t.Fatal(err)
				}
				if parsedToken.Subject() != sa.ResourcePath {
					return false
				}
				privClaims := parsedToken.PrivateClaims()

				return privClaims["tharsis_service_account_id"] == gid.ToGlobalID(gid.ServiceAccountType, sa.Metadata.ID) &&
					privClaims["tharsis_service_account_name"] == sa.Name &&
					privClaims["tharsis_service_account_path"] == sa.ResourcePath
			})).Return([]byte("signedtoken"), nil)

			dbClient := db.Client{
				ServiceAccounts: &mockServiceAccounts,
			}

			serviceAccountAuth := auth.NewIdentityProvider(&mockJWSProvider, "https://tharsis.io")

			getKeySetFunc := func(_ context.Context, _ string) (jwk.Set, error) {
				set := jwk.NewSet()
				set.Add(validKeyPair.pub)
				return set, nil
			}

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			testLogger, _ := logger.NewForTest()

			service := newService(testLogger, &dbClient, serviceAccountAuth, getKeySetFunc, &mockActivityEvents)

			resp, err := service.Login(ctx, &LoginInput{ServiceAccount: test.serviceAccount, Token: test.token})
			if err != nil && test.expectErr == nil {
				t.Fatal(err)
			}

			if test.expectErr == nil {
				expected := LoginResponse{
					Token: []byte("signedtoken"),
				}

				assert.Equal(t, &expected, resp)
			} else {
				assert.EqualError(t, err, test.expectErr.Error())
			}
		})
	}
}

func createKeyPair(t *testing.T) keyPair {
	rsaPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	privKey, err := jwk.New(rsaPrivKey)
	if err != nil {
		t.Fatal(err)
	}

	pubKey, err := jwk.New(rsaPrivKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	if err := jwk.AssignKeyID(pubKey); err != nil {
		t.Fatal(err)
	}

	return keyPair{priv: privKey, pub: pubKey}
}

func createJWT(t *testing.T, key jwk.Key, keyID string, issuer string, sub string, exp time.Time) []byte {
	token := jwt.New()

	_ = token.Set(jwt.ExpirationKey, exp.Unix())
	_ = token.Set(jwt.SubjectKey, sub)
	_ = token.Set(jwt.AudienceKey, "tharsis")
	if issuer != "" {
		_ = token.Set(jwt.IssuerKey, issuer)
	}

	hdrs := jws.NewHeaders()
	_ = hdrs.Set(jws.TypeKey, "JWT")
	_ = hdrs.Set(jws.KeyIDKey, keyID)

	signed, err := jwt.Sign(token, jwa.RS256, key, jwt.WithHeaders(hdrs))
	if err != nil {
		t.Fatal(err)
	}

	return signed
}
