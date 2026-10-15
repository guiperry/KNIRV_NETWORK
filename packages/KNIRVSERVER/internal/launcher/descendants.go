package launcher

import (
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Every managed service (backend_server and the daemons it starts: knirvmodels,
// knirvgateway, knirvgraph, knirvchain, knirvoracle, llama-server, ...) runs in
// its own process group, so a Ctrl+C in the terminal reaches only this process.
// Each service has an orderly stop, but a stop that times out, a backend that is
// SIGKILLed, or a manager that forgot a child would leave processes behind.
// descendantPIDs and reapDescendants are the backstop: they find everything that
// descends from this process, whatever its group, and end whatever survived.

// descendantPIDs returns every live descendant of root, children first. It reads
// /proc, so it returns nil where that is unavailable.
func descendantPIDs(root int) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	children := map[int][]int{}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if ppid, ok := parentPID(pid); ok {
			children[ppid] = append(children[ppid], pid)
		}
	}
	var out []int
	var walk func(int)
	walk = func(p int) {
		for _, c := range children[p] {
			walk(c)
			out = append(out, c)
		}
	}
	walk(root)
	return out
}

// parentPID reads field 4 of /proc/<pid>/stat. The command name (field 2) is in
// parentheses and may itself contain spaces or parentheses, so parse after the
// last ')'.
func parentPID(pid int) (int, bool) {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, false
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return 0, false
	}
	f := strings.Fields(s[i+1:])
	if len(f) < 2 { // state, ppid
		return 0, false
	}
	ppid, err := strconv.Atoi(f[1])
	return ppid, err == nil
}

func processAlive(pid int) bool {
	if err := syscall.Kill(pid, 0); err != nil {
		return err == syscall.EPERM
	}
	// A zombie still answers signal 0; it is already dead for our purposes.
	if b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil {
		s := string(b)
		if i := strings.LastIndexByte(s, ')'); i >= 0 && len(s) > i+2 && s[i+2] == 'Z' {
			return false
		}
	}
	return true
}

// reapDescendants ends every process in pids that is still alive: SIGTERM, up to
// grace for them to exit, then SIGKILL. It returns how many needed SIGKILL.
// Pass the snapshot taken before the orderly stop plus any new descendants, so a
// child that was re-parented to init in the meantime is still found.
func reapDescendants(pids []int, grace time.Duration) (killed int) {
	var live []int
	for _, p := range pids {
		if p > 1 && p != os.Getpid() && processAlive(p) {
			live = append(live, p)
			_ = syscall.Kill(p, syscall.SIGTERM)
		}
	}
	if len(live) == 0 {
		return 0
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		alive := false
		for _, p := range live {
			if processAlive(p) {
				alive = true
				break
			}
		}
		if !alive {
			return 0
		}
		time.Sleep(100 * time.Millisecond)
	}
	for _, p := range live {
		if processAlive(p) {
			log.Printf("shutdown: PID %d ignored SIGTERM, sending SIGKILL", p)
			_ = syscall.Kill(p, syscall.SIGKILL)
			killed++
		}
	}
	return killed
}

// mergePIDs returns a with the entries of b it does not already contain.
func mergePIDs(a, b []int) []int {
	seen := make(map[int]bool, len(a))
	for _, p := range a {
		seen[p] = true
	}
	for _, p := range b {
		if !seen[p] {
			a = append(a, p)
			seen[p] = true
		}
	}
	return a
}
