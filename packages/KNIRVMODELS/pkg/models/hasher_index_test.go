package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/guiperry/knirv-sdk-go/models"

	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/livesearch"
)

func indexedHasher(t *testing.T) (*Hasher, *httptest.Server) {
	t.Helper()
	dir := t.TempDir()
	ix, err := evidence.Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ix.Ingest(evidence.Batch{Tier: evidence.TierReference, Docs: []evidence.Doc{{Text: "Mount Kestrel rises 3100 metres above Lake Vessel in the Alder range.", URL: "https://example.org/k"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := ix.Ingest(evidence.Batch{Tier: evidence.TierUser, Tenant: "acme", Docs: []evidence.Doc{{Text: "Acme internal note: Mount Kestrel base camp opens 2027."}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := ix.Ingest(evidence.Batch{Tier: evidence.TierQuarantine, Docs: []evidence.Doc{{Text: "Forum rumour: Mount Kestrel is 2900 metres.", URL: "https://forum.example/x"}}}); err != nil {
		t.Fatal(err)
	}
	ix.Close()
	h := NewHasher(nil)
	h.UseIndex(dir, "")
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hasher")
	return h, httptest.NewServer(mux)
}

func TestIndexStatusRouteMatchesTheSDKContract(t *testing.T) {
	_, s := indexedHasher(t)
	defer s.Close()
	st, err := sdk.NewClient(s.URL, nil).IndexStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Spans != 3 || st.Tiers["reference"] != 1 || st.Tiers["user"] != 1 || st.Tiers["quarantine"] != 1 || st.Retrieval != "lexical" || st.KeyVersion != "fs1" {
		t.Fatalf("%+v", st)
	}
}

func TestIndexSearchRouteScopesByTenantAndPrivilege(t *testing.T) {
	_, s := indexedHasher(t)
	defer s.Close()
	search := func(hdr map[string]string, body string) (int, string) {
		req, _ := http.NewRequest("POST", s.URL+"/models/hasher/index/search", strings.NewReader(body))
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, e := res.Body.Read(buf)
			b.Write(buf[:n])
			if e != nil {
				break
			}
		}
		return res.StatusCode, b.String()
	}
	body := `{"text":"Mount Kestrel is 3100 metres high.","k":5}`
	_, anon := search(nil, body)
	if !strings.Contains(anon, "3100 metres") || strings.Contains(anon, "base camp") || strings.Contains(anon, "rumour") {
		t.Fatalf("anonymous callers see reference evidence only: %s", anon)
	}
	_, acme := search(map[string]string{TenantHeader: "acme"}, body)
	if !strings.Contains(acme, "base camp") || strings.Contains(acme, "rumour") {
		t.Fatalf("a tenant also sees its own documents: %s", acme)
	}
	_, other := search(map[string]string{TenantHeader: "globex"}, body)
	if strings.Contains(other, "base camp") {
		t.Fatalf("another tenant saw acme's document: %s", other)
	}
	_, userQ := search(map[string]string{TenantHeader: "acme"}, `{"text":"Mount Kestrel is 3100 metres high.","include_quarantine":true}`)
	if strings.Contains(userQ, "rumour") {
		t.Fatal("a plain user must not reach quarantine")
	}
	_, op := search(map[string]string{PrivilegeHeader: "operator"}, `{"text":"Mount Kestrel is 3100 metres high.","include_quarantine":true}`)
	if !strings.Contains(op, "rumour") {
		t.Fatalf("an operator may inspect quarantine: %s", op)
	}
	if code, _ := search(nil, `{"text":""}`); code != http.StatusBadRequest {
		t.Fatal(code)
	}
}

func TestIndexRoutesAre503WhenNoIndexIsConfigured(t *testing.T) {
	h := NewHasher(nil)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hasher")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/models/hasher/index/status", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatal(rec.Code)
	}
}

func fakeWikiServer(t *testing.T, calls *int) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		q := r.URL.Query()
		switch {
		case q.Get("list") == "search":
			w.Write([]byte(`{"query":{"search":[{"title":"Mount Orrery"}]}}`))
		case strings.Contains(q.Get("prop"), "extracts"):
			w.Write([]byte(`{"query":{"pages":[{"title":"Mount Orrery","extract":"Mount Orrery is a mountain of 4400 metres in the Alder range.","revisions":[{"revid":77}]}]}}`))
		default:
			w.Write([]byte(`{"query":{"pages":[{"title":"Mount Orrery","revisions":[{"revid":77}]}]}}`))
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func TestIndexSearchRouteHybridModeGoesLiveAndIndexModeDoesNot(t *testing.T) {
	calls := 0
	wiki := fakeWikiServer(t, &calls)
	h := NewHasher(nil)
	h.UseIndex(t.TempDir(), "")
	h.UseLive(func(ix *evidence.Index, dir string) (*livesearch.Retriever, error) {
		c := livesearch.NewClient(livesearch.Policy{Hosts: []string{strings.TrimPrefix(wiki.URL, "http://")}, AllowInsecure: true, AllowPrivate: true})
		wp := &livesearch.Wikipedia{API: wiki.URL + "/w/api.php", PageAt: wiki.URL + "/wiki/", C: c}
		return livesearch.NewRetriever(ix, dir, []livesearch.Provider{wp}, livesearch.Config{Enabled: true})
	})
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hasher")
	s := httptest.NewServer(mux)
	defer s.Close()
	post := func(body string) (int, map[string]any) {
		res, err := http.Post(s.URL+"/models/hasher/index/search", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var m map[string]any
		_ = json.NewDecoder(res.Body).Decode(&m)
		return res.StatusCode, m
	}
	if code, m := post(`{"text":"Mount Orrery is 4400 metres high.","mode":"index"}`); code != 200 || calls != 0 || len(m["hits"].([]any)) != 0 {
		t.Fatalf("index mode must not touch the network: %d calls=%d %v", code, calls, m)
	}
	code, m := post(`{"text":"Mount Orrery is 4400 metres high.","mode":"hybrid"}`)
	hits, _ := m["hits"].([]any)
	if code != 200 || calls == 0 || m["flash_used"] != true || len(hits) == 0 || m["retrieval"] != "live+lexical" {
		t.Fatalf("%d calls=%d %v", code, calls, m)
	}
	h0 := hits[0].(map[string]any)
	if h0["source"] != "wikipedia" || h0["revision"] != "77" || h0["source_tier"] != "quarantine" {
		t.Fatalf("%v", h0)
	}
	if q, _ := m["outbound_queries"].([]any); len(q) != 1 || !strings.Contains(q[0].(string), "orrery") {
		t.Fatalf("the exact outbound query must be reported: %v", m["outbound_queries"])
	}
	seen := calls
	post(`{"text":"Mount Orrery is 4400 metres high.","mode":"hybrid"}`)
	if calls != seen {
		t.Fatal("the second identical claim must be answered from the cache")
	}
}

func TestLiveModeIs503WhenNotConfigured(t *testing.T) {
	_, s := indexedHasher(t) // UseIndex only, no UseLive
	defer s.Close()
	res, err := http.Post(s.URL+"/models/hasher/index/search", "application/json", strings.NewReader(`{"text":"Mount Kestrel","mode":"hybrid"}`))
	if err != nil || res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("%v %v", err, res)
	}
}

func TestOntologyRoutesSyncCheckAndIsolate(t *testing.T) {
	_, srv := indexedHasher(t)
	defer srv.Close()
	call := func(method, path string, hdr map[string]string, body string) (int, map[string]any) {
		req, _ := http.NewRequest(method, srv.URL+"/models/hasher"+path, strings.NewReader(body))
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	cfg := `{"organization_id":"acme","ontology":{"decision_rights":{"refund_approval":"manager"}}}`
	if code, _ := call("POST", "/ontology/sync", nil, cfg); code != http.StatusForbidden {
		t.Fatalf("a plain user must not load an ontology: %d", code)
	}
	code, out := call("POST", "/ontology/sync", map[string]string{PrivilegeHeader: "operator"}, cfg)
	if code != 200 || out["facts"].(float64) != 1 {
		t.Fatalf("%d %v", code, out)
	}
	claim := `{"subject":"decision rights","property":"refund approval","value":"director"}`
	if code, out := call("POST", "/ontology/check", map[string]string{TenantHeader: "acme"}, claim); code != 200 || out["outcome"] != "refuted" {
		t.Fatalf("%d %v", code, out)
	}
	if _, out := call("POST", "/ontology/check", map[string]string{TenantHeader: "globex"}, claim); out["outcome"] != "no_opinion" {
		t.Fatalf("another tenant saw the ontology: %v", out)
	}
	if code, _ := call("DELETE", "/ontology?tenant=acme", map[string]string{PrivilegeHeader: "operator"}, ""); code != 200 {
		t.Fatal(code)
	}
	if _, out := call("POST", "/ontology/check", map[string]string{TenantHeader: "acme"}, claim); out["outcome"] != "no_opinion" {
		t.Fatalf("a removed ontology still answers: %v", out)
	}
}
