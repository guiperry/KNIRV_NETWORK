package launcher

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// A service in its own process group, with a child of its own, must still be found
// and ended by the shutdown sweep.
func TestReapDescendantsEndsGrandchildrenInOtherGroups(t *testing.T) {
	cmd := exec.Command("sh", "-c", "sleep 60 & sleep 60; wait")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Skip("cannot start sh:", err)
	}
	go cmd.Wait() // reap so the process does not linger as a zombie

	var pids []int
	for i := 0; i < 50 && len(pids) < 3; i++ {
		pids = descendantPIDs(os.Getpid())
		time.Sleep(50 * time.Millisecond)
	}
	if len(pids) < 3 { // sh + two sleeps
		t.Skipf("descendants not visible (no /proc?): %v", pids)
	}
	if got := reapDescendants(pids, 2*time.Second); got != 0 {
		t.Fatalf("plain sleep should end on SIGTERM, %d needed SIGKILL", got)
	}
	for _, p := range pids {
		if processAlive(p) {
			t.Fatalf("PID %d survived the sweep", p)
		}
	}
}

func TestReapDescendantsKillsWhatIgnoresSIGTERM(t *testing.T) {
	cmd := exec.Command("sh", "-c", "trap '' TERM; while :; do sleep 1; done")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Skip("cannot start sh:", err)
	}
	go cmd.Wait()
	time.Sleep(200 * time.Millisecond) // let the trap install
	if got := reapDescendants([]int{cmd.Process.Pid}, 500*time.Millisecond); got != 1 {
		t.Fatalf("want 1 SIGKILL, got %d", got)
	}
	time.Sleep(100 * time.Millisecond)
	if processAlive(cmd.Process.Pid) {
		t.Fatal("process ignoring SIGTERM survived")
	}
}

func TestReapDescendantsNeverTouchesSelfOrInit(t *testing.T) {
	if got := reapDescendants([]int{0, 1, os.Getpid()}, 50*time.Millisecond); got != 0 {
		t.Fatalf("got %d", got)
	}
}
