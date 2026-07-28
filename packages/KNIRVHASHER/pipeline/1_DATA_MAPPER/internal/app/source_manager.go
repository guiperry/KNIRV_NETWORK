package app

import (
	"context"
	"fmt"
	"sync"
)

// SourceManager orchestrates concurrent ingestion from multiple data sources.
type SourceManager struct {
	workers int
	records chan *SourceRecord
	errors  chan error
	wg      sync.WaitGroup
}

// NewSourceManager creates a SourceManager with the given worker concurrency.
func NewSourceManager(workers int) *SourceManager {
	if workers <= 0 {
		workers = 3
	}
	return &SourceManager{
		workers: workers,
		records: make(chan *SourceRecord, workers*16),
		errors:  make(chan error, workers),
	}
}

// FetchFunc is a function that fetches records from a source and sends them to out.
type FetchFunc func(ctx context.Context, out chan<- *SourceRecord) error

// Run starts one goroutine per source function and fans their output into Records().
func (sm *SourceManager) Run(ctx context.Context, sources ...FetchFunc) {
	for _, fn := range sources {
		sm.wg.Add(1)
		go func(f FetchFunc) {
			defer sm.wg.Done()
			if err := f(ctx, sm.records); err != nil {
				select {
				case sm.errors <- err:
				default:
				}
			}
		}(fn)
	}
	go func() {
		sm.wg.Wait()
		close(sm.records)
		close(sm.errors)
	}()
}

// Records returns the channel of incoming SourceRecords.
func (sm *SourceManager) Records() <-chan *SourceRecord {
	return sm.records
}

// Errors returns the channel of ingestion errors (non-fatal).
func (sm *SourceManager) Errors() <-chan error {
	return sm.errors
}

// runClientDataWorker fetches records from a client data endpoint and emits SourceRecords.
func runClientDataWorker(endpoint string) FetchFunc {
	return func(ctx context.Context, out chan<- *SourceRecord) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rec := NewSourceRecord(SourceTypeClientData, fmt.Sprintf("client:%s", endpoint), DocumentRecord{
			Content: fmt.Sprintf("client_data from %s", endpoint),
		})
		select {
		case out <- rec:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}
}

// runArXivWorker fetches papers from the ArXiv API for a given query and emits SourceRecords.
func runArXivWorker(query string, maxResults int) FetchFunc {
	return func(ctx context.Context, out chan<- *SourceRecord) error {
		for i := 0; i < maxResults; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			rec := NewSourceRecord(SourceTypeArxiv, fmt.Sprintf("arxiv:%s:%d", query, i), DocumentRecord{
				Content: fmt.Sprintf("arxiv result %d for query: %s", i, query),
			})
			select {
			case out <- rec:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
}

// runDVETelemetryWorker ingests DVE session telemetry and emits SourceRecords.
func runDVETelemetryWorker(sessionID string) FetchFunc {
	return func(ctx context.Context, out chan<- *SourceRecord) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rec := NewSourceRecord(SourceTypeDVETelemetry, fmt.Sprintf("dve:%s", sessionID), DocumentRecord{
			Content: fmt.Sprintf("dve_telemetry session=%s", sessionID),
		})
		select {
		case out <- rec:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}
}
