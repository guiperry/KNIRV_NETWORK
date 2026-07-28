package connector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// OntologyConnector reads a consistent ontology snapshot. It deliberately
// does not fall through while paging: an ontology is one source tier and the
// next run decides whether a newer snapshot is available.
type OntologyConnector struct {
	BaseURL  string
	Client   *http.Client
	PageSize int
}

type OntologyEntity struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type OntologyRelation struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func (c OntologyConnector) FetchEntities() ([]RawRecord, error) {
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	pageSize := c.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}
	var out []RawRecord
	for offset := 0; ; offset += pageSize {
		var payload json.RawMessage
		if err := getJSON(client, NextOntologyURL(c.BaseURL, "entities", offset, pageSize), &payload); err != nil {
			return nil, err
		}
		var page []OntologyEntity
		if err := json.Unmarshal(payload, &page); err != nil {
			var envelope struct {
				Entities []OntologyEntity `json:"entities"`
				Data     []OntologyEntity `json:"data"`
			}
			if envelopeErr := json.Unmarshal(payload, &envelope); envelopeErr != nil {
				return nil, err
			}
			page = envelope.Entities
			if len(page) == 0 {
				page = envelope.Data
			}
		}
		for i, entity := range page {
			text := strings.TrimSpace(strings.Join([]string{entity.Name, entity.Description}, " — "))
			if text == "" {
				text = entity.Name
			}
			tags := []string{}
			if entity.Type != "" {
				tags = append(tags, entity.Type)
			}
			out = append(out, RawRecord{DatasetID: "knirvserver-ontology", Split: "ontology", Index: int64(offset + i), Text: text, Tags: tags})
		}
		if len(page) < pageSize {
			return out, nil
		}
	}
}

func getJSON(client *http.Client, endpoint string, target any) error {
	resp, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %s", endpoint, resp.Status)
	}
	var body bytes.Buffer
	if _, err := body.ReadFrom(resp.Body); err != nil {
		return err
	}
	return json.Unmarshal(body.Bytes(), target)
}
