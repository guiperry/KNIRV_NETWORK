package api

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// InnerPid is the PID of bwrap's inner process — the one that actually
// performed --unshare-all and became PID 1 of the new pid namespace.
// SandboxSession.Pid is bwrap's outer setup process and stays in the host's
// namespaces, so it cannot be used as an nsenter target. InnerPid is resolved
// once at Start() time via ps --ppid <s.Pid>.
//
// Phase 0 (executed live, confirmed working): joining the inner PID via
// `nsenter -t <InnerPid> -m -p -i -u -n -C -- <cmd>` lands a new process
// inside every namespace the sandbox owns (mnt/pid/ipc/uts/cgroup). The join
// runs as root (matching KNIRVSERVER's own precedent for cgroup control) —
// unprivileged nsenter categorically fails on mnt/pid/ipc/uts/cgroup on the
// tested kernel/util-linux combination.
func (s *SandboxSession) resolveInnerPid() error {
	if s.Pid == 0 {
		return fmt.Errorf("session has no outer PID yet")
	}
	cmd := exec.Command("ps", "--ppid", strconv.Itoa(s.Pid), "-o", "pid=", "--no-headers")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to resolve inner bwrap PID: %v", err)
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return fmt.Errorf("no child process found for outer bwrap PID %d", s.Pid)
	}
	innerPid, err := strconv.Atoi(fields[0])
	if err != nil {
		return fmt.Errorf("failed to parse inner PID %q: %v", fields[0], err)
	}
	s.InnerPid = innerPid
	return nil
}

// resolveTargetPid returns the sandboxed program's host PID. The inner bwrap
// process is PID 1 inside the new PID namespace; its first child is the target
// command that instrumentation tools should attach to.
func (s *SandboxSession) resolveTargetPid() (int, error) {
	if s.InnerPid == 0 {
		if err := s.resolveInnerPid(); err != nil {
			return 0, err
		}
	}
	cmd := exec.Command("ps", "--ppid", strconv.Itoa(s.InnerPid), "-o", "pid=", "--no-headers")
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to resolve sandbox target PID: %w", err)
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return 0, fmt.Errorf("no target child found for inner bwrap PID %d", s.InnerPid)
	}
	return strconv.Atoi(fields[0])
}

// spawnJoined spawns a process inside the sandbox's namespaces. It uses
// `nsenter -t <InnerPid> -m -p -i -u -n -C -- <name> <args...>` to join all
// of the sandbox's namespaces (mnt, pid, ipc, uts, net, cgroup) and then
// executes the requested command inside them.
//
// This is the confirmed working shape from Phase 0's live spike. The join
// runs as root (via the same mechanism realCommandRunner uses) because
// unprivileged setns() cannot reach namespaces beyond the user namespace.
func (s *SandboxSession) spawnJoined(name string, args ...string) (*exec.Cmd, error) {
	if s.InnerPid == 0 {
		if err := s.resolveInnerPid(); err != nil {
			return nil, fmt.Errorf("cannot join sandbox namespaces: %v", err)
		}
	}

	nsenterArgs := append(
		[]string{"-t", strconv.Itoa(s.InnerPid), "-m", "-p", "-i", "-u", "-n", "-C", "--", name},
		args...,
	)
	return s.spawnAsRoot("nsenter", nsenterArgs...)
}

// startToolProcess starts a tool with direct access to its pipes. Session
// spawn() owns pipes for namespace logs, while streaming/RPC lanes need to
// consume stdout/stderr themselves for their own WebSocket protocols.
func (s *SandboxSession) startToolProcess(name string, joined bool, args ...string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	command, commandArgs := name, args
	if joined {
		if s.InnerPid == 0 {
			if err := s.resolveInnerPid(); err != nil {
				return nil, nil, nil, nil, err
			}
		}
		commandArgs = append([]string{"-t", strconv.Itoa(s.InnerPid), "-m", "-p", "-i", "-u", "-n", "-C", "--", name}, args...)
		command = "nsenter"
		if !isRoot() {
			commandArgs = append([]string{"nsenter"}, commandArgs...)
			command = "sudo"
		}
	}
	cmd := exec.CommandContext(s.ctx, command, commandArgs...)
	cmd.Env = s.toolEnv(name)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, nil, err
	}
	return cmd, stdin, stdout, stderr, nil
}

// spawnAsRoot spawns a process with root escalation via sudo when the engine
// is not already running as root. Used by namespace joins and other tool
// operations that need CAP_SYS_ADMIN.
func (s *SandboxSession) spawnAsRoot(name string, args ...string) (*exec.Cmd, error) {
	if isRoot() {
		return s.spawn(name, args...)
	}
	rooted := append([]string{name}, args...)
	return s.spawn("sudo", rooted...)
}

// isRoot reports whether the current process is running as root.
func isRoot() bool {
	return geteuid() == 0
}

// geteuid returns the effective user ID of the current process.
func geteuid() int {
	return geteuidImpl()
}
