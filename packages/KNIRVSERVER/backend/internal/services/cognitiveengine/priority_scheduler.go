package cognitiveengine

import (
	"log"
	"sync"
	"time"
)

type TaskPriority int

const (
	PriorityCritical TaskPriority = iota
	PriorityHigh
	PriorityNormal
	PriorityLow
)

func (p TaskPriority) String() string {
	switch p {
	case PriorityCritical:
		return "critical"
	case PriorityHigh:
		return "high"
	case PriorityNormal:
		return "normal"
	case PriorityLow:
		return "low"
	default:
		return "unknown"
	}
}

type PrioritizedWorkItem struct {
	WorkItem     ValidationWorkItem
	Priority     TaskPriority
	EnqueuedAt   time.Time
	Deadline     time.Time
	IsolatableID string
}

type PriorityScheduler struct {
	queues    map[TaskPriority]chan PrioritizedWorkItem
	workers   int
	wg        sync.WaitGroup
	ctx       interface{ Done() <-chan struct{} }
	processor func(PrioritizedWorkItem)
	mu        sync.RWMutex
}

func NewPriorityScheduler(ctx interface{ Done() <-chan struct{} }, workers int, queueSize int) *PriorityScheduler {
	if workers < 1 {
		workers = 4
	}
	if queueSize < 1 {
		queueSize = 64
	}

	ps := &PriorityScheduler{
		queues:  make(map[TaskPriority]chan PrioritizedWorkItem),
		workers: workers,
		ctx:     ctx,
	}

	for _, p := range []TaskPriority{PriorityCritical, PriorityHigh, PriorityNormal, PriorityLow} {
		ps.queues[p] = make(chan PrioritizedWorkItem, queueSize)
	}

	return ps
}

func (ps *PriorityScheduler) SetProcessor(f func(PrioritizedWorkItem)) {
	ps.processor = f
}

func (ps *PriorityScheduler) Start() {
	for i := 0; i < ps.workers; i++ {
		ps.wg.Add(1)
		go ps.worker(i)
	}
}

func (ps *PriorityScheduler) worker(id int) {
	defer ps.wg.Done()

	log.Printf("[PriorityScheduler] worker %d started", id)

	for {
		item, ok := ps.selectNextItem()
		if !ok {
			log.Printf("[PriorityScheduler] worker %d shutting down", id)
			return
		}

		select {
		case <-ps.ctx.Done():
			log.Printf("[PriorityScheduler] worker %d context cancelled", id)
			return
		default:
			log.Printf("[PriorityScheduler] worker %d processing %s priority item", id, item.Priority.String())
			ps.processItem(item)
		}
	}
}

func (ps *PriorityScheduler) selectNextItem() (PrioritizedWorkItem, bool) {
	for _, priority := range []TaskPriority{PriorityCritical, PriorityHigh, PriorityNormal, PriorityLow} {
		select {
		case item, ok := <-ps.queues[priority]:
			if ok {
				return item, true
			}
		default:
		}
	}

	select {
	case <-ps.ctx.Done():
		return PrioritizedWorkItem{}, false
	default:
		time.Sleep(10 * time.Millisecond)
		return ps.selectNextItem()
	}
}

func (ps *PriorityScheduler) processItem(item PrioritizedWorkItem) {
	if ps.processor != nil {
		ps.processor(item)
	}
}

func (ps *PriorityScheduler) Submit(item ValidationWorkItem, priority TaskPriority) bool {
	pwi := PrioritizedWorkItem{
		WorkItem:   item,
		Priority:   priority,
		EnqueuedAt: time.Now(),
	}

	select {
	case ps.queues[priority] <- pwi:
		return true
	default:
		return false
	}
}

func (ps *PriorityScheduler) SubmitWithDeadline(item ValidationWorkItem, priority TaskPriority, deadline time.Time) bool {
	pwi := PrioritizedWorkItem{
		WorkItem:   item,
		Priority:   priority,
		EnqueuedAt: time.Now(),
		Deadline:   deadline,
	}

	select {
	case ps.queues[priority] <- pwi:
		return true
	default:
		return false
	}
}

func (ps *PriorityScheduler) SubmitCritical(item ValidationWorkItem, isolatableID string) bool {
	pwi := PrioritizedWorkItem{
		WorkItem:     item,
		Priority:     PriorityCritical,
		EnqueuedAt:   time.Now(),
		IsolatableID: isolatableID,
	}

	select {
	case ps.queues[PriorityCritical] <- pwi:
		return true
	default:
		return false
	}
}

func (ps *PriorityScheduler) Stop() {
	for _, ch := range ps.queues {
		close(ch)
	}
	ps.wg.Wait()
}

func (ps *PriorityScheduler) QueueDepth(priority TaskPriority) int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.queues[priority])
}

func (ps *PriorityScheduler) TotalQueueDepth() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	total := 0
	for _, ch := range ps.queues {
		total += len(ch)
	}
	return total
}

func (ps *PriorityScheduler) IsHighPriority(item ValidationWorkItem, guardrailViolations int) TaskPriority {
	if guardrailViolations > 0 {
		return PriorityCritical
	}
	return PriorityNormal
}
