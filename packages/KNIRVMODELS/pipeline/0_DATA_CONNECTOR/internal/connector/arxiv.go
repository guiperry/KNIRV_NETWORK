package connector

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const arxivAPIURL = "https://export.arxiv.org/api/query"

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
	return fetchArxivPage(client, arxivAPIURL, categories, 0, max)
}

type ArxivCursor struct {
	NextOffset int `json:"next_offset"`
}

// FetchNextArxiv stages one paper per pipeline pass and persists the next
// result offset. This allows the controller's existing batch loop to carry one
// complete paper through mapper, encoder, and trainer before requesting another.
func FetchNextArxiv(client *http.Client, categories []string, max int, statePath string) ([]RawRecord, bool, error) {
	cursor := ArxivCursor{}
	if data, err := os.ReadFile(statePath); err == nil {
		_ = json.Unmarshal(data, &cursor)
	}
	if max > 0 && cursor.NextOffset >= max {
		return nil, true, nil
	}
	records, exhausted, err := fetchArxivPage(client, arxivAPIURL, categories, cursor.NextOffset, 1)
	if err != nil || exhausted || len(records) == 0 {
		return records, exhausted, err
	}
	cursor.NextOffset++
	if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
		return nil, false, err
	}
	data, err := json.Marshal(cursor)
	if err != nil {
		return nil, false, err
	}
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return nil, false, err
	}
	return records[:1], false, nil
}

func fetchArxivPage(client *http.Client, baseURL string, categories []string, start, max int) ([]RawRecord, bool, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if len(categories) == 0 {
		categories = []string{"cs.LG"}
	}
	if max <= 0 {
		max = 50
	}
	u := baseURL + "?search_query=" + url.QueryEscape("cat:"+strings.Join(categories, " OR cat:")) + "&start=" + strconv.Itoa(start) + "&max_results=" + strconv.Itoa(max)
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
		out = append(out, RawRecord{
			DatasetID: "arxiv:" + strings.Join(categories, ","),
			Split:     "papers",
			Index:     int64(start + i),
			Heading:   strings.Join(strings.Fields(e.Title), " "),
			Text:      strings.TrimSpace(e.Summary),
			Tags:      tags,
		})
	}
	return out, len(feed.Entries) == 0, nil
}
