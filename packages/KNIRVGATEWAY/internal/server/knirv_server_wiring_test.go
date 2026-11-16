package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

// TestSetupRoutesWiresKnirvSocketRoutesInOrder parses server.go and checks the route
// registrations this change depends on. Each helper must be called from setupRoutes, and
// in an order where gorilla/mux matches the intended route first:
//   - the KNIRVSERVER app routes and the /git/ 404 come before the generic /api/v1/
//     backend proxy, so the system routes and /git/ never fall through to the backend;
//   - the /ws app route comes after the gateway's inner-agent /ws/dve/... tunnels;
//   - the /ws app route comes before the backend /ws tunnel, which it replaces.
//
// Reverting or moving one of these calls fails here. The route handlers themselves are
// covered by the router-level tests in knirv_server_app_proxy_test.go.
func TestSetupRoutesWiresKnirvSocketRoutesInOrder(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "server.go", nil, 0)
	if err != nil {
		t.Fatalf("parse server.go: %v", err)
	}

	callPos := map[string]token.Pos{}
	litPos := map[string]token.Pos{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if fn, ok := x.Fun.(*ast.Ident); ok {
				if _, seen := callPos[fn.Name]; !seen {
					callPos[fn.Name] = x.Pos()
				}
			}
		case *ast.BasicLit:
			if x.Kind == token.STRING {
				if s, err := strconv.Unquote(x.Value); err == nil {
					if _, seen := litPos[s]; !seen {
						litPos[s] = x.Pos()
					}
				}
			}
		}
		return true
	})

	for _, name := range []string{
		"registerKnirvSocketRoutes", "registerGitPublicNotFound", "newKnirvServerAppProxy",
		"registerKnirvServerAppRoutes", "registerKnirvServerAppWSRoutes", "webguiFileOrAppFallback",
	} {
		if _, ok := callPos[name]; !ok {
			t.Fatalf("server.go never calls %s", name)
		}
	}

	backendAPI, ok := litPos["/api/v1/"]
	if !ok {
		t.Fatal(`server.go no longer registers the generic "/api/v1/" backend proxy`)
	}
	if !(callPos["registerKnirvServerAppRoutes"] < backendAPI) {
		t.Errorf("registerKnirvServerAppRoutes must be registered before the generic /api/v1/ backend proxy")
	}
	if !(callPos["registerGitPublicNotFound"] < backendAPI) {
		t.Errorf("registerGitPublicNotFound must be registered before the generic /api/v1/ backend proxy")
	}
	if !(callPos["registerKnirvSocketRoutes"] < backendAPI) {
		t.Errorf("registerKnirvSocketRoutes must be registered before the generic /api/v1/ backend proxy")
	}

	innerTunnel, ok := litPos["/ws/dve/{dveId}/inner/stream/{sessionId}"]
	if !ok {
		t.Fatal("inner-agent WebSocket tunnel route not found")
	}
	if !(innerTunnel < callPos["registerKnirvServerAppWSRoutes"]) {
		t.Errorf("registerKnirvServerAppWSRoutes must come after the inner-agent /ws/dve tunnels")
	}
	backendWS, ok := litPos["/ws/dve/{dveId}/advisor/stream/{sessionId}"]
	if !ok {
		t.Fatal("backend WebSocket tunnel route not found")
	}
	if !(callPos["registerKnirvServerAppWSRoutes"] < backendWS) {
		t.Errorf("registerKnirvServerAppWSRoutes must come before the backend WebSocket tunnel routes")
	}
}
