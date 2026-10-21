package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAdmissionServesInteractiveBeforeEarlierTeacher(t *testing.T) {
	a := NewAdmission(1)
	hold, _ := a.Acquire(context.Background(), 1)
	var mu sync.Mutex
	var order []int
	var wg sync.WaitGroup
	start := func(class int) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rel, err := a.Acquire(context.Background(), class)
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			order = append(order, class)
			mu.Unlock()
			rel()
		}()
	}
	start(3)
	waitDepth(t, a, 1)
	start(2)
	waitDepth(t, a, 2)
	start(0)
	waitDepth(t, a, 3)
	hold()
	wg.Wait()
	if len(order) != 3 || order[0] != 0 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("order %v", order)
	}
}

func waitDepth(t *testing.T, a *Admission, n int) {
	t.Helper()
	for i := 0; i < 200; i++ {
		a.mu.Lock()
		got := len(a.waiters)
		a.mu.Unlock()
		if got == n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("queue never reached depth %d", n)
}

func TestAdmissionAgingPreventsStarvation(t *testing.T) {
	a := NewAdmission(1)
	clock := time.Unix(0, 0)
	a.now = func() time.Time { return clock }
	hold, _ := a.Acquire(context.Background(), 0)
	got := make(chan int, 2)
	go func() { rel, _ := a.Acquire(context.Background(), 3); got <- 3; rel() }()
	waitDepth(t, a, 1)
	clock = clock.Add(4 * agingInterval) // the teacher call has now waited 4 intervals: class 3 -> -1
	go func() { rel, _ := a.Acquire(context.Background(), 0); got <- 0; rel() }()
	waitDepth(t, a, 2)
	hold()
	if first := <-got; first != 3 {
		t.Fatalf("aged waiter should go first, got class %d", first)
	}
	<-got
}

func TestAdmissionCancelLeavesTheQueue(t *testing.T) {
	a := NewAdmission(1)
	hold, _ := a.Acquire(context.Background(), 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := a.Acquire(ctx, 1); done <- err }()
	waitDepth(t, a, 1)
	cancel()
	if err := <-done; err == nil {
		t.Fatal("cancelled acquire succeeded")
	}
	waitDepth(t, a, 0)
	hold()
	rel, err := a.Acquire(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	rel()
}

func TestAdmissionRejectsWhenFull(t *testing.T) {
	a := NewAdmission(1)
	hold, _ := a.Acquire(context.Background(), 1)
	defer hold()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < maxQueued; i++ {
		go func() { _, _ = a.Acquire(ctx, 1) }()
	}
	waitDepth(t, a, maxQueued)
	if _, err := a.Acquire(context.Background(), 1); err != ErrQueueFull {
		t.Fatalf("got %v", err)
	}
}

func TestPriorityFieldIsStrippedAndQueued(t *testing.T) {
	var seen string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen = string(b)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer up.Close()
	h, _ := NewServer(strings.TrimPrefix(up.URL, "http://"), "m", "", nil, Options{AdmitSlots: 1})
	s := httptest.NewServer(h)
	defer s.Close()
	resp, err := http.Post(s.URL+"/v1/chat/completions", "application/json",
		strings.NewReader(`{"knirv_priority":0,"knirv_stage":"x","messages":[]}`))
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("%v %v", err, resp)
	}
	if strings.Contains(seen, "knirv_") {
		t.Fatalf("upstream saw %s", seen)
	}
	st, _ := http.Get(s.URL + "/v1/queue/stats")
	b, _ := io.ReadAll(st.Body)
	if !strings.Contains(string(b), `"admitted":[1,0,0,0]`) {
		t.Fatalf("stats %s", b)
	}
}
