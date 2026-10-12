package controller

import (
	"context"
	"testing"
	"time"
)

func TestSchedulerStopsWithItsContextAndHonoursOptOut(t *testing.T) {
	c := &Controller{}
	t.Setenv("KNIRV_HEAD_AUTOTRAIN", "0")
	done := make(chan struct{})
	go func() { c.RunHEADScheduler(context.Background(), time.Minute); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("opt-out must return immediately")
	}

	t.Setenv("KNIRV_HEAD_AUTOTRAIN", "")
	ctx, cancel := context.WithCancel(context.Background())
	done = make(chan struct{})
	go func() { c.RunHEADScheduler(ctx, time.Minute); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler ignored its context")
	}
}
