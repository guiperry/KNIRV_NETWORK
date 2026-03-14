package cognitiveengine

import (
	"context"
	"sync"

	"backend_server/internal/objects"
)

// ValidationWorkItem pairs a ValidationResult with its parent ValidationTask
// for concurrent processing by the worker pool.
type ValidationWorkItem struct {
	Result *objects.ValidationResult
	Task   *objects.ValidationTask
}

// TaskWorkerPool manages a fixed pool of goroutines that each consume from a
// shared buffered queue.  This decouples result ingestion from the learning
// state mutation and improves throughput for high-volume validation workloads.
type TaskWorkerPool struct {
	queue   chan ValidationWorkItem
	workers int
	wg      sync.WaitGroup
}

// NewTaskWorkerPool creates a pool with `workers` goroutines and a queue
// buffered to `queueCap` items.
func NewTaskWorkerPool(workers, queueCap int) *TaskWorkerPool {
	if workers < 1 {
		workers = 1
	}
	if queueCap < workers {
		queueCap = workers * 2
	}
	return &TaskWorkerPool{
		queue:   make(chan ValidationWorkItem, queueCap),
		workers: workers,
	}
}

// Start launches the worker goroutines.  Each worker calls processFunc for
// every dequeued item.  Workers exit when the context is cancelled or the
// queue channel is closed (via Stop).
func (p *TaskWorkerPool) Start(ctx context.Context, processFunc func(ValidationWorkItem)) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case item, ok := <-p.queue:
					if !ok {
						return
					}
					processFunc(item)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

// Submit enqueues a work item.  Returns true if the item was accepted, false
// if the queue is full (back-pressure signal – caller should log and continue).
func (p *TaskWorkerPool) Submit(item ValidationWorkItem) bool {
	select {
	case p.queue <- item:
		return true
	default:
		return false
	}
}

// Stop closes the input queue and waits for all in-flight items to finish.
func (p *TaskWorkerPool) Stop() {
	close(p.queue)
	p.wg.Wait()
}

// QueueDepth returns the current number of items waiting to be processed.
func (p *TaskWorkerPool) QueueDepth() int {
	return len(p.queue)
}
