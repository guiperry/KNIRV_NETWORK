package connector

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type arxivFeed struct {
	Entries []struct {
		ID         string `xml:"id"`
		Title      string `xml:"title"`
		Summary    string `xml:"summary"`
		Categories []struct {
			Term string `xml:"term,attr"`
		} `xml:"category"`
	} `xml:"entry"`
}

func FetchArxiv(client *http.Client, categories []string, max int) ([]RawRecord, bool, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if len(categories) == 0 {
		categories = []string{"cs.LG"}
	}
	if max <= 0 {
		max = 50
	}
	u := "https://export.arxiv.org/api/query?search_query=" + url.QueryEscape("cat:"+strings.Join(categories, " OR cat:")) + "&start=0&max_results=" + strconv.Itoa(max)
	resp, err := client.Get(u)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("arxiv: HTTP %s", resp.Status)
	}
	var feed arxivFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, false, err
	}
	out := make([]RawRecord, 0, len(feed.Entries))
	for i, e := range feed.Entries {
		tags := make([]string, 0, len(e.Categories))
		for _, c := range e.Categories {
			tags = append(tags, c.Term)
		}
		out = append(out, RawRecord{DatasetID: "arxiv:" + strings.Join(categories, ","), Split: "papers", Index: int64(i), Text: strings.TrimSpace(e.Title + "\n\n" + e.Summary), Tags: tags})
	}
	return out, len(feed.Entries) == 0, nil
}
