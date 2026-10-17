package transformer

import "testing"

func TestBackwardVerifyIsNotAPassThrough(t *testing.T) {
	bv := NewBidirectionalVerifier() // hashnet unavailable by default
	good := "package main\n\n//export GuardrailClass\nfunc GuardrailClass() uint32 {\n\treturn 1\n}\n"
	if ok, msg := bv.BackwardVerify(good, "rule"); !ok {
		t.Fatalf("well-formed rule failed backward: %s", msg)
	}
	cases := map[string]string{
		"empty":           "",
		"export no func":  "//export GuardrailClass\nvar x = 1\n",
		"wrong export":    "//export ErrorClass\nfunc ErrorClass() uint32 { return 1 }\n",
		"func name drift": "//export GuardrailClass\nfunc Other() uint32 { return 1 }\n",
	}
	for name, src := range cases {
		if ok, _ := bv.BackwardVerify(src, "rule"); ok {
			t.Errorf("%s: backward passed, want fail", name)
		}
	}
	if ok, _ := bv.BackwardVerify(good, "bogus"); ok {
		t.Error("unknown wasm type passed")
	}
}

func TestBackwardVerifyAcceptsTemplates(t *testing.T) {
	bv := NewBidirectionalVerifier()
	for typ, src := range map[string]string{
		"rule":       NewRuleTemplate().Build(),
		"resolution": NewResolutionTemplate().Build(),
		"patch":      NewPatchTemplate().Build(),
	} {
		if ok, msg := bv.BackwardVerify(src, typ); !ok {
			t.Fatalf("%s template failed backward: %s", typ, msg)
		}
	}
}
