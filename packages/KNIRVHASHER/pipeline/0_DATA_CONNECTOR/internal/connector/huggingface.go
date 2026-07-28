package connector

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HuggingFaceConnector consumes the dataset-server rows API in pages. An
// empty page is reported as Exhausted so the orchestrator can advance to
// arXiv rather than retrying the same source forever.
type HuggingFaceConnector struct {
	Config         HuggingFaceConfig
	Client         *http.Client
	BaseURL        string
	MaxRetries     int
	RetryBaseDelay time.Duration
	Sleep          func(time.Duration)
}
type hfPage struct {
	Rows []struct {
		Row map[string]any `json:"row"`
	} `json:"rows"`
}

func (c HuggingFaceConnector) Page(dataset, split string, offset, length int) (records []RawRecord, exhausted bool, err error) {
	base := c.BaseURL
	if base == "" {
		base = "https://datasets-server.huggingface.co/rows"
	}
	q := url.Values{"dataset": {dataset}, "config": {"default"}, "split": {split}, "offset": {strconv.Itoa(offset)}, "length": {strconv.Itoa(length)}}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	maxRetries := c.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	baseDelay := c.RetryBaseDelay
	if baseDelay <= 0 {
		baseDelay = time.Second
	}
	sleep := c.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest(http.MethodGet, base+"?"+q.Encode(), nil)
		if err != nil {
			return nil, false, err
		}
		if token := c.Config.Token; token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, false, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if attempt >= maxRetries {
				// A persistently throttled source is unavailable for this run.
				// Report it as exhausted so the priority runner advances to
				// arXiv instead of terminating the complete pipeline.
				log.Printf("0_DATA_CONNECTOR: HuggingFace remains rate limited after %d retries; falling back to arXiv", maxRetries)
				return nil, true, nil
			}
			delay := retryAfter(resp.Header.Get("Retry-After"), baseDelay, attempt)
			log.Printf("0_DATA_CONNECTOR: HuggingFace rate limited dataset=%s offset=%d retry=%d/%d delay=%s", dataset, offset, attempt+1, maxRetries, delay)
			sleep(delay)
			continue
		}
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return nil, true, nil
		}
		if resp.StatusCode != http.StatusOK {
			status := resp.Status
			resp.Body.Close()
			return nil, false, fmt.Errorf("huggingface rows: HTTP %s", status)
		}

		var page hfPage
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			return nil, false, err
		}
		if len(page.Rows) == 0 {
			return nil, true, nil
		}
		for i, row := range page.Rows {
			records = append(records, RawRecord{DatasetID: dataset, Split: split, Index: int64(offset + i), Text: NormalizeRow(row.Row)})
		}
		return records, false, nil
	}
}

func retryAfter(value string, base time.Duration, attempt int) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		delay := time.Duration(seconds) * time.Second
		if delay > 30*time.Second {
			return 30 * time.Second
		}
		return delay
	}
	if when, err := http.ParseTime(value); err == nil {
		delay := time.Until(when)
		if delay < 0 {
			return 0
		}
		if delay > 30*time.Second {
			return 30 * time.Second
		}
		return delay
	}
	delay := base << attempt
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}
