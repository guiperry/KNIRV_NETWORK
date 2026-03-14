package teesecurity

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// executeInContainer forks a child process with namespaces and executes the skill code
func (ncr *NativeContainerRuntime) executeInContainer(
	ctx context.Context,
	opts ContainerOptions,
	containerID string,
	cgroupMgr *CgroupManager,
) (*ContainerResult, error) {
	result := &ContainerResult{ContainerID: containerID}

	// Prepare container filesystem
	sandboxPath := filepath.Join(ncr.containerDir, containerID)
	if err := os.MkdirAll(sandboxPath, 0700); err != nil {
		return result, fmt.Errorf("failed to create sandbox: %w", err)
	}
	defer os.RemoveAll(sandboxPath)

	// Write skill code
	skillPath := filepath.Join(sandboxPath, "skill.sh")
	if err := os.WriteFile(skillPath, []byte(opts.SkillCode), 0700); err != nil {
		return result, fmt.Errorf("failed to write skill code: %w", err)
	}

	// Build command to execute
	cmd := exec.CommandContext(ctx, "/bin/bash", skillPath)
	cmd.Dir = sandboxPath
	cmd.Env = append(os.Environ(), opts.Env...)

	// Set namespace clone flags
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: unix.CLONE_NEWPID | unix.CLONE_NEWNET | unix.CLONE_NEWNS |
			unix.CLONE_NEWUTS | unix.CLONE_NEWIPC | unix.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
		GidMappingsEnableSetgroups: false,
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return result, fmt.Errorf("failed to start container: %w", err)
	}

	// Add to cgroup
	if err := cgroupMgr.AddProcess(cmd.Process.Pid); err != nil {
		cmd.Process.Kill()
		return result, fmt.Errorf("failed to add to cgroup: %w", err)
	}

	// Wait for completion
	startTime := time.Now()
	if err := cmd.Wait(); err != nil {
		result.ExitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
	} else {
		result.ExitCode = 0
	}

	result.ExecutionTime = int64(time.Since(startTime).Milliseconds())

	// Get cgroup stats
	stats, _ := cgroupMgr.GetStats()
	result.ResourceUsage = &ResourceUsage{
		MemoryPeak:  stats.MemoryPeak,
		MemoryUsage: stats.MemoryUsage,
		PIDsUsed:    stats.PIDsCurrent,
	}

	return result, nil
}
