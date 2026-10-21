package transformer

import (
	"context"
	"encoding/json"
	"go/parser"
	"go/token"
	"os/exec"
	"strings"
	"testing"
)

var goodFields = map[WASMType]string{
	WASMTypeRule:       `{"policy_id":7,"severity":3,"action":1,"confidence":0.8}`,
	WASMTypeResolution: `{"error_code":42,"hint":"retry with backoff","severity":2,"retryable":true}`,
	WASMTypePatch:      `{"target_component":"gateway","patch_data":"set limit=10","apply_priority":5}`,
}

func TestRenderedModulesAreValidGoAndVerify(t *testing.T) {
	f := &WASMFinisher{Verifier: NewBidirectionalVerifier()}
	for typ, fields := range goodFields {
		src, err := RenderFields(typ, json.RawMessage(fields))
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		if _, err := parser.ParseFile(token.NewFileSet(), "m.go", src, parser.AllErrors); err != nil {
			t.Errorf("%s: rendered source does not parse: %v\n%s", typ, err, src)
		}
		if _, ok, msg, err := f.Check(typ, json.RawMessage(fields)); err != nil || !ok {
			t.Errorf("%s: backward check rejected a valid module: %v %s", typ, err, msg)
		}
	}
}

// Text the model supplies reaches the module only as an escaped string literal.
func TestFieldTextCannotBreakOutOfItsLiteral(t *testing.T) {
	evil := "x\"; import \"os\"\nfunc init(){ os.Exit(1) }\n//"
	raw, _ := json.Marshal(map[string]any{"error_code": 1, "hint": evil, "severity": 1, "retryable": false})
	src, err := RenderFields(WASMTypeResolution, raw)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "m.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("escaped text must still parse: %v", err)
	}
	if len(file.Imports) != 1 || file.Imports[0].Path.Value != `"unsafe"` {
		t.Fatalf("injected import: %v", file.Imports)
	}
	for _, d := range file.Decls {
		if fn, ok := d.(interface{ Pos() token.Pos }); ok {
			_ = fn
		}
	}
	if strings.Contains(src, "\nfunc init()") {
		t.Fatal("injected function")
	}
}

func TestBadFieldsAreRefusedNotClamped(t *testing.T) {
	cases := []struct {
		typ WASMType
		raw string
	}{
		{WASMTypeRule, `{"policy_id":1,"severity":300,"action":0,"confidence":0.5}`},
		{WASMTypeRule, `{"policy_id":1,"severity":1,"action":0,"confidence":1.5}`},
		{WASMTypeRule, `{"policy_id":1,"severity":1,"action":0,"confidence":0.5,"extra":true}`},
		{WASMTypeRule, `{"policy_id":-1,"severity":1,"action":0,"confidence":0.5}`},
		{WASMTypeResolution, `{"error_code":1,"hint":"` + strings.Repeat("a", 300) + `","severity":1,"retryable":false}`},
		{WASMTypePatch, `{"target_component":"c","patch_data":"d","apply_priority":-4}`},
		{WASMTypePatch, `not json`},
		{WASMType("bogus"), `{}`},
	}
	for _, c := range cases {
		if _, err := RenderFields(c.typ, json.RawMessage(c.raw)); err == nil {
			t.Errorf("%s %.60s: accepted", c.typ, c.raw)
		}
	}
}

func TestAFailedCompileIsNeverReportedAsVerified(t *testing.T) {
	f := &WASMFinisher{Compiler: NewWASMCompiler("/nonexistent/tinygo", t.TempDir()), Verifier: NewBidirectionalVerifier()}
	d, err := f.Finish(context.Background(), WASMTypeRule, json.RawMessage(goodFields[WASMTypeRule]), "r")
	if err != nil {
		t.Fatal(err)
	}
	if d.BidirectionalVerified || !strings.Contains(d.BackwardVerifierMsg, "compile failed") {
		t.Fatalf("a compile failure must not read as success: %+v", d)
	}
}

func TestFinishCompilesWithRealTinyGo(t *testing.T) {
	path, err := exec.LookPath("tinygo")
	if err != nil {
		t.Skip("tinygo not installed")
	}
	f := &WASMFinisher{Compiler: NewWASMCompiler(path, t.TempDir()), Verifier: NewBidirectionalVerifier()}
	for typ, fields := range goodFields {
		d, err := f.Finish(context.Background(), typ, json.RawMessage(fields), "r")
		if err != nil {
			t.Fatal(err)
		}
		if !d.BidirectionalVerified || d.WASMHash == "" || d.WASMPath == "" {
			t.Errorf("%s: %+v", typ, d)
		}
	}
}
