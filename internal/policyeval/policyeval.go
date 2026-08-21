// Package policyeval provides server-side OPA policy evaluation shared by the executor's
// post-plan policy-eval job and the test/evaluate-policy feature.
//
// Rego evaluation follows the Conftest convention: rules named "deny" or "violation" (in any
// package) produce violation messages; a non-empty union of those messages means the evaluation
// FAILED, an empty union means it PASSED. Messages may be strings or objects with a "msg" field.
// Policies are authored in Rego v1 syntax.
package policyeval

import (
	"context"
	"fmt"
	"time"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage/inmem"
)

// Bundle holds a single policy set's rego modules and data documents.
type Bundle struct {
	// Modules maps a module name (e.g. file path) to its Rego source.
	Modules map[string]string
	// Data holds JSON data documents merged into the OPA store (optional).
	Data map[string]interface{}
	// ShaSum is the hex-encoded SHA-256 of the raw package (tar.gz) stream the bundle was
	// extracted from. The caller compares it against the pinned digest to verify integrity.
	ShaSum string
}

// maxEvaluationTimeout bounds a single policy-set evaluation. Policy bundles are arbitrary Rego from
// the package registry; a bounded deadline keeps a pathological or hostile policy (e.g. a deep
// recursion or a huge comprehension) from pinning the evaluating goroutine indefinitely. It caps the
// caller's context rather than replacing it, so an earlier caller deadline still wins.
const maxEvaluationTimeout = 60 * time.Second

// sandboxCapabilities returns the OPA capabilities a policy bundle is compiled and evaluated against,
// starting from this OPA version's defaults and dropping every nondeterministic built-in. That set
// includes the network built-ins a hostile policy would use to exfiltrate the Terraform plan or pivot
// into internal networks (http.send, net.lookup_ip_addr) as well as runtime/randomness/clock built-ins
// (opa.runtime, rand.*, time.*, uuid.*). Referencing a dropped built-in fails compilation, so a policy
// that reaches for one is rejected outright rather than evaluated and silently reported as passed.
func sandboxCapabilities() *ast.Capabilities {
	caps := ast.CapabilitiesForThisVersion()
	allowed := make([]*ast.Builtin, 0, len(caps.Builtins))
	for _, b := range caps.Builtins {
		if b.Nondeterministic {
			continue
		}
		allowed = append(allowed, b)
	}
	caps.Builtins = allowed
	return caps
}

// Evaluate compiles and evaluates a policy bundle's modules against the input, returning the
// union of "deny"/"violation" messages (Conftest convention). An empty slice means the policy
// set passed.
//
// Evaluation is fail-closed: an empty result set, or a "deny"/"violation" rule written as a scalar
// rather than a set/array of messages, is reported as an error rather than silently treated as a
// pass, so a malformed or truncated policy can never masquerade as a clean result.
func Evaluate(ctx context.Context, bundle Bundle, input interface{}) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, maxEvaluationTimeout)
	defer cancel()

	opts := []func(*rego.Rego){
		rego.Query("data"),
		rego.SetRegoVersion(ast.RegoV1),
		rego.Capabilities(sandboxCapabilities()),
	}
	for name, src := range bundle.Modules {
		opts = append(opts, rego.Module(name, src))
	}
	if len(bundle.Data) > 0 {
		opts = append(opts, rego.Store(inmem.NewFromObject(bundle.Data)))
	}

	query, err := rego.New(opts...).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compile policy set: %w", err)
	}

	rs, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policy set: %w", err)
	}

	// The "data" query always yields exactly one result carrying the whole data document. Zero results
	// (or a result with no expression value) is not a clean pass — it means evaluation did not produce
	// the document we count violations from — so fail closed rather than reporting the set as passed.
	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return nil, fmt.Errorf("policy set evaluation produced no result document")
	}

	return collectViolations(rs[0].Expressions[0].Value)
}

// collectViolations walks the evaluated data document and collects the messages produced by any
// "deny" or "violation" rule in any package. It fails closed: a deny/violation rule whose value is
// not a set/array of messages is a malformed rule, reported as an error rather than as zero messages.
func collectViolations(node interface{}) ([]string, error) {
	m, ok := node.(map[string]interface{})
	if !ok {
		return nil, nil
	}

	var out []string
	for k, v := range m {
		switch k {
		case "deny", "violation":
			msgs, err := messagesFromRule(k, v)
			if err != nil {
				return nil, err
			}
			out = append(out, msgs...)
		default:
			nested, err := collectViolations(v)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
		}
	}

	return out, nil
}

// messagesFromRule converts a deny/violation rule value into a slice of message strings. The rule
// must be a set/array (a partial set like `deny contains msg`, or a complete `deny = [...]`); a
// scalar value (e.g. `deny := "msg"`) is a malformed rule and returns an error rather than being
// silently ignored, which would otherwise report a failing policy as passed. Elements may be plain
// strings or objects with a "msg" field.
func messagesFromRule(ruleName string, v interface{}) ([]string, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf(
			"%q rule must be a set or array of messages, got %T; use `%s contains msg if { ... }`",
			ruleName, v, ruleName)
	}

	msgs := make([]string, 0, len(arr))
	for _, item := range arr {
		switch t := item.(type) {
		case string:
			msgs = append(msgs, t)
		case map[string]interface{}:
			if msg, ok := t["msg"].(string); ok {
				msgs = append(msgs, msg)
			} else {
				msgs = append(msgs, fmt.Sprintf("%v", t))
			}
		default:
			msgs = append(msgs, fmt.Sprintf("%v", t))
		}
	}

	return msgs, nil
}
