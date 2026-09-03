package rules

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/in-toto/in-toto-golang/in_toto"
	ssldsse "github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/dsse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// generateAttestationTestKey returns an ephemeral ECDSA key pair and its PEM-encoded public key,
// generated fresh for this test process rather than committed to the repo.
func generateAttestationTestKey(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	pemBytes, err := cryptoutils.MarshalPublicKeyToPEM(&priv.PublicKey)
	require.NoError(t, err)
	return priv, string(pemBytes)
}

// signAttestation builds an in-toto statement naming moduleDigestHex as a sha256 subject with the
// given predicate type, wraps it in a DSSE envelope signed by priv, and returns the
// base64-of-JSON-envelope form VerifyModuleAttestation expects (matching what a module's
// GetAttestations returns in production).
func signAttestation(t *testing.T, priv *ecdsa.PrivateKey, moduleDigestHex, predicateType string) string {
	t.Helper()
	return signAttestationAs(t, priv, moduleDigestHex, predicateType, in_toto.PayloadType, in_toto.StatementInTotoV01)
}

// signAttestationAs is signAttestation with the DSSE payload type and in-toto statement type both
// overridable, so a test can build an otherwise well-formed, validly-signed envelope that declares
// something other than the standard in-toto values -- the type-confusion case
// verifyAttestationSatisfied's payload/statement type checks exist to reject.
func signAttestationAs(t *testing.T, priv *ecdsa.PrivateKey, moduleDigestHex, predicateType, payloadType, statementType string) string {
	t.Helper()

	statement := in_toto.Statement{
		StatementHeader: in_toto.StatementHeader{
			Type:          statementType,
			PredicateType: predicateType,
			Subject: []in_toto.Subject{
				{Name: "module", Digest: map[string]string{"sha256": moduleDigestHex}},
			},
		},
		Predicate: map[string]string{},
	}
	payload, err := json.Marshal(statement)
	require.NoError(t, err)

	signer, err := signature.LoadSigner(priv, crypto.SHA256)
	require.NoError(t, err)

	envelopeSigner, err := ssldsse.NewEnvelopeSigner(&dsse.SignerAdapter{SignatureSigner: signer})
	require.NoError(t, err)

	env, err := envelopeSigner.SignPayload(context.Background(), payloadType, payload)
	require.NoError(t, err)

	envBytes, err := json.Marshal(env)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(envBytes)
}

func TestVerifyModuleAttestation(t *testing.T) {
	const moduleDigestHex = "7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c"
	moduleDigest, err := hex.DecodeString(moduleDigestHex)
	require.NoError(t, err)

	signingKey, signingKeyPEM := generateAttestationTestKey(t)
	otherKey, _ := generateAttestationTestKey(t)

	validAttestation := signAttestation(t, signingKey, moduleDigestHex, "https://slsa.dev/provenance/v1")
	wrongKeyAttestation := signAttestation(t, otherKey, moduleDigestHex, "https://slsa.dev/provenance/v1")
	wrongDigestAttestation := signAttestation(t, signingKey, "0000000000000000000000000000000000000000000000000000000000ff", "https://slsa.dev/provenance/v1")
	// Validly signed under signingKey -- the type-confusion case: the signature and subject digest
	// are both correct, but the envelope declares a different DSSE payload type / in-toto statement
	// type than what this policy is willing to trust, e.g. reuse of the same key to sign some other
	// kind of DSSE-wrapped artifact.
	wrongPayloadTypeAttestation := signAttestationAs(t, signingKey, moduleDigestHex, "https://slsa.dev/provenance/v1", "application/vnd.other+json", in_toto.StatementInTotoV01)
	wrongStatementTypeAttestation := signAttestationAs(t, signingKey, moduleDigestHex, "https://slsa.dev/provenance/v1", in_toto.PayloadType, "https://example.com/Other/v1")

	const runID = "run-1"
	const currentStateVersionID = "state-version-1"
	const moduleSourceStr = "registry.example.com/group/module/aws"

	newModuleSource := func(t *testing.T) *registry.MockModuleRegistrySource {
		src := registry.NewMockModuleRegistrySource(t)
		src.On("IsTharsisModule").Return(true).Maybe()
		src.On("Source").Return(moduleSourceStr).Maybe()
		return src
	}

	tests := []struct {
		name            string
		input           func(t *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput
		stateVersion    *models.StateVersion
		stateVersionRun *models.Run
		attestations    []string
		expectDiag      bool
		expectErr       bool
	}{
		{
			name: "valid signature and predicate passes",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
					PredicateType:         ptr.String("https://slsa.dev/provenance/v1"),
				}
			},
			attestations: []string{validAttestation},
		},
		{
			name: "no predicate type required accepts any predicate",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
				}
			},
			attestations: []string{validAttestation},
		},
		{
			name: "wrong key fails",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
				}
			},
			attestations: []string{wrongKeyAttestation},
			expectDiag:   true,
		},
		{
			name: "right key wrong predicate type fails",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
					PredicateType:         ptr.String("https://example.com/other-predicate"),
				}
			},
			attestations: []string{validAttestation},
			expectDiag:   true,
		},
		{
			name: "subject digest mismatch fails",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
				}
			},
			attestations: []string{wrongDigestAttestation},
			expectDiag:   true,
		},
		{
			// Signature and subject digest are both correct; only the DSSE payload type is wrong.
			// Accepting this would mean a key reused to sign some other DSSE-wrapped artifact type
			// could satisfy this policy as long as the payload happened to parse as an in-toto
			// statement with the right subject -- see verifyAttestationSatisfied's payload/statement
			// type checks.
			name: "correct signature and digest but wrong dsse payload type fails",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
				}
			},
			attestations: []string{wrongPayloadTypeAttestation},
			expectDiag:   true,
		},
		{
			// Same type-confusion case one layer in: the DSSE payload type is the standard in-toto
			// value, but the statement's own _type field is not in-toto v0.1.
			name: "correct signature and digest but wrong in-toto statement type fails",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
				}
			},
			attestations: []string{wrongStatementTypeAttestation},
			expectDiag:   true,
		},
		{
			// The first candidate is undecodable garbage; the second is a real, validly-signed
			// attestation. A decode failure on one candidate must not abort evaluation of the rest
			// of the list -- see verifyAttestationSatisfied.
			name: "a malformed attestation earlier in the list does not block a valid one later in it",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
					PredicateType:         ptr.String("https://slsa.dev/provenance/v1"),
				}
			},
			attestations: []string{"not-valid-base64!!!", validAttestation},
		},
		{
			name: "no attestations at all fails",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
				}
			},
			attestations: []string{},
			expectDiag:   true,
		},
		{
			name: "no module source fails closed",
			input: func(_ *testing.T, _ *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource: nil,
					PublicKey:    signingKeyPEM,
				}
			},
			expectDiag: true,
		},
		{
			name: "non-tharsis module source fails closed",
			input: func(_ *testing.T, _ *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				nonTharsis := registry.NewMockModuleRegistrySource(t)
				nonTharsis.On("IsTharsisModule").Return(false)
				return &VerifyModuleAttestationInput{
					ModuleSource: nonTharsis,
					PublicKey:    signingKeyPEM,
				}
			},
			expectDiag: true,
		},
		{
			name: "missing module digest errors",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
				}
			},
			expectErr: true,
		},
		{
			name: "missing module semantic version errors",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource: src,
					ModuleDigest: moduleDigest,
					PublicKey:    signingKeyPEM,
				}
			},
			expectErr: true,
		},
		{
			name: "malformed public key errors",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             "not-a-key",
				}
			},
			attestations: []string{validAttestation},
			expectErr:    true,
		},
		{
			name: "lineage: manual state push fails",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
					VerifyStateLineage:    true,
					CurrentStateVersionID: ptr.String(currentStateVersionID),
				}
			},
			stateVersion: &models.StateVersion{Metadata: models.ResourceMetadata{ID: currentStateVersionID}, RunID: nil},
			expectDiag:   true,
		},
		{
			name: "lineage: different module source fails",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
					VerifyStateLineage:    true,
					CurrentStateVersionID: ptr.String(currentStateVersionID),
				}
			},
			stateVersion:    &models.StateVersion{Metadata: models.ResourceMetadata{ID: currentStateVersionID}, RunID: ptr.String(runID)},
			stateVersionRun: &models.Run{Metadata: models.ResourceMetadata{ID: runID}, ModuleSource: ptr.String("registry.example.com/group/other-module/aws")},
			expectDiag:      true,
		},
		{
			name: "lineage: destroy run bypasses module source check",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
					VerifyStateLineage:    true,
					CurrentStateVersionID: ptr.String(currentStateVersionID),
				}
			},
			stateVersion:    &models.StateVersion{Metadata: models.ResourceMetadata{ID: currentStateVersionID}, RunID: ptr.String(runID)},
			stateVersionRun: &models.Run{Metadata: models.ResourceMetadata{ID: runID}, IsDestroy: true, Status: models.RunApplied},
			attestations:    []string{validAttestation},
		},
		{
			// A destroy that errored or was canceled mid-apply can leave live resources behind
			// while the run is still IsDestroy == true, so it must not get the same free pass as
			// one that actually completed -- otherwise a later run under a different module source
			// would inherit those un-attested resources.
			name: "lineage: incomplete destroy run does not bypass the check",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
					VerifyStateLineage:    true,
					CurrentStateVersionID: ptr.String(currentStateVersionID),
				}
			},
			stateVersion:    &models.StateVersion{Metadata: models.ResourceMetadata{ID: currentStateVersionID}, RunID: ptr.String(runID)},
			stateVersionRun: &models.Run{Metadata: models.ResourceMetadata{ID: runID}, IsDestroy: true, Status: models.RunErrored},
			expectDiag:      true,
		},
		{
			// A destroy scoped with -target only removes the named resources and their
			// dependents, not necessarily everything the module manages, even though the run
			// itself completed successfully (RunApplied). It must not get the destroy exemption:
			// resources left behind could be from a different, unattested module version than the
			// destroy run's own module source would suggest, so the run's ModuleSource -- left nil
			// here -- is deliberately not what this case is testing against.
			name: "lineage: targeted destroy run does not bypass the check",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
					VerifyStateLineage:    true,
					CurrentStateVersionID: ptr.String(currentStateVersionID),
				}
			},
			stateVersion: &models.StateVersion{Metadata: models.ResourceMetadata{ID: currentStateVersionID}, RunID: ptr.String(runID)},
			stateVersionRun: &models.Run{
				Metadata:        models.ResourceMetadata{ID: runID},
				IsDestroy:       true,
				Status:          models.RunApplied,
				TargetAddresses: []string{"aws_instance.example"},
			},
			expectDiag: true,
		},
		{
			name: "lineage: matching module source passes",
			input: func(_ *testing.T, src *registry.MockModuleRegistrySource) *VerifyModuleAttestationInput {
				return &VerifyModuleAttestationInput{
					ModuleSource:          src,
					ModuleDigest:          moduleDigest,
					ModuleSemanticVersion: ptr.String("1.0.0"),
					PublicKey:             signingKeyPEM,
					VerifyStateLineage:    true,
					CurrentStateVersionID: ptr.String(currentStateVersionID),
				}
			},
			stateVersion:    &models.StateVersion{Metadata: models.ResourceMetadata{ID: currentStateVersionID}, RunID: ptr.String(runID)},
			stateVersionRun: &models.Run{Metadata: models.ResourceMetadata{ID: runID}, ModuleSource: ptr.String(moduleSourceStr)},
			attestations:    []string{validAttestation},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := newModuleSource(t)
			if tt.attestations != nil {
				src.On("GetAttestations", mock.Anything, mock.Anything, mock.Anything).Return(tt.attestations, nil)
			}

			mockStateVersions := db.NewMockStateVersions(t)
			mockRuns := db.NewMockRuns(t)
			if tt.stateVersion != nil {
				mockStateVersions.On("GetStateVersionByID", mock.Anything, currentStateVersionID).Return(tt.stateVersion, nil)
				if tt.stateVersionRun != nil {
					mockRuns.On("GetRunByID", mock.Anything, runID).Return(tt.stateVersionRun, nil)
				}
			}

			dbClient := &db.Client{StateVersions: mockStateVersions, Runs: mockRuns}

			diag, err := VerifyModuleAttestation(context.Background(), dbClient, tt.input(t, src))

			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.expectDiag {
				assert.NotEmpty(t, diag)
			} else {
				assert.Empty(t, diag)
			}
		})
	}
}
