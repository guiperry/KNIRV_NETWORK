package dve_workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// OverlayWorkspace manages an OverlayFS mount for a DVE workspace.
// Each DVE gets its own upper/work/merged directories sharing a common lower BusyBox rootfs.
type OverlayWorkspace struct {
	DVEID     string
	LowerDir  string // busybox-rootfs (shared, read-only)
	UpperDir  string // per-DVE writable CoW layer
	WorkDir   string // OverlayFS internal scratch
	MergedDir string // DVE's visible root
	mounted   bool
}

// NewOverlayWorkspace allocates directory structure for a DVE.
// Does NOT mount — call Mount() inside the target namespace.
func NewOverlayWorkspace(workspaceRoot, busyboxRootfs, dveID string) (*OverlayWorkspace, error) {
	base := filepath.Join(workspaceRoot, dveID)
	dirs := []string{
		filepath.Join(base, "upper"),
		filepath.Join(base, "work"),
		filepath.Join(base, "merged"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return &OverlayWorkspace{
		DVEID:     dveID,
		LowerDir:  busyboxRootfs,
		UpperDir:  filepath.Join(base, "upper"),
		WorkDir:   filepath.Join(base, "work"),
		MergedDir: filepath.Join(base, "merged"),
	}, nil
}

// Mount performs the overlayfs mount.
// Must be called from within a mount namespace (see namespace.go).
func (w *OverlayWorkspace) Mount() error {
	opts := fmt.Sprintf(
		"lowerdir=%s,upperdir=%s,workdir=%s",
		w.LowerDir, w.UpperDir, w.WorkDir,
	)
	err := unix.Mount("overlay", w.MergedDir, "overlay", 0, opts)
	if err != nil {
		// Fallback: fuse-overlayfs for rootless environments without kernel overlay support
		err = w.mountFuseOverlay(opts)
	}
	if err != nil {
		return fmt.Errorf("overlayfs mount failed: %w", err)
	}
	w.mounted = true
	return nil
}

// Unmount tears down the overlayfs mount and does NOT delete upper layer data.
func (w *OverlayWorkspace) Unmount() error {
	if !w.mounted {
		return nil
	}
	if err := unix.Unmount(w.MergedDir, unix.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount %s: %w", w.MergedDir, err)
	}
	w.mounted = false
	return nil
}

// Destroy unmounts and removes all per-DVE directories (upper/work/merged).
// Call when the DVE is permanently terminated, not just suspended.
func (w *OverlayWorkspace) Destroy() error {
	_ = w.Unmount()
	base := filepath.Dir(w.UpperDir)
	return os.RemoveAll(base)
}

func (w *OverlayWorkspace) mountFuseOverlay(opts string) error {
	// fuse-overlayfs binary path is configurable
	cmd := exec.Command("fuse-overlayfs", "-o", opts, w.MergedDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fuse-overlayfs: %w — %s", err, out)
	}
	return nil
}
