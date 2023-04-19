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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	jwsprovider "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type keyPair struct {
	priv jwk.Key
	pub  jwk.Key
}

func TestCreateToken(t *testing.T) {
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
			name:           "create service account token with service account resource path",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
		},
		{
			name:           "create service account token with service account ID",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         basicPolicy,
		},
		{
			name:           "subject claim doesn't match",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "invalidsubject", time.Now().Add(time.Minute)),
			policy:         basicPolicy,
			expectErr:      errors.New("of the trust policies for issuer https://test.tharsis, none was satisfied"),
		},
		{
			name:           "expired token",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "invalidsubject", time.Now().Add(-time.Minute)),
			policy:         basicPolicy,
			expectErr:      errExpiredToken,
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
			expectErr: errFailedCreateToken,
		},
		{
			name:           "empty trust policy",
			serviceAccount: serviceAccountID,
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, sub, time.Now().Add(time.Minute)),
			policy:         []models.OIDCTrustPolicy{},
			expectErr:      errFailedCreateToken,
		},
		{
			name:           "invalid token",
			serviceAccount: "groupA/serviceAccount1",
			token:          []byte("invalidtoken"),
			policy:         basicPolicy,
			expectErr:      errors.New("failed to decode token: failed to parse token: invalid character 'i' looking for beginning of value"),
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
			expectErr:      errFailedCreateToken,
		},
		{
			name:           "negative: multiple trust policies with same issuer: all mismatch",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "noMatchSubject", time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "firstSubject"},
				},
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "secondSubject"},
				},
			},
			expectErr: errors.New("of the trust policies for issuer https://test.tharsis, none was satisfied"),
		},
		{
			name:           "positive: multiple trust policies with same issuer: match first",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "firstSubject", time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "firstSubject"},
				},
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "secondSubject"},
				},
			},
		},
		{
			name:           "positive: multiple trust policies with same issuer: match second",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, issuer, "secondSubject", time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "firstSubject"},
				},
				{
					Issuer:      issuer,
					BoundClaims: map[string]string{"sub": "secondSubject"},
				},
			},
		},
		{
			name:           "positive: trust policy issuer has forward slash, token issuer does not",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, "https://test.tharsis", sub, time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      "https://test.tharsis/",
					BoundClaims: map[string]string{},
				},
			},
		},
		{
			name:           "positive: token issuer has forward slash, trust policy issuer does not",
			serviceAccount: "groupA/serviceAccount1",
			token:          createJWT(t, validKeyPair.priv, keyID, "https://test.tharsis/", sub, time.Now().Add(time.Minute)),
			policy: []models.OIDCTrustPolicy{
				{
					Issuer:      "https://test.tharsis",
					BoundClaims: map[string]string{},
				},
			},
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

			mockJWSProvider := jwsprovider.MockProvider{}
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

			configFetcher := auth.NewOpenIDConfigFetcher()

			getKeySetFunc := func(_ context.Context, _ string, _ *auth.OpenIDConfigFetcher) (jwk.Set, error) {
				set := jwk.NewSet()
				set.Add(validKeyPair.pub)
				return set, nil
			}

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			testLogger, _ := logger.NewForTest()

			service := newService(testLogger, &dbClient, serviceAccountAuth, configFetcher, getKeySetFunc, &mockActivityEvents)

			resp, err := service.CreateToken(ctx, &CreateTokenInput{ServiceAccount: test.serviceAccount, Token: test.token})
			if err != nil && test.expectErr == nil {
				t.Fatal(err)
			}

			if test.expectErr == nil {
				expected := CreateTokenResponse{
					Token:     []byte("signedtoken"),
					ExpiresIn: int32(serviceAccountLoginDuration / time.Second),
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
