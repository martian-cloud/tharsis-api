package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"unicode/utf8"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// maxPolicyMessagesSize bounds the total bytes of the messages one policy may report. Exceeding it is
// rejected rather than truncated: a policy producing this much output is misconfigured, and silently
// keeping a fraction of it would report a verdict on evidence the user cannot see in full.
const maxPolicyMessagesSize = 1 << 20 // 1 MiB

// maxPolicyCheckMessagesSummarySize caps the check's summary of the messages its policies reported.
// Small on purpose: it rides on the check's own row and is read by views listing many checks, so it
// holds a preview rather than the full output.
const maxPolicyCheckMessagesSummarySize = 2 << 10 // 2 KiB

// RunPolicyOutcome is a single policy's reported result, keyed by the run's PolicyCheckPolicy ID (the
// evaluator echoes the id it received). Messages holds one entry per violation, in display order, and
// is empty when the policy passed. The enforcement level is taken from the pinned policy, not
// supplied by the caller.
type RunPolicyOutcome struct {
	PolicyID string
	Messages []string
	Passed   bool
}

// ReportRunPolicyOutcomes stamps the per-policy-set results onto a run's policy check node and
// sets the check verdict, in a single transaction. It is the sole writer of the check verdict:
// the verdict is derived from the failed policies' effective enforcement — errored if a
// hard-mandatory policy set failed, awaiting_override if a soft-mandatory failed (no hard
// fail), passed otherwise (advisory-only failures do not block). The owning task stage node then
// aggregates the check verdicts and projects the stage status onto the run.
type ReportRunPolicyOutcomes struct {
	dbClient      *db.Client
	artifactStore workspace.ArtifactStore
	PolicyCheckID string
	Outcomes      []RunPolicyOutcome

	// Populated by Prepare.
	runID string
	// messageKeys and messageRetainFns are keyed by policy id and hold only the policies that
	// reported messages; a policy absent from them reported none.
	messageKeys      map[string]string
	messageRetainFns map[string]db.RetainObjectRefFunc
	// messagesSummary previews the messages across every policy, for the check row itself. Nil when
	// no policy reported any.
	messagesSummary *models.PolicyCheckMessagesSummary
}

// Prepare resolves the run owning the policy check node and writes each policy's violation messages
// to object storage. It runs before the transaction is opened, which is safe because every upload
// mints a fresh key: a losing concurrent report cannot overwrite another's object, and the returned
// retain callbacks are only invoked once Execute is inside the transaction. An upload whose
// transaction never commits is left unlinked for the janitor.
//
// A policy reporting more than maxPolicyMessagesSize is rejected with EInvalid, which fails the
// policy-eval job rather than recording a partial result. Invalid UTF-8 needs no handling here: both
// the uploaded object and the check summary go through encoding/json, which replaces invalid bytes.
func (c *ReportRunPolicyOutcomes) Prepare(ctx context.Context) error {
	run, err := c.dbClient.Runs.GetRunByNodeID(ctx, c.PolicyCheckID)
	if err != nil {
		return errors.Wrap(err, "failed to get run by policy check node ID")
	}
	if run == nil {
		return errors.New("policy check node with id %s not found", c.PolicyCheckID, errors.WithErrorCode(errors.ENotFound))
	}
	c.runID = run.Metadata.ID

	c.messageKeys = map[string]string{}
	c.messageRetainFns = map[string]db.RetainObjectRefFunc{}

	var summarySource []string
	for _, outcome := range c.Outcomes {
		if len(outcome.Messages) == 0 {
			continue
		}

		body, err := json.Marshal(outcome.Messages)
		if err != nil {
			return errors.Wrap(err, "failed to marshal policy messages")
		}

		// Measured on the encoded array rather than the strings, so the bound covers what is actually
		// stored — a policy reporting a huge number of tiny messages is as expensive as one reporting
		// a few enormous ones.
		if len(body) > maxPolicyMessagesSize {
			return errors.New("policy %s reported %d bytes of messages, above the %d byte limit",
				outcome.PolicyID, len(body), maxPolicyMessagesSize, errors.WithErrorCode(errors.EInvalid))
		}

		summarySource = append(summarySource, outcome.Messages...)

		retainFn, key, err := c.artifactStore.UploadPolicyCheckPolicyMessages(ctx, run, bytes.NewReader(body))
		if err != nil {
			return errors.Wrap(err, "failed to write policy messages to object storage")
		}
		c.messageKeys[outcome.PolicyID] = key
		c.messageRetainFns[outcome.PolicyID] = retainFn
	}

	if len(summarySource) > 0 {
		messages, truncated := summarizePolicyMessages(summarySource)
		c.messagesSummary = &models.PolicyCheckMessagesSummary{Messages: messages, Truncated: truncated}
	}

	return nil
}

// summarizePolicyMessages fills the check's summary budget with the messages its policies reported,
// saying whether it had to leave anything out. Unlike the per-policy limit above this truncates rather
// than rejecting, because the summary is a preview by construction — the full list of any one policy is
// in object storage.
//
// The message that overruns the budget is cut to the space left rather than dropped, so the summary
// uses every byte it is allowed: the opening lines of the next violation tell a reader more than the
// gap they would otherwise leave. What was cut or dropped is reported as a flag rather than a marker
// entry, because the caller renders this as a list and says separately that it is partial.
func summarizePolicyMessages(messages []string) ([]string, bool) {
	kept := make([]string, 0, len(messages))
	var total int
	for _, message := range messages {
		if total+len(message) > maxPolicyCheckMessagesSummarySize {
			// The head only comes out empty when the remaining space cannot hold a single rune, in
			// which case there is nothing to add but the summary is still partial.
			if head := truncateToRuneBoundary(message, maxPolicyCheckMessagesSummarySize-total); head != "" {
				kept = append(kept, head)
			}
			return kept, true
		}

		kept = append(kept, message)
		total += len(message)
	}

	return kept, false
}

// truncateToRuneBoundary cuts s to at most limit bytes without splitting the rune it lands in.
func truncateToRuneBoundary(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	s = s[:limit]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

// Execute writes the check's outcomes and sets the check verdict.
func (c *ReportRunPolicyOutcomes) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.runID)
	if err != nil {
		return err
	}

	check := run.PolicyCheckByID(c.PolicyCheckID)
	if check == nil {
		return errors.New("run %s has no matching policy check node %s", c.runID, c.PolicyCheckID, errors.WithErrorCode(errors.ENotFound))
	}

	// Index the pinned policies by their id so each reported outcome can be stamped onto its
	// matching entry (there is one entry per policy, so id is unique — unlike set+version).
	policyByID := map[string]*models.PolicyCheckPolicy{}
	for i := range check.Policies {
		p := check.Policies[i]
		policyByID[p.ID] = p
	}

	var hardFailed, softFailed bool
	for _, outcome := range c.Outcomes {
		policy := policyByID[outcome.PolicyID]
		if policy == nil {
			return errors.New("failed to report run status: policy %q does not exist in run %q", outcome.PolicyID, c.runID, errors.WithErrorCode(errors.EInvalid))
		}

		if outcome.Passed {
			policy.Status = models.PolicyCheckPolicyPassed
		} else {
			policy.Status = models.PolicyCheckPolicyFailed
			switch policy.EnforcementLevel {
			case models.PolicyEnforcementHardMandatory:
				hardFailed = true
			case models.PolicyEnforcementSoftMandatory:
				softFailed = true
			}
		}

		// Point the policy at the object Prepare uploaded, or clear the key when this report carried
		// no messages for it, so a key from an earlier report cannot outlive its result.
		if key, ok := c.messageKeys[outcome.PolicyID]; ok {
			policy.MessagesObjectStoreKey = &key
			if err := c.messageRetainFns[outcome.PolicyID](ctx, c.runID); err != nil {
				return errors.Wrap(err, "failed to link policy messages object store ref")
			}
		} else {
			policy.MessagesObjectStoreKey = nil
		}
	}

	check.MessagesSummary = c.messagesSummary

	// Recomputed across every check rather than set from this one's outcomes: the flag is a run-level
	// summary, and this report may have just cleared the only advisory failure the run had.
	run.HasAdvisoryFailures = run.ComputeHasAdvisoryFailures()

	verdict := computePolicyCheckVerdict(hardFailed, softFailed)

	verdictChanges, err := statemachine.SetPolicyCheckStatus(run, check.GetPath(), verdict)
	if err != nil {
		return errors.Wrap(err, "failed to set policy check verdict")
	}
	return input.RunStore.AddRunChanges(run, verdictChanges...)
}

// computePolicyCheckVerdict maps the failed outcomes' effective enforcement onto the check
// verdict: errored (a hard-mandatory policy set failed), awaiting_override (a soft-mandatory
// failed with no hard fail), or passed (no mandatory failures; advisory failures are informational).
func computePolicyCheckVerdict(hardFailed, softFailed bool) models.PolicyCheckStatus {
	switch {
	case hardFailed:
		return models.PolicyCheckErrored
	case softFailed:
		return models.PolicyCheckSoftFailed
	default:
		return models.PolicyCheckPassed
	}
}
