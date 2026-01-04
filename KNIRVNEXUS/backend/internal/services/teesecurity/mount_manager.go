package teesecurity

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// MountManager handles mount namespace configuration and pivot_root
type MountManager struct {
	config MountConfig
}

// NewMountManager creates a new MountManager instance
func NewMountManager(config MountConfig) *MountManager {
	return &MountManager{config: config}
}

// SetupMountNamespace configures the mount namespace
// This must be called from within the container process (after fork)
func (mm *MountManager) SetupMountNamespace() error {
	// 1. Make everything private (no mount propagation to/from host)
	if err := unix.Mount("", "/", "", unix.MS_PRIVATE|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to make / private: %w", err)
	}

	// 2. Prepare new root
	if err := mm.prepareNewRoot(); err != nil {
		return err
	}

	// 3. Perform pivot_root
	if err := mm.pivotRoot(); err != nil {
		return err
	}

	// 4. Mount special filesystems
	if err := mm.mountSpecialFS(); err != nil {
		return err
	}

	// 5. Apply bind mounts
	if err := mm.applyBindMounts(); err != nil {
		return err
	}

	// 6. Make root read-only if configured
	if mm.config.ReadOnlyRoot {
		if err := mm.makeReadOnly(); err != nil {
			return err
		}
	}

	return nil
}

// prepareNewRoot prepares the new root filesystem
func (mm *MountManager) prepareNewRoot() error {
	// Create new root if it doesn't exist
	if err := os.MkdirAll(mm.config.RootFS, 0755); err != nil {
		return fmt.Errorf("failed to create new root: %w", err)
	}

	// Mount new root as bind mount
	if err := unix.Mount(mm.config.RootFS, mm.config.RootFS, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to bind mount new root: %w", err)
	}

	return nil
}

// pivotRoot performs the pivot_root system call
func (mm *MountManager) pivotRoot() error {
	// Create put_old directory
	putOld := filepath.Join(mm.config.RootFS, ".pivot_root")
	if err := os.MkdirAll(putOld, 0700); err != nil {
		return fmt.Errorf("failed to create put_old: %w", err)
	}

	// pivot_root(new_root, put_old)
	if err := unix.PivotRoot(mm.config.RootFS, putOld); err != nil {
		return fmt.Errorf("failed to pivot_root: %w", err)
	}

	// Change to new root
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("failed to chdir to /: %w", err)
	}

	// Unmount old root
	putOld = "/.pivot_root"
	if err := unix.Unmount(putOld, unix.MNT_DETACH); err != nil {
		return fmt.Errorf("failed to unmount old root: %w", err)
	}

	// Remove put_old
	if err := os.RemoveAll(putOld); err != nil {
		return fmt.Errorf("failed to remove put_old: %w", err)
	}

	return nil
}

// mountSpecialFS mounts special filesystems like /proc, /sys, /dev
func (mm *MountManager) mountSpecialFS() error {
	// Mount /proc
	if mm.config.MountProc {
		if err := os.MkdirAll("/proc", 0755); err != nil {
			return err
		}
		if err := unix.Mount("proc", "/proc", "proc", unix.MS_NOSUID|unix.MS_NOEXEC|unix.MS_NODEV, ""); err != nil {
			return fmt.Errorf("failed to mount /proc: %w", err)
		}
	}

	// Mount /sys (read-only)
	if mm.config.MountSys {
		if err := os.MkdirAll("/sys", 0755); err != nil {
			return err
		}
		if err := unix.Mount("sysfs", "/sys", "sysfs", unix.MS_NOSUID|unix.MS_NOEXEC|unix.MS_NODEV|unix.MS_RDONLY, ""); err != nil {
			return fmt.Errorf("failed to mount /sys: %w", err)
		}
	}

	// Mount /dev (minimal)
	if mm.config.MountDev {
		if err := mm.mountMinimalDev(); err != nil {
			return err
		}
	}

	return nil
}

// mountMinimalDev creates a minimal /dev with essential device nodes
func (mm *MountManager) mountMinimalDev() error {
	if err := os.MkdirAll("/dev", 0755); err != nil {
		return err
	}

	// Mount tmpfs on /dev
	if err := unix.Mount("tmpfs", "/dev", "tmpfs", unix.MS_NOSUID|unix.MS_NOEXEC, "mode=755"); err != nil {
		return fmt.Errorf("failed to mount /dev: %w", err)
	}

	// Create essential device nodes
	devices := []struct {
		path string
		mode uint32
		dev  int
	}{
		{"/dev/null", unix.S_IFCHR | 0666, int(unix.Mkdev(1, 3))},
		{"/dev/zero", unix.S_IFCHR | 0666, int(unix.Mkdev(1, 5))},
		{"/dev/random", unix.S_IFCHR | 0666, int(unix.Mkdev(1, 8))},
		{"/dev/urandom", unix.S_IFCHR | 0666, int(unix.Mkdev(1, 9))},
	}

	for _, d := range devices {
		if err := unix.Mknod(d.path, d.mode, d.dev); err != nil {
			if !os.IsExist(err) {
				return fmt.Errorf("failed to create %s: %w", d.path, err)
			}
		}
	}

	return nil
}

// applyBindMounts applies additional bind mounts
func (mm *MountManager) applyBindMounts() error {
	for _, bm := range mm.config.BindMounts {
		// Create mount point
		if err := os.MkdirAll(bm.Destination, 0755); err != nil {
			return fmt.Errorf("failed to create mount point %s: %w", bm.Destination, err)
		}

		// Bind mount
		flags := unix.MS_BIND | unix.MS_REC
		if bm.ReadOnly {
			flags |= unix.MS_RDONLY
		}

		if err := unix.Mount(bm.Source, bm.Destination, "", uintptr(flags), ""); err != nil {
			return fmt.Errorf("failed to bind mount %s -> %s: %w", bm.Source, bm.Destination, err)
		}
	}

	return nil
}

// makeReadOnly makes the root filesystem read-only
func (mm *MountManager) makeReadOnly() error {
	// Remount root as read-only
	if err := unix.Mount("", "/", "", unix.MS_REMOUNT|unix.MS_RDONLY|unix.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to remount root as read-only: %w", err)
	}

	return nil
}
