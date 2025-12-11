package crosschain

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// TransferQueue manages a queue of pending transfers
type TransferQueue struct {
	queue   *list.List
	items   map[string]*list.Element
	maxSize int
	mu      sync.RWMutex
}

// QueueItem represents an item in the transfer queue
type QueueItem struct {
	Transfer  *CrossChainTransfer
	AddedAt   time.Time
	ProcessAt time.Time
	Retries   int
}

// NewTransferQueue creates a new transfer queue
func NewTransferQueue() *TransferQueue {
	return &TransferQueue{
		queue:   list.New(),
		items:   make(map[string]*list.Element),
		maxSize: 10000, // Maximum 10k pending transfers
	}
}

// Enqueue adds a transfer to the queue
func (tq *TransferQueue) Enqueue(transfer *CrossChainTransfer) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Check if already in queue
	if _, exists := tq.items[transfer.TransferID]; exists {
		return nil // Already queued
	}

	// Check queue size
	if tq.queue.Len() >= tq.maxSize {
		return ErrQueueFull
	}

	// Create queue item
	item := &QueueItem{
		Transfer:  transfer,
		AddedAt:   time.Now(),
		ProcessAt: time.Now(),
		Retries:   0,
	}

	// Add to queue
	element := tq.queue.PushBack(item)
	tq.items[transfer.TransferID] = element

	return nil
}

// Dequeue removes and returns the next transfer from the queue
func (tq *TransferQueue) Dequeue() *CrossChainTransfer {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Get front element
	element := tq.queue.Front()
	if element == nil {
		return nil
	}

	// Remove from queue
	item := element.Value.(*QueueItem)
	tq.queue.Remove(element)
	delete(tq.items, item.Transfer.TransferID)

	return item.Transfer
}

// Peek returns the next transfer without removing it
func (tq *TransferQueue) Peek() *CrossChainTransfer {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	element := tq.queue.Front()
	if element == nil {
		return nil
	}

	item := element.Value.(*QueueItem)
	return item.Transfer
}

// Remove removes a transfer from the queue
func (tq *TransferQueue) Remove(transferID string) bool {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	element, exists := tq.items[transferID]
	if !exists {
		return false
	}

	tq.queue.Remove(element)
	delete(tq.items, transferID)
	return true
}

// Contains checks if a transfer is in the queue
func (tq *TransferQueue) Contains(transferID string) bool {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	_, exists := tq.items[transferID]
	return exists
}

// Size returns the number of items in the queue
func (tq *TransferQueue) Size() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	return tq.queue.Len()
}

// IsEmpty returns true if the queue is empty
func (tq *TransferQueue) IsEmpty() bool {
	return tq.Size() == 0
}

// IsFull returns true if the queue is full
func (tq *TransferQueue) IsFull() bool {
	return tq.Size() >= tq.maxSize
}

// Clear removes all items from the queue
func (tq *TransferQueue) Clear() {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	tq.queue.Init()
	tq.items = make(map[string]*list.Element)
}

// GetReadyTransfers returns transfers that are ready to be processed
func (tq *TransferQueue) GetReadyTransfers() []*CrossChainTransfer {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	now := time.Now()
	ready := make([]*CrossChainTransfer, 0)

	for element := tq.queue.Front(); element != nil; element = element.Next() {
		item := element.Value.(*QueueItem)
		if now.After(item.ProcessAt) || now.Equal(item.ProcessAt) {
			ready = append(ready, item.Transfer)
		}
	}

	return ready
}

// Retry schedules a transfer for retry
func (tq *TransferQueue) Retry(transferID string, delay time.Duration) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	element, exists := tq.items[transferID]
	if !exists {
		return ErrTransferNotFound
	}

	item := element.Value.(*QueueItem)
	item.Retries++
	item.ProcessAt = time.Now().Add(delay)

	return nil
}

// GetStats returns queue statistics
func (tq *TransferQueue) GetStats() QueueStats {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	stats := QueueStats{
		Size:    tq.queue.Len(),
		MaxSize: tq.maxSize,
	}

	// Count retries
	for element := tq.queue.Front(); element != nil; element = element.Next() {
		item := element.Value.(*QueueItem)
		if item.Retries > 0 {
			stats.Retries++
		}
	}

	return stats
}

// QueueStats represents queue statistics
type QueueStats struct {
	Size    int `json:"size"`
	MaxSize int `json:"max_size"`
	Retries int `json:"retries"`
}

// Custom errors
var (
	ErrQueueFull        = fmt.Errorf("transfer queue is full")
	ErrTransferNotFound = fmt.Errorf("transfer not found in queue")
)
