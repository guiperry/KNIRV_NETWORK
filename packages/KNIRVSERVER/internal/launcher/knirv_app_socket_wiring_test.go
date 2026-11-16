package launcher

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestPublicTCPListenerWiredThroughGuard checks that the public TCP http.Server in Start
// takes its handler from publicTCPHandler. Using app.router directly would expose the
// web app, /api/*, /tunnel, /ws and /git/* on TCP again, and the handler tests above
// would still pass because they call publicTCPHandler themselves.
func TestPublicTCPListenerWiredThroughGuard(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "launcher.go", nil, 0)
	if err != nil {
		t.Fatalf("parse launcher.go: %v", err)
	}
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok || !isHTTPServerType(lit.Type) {
			return true
		}
		var hasAddr bool
		var handler ast.Expr
		for _, el := range lit.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, _ := kv.Key.(*ast.Ident)
			if key == nil {
				continue
			}
			switch key.Name {
			case "Addr":
				hasAddr = true
			case "Handler":
				handler = kv.Value
			}
		}
		// The public listener is the http.Server with Addr set (the socket servers have none).
		if !hasAddr {
			return true
		}
		found = true
		call, ok := handler.(*ast.CallExpr)
		if !ok {
			t.Fatalf("public TCP http.Server Handler must be publicTCPHandler(...), got %T", handler)
		}
		if id, ok := call.Fun.(*ast.Ident); !ok || id.Name != "publicTCPHandler" {
			t.Fatalf("public TCP http.Server Handler must be publicTCPHandler(...)")
		}
		return true
	})
	if !found {
		t.Fatal("public TCP http.Server literal not found in launcher.go")
	}
}

func isHTTPServerType(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "http" && sel.Sel.Name == "Server"
}
