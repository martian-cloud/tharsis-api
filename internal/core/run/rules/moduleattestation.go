package rules

import (
	"context"
	"crypto"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/in-toto/in-toto-golang/in_toto"
	ssldsse "github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/dsse"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// VerifyModuleAttestationInput is the input to VerifyModuleAttestation.
type VerifyModuleAttestationInput struct {
	// ModuleSource is the run's resolved registry module source. A run built from a
	// configuration version (no registry module source) has none: the policy fails closed on a
	// nil ModuleSource rather than passing it, unlike the managed identity rule this shares its
	// verification core with.
	ModuleSource          registry.ModuleRegistrySource
	ModuleDigest          []byte
	ModuleSemanticVersion *string
	// PublicKey is a PEM-encoded public key the attestation's signature must verify against.
	PublicKey string
	// PredicateType, when set, is the in-toto predicate type the attestation must declare. Nil
	// means any predicate type satisfies the policy.
	PredicateType *string
	// VerifyStateLineage additionally requires the workspace's current state to have been
	// written by a run using the same module source.
	VerifyStateLineage bool
	// CurrentStateVersionID is the workspace's current state version at evaluation time. Only
	// consulted when VerifyStateLineage is true.
	CurrentStateVersionID *string
}

// VerifyModuleAttestation reports whether the module named by input.ModuleSource satisfies a
// module attestation policy: it must carry an in-toto attestation signed by PublicKey, with a
// matching PredicateType when one is required, and — when VerifyStateLineage is set — the
// workspace's current state must have been written by a run using the same module source.
//
// A non-empty return value is a diagnostic: verification completed and the module did not satisfy
// the policy. A non-nil error means verification itself could not be completed (registry I/O, a
// malformed public key) — the caller should treat that as an evaluation error, not a policy
// outcome, since redelivering the check may succeed once the underlying problem clears.
func VerifyModuleAttestation(ctx context.Context, dbClient *db.Client, input *VerifyModuleAttestationInput) (string, error) {
	// A run with no registry module source (built from a configuration version) fails the
	// policy: there is nothing to verify an attestation against. This is the opposite of the
	// managed identity rule's behavior for the same condition (see enforceModuleAttestationRuleType),
	// which is intentional — see requirement 4 in the module attestation policy plan.
	if input.ModuleSource == nil || !input.ModuleSource.IsTharsisModule() {
		return "module attestation policy requires the run's module to come from a Tharsis module registry", nil
	}

	if input.ModuleDigest == nil {
		return "", errors.New("module digest must be defined to evaluate a module attestation policy")
	}
	if input.ModuleSemanticVersion == nil {
		return "", errors.New("module semantic version must be defined to evaluate a module attestation policy")
	}

	if input.VerifyStateLineage {
		diag, err := verifyStateLineage(ctx, dbClient, input.CurrentStateVersionID, input.ModuleSource)
		if err != nil {
			return "", err
		}
		if diag != "" {
			return diag, nil
		}
	}

	moduleDigest := hex.EncodeToString(input.ModuleDigest)

	attestations, err := input.ModuleSource.GetAttestations(ctx, *input.ModuleSemanticVersion, moduleDigest)
	if err != nil {
		return "", err
	}

	return verifyAttestationSatisfied(ctx, attestations, moduleDigest, input.PublicKey, input.PredicateType)
}

// verifyStateLineage requires that the workspace's current state version was written by a run (not
// a manual state push) and, unless that run was a whole-module destroy that completed
// successfully, that the run's module source matches moduleSource. A destroy scoped with
// TargetAddresses does not get the destroy exemption: it may have removed only some of the
// module's resources, and the ones left behind could be from a different, unattested module
// version than the destroy run's own module source would suggest -- so it fails lineage outright
// rather than being compared against moduleSource, which would not actually speak to what remains.
// Returns a non-empty diagnostic when lineage doesn't hold; a nil currentStateVersionID (no current
// state yet) has nothing to verify and passes.
func verifyStateLineage(
	ctx context.Context,
	dbClient *db.Client,
	currentStateVersionID *string,
	moduleSource registry.ModuleRegistrySource,
) (string, error) {
	if currentStateVersionID == nil {
		return "", nil
	}

	stateVersion, err := dbClient.StateVersions.GetStateVersionByID(ctx, *currentStateVersionID)
	if err != nil {
		return "", err
	}
	if stateVersion == nil {
		return "", fmt.Errorf("failed to get state version with ID %s", *currentStateVersionID)
	}

	if stateVersion.RunID == nil {
		return "workspace's current state version was modified manually, which is not permitted when a module attestation policy verifies state lineage", nil
	}

	run, err := dbClient.Runs.GetRunByID(ctx, *stateVersion.RunID)
	if err != nil {
		return "", err
	}
	if run == nil {
		return "", fmt.Errorf("failed to get run with ID %s associated with state version %s", *stateVersion.RunID, *currentStateVersionID)
	}

	// A destroy run's module source is not compared against moduleSource: destroying is what
	// removes the module's resources from state, not what places them there, so a destroy is
	// exempt from the lineage requirement regardless of which module wrote the state it destroyed
	// -- but only when that destroy actually removed everything. Two ways it might not have:
	//
	//   - It didn't complete. An errored or canceled apply can leave live resources behind while
	//     the run is still IsDestroy == true. RunApplied is reached only once the apply chain
	//     completes successfully (see statemachine/run.go), so it is what rules this out.
	//
	//   - It was targeted (-target). TargetAddresses being set means only the named resources and
	//     their dependents were removed, not necessarily everything the module manages, even on a
	//     fully successful (RunApplied) destroy.
	//
	// Either way the run gets no exemption -- and unlike a whole-module destroy, it isn't compared
	// against moduleSource either: the destroy run's module source describes what it destroyed, not
	// what (if anything) it left behind, so that comparison would not actually speak to what remains.
	if run.IsDestroy {
		if run.Status != models.RunApplied {
			return "workspace's current state version was written by a destroy run that did not complete successfully, and the module attestation policy verifies state lineage", nil
		}
		if len(run.TargetAddresses) > 0 {
			return "workspace's current state version was written by a targeted destroy run (-target), which may not have removed every resource the module manages, and the module attestation policy verifies state lineage", nil
		}
		return "", nil
	}

	if run.ModuleSource == nil || *run.ModuleSource != moduleSource.Source() {
		return "workspace's current state version was not created by the same module source, and the module attestation policy verifies state lineage", nil
	}

	return "", nil
}

// verifyAttestationSatisfied reports whether any attestation in the given list is a validly
// signed DSSE envelope of the standard in-toto payload type, under publicKeyPEM, carrying an
// in-toto v0.1 statement whose subject names moduleDigest and (when predicateType is set) declares
// that predicate type. A malformed candidate (undecodable, unparsable, wrong payload/statement
// type) is diagnosed and skipped, same as a signature that fails verification, so one bad
// attestation in the list cannot block a later, validly-signed one. Returns a non-empty diagnostic
// collecting why each candidate attestation was rejected when none match.
func verifyAttestationSatisfied(
	ctx context.Context,
	attestations []string,
	moduleDigest string,
	publicKeyPEM string,
	predicateType *string,
) (string, error) {
	pub, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(publicKeyPEM))
	if err != nil {
		return "", err
	}

	verifier, err := signature.LoadVerifier(pub, crypto.SHA256)
	if err != nil {
		return "", err
	}

	if len(attestations) == 0 {
		return "no attestations found for this module version", nil
	}

	diagnostics := []string{}
	for _, attestation := range attestations {
		decodedSig, err := base64.StdEncoding.DecodeString(attestation)
		if err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("failed to decode attestation signature: %v", err))
			continue
		}

		env := ssldsse.Envelope{}
		if err = json.Unmarshal(decodedSig, &env); err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("failed to unmarshal dsse envelope: %v", err))
			continue
		}

		dssev, err := ssldsse.NewEnvelopeVerifier(&dsse.VerifierAdapter{SignatureVerifier: verifier})
		if err != nil {
			return "", fmt.Errorf("failed to create new dsse envelope verifier: %v", err)
		}

		if _, err = dssev.Verify(ctx, &env); err != nil {
			diagnostics = append(diagnostics, "signature is not valid for required public key")
			continue
		}

		// The DSSE signature covers PayloadType (it is part of the pre-authentication encoding
		// dssev.Verify checks), but Verify only proves the envelope was signed by this key with
		// this declared type -- it says nothing about what that type is. Without this check, a
		// validly-signed DSSE envelope of some other payload type would still be accepted here as
		// long as its payload happened to decode as JSON shaped like an in-toto statement with the
		// right subject digest: a type-confusion risk if the signing key is ever reused for another
		// DSSE-signed artifact type, worst when predicateType is left unset below.
		if env.PayloadType != in_toto.PayloadType {
			diagnostics = append(diagnostics, fmt.Sprintf("unexpected dsse payload type, expected=%s actual=%s", in_toto.PayloadType, env.PayloadType))
			continue
		}

		decodedPredicate, err := base64.StdEncoding.DecodeString(env.Payload)
		if err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("failed to decode dsse payload: %v", err))
			continue
		}
		var statement in_toto.Statement
		if err := json.Unmarshal(decodedPredicate, &statement); err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("failed to decode attestation predicate: %v", err))
			continue
		}

		// Same reasoning as the PayloadType check above, one layer in: the statement's own _type
		// field is what an in-toto consumer is supposed to gate on before trusting the rest of the
		// document's shape.
		if statement.Type != in_toto.StatementInTotoV01 {
			diagnostics = append(diagnostics, fmt.Sprintf("unexpected in-toto statement type, expected=%s actual=%s", in_toto.StatementInTotoV01, statement.Type))
			continue
		}

		if statement.Subject == nil {
			diagnostics = append(diagnostics, "no subject in intoto statement")
			continue
		}

		foundSubject := false
		for _, subj := range statement.Subject {
			if shaSum, ok := subj.Digest["sha256"]; ok && shaSum == moduleDigest {
				foundSubject = true
				break
			}
		}
		if !foundSubject {
			diagnostics = append(diagnostics, fmt.Sprintf("subject with digest %s not found in module attestation", moduleDigest))
			continue
		}

		if predicateType != nil && statement.PredicateType != *predicateType {
			diagnostics = append(diagnostics, fmt.Sprintf("invalid predicate type, expected=%s actual=%s", *predicateType, statement.PredicateType))
			continue
		}

		return "", nil
	}

	return fmt.Sprintf("no attestation satisfies the module attestation policy (%s)", strings.Join(diagnostics, "; ")), nil
}
