package teesecurity

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// NamespaceManager manages Linux namespace creation and configuration
type NamespaceManager struct {
	config NamespaceConfig
}

// NewNamespaceManager creates a new NamespaceManager instance
func NewNamespaceManager(config NamespaceConfig) *NamespaceManager {
	return &NamespaceManager{config: config}
}

// CreateNamespaces creates all enabled namespaces using unshare
func (nm *NamespaceManager) CreateNamespaces() ([]Namespace, error) {
	var namespaces []Namespace

	flags := 0
	if nm.config.EnablePID {
		flags |= unix.CLONE_NEWPID
		namespaces = append(namespaces, Namespace{Type: "pid"})
	}
	if nm.config.EnableNetwork {
		flags |= unix.CLONE_NEWNET
		namespaces = append(namespaces, Namespace{Type: "net"})
	}
	if nm.config.EnableMount {
		flags |= unix.CLONE_NEWNS
		namespaces = append(namespaces, Namespace{Type: "mnt"})
	}
	if nm.config.EnableUTS {
		flags |= unix.CLONE_NEWUTS
		namespaces = append(namespaces, Namespace{Type: "uts"})
	}
	if nm.config.EnableIPC {
		flags |= unix.CLONE_NEWIPC
		namespaces = append(namespaces, Namespace{Type: "ipc"})
	}
	if nm.config.EnableUser {
		flags |= unix.CLONE_NEWUSER
		namespaces = append(namespaces, Namespace{Type: "user"})
	}

	// Use unix.Unshare to create namespaces in current process
	if err := unix.Unshare(flags); err != nil {
		return nil, fmt.Errorf("failed to create namespaces: %w", err)
	}

	return namespaces, nil
}

// SetupUserNamespace configures user namespace UID/GID mappings
func (nm *NamespaceManager) SetupUserNamespace(uid, gid int) error {
	// Write UID mapping
	uidMap := fmt.Sprintf("0 %d 1", uid)
	if err := os.WriteFile("/proc/self/uid_map", []byte(uidMap), 0644); err != nil {
		return fmt.Errorf("failed to write uid_map: %w", err)
	}

	// Write GID mapping
	gidMap := fmt.Sprintf("0 %d 1", gid)
	if err := os.WriteFile("/proc/self/gid_map", []byte(gidMap), 0644); err != nil {
		return fmt.Errorf("failed to write gid_map: %w", err)
	}

	// Disable setgroups to prevent privilege escalation
	if err := os.WriteFile("/proc/self/setgroups", []byte("deny"), 0644); err != nil {
		return fmt.Errorf("failed to write setgroups: %w", err)
	}

	return nil
}

// SetupUTSNamespace configures the UTS namespace (hostname)
func (nm *NamespaceManager) SetupUTSNamespace(hostname string) error {
	if err := unix.Sethostname([]byte(hostname)); err != nil {
		return fmt.Errorf("failed to set hostname: %w", err)
	}
	return nil
}

// GetNamespaceFD returns a file descriptor to a namespace
func GetNamespaceFD(pid int, namespaceType string) (int, error) {
	var nsType string
	switch namespaceType {
	case "pid":
		nsType = "pid_for_children"
	case "net":
		nsType = "net"
	case "mnt":
		nsType = "mnt"
	case "uts":
		nsType = "uts"
	case "ipc":
		nsType = "ipc"
	case "user":
		nsType = "user"
	default:
		return -1, fmt.Errorf("unknown namespace type: %s", namespaceType)
	}

	path := fmt.Sprintf("/proc/%d/ns/%s", pid, nsType)
	fd, err := unix.Open(path, unix.O_RDONLY, 0)
	if err != nil {
		return -1, fmt.Errorf("failed to open namespace %s: %w", path, err)
	}

	return fd, nil
}

// CloseNamespaceFD closes a namespace file descriptor
func CloseNamespaceFD(fd int) error {
	return unix.Close(fd)
}
