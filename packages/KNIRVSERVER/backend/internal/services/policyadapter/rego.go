package policyadapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage/inmem"
)

type RegoEvaluator struct {
	partials map[string]rego.PreparedEvalQuery
}

type RegoResult struct {
	Allowed   bool                   `json:"allowed"`
	Decision  string                 `json:"decision"`
	Reason    string                 `json:"reason,omitempty"`
	Bindings  map[string]interface{} `json:"bindings,omitempty"`
}

func NewRegoEvaluator() *RegoEvaluator {
	return &RegoEvaluator{
		partials: make(map[string]rego.PreparedEvalQuery),
	}
}

func (re *RegoEvaluator) LoadPolicy(policyID, regoCode string) error {
	compiler, err := ast.CompileModules(map[string]string{
		policyID + ".rego": regoCode,
	})
	if err != nil {
		return fmt.Errorf("rego compile error: %w", err)
	}

	query, err := rego.New(
		rego.Query("data.knirv.allow"),
		rego.Compiler(compiler),
		rego.Store(inmem.New()),
	).PrepareForEval(context.Background())
	if err != nil {
		return fmt.Errorf("rego prepare error: %w", err)
	}

	re.partials[policyID] = query
	return nil
}

func (re *RegoEvaluator) Evaluate(policyID string, input map[string]interface{}) (*RegoResult, error) {
	query, ok := re.partials[policyID]
	if !ok {
		return nil, fmt.Errorf("policy not loaded: %s", policyID)
	}

	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("input marshal error: %w", err)
	}

	var parsedInput map[string]interface{}
	if err := json.Unmarshal(inputBytes, &parsedInput); err != nil {
		return nil, fmt.Errorf("input unmarshal error: %w", err)
	}

	results, err := query.Eval(context.Background(), rego.EvalInput(parsedInput))
	if err != nil {
		return nil, fmt.Errorf("rego eval error: %w", err)
	}

	if len(results) == 0 {
		return &RegoResult{Allowed: false, Decision: "deny", Reason: "no rules matched"}, nil
	}

	allowed, ok := results[0].Expressions[0].Value.(bool)
	if !ok {
		return nil, fmt.Errorf("unexpected rego result type")
	}

	if allowed {
		return &RegoResult{Allowed: true, Decision: "allow"}, nil
	}

	return &RegoResult{Allowed: false, Decision: "deny"}, nil
}

func (re *RegoEvaluator) RemovePolicy(policyID string) {
	delete(re.partials, policyID)
}

func (re *RegoEvaluator) ListPolicies() []string {
	ids := make([]string, 0, len(re.partials))
	for id := range re.partials {
		ids = append(ids, id)
	}
	return ids
}
