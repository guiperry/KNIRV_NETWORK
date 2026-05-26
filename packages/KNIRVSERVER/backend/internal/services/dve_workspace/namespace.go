package dve_workspace

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// DVENamespace manages a user+mount namespace for a DVE workspace.
// The host process calls os/exec with SysProcAttr to clone namespaces
// before running the mount. This requires no root privileges on Linux >= 3.18.
type DVENamespace struct {
	Workspace *OverlayWorkspace
	UID, GID  int
	cmd       *exec.Cmd
}

// SpawnNamespaced starts a long-running helper process inside new
// user + mount namespaces, then performs the OverlayFS mount from within.
// The helper process stays alive for the DVE lifetime; killing it
// releases the namespace and triggers lazy unmount.
func SpawnNamespaced(ws *OverlayWorkspace, uid, gid int, fuseOverlayFSBin string) (*DVENamespace, error) {
	cmd := exec.Command("/proc/self/exe", "--dve-ns-helper")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: uid, Size: 65536},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: gid, Size: 65536},
		},
	}
	// Pass workspace paths to helper via env
	cmd.Env = []string{
		"DVE_LOWER=" + ws.LowerDir,
		"DVE_UPPER=" + ws.UpperDir,
		"DVE_WORK=" + ws.WorkDir,
		"DVE_MERGED=" + ws.MergedDir,
		"DVE_FUSE_BIN=" + fuseOverlayFSBin,
	}
	// Capture stderr for debugging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("spawn ns helper: %w", err)
	}
	// Read stderr in background to prevent deadlock
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				log.Printf("DVENamespace[%s]: %s", ws.DVEID, string(buf[:n]))
			}
			if err != nil {
				return
			}
		}
	}()
	ws.mounted = true
	return &DVENamespace{Workspace: ws, UID: uid, GID: gid, cmd: cmd}, nil
}

// Teardown sends SIGTERM to the helper, which causes lazy unmount of merged/.
func (n *DVENamespace) Teardown() error {
	if n.cmd != nil && n.cmd.Process != nil {
		if err := n.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			log.Printf("DVENamespace: signal error: %v", err)
		}
		_, _ = n.cmd.Process.Wait()
	}
	return nil
}

// NamespaceHelper runs inside the user+mount namespace to perform the OverlayFS mount.
// It reads DVE_* env vars, mounts overlay, then blocks until stdin is closed.
func NamespaceHelper() {
	lower := os.Getenv("DVE_LOWER")
	upper := os.Getenv("DVE_UPPER")
	work := os.Getenv("DVE_WORK")
	merged := os.Getenv("DVE_MERGED")

	if lower == "" || merged == "" {
		fmt.Fprintf(os.Stderr, "DVE_* environment variables not set\n")
		os.Exit(1)
	}

	ws := &OverlayWorkspace{
		LowerDir:  lower,
		UpperDir:  upper,
		WorkDir:   work,
		MergedDir: merged,
	}

	if err := ws.Mount(); err != nil {
		fmt.Fprintf(os.Stderr, "overlay mount failed: %v\n", err)
		os.Exit(1)
	}

	// Block until stdin is closed (parent process signals teardown)
	buf := make([]byte, 1)
	os.Stdin.Read(buf)

	// Unmount
	if err := ws.Unmount(); err != nil {
		fmt.Fprintf(os.Stderr, "overlay unmount failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
