package rules

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestEnforceRules(t *testing.T) {
	managedIdentity := models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "123",
		},
		ResourcePath: "test-group/test-managed-identity",
	}

	runID := "run-1"
	moduleID := "module-1"
	currentStateVersionID := "state-version-1"
	pubKey := "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE998KMh+Icdiqo9sz7KT/dyvImVQs\nJRWsKi78jT0htK6/B5bgxaNWYX1FElTrdEwVlF3AhU0n1gdffZkerSduIQ==\n-----END PUBLIC KEY-----"
	validModuleDigestHex := "7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7d"
	validAttestation := "eyJwYXlsb2FkVHlwZSI6ImFwcGxpY2F0aW9uL3ZuZC5pbi10b3RvK2pzb24iLCJwYXlsb2FkIjoiZXlKZmRIbHdaU0k2SW1oMGRIQnpPaTh2YVc0dGRHOTBieTVwYnk5VGRHRjBaVzFsYm5RdmRqQXVNU0lzSW5CeVpXUnBZMkYwWlZSNWNHVWlPaUpqYjNOcFoyNHVjMmxuYzNSdmNtVXVaR1YyTDJGMGRHVnpkR0YwYVc5dUwzWXhJaXdpYzNWaWFtVmpkQ0k2VzNzaWJtRnRaU0k2SW1Kc2IySWlMQ0prYVdkbGMzUWlPbnNpYzJoaE1qVTJJam9pTjJGbE5EY3haV1F4T0RNNU5UTXpPVFUzTW1ZMU1qWTFZamd6TlRnMk1HVXlPR0V5WmpnMU1ERTJORFUxTWpFMFkySXlNVFJpWVdabE5EUXlNbU0zWkNKOWZWMHNJbkJ5WldScFkyRjBaU0k2ZXlKRVlYUmhJam9pZTF3aWRtVnlhV1pwWldSY0lqcDBjblZsZlZ4dUlpd2lWR2x0WlhOMFlXMXdJam9pTWpBeU1pMHhNaTB4TWxReE5EbzFOam8wTVZvaWZYMD0iLCJzaWduYXR1cmVzIjpbeyJrZXlpZCI6IiIsInNpZyI6Ik1FVUNJUURIZGk2UkI2YktESVlPZ3duZkwvaVU5UlQ2a2xyaGRUaEt1NHkzK29JZGNBSWdaVmRQeUczaGhsQTJNZnJxYTkvVUsrOFF4c2d4T2pYcGxGd2JxWW1nQnkwPSJ9XX0="

	validModuleDigest, err := hex.DecodeString(validModuleDigestHex)
	if err != nil {
		t.Fatal(err)
	}

	// Test cases
	tests := []struct {
		callerType      string
		runDetails      *RunDetails
		stateVersion    *models.StateVersion
		stateVersionRun *models.Run
		name            string
		expectErrorCode errors.CodeType
		rules           []models.ManagedIdentityAccessRule
		attestations    []models.TerraformModuleAttestation
		teams           []models.Team
	}{
		{
			name:       "user is allowed to create run because user is team member and team is in managed identity access rule",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage: models.JobPlanType,
			},
			teams: []models.Team{
				{
					Metadata: models.ResourceMetadata{
						ID: "42",
					},
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					AllowedTeamIDs:    []string{"42"},
				},
			},
		},
		{
			name:       "user is not allowed to create run because user is in not in the required team",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage: models.JobPlanType,
			},
			teams: []models.Team{
				{
					Metadata: models.ResourceMetadata{
						ID: "01",
					},
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					AllowedTeamIDs:    []string{"42"},
				},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "passing eligible principals rule matching run stage and allowed user",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage: models.JobPlanType,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					AllowedUserIDs:    []string{"123"},
				},
			},
		},
		{
			name:       "eligible principals rule does not include required user",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage: models.JobPlanType,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					AllowedUserIDs:    []string{"invalid"},
				},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "no users are allowed to apply the managed identity",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage: models.JobPlanType,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					AllowedUserIDs:    []string{},
				},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "service account is allowed to apply run because service account is in managed identity access rule",
			callerType: "serviceAccount",
			runDetails: &RunDetails{
				RunStage: models.JobApplyType,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:                 models.JobApplyType,
					ManagedIdentityID:        managedIdentity.Metadata.ID,
					AllowedUserIDs:           []string{},
					AllowedServiceAccountIDs: []string{"sa1"},
				},
			},
		},
		{
			name:       "service account is forbidden to apply run because managed identity access rule doesn't allow it",
			callerType: "serviceAccount",
			runDetails: &RunDetails{
				RunStage: models.JobApplyType,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:                 models.JobApplyType,
					ManagedIdentityID:        managedIdentity.Metadata.ID,
					AllowedUserIDs:           []string{},
					AllowedServiceAccountIDs: []string{"sa2"},
				},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "service account is allowed to apply run because managed identity doesn't have any access rules",
			callerType: "serviceAccount",
			runDetails: &RunDetails{
				RunStage: models.JobPlanType,
			},
			rules: []models.ManagedIdentityAccessRule{},
		},
		{
			name:       "passing eligible principals rule with 2 rules of the same type",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage: models.JobPlanType,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					AllowedUserIDs:    []string{"invalid"},
				},
				{
					Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					AllowedUserIDs:    []string{"123"},
				},
			},
		},
		{
			name:       "eligible principals rule with 2 rules of the same type but no rules are satisfied",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage: models.JobPlanType,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					AllowedUserIDs:    []string{"invalid1"},
				},
				{
					Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					AllowedUserIDs:    []string{"invalid2"},
				},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "passing module attestion exists for module",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage:     models.JobPlanType,
				ModuleID:     &moduleID,
				ModuleDigest: validModuleDigest,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey},
					},
				},
			},
			attestations: []models.TerraformModuleAttestation{
				{
					ModuleID:      moduleID,
					SchemaType:    "https://in-toto.io/Statement/v0.1",
					PredicateType: "cosign.sigstore.dev/attestation/v1",
					Digests:       []string{validModuleDigestHex},
					Data:          validAttestation,
				},
			},
		},
		{
			name:       "passing with multiple attestion rules",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage:     models.JobPlanType,
				ModuleID:     &moduleID,
				ModuleDigest: validModuleDigest,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey, PredicateType: ptr.String("http://invalid-predicate")},
					},
				},
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey},
					},
				},
			},
			attestations: []models.TerraformModuleAttestation{
				{
					ModuleID:      moduleID,
					SchemaType:    "https://in-toto.io/Statement/v0.1",
					PredicateType: "cosign.sigstore.dev/attestation/v1",
					Digests:       []string{validModuleDigestHex},
					Data:          validAttestation,
				},
			},
		},
		{
			name:       "attestation signature matches but predicate does not",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage:     models.JobPlanType,
				ModuleID:     &moduleID,
				ModuleDigest: validModuleDigest,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey, PredicateType: ptr.String("http://invalid-predicate")},
					},
				},
			},
			attestations: []models.TerraformModuleAttestation{
				{
					ModuleID:      moduleID,
					SchemaType:    "https://in-toto.io/Statement/v0.1",
					PredicateType: "cosign.sigstore.dev/attestation/v1",
					Digests:       []string{validModuleDigestHex},
					Data:          validAttestation,
				},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "attestation signature does not match public key",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage:     models.JobPlanType,
				ModuleID:     &moduleID,
				ModuleDigest: validModuleDigest,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE+SkwqyxreyOIQ5IoMvVB8BQskaOW\nQGncVVeiM1zebs6I4eJGc3labfRR6IeSO9a0EGP5AhsjiG7ywcBmhzRpfw==\n-----END PUBLIC KEY-----"},
					},
				},
			},
			attestations: []models.TerraformModuleAttestation{
				{
					ModuleID:      moduleID,
					SchemaType:    "https://in-toto.io/Statement/v0.1",
					PredicateType: "cosign.sigstore.dev/attestation/v1",
					Digests:       []string{validModuleDigestHex},
					Data:          validAttestation,
				},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "negative: attestation policy with multiple required attestations",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage:     models.JobPlanType,
				ModuleID:     &moduleID,
				ModuleDigest: validModuleDigest,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey},
						{PublicKey: "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE+SkwqyxreyOIQ5IoMvVB8BQskaOW\nQGncVVeiM1zebs6I4eJGc3labfRR6IeSO9a0EGP5AhsjiG7ywcBmhzRpfw==\n-----END PUBLIC KEY-----"},
					},
				},
			},
			attestations: []models.TerraformModuleAttestation{
				{
					ModuleID:      moduleID,
					SchemaType:    "https://in-toto.io/Statement/v0.1",
					PredicateType: "cosign.sigstore.dev/attestation/v1",
					Digests:       []string{validModuleDigestHex},
					Data:          validAttestation,
				},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "module attestion does not exist for module",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage:     models.JobPlanType,
				ModuleID:     &moduleID,
				ModuleDigest: validModuleDigest,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey},
					},
				},
			},
			attestations:    []models.TerraformModuleAttestation{},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "passing multiple rules with different types",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage:     models.JobPlanType,
				ModuleID:     &moduleID,
				ModuleDigest: validModuleDigest,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					AllowedUserIDs:    []string{"123"},
				},
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey},
					},
				},
			},
			attestations: []models.TerraformModuleAttestation{
				{
					ModuleID:      moduleID,
					SchemaType:    "https://in-toto.io/Statement/v0.1",
					PredicateType: "cosign.sigstore.dev/attestation/v1",
					Digests:       []string{validModuleDigestHex},
					Data:          validAttestation,
				},
			},
		},
		{
			name:       "multiple rules with different types not passing",
			callerType: "user",
			runDetails: &RunDetails{
				RunStage:     models.JobPlanType,
				ModuleID:     &moduleID,
				ModuleDigest: validModuleDigest,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					AllowedUserIDs:    []string{"123"},
				},
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey, PredicateType: ptr.String("http://invalid-predicate")},
					},
				},
			},
			attestations: []models.TerraformModuleAttestation{
				{
					ModuleID:      moduleID,
					SchemaType:    "https://in-toto.io/Statement/v0.1",
					PredicateType: "cosign.sigstore.dev/attestation/v1",
					Digests:       []string{validModuleDigestHex},
					Data:          validAttestation,
				},
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "workspace's current state version was created manually",
			callerType: "user",
			runDetails: &RunDetails{
				CurrentStateVersionID: &currentStateVersionID,
				RunStage:              models.JobPlanType,
				ModuleID:              &moduleID,
				ModuleDigest:          validModuleDigest,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey},
					},
				},
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: currentStateVersionID,
				},
				// RunID field being empty means it was created manually.
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "workspace's current state version was created without a module source",
			callerType: "user",
			runDetails: &RunDetails{
				CurrentStateVersionID: &currentStateVersionID,
				RunStage:              models.JobPlanType,
				ModuleID:              &moduleID,
				ModuleDigest:          validModuleDigest,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey},
					},
				},
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: currentStateVersionID,
				},
				RunID: &runID,
			},
			stateVersionRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: runID,
				},
				// ModuleSource field being nil means there was no module being used.
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "workspace's current state version was created by another module than expected",
			callerType: "user",
			runDetails: &RunDetails{
				CurrentStateVersionID: &currentStateVersionID,
				RunStage:              models.JobPlanType,
				ModuleID:              &moduleID,
				ModuleDigest:          validModuleDigest,
				ModuleSource:          ptr.String("some-module-source"),
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey},
					},
				},
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: currentStateVersionID,
				},
				RunID: &runID,
			},
			stateVersionRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: runID,
				},
				ModuleSource: ptr.String("some-other-module-source"),
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "run associated with workspace's current state version was a destroy type",
			callerType: "user",
			runDetails: &RunDetails{
				CurrentStateVersionID: &currentStateVersionID,
				RunStage:              models.JobPlanType,
				ModuleID:              &moduleID,
				ModuleDigest:          validModuleDigest,
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					Type:              models.ManagedIdentityAccessRuleModuleAttestation,
					RunStage:          models.JobPlanType,
					ManagedIdentityID: managedIdentity.Metadata.ID,
					ModuleAttestationPolicies: []models.ManagedIdentityAccessRuleModuleAttestationPolicy{
						{PublicKey: pubKey},
					},
				},
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: currentStateVersionID,
				},
				RunID: &runID,
			},
			stateVersionRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: runID,
				},
				IsDestroy: true,
			},
			attestations: []models.TerraformModuleAttestation{
				{
					ModuleID:      moduleID,
					SchemaType:    "https://in-toto.io/Statement/v0.1",
					PredicateType: "cosign.sigstore.dev/attestation/v1",
					Digests:       []string{validModuleDigestHex},
					Data:          validAttestation,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbClient := db.Client{}

			// Select userCaller or serviceAccountCaller.
			var testCaller auth.Caller
			switch test.callerType {
			case "user":
				testCaller = auth.NewUserCaller(
					&models.User{
						Metadata: models.ResourceMetadata{
							ID: "123",
						},
						Admin:    false,
						Username: "user1",
					},
					nil,
					&dbClient,
				)
			case "serviceAccount":
				testCaller = auth.NewServiceAccountCaller(
					"sa1",
					"groupA/sa1",
					nil,
					nil,
				)
			}

			ctx, cancel := context.WithCancel(auth.WithCaller(context.Background(), testCaller))
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockStateVersions := db.NewMockStateVersions(t)
			mockRuns := db.NewMockRuns(t)
			mockTerraformModuleAttestations := db.NewMockTerraformModuleAttestations(t)
			mockTeams := db.NewMockTeams(t)

			mockManagedIdentities.On("GetManagedIdentityAccessRules", ctx, &db.GetManagedIdentityAccessRulesInput{
				Filter: &db.ManagedIdentityAccessRuleFilter{
					ManagedIdentityID: &managedIdentity.Metadata.ID,
				},
			}).Return(&db.ManagedIdentityAccessRulesResult{
				ManagedIdentityAccessRules: test.rules,
			}, nil)

			if test.attestations != nil {
				// Determine how many times the mock will be called
				callCount := 0
				for _, rule := range test.rules {
					if rule.Type == models.ManagedIdentityAccessRuleModuleAttestation && rule.RunStage == test.runDetails.RunStage {
						// The mock will be called once for each module attestation rule
						callCount++
					}
				}

				moduleDigest := hex.EncodeToString(test.runDetails.ModuleDigest)
				mockTerraformModuleAttestations.On("GetModuleAttestations", ctx, &db.GetModuleAttestationsInput{
					Filter: &db.TerraformModuleAttestationFilter{
						ModuleID: test.runDetails.ModuleID,
						Digest:   &moduleDigest,
					},
				}).Return(&db.ModuleAttestationsResult{
					ModuleAttestations: test.attestations,
				}, nil).Times(callCount)
			}

			if test.stateVersion != nil {
				mockStateVersions.On("GetStateVersion", mock.Anything, currentStateVersionID).Return(test.stateVersion, nil)

				if test.stateVersionRun != nil {
					mockRuns.On("GetRun", mock.Anything, runID).Return(test.stateVersionRun, nil)
				}
			}

			if test.teams != nil {
				mockTeams.On("GetTeams", ctx, mock.Anything).
					Return(&db.TeamsResult{Teams: test.teams}, nil).Maybe()
			} else {
				mockTeams.On("GetTeams", ctx, mock.Anything).
					Return(&db.TeamsResult{Teams: []models.Team{}}, nil).Maybe()
			}

			dbClient.ManagedIdentities = mockManagedIdentities
			dbClient.TerraformModuleAttestations = mockTerraformModuleAttestations
			dbClient.Teams = mockTeams
			dbClient.StateVersions = mockStateVersions
			dbClient.Runs = mockRuns

			enforcer := NewRuleEnforcer(&dbClient)

			err := enforcer.EnforceRules(ctx, &managedIdentity, test.runDetails)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err), "unexpected error returned %v", err)
				return
			}

			require.Nil(t, err)
		})
	}
}
