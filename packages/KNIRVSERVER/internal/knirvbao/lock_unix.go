//go:build unix

package knirvbao

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// The lock is never unlinked. OpenBao itself removes a socket path before
// binding it, so serializing cooperating launchers prevents one manager from
// unlinking another manager's live socket.
func (m *Manager) lockRoot(root string) error {
	if m.lockFile != nil {
		return nil
	}
	if err := privateDir(root); err != nil {
		return err
	}
	fd, err := syscall.Open(filepath.Join(root, "manager.lock"), syscall.O_RDWR|syscall.O_CREAT|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), "manager.lock")
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || stat.Uid != uint32(os.Getuid()) {
		_ = file.Close()
		return errors.New("OpenBao manager lock is not a private owned file")
	}
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return errResourceBusy
		}
		return fmt.Errorf("lock OpenBao data directory: %w", err)
	}
	m.lockFile = file
	return nil
}

func (m *Manager) releaseLock() {
	if m.lockFile == nil {
		return
	}
	_ = syscall.Flock(int(m.lockFile.Fd()), syscall.LOCK_UN)
	_ = m.lockFile.Close()
	m.lockFile = nil
}
