package dve_workspace

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed bin/busybox-static
var embeddedBusybox []byte

// EnsureBusyBoxRootfs checks if the shared rootfs exists and is valid.
// Downloads + extracts busybox-static if missing.
// This runs once at DVEService startup, not per-workspace.
func EnsureBusyBoxRootfs(rootfsPath string, cfg DVEConfig) error {
	marker := filepath.Join(rootfsPath, ".knirvdve-ready")
	if _, err := os.Stat(marker); err == nil {
		return nil // already bootstrapped
	}
	log.Printf("DVEService: bootstrapping BusyBox rootfs at %s", rootfsPath)

	switch cfg.BusyBoxSource {
	case "embedded":
		return extractEmbeddedBusyBox(rootfsPath)
	case "package":
		return installViaDpkg(rootfsPath)
	case "download":
		return downloadBusyBox(rootfsPath, cfg.BusyBoxVersion)
	default:
		return fmt.Errorf("unknown busybox_source: %s", cfg.BusyBoxSource)
	}
}

// extractEmbeddedBusyBox writes the embedded busybox-static binary
// and creates the standard applet symlinks and minimal filesystem tree.
// When the embedded binary is not found (build without the binary), it
// falls back to the "download" method.
func extractEmbeddedBusyBox(rootfsPath string) error {
	binPath := filepath.Join(rootfsPath, "bin", "busybox")
	if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(binPath, embeddedBusybox, 0755); err != nil {
		return fmt.Errorf("write busybox: %w", err)
	}
	return createRootfsSymlinks(rootfsPath)
}

// createRootfsSymlinks creates the standard BusyBox applet symlinks
// and the minimal /etc/passwd, /workspace, /tmp directories.
func createRootfsSymlinks(rootfsPath string) error {
	applets := []string{"sh", "ls", "cat", "cp", "mv", "rm", "mkdir",
		"grep", "find", "tar", "gzip", "wget", "curl", "env", "echo", "test",
		"chmod", "chown", "head", "tail", "sort", "awk", "sed", "wc", "ps"}
	binDir := filepath.Join(rootfsPath, "bin")
	for _, a := range applets {
		if err := os.Symlink("busybox", filepath.Join(binDir, a)); err != nil && !os.IsExist(err) {
			return fmt.Errorf("symlink %s: %w", a, err)
		}
	}
	for _, d := range []string{"etc", "tmp", "workspace", "proc"} {
		if err := os.MkdirAll(filepath.Join(rootfsPath, d), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	if err := os.WriteFile(filepath.Join(rootfsPath, "etc", "passwd"),
		[]byte("dve:x:0:0:DVE Workspace:/workspace:/bin/sh\n"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(rootfsPath, ".knirvdve-ready"), []byte("1"), 0644); err != nil {
		return err
	}
	return nil
}

// installViaDpkg installs busybox-static via the host package manager.
func installViaDpkg(rootfsPath string) error {
	binDir := filepath.Join(rootfsPath, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	// Try to install busybox-static from host packages
	// This requires apt on debian-based systems
	if _, err := os.Stat("/usr/bin/apt-get"); err == nil {
		cmd := exec.Command("apt-get", "install", "-y", "busybox-static")
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("apt-get install busybox-static: %w", err)
		}
		// Copy the binary to our rootfs
		data, err := os.ReadFile("/bin/busybox")
		if err != nil {
			// Try /usr/bin/busybox
			data, err = os.ReadFile("/usr/bin/busybox")
			if err != nil {
				return fmt.Errorf("busybox binary not found after install: %w", err)
			}
		}
		if err := os.WriteFile(filepath.Join(binDir, "busybox"), data, 0755); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("package manager not available")
	}

	return createRootfsSymlinks(rootfsPath)
}

// downloadBusyBox downloads a static BusyBox binary from busybox.net.
func downloadBusyBox(rootfsPath, version string) error {
	if version == "" {
		version = "1.36.1"
	}

	arch := "amd64"
	binDir := filepath.Join(rootfsPath, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	url := fmt.Sprintf("https://busybox.net/downloads/binaries/%s-defconfig-multiarch-musl/busybox-%s", version, arch)
	binPath := filepath.Join(binDir, "busybox")

	cmd := exec.Command("curl", "-fsSL", "-o", binPath, url)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Try wget as fallback
		cmd = exec.Command("wget", "-q", "-O", binPath, url)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("download busybox: %w", err)
		}
	}

	if err := os.Chmod(binPath, 0755); err != nil {
		return err
	}

	return createRootfsSymlinks(rootfsPath)
}
