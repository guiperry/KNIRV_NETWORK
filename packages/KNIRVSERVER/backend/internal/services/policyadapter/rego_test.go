package policyadapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const allowPolicy = `package knirv

default allow := false

allow := true {
	input.role == "admin"
}

allow := true {
	input.action == "read"
	input.resource == "public"
}
`

const denyAllPolicy = `package knirv

default allow := false
`

func TestNewRegoEvaluator(t *testing.T) {
	re := NewRegoEvaluator()
	assert.NotNil(t, re)
	assert.Empty(t, re.ListPolicies())
}

func TestLoadPolicy(t *testing.T) {
	re := NewRegoEvaluator()
	err := re.LoadPolicy("test-policy", allowPolicy)
	require.NoError(t, err)

	policies := re.ListPolicies()
	assert.Contains(t, policies, "test-policy")
}

func TestLoadPolicyInvalidRego(t *testing.T) {
	re := NewRegoEvaluator()
	err := re.LoadPolicy("invalid", "this is not rego {")
	assert.Error(t, err)
}

func TestEvaluateAdminAllowed(t *testing.T) {
	re := NewRegoEvaluator()
	err := re.LoadPolicy("admin-policy", allowPolicy)
	require.NoError(t, err)

	result, err := re.Evaluate("admin-policy", map[string]interface{}{
		"role": "admin",
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, "allow", result.Decision)
}

func TestEvaluateActionDenied(t *testing.T) {
	re := NewRegoEvaluator()
	err := re.LoadPolicy("deny-policy", allowPolicy)
	require.NoError(t, err)

	result, err := re.Evaluate("deny-policy", map[string]interface{}{
		"role": "user",
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "deny", result.Decision)
}

func TestEvaluateNonexistentPolicy(t *testing.T) {
	re := NewRegoEvaluator()
	_, err := re.Evaluate("nonexistent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy not loaded")
}

func TestEvaluateDenyAllPolicy(t *testing.T) {
	re := NewRegoEvaluator()
	err := re.LoadPolicy("deny-all", denyAllPolicy)
	require.NoError(t, err)

	result, err := re.Evaluate("deny-all", map[string]interface{}{"role": "admin"})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "deny", result.Decision)
}

func TestRemovePolicy(t *testing.T) {
	re := NewRegoEvaluator()
	err := re.LoadPolicy("remove-me", allowPolicy)
	require.NoError(t, err)
	assert.Len(t, re.ListPolicies(), 1)

	re.RemovePolicy("remove-me")
	assert.Empty(t, re.ListPolicies())

	_, err = re.Evaluate("remove-me", nil)
	assert.Error(t, err)
}

func TestMultiplePolicies(t *testing.T) {
	re := NewRegoEvaluator()
	require.NoError(t, re.LoadPolicy("p1", allowPolicy))
	require.NoError(t, re.LoadPolicy("p2", denyAllPolicy))

	policies := re.ListPolicies()
	assert.Len(t, policies, 2)
	assert.Contains(t, policies, "p1")
	assert.Contains(t, policies, "p2")
}

func TestEvaluateNoMatchingRules(t *testing.T) {
	emptyPolicy := `package knirv
default allow := false
`
	re := NewRegoEvaluator()
	require.NoError(t, re.LoadPolicy("empty", emptyPolicy))

	result, err := re.Evaluate("empty", map[string]interface{}{
		"something": "anything",
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "deny", result.Decision)
}
