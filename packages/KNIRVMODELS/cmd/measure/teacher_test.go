package main

import (
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeInference serves what the harness router does for teacher jobs on a unix socket.
// answer picks the reply text from the user prompt; active is the /harness/status
// "active" backend.
func fakeInference(t *testing.T, active string, answer func(user string) string) string {
	t.Helper()
	sock := filepath.Join(os.TempDir(), "mt-"+strings.ReplaceAll(t.Name(), "/", "_")+".sock")
	os.Remove(sock)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Skip(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/harness/status", func(w http.ResponseWriter, r *http.Request) {
		doc := map[string]any{"active": active, "colibri_model": "qwen36-35b"}
		if active != "colibri" {
			doc["primary_note"] = "colibri skipped: this host does not meet the minimum requirements: CPU has no AVX2"
		}
		json.NewEncoder(w).Encode(doc)
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Messages []struct{ Content string } }
		json.NewDecoder(r.Body).Decode(&req)
		user := req.Messages[len(req.Messages)-1].Content
		w.Header().Set("X-Knirv-Backend", "colibri")
		w.Header().Set("X-Knirv-Model", "qwen3.6-colibri")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]string{"content": answer(user)}}},
			"usage":   map[string]int{"completion_tokens": 1},
		})
	})
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close(); os.Remove(sock) })
	return sock
}

// letterFor finds the option letter whose description contains want.
func letterFor(user, want string) string {
	for _, l := range strings.Split(user, "\n") {
		if strings.Contains(l, want) && len(l) > 2 && l[1] == ':' {
			return l[:1]
		}
	}
	return "A"
}

func writeLines(t *testing.T, path string, vs ...any) {
	t.Helper()
	for _, v := range vs {
		if err := appendJSONL(path, v); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSuggestWritesSuggestionsAndNeverTouchesGold(t *testing.T) {
	sock := fakeInference(t, "colibri", func(u string) string { return letterFor(u, "clearly entails") })
	dir := t.TempDir()
	cand := filepath.Join(dir, "cand.jsonl")
	writeLines(t, cand, GoldCase{Claim: "Paris is in France", Evidence: "Paris is the capital of France."})
	if err := cmdSuggest([]string{"-dir", dir, "-candidates", cand, "-socket", sock, "-samples", "4"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(goldPath(dir)); err == nil {
		t.Fatal("suggest must never write gold.jsonl")
	}
	sugg, err := readJSONL[Suggestion](suggestionsPath(dir))
	if err != nil || len(sugg) != 1 {
		t.Fatalf("%v %v", sugg, err)
	}
	s := sugg[0]
	if s.Suggested != LabelSupported || !s.Confident || s.Provenance.Teacher != "colibri" || s.Provenance.Catalog != "qwen36-35b" {
		t.Fatalf("%+v", s)
	}
	// Rerunning suggests nothing new.
	if err := cmdSuggest([]string{"-dir", dir, "-candidates", cand, "-socket", sock}); err != nil {
		t.Fatal(err)
	}
	if again, _ := readJSONL[Suggestion](suggestionsPath(dir)); len(again) != 1 {
		t.Fatalf("a second run duplicated suggestions: %d", len(again))
	}
}

// An unsure teacher suggests NEI rather than guessing (order disagreement rule).
func TestSuggestIsNEIWhenTheTeacherIsNotSure(t *testing.T) {
	calls := 0
	sock := fakeInference(t, "colibri", func(u string) string { calls++; return []string{"A", "B", "C"}[calls%3] })
	dir := t.TempDir()
	cand := filepath.Join(dir, "cand.jsonl")
	writeLines(t, cand, GoldCase{Claim: "c", Evidence: "e"})
	if err := cmdSuggest([]string{"-dir", dir, "-candidates", cand, "-socket", sock, "-samples", "6"}); err != nil {
		t.Fatal(err)
	}
	s, _ := readJSONL[Suggestion](suggestionsPath(dir))
	if len(s) != 1 || s[0].Suggested != LabelNEI || s[0].Confident {
		t.Fatalf("%+v", s)
	}
}

func TestSuggestIsBlockedWhenColibriIsNotServing(t *testing.T) {
	sock := fakeInference(t, "llama.cpp", func(string) string { return "A" })
	dir := t.TempDir()
	cand := filepath.Join(dir, "cand.jsonl")
	writeLines(t, cand, GoldCase{Claim: "c", Evidence: "e"})
	if err := cmdSuggest([]string{"-dir", dir, "-candidates", cand, "-socket", sock}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(suggestionsPath(dir)); err == nil {
		t.Fatal("nothing may be suggested without the teacher")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "results", "suggest.json"))
	if !strings.Contains(string(b), `"blocked"`) || !strings.Contains(string(b), "no AVX2") || !strings.Contains(string(b), "never use llama.cpp") {
		t.Fatalf("the reason must be recorded:\n%s", b)
	}
}

func TestGoldAcceptNeedsAPersonsLabelAndMarksTheSource(t *testing.T) {
	dir := t.TempDir()
	s := Suggestion{ID: goldID("c", "e"), Claim: "c", Evidence: "e", Suggested: LabelSupported}
	s.Provenance.Catalog = "qwen36-35b"
	writeLines(t, suggestionsPath(dir), s)
	accept := func(args ...string) error {
		fs := newFlagSet()
		return cmdGoldAccept(fs, &dir, args)
	}
	if err := accept("-id", s.ID, "-labeler", "owner"); err == nil {
		t.Fatal("a label or -agree is required")
	}
	if err := accept("-id", s.ID, "-labeler", "owner", "-label", "REFUTED", "-agree"); err == nil {
		t.Fatal("-label and -agree together are ambiguous")
	}
	if err := accept("-id", s.ID, "-labeler", "owner", "-label", "refuted"); err != nil {
		t.Fatal(err)
	}
	gold, _ := loadGold(dir)
	if len(gold) != 1 || gold[0].Label != LabelRefuted || gold[0].Labeler != "owner" ||
		!strings.Contains(gold[0].Source, "teacher-suggested:qwen36-35b:overridden-from-SUPPORTED") {
		t.Fatalf("%+v", gold)
	}
	if err := accept("-id", s.ID, "-labeler", "owner", "-agree"); err == nil {
		t.Fatal("the same case must not be added twice")
	}
}

func TestRetrievalJudgeFindsRelevantPassagesTheGoldSetLacks(t *testing.T) {
	// The fake calls a passage RELEVANT when it mentions "France".
	sock := fakeInference(t, "colibri", func(u string) string {
		if strings.Contains(u, "PASSAGE:\n<<<\n") && strings.Contains(strings.SplitN(u, "PASSAGE:", 2)[1], "France") {
			return letterFor(u, "directly supports")
		}
		return letterFor(u, "does not help")
	})
	dir := t.TempDir()
	writeLines(t, goldPath(dir),
		GoldCase{ID: "1", Claim: "Paris is the capital of France", Evidence: "Paris is the capital of France.", Label: LabelSupported, Labeler: "o", Group: "g1", DataClass: "private"},
		GoldCase{ID: "2", Claim: "Berlin is in Germany", Evidence: "Berlin is a city in Germany.", Label: LabelSupported, Labeler: "o", Group: "g2", DataClass: "private"},
	)
	dist := filepath.Join(dir, "dist.jsonl")
	writeLines(t, dist, map[string]string{"text": "France borders Spain and Germany and its capital is Paris."})
	if err := cmdRetrievalJudge([]string{"-dir", dir, "-socket", sock, "-k", "3", "-samples", "2", "-distractors", dist}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "results", "retrieval-judge.json"))
	if err != nil {
		t.Fatal(err)
	}
	var res Result
	json.Unmarshal(b, &res)
	if res.Status != statusMeasured {
		t.Fatalf("%s %s", res.Status, res.Reason)
	}
	if n, _ := res.Data["relevant_passages_without_a_gold_label"].(float64); n < 1 {
		t.Fatalf("the distractor about France is relevant to claim 1 and has no label: %v", res.Data)
	}
	if _, err := os.Stat(filepath.Join(dir, "retrieval_judgments.jsonl")); err != nil {
		t.Fatal("every judgment must be recorded with its provenance")
	}
	// The gold set itself is untouched.
	if g, _ := loadGold(dir); len(g) != 2 {
		t.Fatalf("gold changed: %d", len(g))
	}
}

func newFlagSet() *flag.FlagSet { return flag.NewFlagSet("gold accept", flag.ContinueOnError) }
