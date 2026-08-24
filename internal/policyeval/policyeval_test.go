package policyeval

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluate_DenyFires(t *testing.T) {
	ctx := context.Background()

	bundle := Bundle{
		Modules: map[string]string{
			"policy.rego": `package tharsis.test

deny contains msg if {
    input.isDestroy
    msg := "destroy runs are not allowed"
}`,
		},
	}

	violations, err := Evaluate(ctx, bundle, map[string]interface{}{"isDestroy": true})
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, "destroy runs are not allowed", violations[0])
}

func TestEvaluate_DenyDoesNotFire(t *testing.T) {
	ctx := context.Background()

	bundle := Bundle{
		Modules: map[string]string{
			"policy.rego": `package tharsis.test

deny contains msg if {
    input.isDestroy
    msg := "destroy runs are not allowed"
}`,
		},
	}

	violations, err := Evaluate(ctx, bundle, map[string]interface{}{"isDestroy": false})
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestEvaluate_NoDenyOrViolationPasses(t *testing.T) {
	ctx := context.Background()

	bundle := Bundle{
		Modules: map[string]string{
			"policy.rego": `package tharsis.test

allow if {
    input.isDestroy
}`,
		},
	}

	violations, err := Evaluate(ctx, bundle, map[string]interface{}{"isDestroy": true})
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestEvaluate_ViolationWithMsgObject(t *testing.T) {
	ctx := context.Background()

	bundle := Bundle{
		Modules: map[string]string{
			"policy.rego": `package tharsis.test

violation contains {"msg": msg} if {
    input.count > 0
    msg := "count must be zero"
}`,
		},
	}

	violations, err := Evaluate(ctx, bundle, map[string]interface{}{"count": 1})
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, "count must be zero", violations[0])
}

func TestEvaluate_InvalidRegoReturnsError(t *testing.T) {
	ctx := context.Background()

	bundle := Bundle{
		Modules: map[string]string{
			"policy.rego": `package tharsis.test

deny[msg] {
    this is not valid rego
}`,
		},
	}

	_, err := Evaluate(ctx, bundle, map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile policy set")
}

func TestEvaluate_ScalarDenyRuleFailsClosed(t *testing.T) {
	ctx := context.Background()

	// A deny rule written as a scalar rather than a partial set produces no messages under the
	// Conftest convention. Reporting that as a pass would let a malformed policy wave a run through,
	// so evaluation must fail closed with an error instead.
	bundle := Bundle{
		Modules: map[string]string{
			"policy.rego": `package tharsis.test

deny := "destroy runs are not allowed" if {
    input.isDestroy
}`,
		},
	}

	_, err := Evaluate(ctx, bundle, map[string]interface{}{"isDestroy": true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a set or array")
}

func TestEvaluate_HTTPSendIsRejected(t *testing.T) {
	ctx := context.Background()

	// http.send is dropped from the sandboxed capabilities, so a policy that reaches for it (to
	// exfiltrate the plan or pivot into internal networks) fails to compile rather than evaluating
	// and reporting "passed".
	bundle := Bundle{
		Modules: map[string]string{
			"policy.rego": `package tharsis.test

deny contains msg if {
    resp := http.send({"method": "get", "url": "http://169.254.169.254/"})
    msg := sprintf("%v", [resp])
}`,
		},
	}

	_, err := Evaluate(ctx, bundle, map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile policy set")
}

func TestEvaluate_NetLookupIsRejected(t *testing.T) {
	ctx := context.Background()

	bundle := Bundle{
		Modules: map[string]string{
			"policy.rego": `package tharsis.test

deny contains msg if {
    addrs := net.lookup_ip_addr("internal.example.com")
    msg := sprintf("%v", [addrs])
}`,
		},
	}

	_, err := Evaluate(ctx, bundle, map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile policy set")
}
