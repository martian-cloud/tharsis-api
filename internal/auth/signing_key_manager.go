package auth

//go:generate go tool mockery --name SigningKeyManager --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	jwsplugin "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (
	// privateClaimPrefix is the prefix used for private claims in the JWT
	privateClaimPrefix = "tharsis_"
	// expiredKeyCheckPeriod is how often to check for expired keys that need to be commissioned
	expiredKeyCheckPeriod = time.Hour
	// keyCreationTimeout is the duration to wait before considering a key creation failed and cleaning it up
	keyCreationTimeout = 1 * time.Minute
)

// TokenInput provides options for creating a new service account token
type TokenInput struct {
	Expiration *time.Time
	Claims     map[string]string
	Subject    string
	JwtID      string
	Audience   string
}

// VerifyTokenOutput is the response from verifying a token
type VerifyTokenOutput struct {
	Token         jwt.Token
	PrivateClaims map[string]string
}

// SigningKeyManager is an interface for generating and verifying JWT tokens
type SigningKeyManager interface {
	// GenerateToken creates a new JWT token
	GenerateToken(ctx context.Context, input *TokenInput) ([]byte, error)
	// VerifyToken verifies that the token is valid
	VerifyToken(ctx context.Context, token string, validateOptions ...jwt.ValidateOption) (*VerifyTokenOutput, error)
	// GetKeys returns the JSON Web Key Set (JWKS)
	GetKeys(ctx context.Context) ([]byte, error)
}

type signingKeyManager struct {
	jwsPlugin                jwsplugin.Provider
	issuerURL                string
	dbClient                 *db.Client
	eventManager             *events.EventManager
	keySet                   jwk.Set
	keySetLock               sync.RWMutex
	logger                   logger.Logger
	keyRotationPeriod        time.Duration
	keyDecommissioningPeriod time.Duration
	jwsProviderPluginType    string
}

// NewSigningKeyManager initializes the SigningKeyManager type
func NewSigningKeyManager(
	ctx context.Context,
	logger logger.Logger,
	jwsPlugin jwsplugin.Provider,
	dbClient *db.Client,
	eventManager *events.EventManager,
	cfg *config.Config,
) (SigningKeyManager, error) {
	return newSigningKeyManager(ctx, logger, jwsPlugin, dbClient, eventManager, cfg, true)
}

func newSigningKeyManager(
	ctx context.Context,
	logger logger.Logger,
	jwsPlugin jwsplugin.Provider,
	dbClient *db.Client,
	eventManager *events.EventManager,
	cfg *config.Config,
	startBackgroundTasks bool,
) (SigningKeyManager, error) {
	if cfg.AsymmetricSigningKeyRotationPeriodDays > 0 && !jwsPlugin.SupportsKeyRotation() {
		return nil, fmt.Errorf("the configured JWS provider plugin %q does not support key rotation", cfg.JWSProviderPluginType)
	}

	signingKeyManager := &signingKeyManager{
		jwsPlugin:                jwsPlugin,
		issuerURL:                cfg.JWTIssuerURL,
		dbClient:                 dbClient,
		eventManager:             eventManager,
		keySet:                   jwk.NewSet(),
		logger:                   logger,
		keyRotationPeriod:        time.Duration(cfg.AsymmetricSigningKeyRotationPeriodDays) * 24 * time.Hour,
		keyDecommissioningPeriod: time.Duration(cfg.AsymmetricSigningKeyDecommissionPeriodDays) * 24 * time.Hour,
		jwsProviderPluginType:    cfg.JWSProviderPluginType,
	}

	if err := signingKeyManager.initializeKeySet(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize identity provider: %w", err)
	}

	if startBackgroundTasks {
		signingKeyManager.startBackgroundTasks(ctx)
	}

	return signingKeyManager, nil
}

func (s *signingKeyManager) GenerateToken(ctx context.Context, input *TokenInput) ([]byte, error) {
	currentTimestamp := time.Now().Unix()

	token := jwt.New()

	if input.Expiration != nil {
		if err := token.Set(jwt.ExpirationKey, input.Expiration.Unix()); err != nil {
			return nil, err
		}
	}
	if err := token.Set(jwt.NotBeforeKey, currentTimestamp); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.IssuedAtKey, currentTimestamp); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.IssuerKey, s.issuerURL); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.SubjectKey, input.Subject); err != nil {
		return nil, err
	}

	aud := input.Audience
	if aud == "" {
		aud = "tharsis"
	}
	if err := token.Set(jwt.AudienceKey, aud); err != nil {
		return nil, err
	}
	if input.JwtID != "" {
		if err := token.Set(jwt.JwtIDKey, input.JwtID); err != nil {
			return nil, err
		}
	}

	for k, v := range input.Claims {
		if err := token.Set(fmt.Sprintf("%s%s", privateClaimPrefix, k), v); err != nil {
			return nil, err
		}
	}

	payload, err := jwt.NewSerializer().Serialize(token)
	if err != nil {
		return nil, err
	}

	activeKey, err := s.getActiveKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active key when signing JWT: %w", err)
	}

	// Use plugin to sign token
	return s.jwsPlugin.Sign(ctx, payload, activeKey.Metadata.ID, activeKey.PluginData, activeKey.PubKeyID)
}

func (s *signingKeyManager) VerifyToken(ctx context.Context, token string, validateOptions ...jwt.ValidateOption) (*VerifyTokenOutput, error) {
	tokenBytes := []byte(token)

	jwsMsg, err := jws.Parse(tokenBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token headers %w", err)
	}

	signatures := jwsMsg.Signatures()
	if len(signatures) < 1 {
		return nil, errors.New("token is missing signature")
	}

	kid := signatures[0].ProtectedHeaders().KeyID()
	if kid == "" {
		return nil, errors.New("token is missing key ID")
	}

	s.keySetLock.RLock()
	_, found := s.keySet.LookupKeyID(kid)
	s.keySetLock.RUnlock()

	if !found {
		if err = s.syncKeySet(ctx); err != nil {
			return nil, errors.New("failed to load keys")
		}
	}

	s.keySetLock.RLock()
	defer s.keySetLock.RUnlock()

	options := []jwt.ParseOption{
		jwt.WithVerify(true),
		jwt.WithKeySet(s.keySet),
		jwt.WithValidate(true),
		jwt.WithIssuer(s.issuerURL),
	}
	for _, o := range validateOptions {
		options = append(options, o)
	}

	// Parse and validate jwt
	decodedToken, err := jwt.Parse(tokenBytes, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token %w", err)
	}

	return &VerifyTokenOutput{
		Token:         decodedToken,
		PrivateClaims: getPrivateClaims(decodedToken),
	}, nil
}

func (s *signingKeyManager) GetKeys(_ context.Context) ([]byte, error) {
	s.keySetLock.RLock()
	defer s.keySetLock.RUnlock()

	return json.Marshal(s.keySet)
}

func (s *signingKeyManager) initializeKeySet(ctx context.Context) error {
	// Cleanup that failed to create and are stuck in the creating state
	if err := s.cleanupFailedKeys(ctx); err != nil {
		return err
	}

	// Check if signing key already exists or is in the process of being created
	signingKeys, err := s.dbClient.AsymSigningKeys.GetAsymSigningKeys(ctx, &db.GetAsymSigningKeysInput{
		Filter: &db.AsymSigningKeyFilter{
			Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusCreating, models.AsymSigningKeyStatusActive},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to query for existing signing keys: %w", err)
	}

	var activeKey *models.AsymSigningKey
	if len(signingKeys.AsymSigningKeys) == 0 {
		// There are no creating or active keys so we will create a new one
		activeKey, err = s.createKey(ctx)
		if err != nil {
			return err
		}
	} else {
		for _, key := range signingKeys.AsymSigningKeys {
			if key.Status == models.AsymSigningKeyStatusActive {
				activeKey = &key
				break
			}
		}
		if activeKey == nil {
			activeKey, err = s.waitForActiveKey(ctx)
			if err != nil {
				return err
			}
		}
	}

	if activeKey.PluginType != s.jwsProviderPluginType {
		return fmt.Errorf("the existing signing key is of type %s, but the configured JWS provider is of type %s", activeKey.PluginType, s.jwsProviderPluginType)
	}

	s.logger.Infof("using %q signing key %q", s.jwsProviderPluginType, activeKey.Metadata.ID)

	return s.syncKeySet(ctx)
}

func (s *signingKeyManager) startBackgroundTasks(ctx context.Context) {
	go s.listenForDBEvents(ctx)

	if s.keyRotationPeriod > 0 {
		go func() {
			for {
				select {
				case <-time.After(expiredKeyCheckPeriod):
					if err := s.deleteDecommissionedKeys(ctx); err != nil {
						s.logger.Errorf("failed to decommission signing key: %v", err)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
		go func() {
			for {
				select {
				case <-time.After(expiredKeyCheckPeriod):
					if err := s.checkForExpiredKey(ctx); err != nil {
						s.logger.Errorf("failed to check for expired signing key: %v", err)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

func (s *signingKeyManager) listenForDBEvents(ctx context.Context) {
	// Monitor db events for key updates
	subscriber := s.eventManager.Subscribe([]events.Subscription{{Type: events.AsymSigningKeySubscription}})
	defer s.eventManager.Unsubscribe(subscriber)

	for {
		_, err := subscriber.GetEvent(ctx)
		if err != nil {
			if errors.IsContextCanceledError(err) {
				return
			}
			s.logger.Errorf("error occurred while waiting for asym signing key event: %v", err)
			continue
		}

		s.logger.Info("received asymmetric signing key event, syncing local key set...")

		// Update key set
		if err := s.syncKeySet(ctx); err != nil {
			s.logger.Errorf("failed to sync signing key set: %v", err)
		}
	}
}

func (s *signingKeyManager) deleteDecommissionedKeys(ctx context.Context) error {
	// Get decommissioned keys
	decommissionedKeys, err := s.dbClient.AsymSigningKeys.GetAsymSigningKeys(ctx, &db.GetAsymSigningKeysInput{
		Filter: &db.AsymSigningKeyFilter{
			Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusDecommissioning},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to query for decommissioned signing keys: %v", err)
	}

	for _, key := range decommissionedKeys.AsymSigningKeys {
		if time.Since(*key.Metadata.LastUpdatedTimestamp) > s.keyDecommissioningPeriod {
			s.logger.Infof("deleting decommissioned signing key: %q", key.Metadata.ID)

			// Delete key from db
			if err := s.dbClient.AsymSigningKeys.DeleteAsymSigningKey(ctx, &key); err != nil {
				s.logger.Errorf("failed to delete decommissioned signing key from db: %v", err)
				continue
			}

			// Delete key from plugin
			if err := s.jwsPlugin.Delete(ctx, key.Metadata.ID, key.PluginData); err != nil {
				s.logger.Errorf("failed to delete decommissioned signing key %q using plugin %q: %v", key.Metadata.ID, s.jwsProviderPluginType, err)
				continue
			}

			s.logger.Infof("successfully deleted decommissioned signing key: %q", key.Metadata.ID)
		}
	}

	return nil
}

func (s *signingKeyManager) checkForExpiredKey(ctx context.Context) error {
	activeKey, err := s.getActiveKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active signing key: %v", err)
	}

	// Check if active key is older than key rotation period
	if time.Since(*activeKey.Metadata.CreationTimestamp) > s.keyRotationPeriod {
		if err := s.rotateKey(ctx, activeKey); err != nil {
			if errors.ErrorCode(err) != errors.EOptimisticLock {
				s.logger.Errorf("failed to rotate expired signing key %q: %v", activeKey.Metadata.ID, err)
			}
			return nil
		}
		s.logger.Info("successfully rotated signing key")
	}

	return nil
}

func (s *signingKeyManager) rotateKey(ctx context.Context, expiredKey *models.AsymSigningKey) error {
	// Start db transaction
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for rotateKey: %v", txErr)
		}
	}()

	expiredKey.Status = models.AsymSigningKeyStatusDecommissioning
	if _, err := s.dbClient.AsymSigningKeys.UpdateAsymSigningKey(txContext, expiredKey); err != nil {
		return fmt.Errorf("failed to mark signing key as decommissioning %q: %w", expiredKey.Metadata.ID, err)
	}

	if _, err := s.createKey(txContext); err != nil {
		return fmt.Errorf("failed to create new signing key: %w", err)
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *signingKeyManager) syncKeySet(ctx context.Context) error {
	signingKeys, err := s.dbClient.AsymSigningKeys.GetAsymSigningKeys(ctx, &db.GetAsymSigningKeysInput{})
	if err != nil {
		return fmt.Errorf("failed to query for existing signing keys: %w", err)
	}

	jwkList := []jwk.Key{}

	for _, key := range signingKeys.AsymSigningKeys {
		if key.PublicKey != nil {
			jwkKey, err := jwk.ParseKey(key.PublicKey)
			if err != nil {
				return fmt.Errorf("failed to convert public signing key to jwk: %w", err)
			}
			jwkList = append(jwkList, jwkKey)
		}
	}

	s.keySetLock.Lock()
	defer s.keySetLock.Unlock()

	s.keySet.Clear()

	for _, k := range jwkList {
		if err := s.keySet.AddKey(k); err != nil {
			return err
		}
	}

	return nil
}

func (s *signingKeyManager) cleanupFailedKeys(ctx context.Context) error {
	keys, err := s.dbClient.AsymSigningKeys.GetAsymSigningKeys(ctx, &db.GetAsymSigningKeysInput{
		Filter: &db.AsymSigningKeyFilter{
			Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusCreating},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to query for existing signing keys: %w", err)
	}

	for _, key := range keys.AsymSigningKeys {
		// Check if created time is older than creation timeout
		if time.Since(*key.Metadata.CreationTimestamp) > keyCreationTimeout {
			s.logger.Infof("cleaning up failed signing key: %s", key.Metadata.ID)
			if err := s.dbClient.AsymSigningKeys.DeleteAsymSigningKey(ctx, &key); err != nil {
				return fmt.Errorf("failed to delete stale signing key %s: %w", key.Metadata.ID, err)
			}
		}
	}

	return nil
}

func (s *signingKeyManager) waitForActiveKey(ctx context.Context) (*models.AsymSigningKey, error) {
	s.logger.Info("waiting for signing key to become active")

	// Check every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeoutCh := time.After(keyCreationTimeout)

	for {
		select {
		case <-ticker.C:
			keys, err := s.dbClient.AsymSigningKeys.GetAsymSigningKeys(ctx, &db.GetAsymSigningKeysInput{
				Filter: &db.AsymSigningKeyFilter{
					Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusActive},
				},
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1), // Only one signing key can be active at a time
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to query for existing signing keys: %w", err)
			}
			if len(keys.AsymSigningKeys) > 0 {
				return &keys.AsymSigningKeys[0], nil
			}
		case <-timeoutCh:
			return nil, fmt.Errorf("timed out waiting for active signing key")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (s *signingKeyManager) createKey(ctx context.Context) (*models.AsymSigningKey, error) {
	s.logger.Infof("creating new signing key using plugin %q", s.jwsProviderPluginType)

	// Create key with status of creating
	key, err := s.dbClient.AsymSigningKeys.CreateAsymSigningKey(ctx, &models.AsymSigningKey{
		Status:     models.AsymSigningKeyStatusCreating,
		PluginType: s.jwsProviderPluginType,
	})
	if err != nil {
		if errors.ErrorCode(err) == errors.EOptimisticLock {
			s.logger.Info("signing key already being created by another instance")
			// Another instance created the key, wait for it to be active
			return s.waitForActiveKey(ctx)
		}
		return nil, fmt.Errorf("failed to create signing key: %w", err)
	}

	createResponse, err := s.jwsPlugin.Create(ctx, key.Metadata.ID)
	if err != nil {
		// Delete the key if we fail to create it in the plugin
		if delErr := s.dbClient.AsymSigningKeys.DeleteAsymSigningKey(ctx, key); delErr != nil {
			return nil, fmt.Errorf("failed to create signing key using plugin %q: %v, failed to delete key: %v", s.jwsProviderPluginType, err, delErr)
		}
		return nil, fmt.Errorf("failed to create signing key using plugin %q: %w", s.jwsProviderPluginType, err)
	}

	pubKeyBytes, err := json.Marshal(createResponse.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JWK key %v", err)
	}

	// Update key with public key and plugin data
	key.PubKeyID = createResponse.PublicKey.KeyID()
	key.PublicKey = pubKeyBytes
	key.PluginData = createResponse.KeyData
	key.Status = models.AsymSigningKeyStatusActive

	key, err = s.dbClient.AsymSigningKeys.UpdateAsymSigningKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to update signing key in db %q: %w", key.Metadata.ID, err)
	}

	s.logger.Infof("created new signing key %q", key.Metadata.ID)

	return key, nil
}

func (s *signingKeyManager) getActiveKey(ctx context.Context) (*models.AsymSigningKey, error) {
	keys, err := s.dbClient.AsymSigningKeys.GetAsymSigningKeys(ctx, &db.GetAsymSigningKeysInput{
		Filter: &db.AsymSigningKeyFilter{
			Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusActive},
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1), // Only one signing key can be active at a time
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query for existing signing keys: %w", err)
	}

	if len(keys.AsymSigningKeys) == 0 {
		return nil, fmt.Errorf("no active signing key found")
	}

	return &keys.AsymSigningKeys[0], nil
}

// GetPrivateClaims returns a map of the token's private claims
func getPrivateClaims(token jwt.Token) map[string]string {
	claimsMap := make(map[string]string)

	privClaims := token.PrivateClaims()
	for k, v := range privClaims {
		if strings.HasPrefix(k, privateClaimPrefix) {
			claimsMap[strings.TrimPrefix(k, privateClaimPrefix)] = v.(string)
		}
	}

	return claimsMap
}

func getPrivateClaim(claim string, token jwt.Token) (string, bool) {
	if claim, ok := token.Get(privateClaimPrefix + claim); ok {
		if val, ok := claim.(string); ok {
			return val, true
		}
	}
	return "", false
}
