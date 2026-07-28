package connector

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestFetchNextArxivStagesOnePaperAndAdvancesCursor(t *testing.T) {
	starts := []int{}
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		start, _ := strconv.Atoi(req.URL.Query().Get("start"))
		starts = append(starts, start)
		body := fmt.Sprintf(`<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>https://arxiv.org/abs/%d</id>
    <title>
      Paper Heading %d
    </title>
    <summary>Paper abstract %d with enough content for ingestion.</summary>
    <category term="cs.LG"/>
  </entry>
</feed>`, start, start, start)
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})}
	statePath := t.TempDir() + "/arxiv_cursor.json"

	first, exhausted, err := FetchNextArxiv(client, []string{"cs.LG"}, 50, statePath)
	if err != nil || exhausted || len(first) != 1 {
		t.Fatalf("first records=%d exhausted=%v err=%v", len(first), exhausted, err)
	}
	if first[0].Heading != "Paper Heading 0" || first[0].Index != 0 {
		t.Fatalf("first=%+v", first[0])
	}
	second, exhausted, err := FetchNextArxiv(client, []string{"cs.LG"}, 50, statePath)
	if err != nil || exhausted || len(second) != 1 {
		t.Fatalf("second records=%d exhausted=%v err=%v", len(second), exhausted, err)
	}
	if second[0].Heading != "Paper Heading 1" || second[0].Index != 1 {
		t.Fatalf("second=%+v", second[0])
	}
	if len(starts) != 2 || starts[0] != 0 || starts[1] != 1 {
		t.Fatalf("starts=%v", starts)
	}
}
