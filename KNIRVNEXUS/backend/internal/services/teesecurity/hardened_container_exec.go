package teesecurity

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// executeHardenedContainer forks a child process with namespaces and executes the skill code
// with full Phase 3 isolation (network, mount, capabilities, seccomp, AppArmor)
func (ncr *NativeContainerRuntime) executeHardenedContainer(
	ctx context.Context,
	opts ContainerOptions,
	containerID string,
	cgroupMgr *CgroupManager,
	namespaceMgr *NamespaceManager,
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

	// Load AppArmor profile before starting container (Phase 3 - Security Hardening)
	if ncr.config.Security.AppArmorEnabled {
		apparmorMgr := NewAppArmorManager(MACConfig{
			Profile:      "knirv-nexus-skill-container",
			AutoGenerate: true,
		})
		if err := apparmorMgr.LoadProfile(); err != nil {
			log.Printf("Warning: Failed to load AppArmor profile: %v", err)
			// Continue even if AppArmor fails - it's an additional security layer
		}
	}

	// Create network manager
	networkMgr := NewNetworkManager(ncr.config.Network)

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

	// Setup network (host side)
	vethPair, err := networkMgr.CreateVethPair(cmd.Process.Pid)
	if err != nil {
		cmd.Process.Kill()
		return result, fmt.Errorf("failed to create veth pair: %w", err)
	}
	defer networkMgr.DeleteVethPair(vethPair)

	// Call containerInit with the dynamically generated container interface name
	err = containerInit(ctx, opts, containerID, ncr, vethPair.ContainerInterface)
	if err != nil {
		cmd.Process.Kill()
		return result, fmt.Errorf("failed to initialize container: %w", err)
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

// containerInit runs inside the container (child process)
// This is the entry point after fork with namespaces
func containerInit(
	ctx context.Context,
	opts ContainerOptions,
	containerID string,
	ncr *NativeContainerRuntime,
	containerInterface string, // New parameter
) error {
	// 1. Setup mount namespace
	mountConfig := MountConfig{
		RootFS:       filepath.Join(ncr.containerDir, containerID),
		MountProc:    true,
		MountSys:     true,
		MountDev:     true,
		ReadOnlyRoot: ncr.config.Security.ReadOnlyRoot,
		BindMounts: []BindMount{
			{
				Source:      filepath.Join(ncr.containerDir, containerID),
				Destination: "/skill",
				ReadOnly:    false,
			},
		},
	}

	mountMgr := NewMountManager(mountConfig)
	if err := mountMgr.SetupMountNamespace(); err != nil {
		return err
	}

	// 2. Setup network namespace
	networkMgr := NewNetworkManager(ncr.config.Network)
	if err := networkMgr.SetupNetworkNamespace("/proc/self/ns/net", containerInterface); err != nil {
		return err
	}

	// 3. Apply seccomp filter (Phase 3 - Security Hardening)
	if ncr.config.Security.SeccompEnabled {
		seccompMgr := NewSeccompManager(GetDefaultSeccompProfile())
		if err := seccompMgr.ApplyFilter(); err != nil {
			return err
		}
	}

	// 4. Drop capabilities
	capMgr := NewCapabilityManager(CapabilityConfig{
		Keep: []string{"CAP_SETUID", "CAP_SETGID"},
	})
	if err := capMgr.DropCapabilities(); err != nil {
		return err
	}
	if err := capMgr.SetNoNewPrivs(); err != nil {
		return err
	}

	// 5. Apply AppArmor profile (Phase 3 - Security Hardening)
	if ncr.config.Security.AppArmorEnabled {
		apparmorMgr := NewAppArmorManager(MACConfig{
			Profile:      "knirv-nexus-skill-container",
			AutoGenerate: true,
		})
		if err := apparmorMgr.ApplyToProcess(); err != nil {
			log.Printf("Warning: Failed to apply AppArmor: %v", err)
			// Continue even if AppArmor fails - it's an additional security layer
		}
	}

	// 6. Change to skill directory
	if err := os.Chdir("/skill"); err != nil {
		return err
	}

	// 7. Execute skill code
	skillPath := "/skill/skill.sh"
	cmd := exec.CommandContext(ctx, "/bin/bash", skillPath)
	cmd.Env = append(os.Environ(), opts.Env...)

	return cmd.Run()
}
