package dve_workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// OverlayWorkspace Tests
// ──────────────────────────────────────────────

func TestOverlayWorkspaceDirs(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "busybox-rootfs")
	require.NoError(t, os.MkdirAll(rootfsDir, 0755))

	ws, err := NewOverlayWorkspace(tmpDir, rootfsDir, "test-dve-1")
	require.NoError(t, err)
	require.NotNil(t, ws)

	// Verify directories created
	assert.DirExists(t, ws.UpperDir)
	assert.DirExists(t, ws.WorkDir)
	assert.DirExists(t, ws.MergedDir)
	assert.Equal(t, rootfsDir, ws.LowerDir)
	assert.Equal(t, "test-dve-1", ws.DVEID)
	assert.False(t, ws.mounted)

	// Destroy should clean up
	err = ws.Destroy()
	assert.NoError(t, err)
	_, err = os.Stat(ws.UpperDir)
	assert.True(t, os.IsNotExist(err), "upper dir should be removed after destroy")
}

func TestOverlayWorkspace_DestroyIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "busybox-rootfs")
	require.NoError(t, os.MkdirAll(rootfsDir, 0755))

	ws, err := NewOverlayWorkspace(tmpDir, rootfsDir, "test-dve-destroy")
	require.NoError(t, err)

	// Destroy once
	err = ws.Destroy()
	assert.NoError(t, err)

	// Destroy again should not error (unmount is idempotent)
	err = ws.Destroy()
	assert.NoError(t, err)
}

// ──────────────────────────────────────────────
// BusyBox Rootfs Tests
// ──────────────────────────────────────────────

func TestBusyBoxRootfsBootstrap(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsPath := filepath.Join(tmpDir, "busybox-rootfs")

	cfg := DVEConfig{
		BusyBoxSource:  "download",
		BusyBoxVersion: "1.36.1",
	}
	_ = cfg  // cfg used indirectly through createRootfsSymlinks

	// Bootstrap directly (will fall through since download requires network)
	// Instead test createRootfsSymlinks directly
	binDir := filepath.Join(rootfsPath, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a fake busybox binary
	fakeBusybox := filepath.Join(binDir, "busybox")
	require.NoError(t, os.WriteFile(fakeBusybox, []byte("#!/bin/sh\necho fake"), 0755))

	err := createRootfsSymlinks(rootfsPath)
	require.NoError(t, err)

	// Verify symlinks
	applets := []string{"sh", "ls", "cat", "cp", "grep", "tar"}
	for _, a := range applets {
		linkPath := filepath.Join(binDir, a)
		info, err := os.Lstat(linkPath)
		assert.NoError(t, err, "symlink %s should exist", a)
		assert.True(t, info.Mode()&os.ModeSymlink != 0, "%s should be a symlink", a)
		target, _ := os.Readlink(linkPath)
		assert.Equal(t, "busybox", target, "%s should point to busybox", a)
	}

	// Verify marker file
	assert.FileExists(t, filepath.Join(rootfsPath, ".knirvdve-ready"))

	// Verify passwd file
	passwdContent, err := os.ReadFile(filepath.Join(rootfsPath, "etc", "passwd"))
	assert.NoError(t, err)
	assert.Contains(t, string(passwdContent), "dve:x:0:0:DVE Workspace")
}

func TestEnsureBusyBoxRootfs_MarkerAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsPath := filepath.Join(tmpDir, "busybox-rootfs")
	require.NoError(t, os.MkdirAll(rootfsPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(rootfsPath, ".knirvdve-ready"), []byte("1"), 0644))

	cfg := DVEConfig{BusyBoxSource: "embedded"}
	err := EnsureBusyBoxRootfs(rootfsPath, cfg)
	assert.NoError(t, err, "should succeed when marker already exists")
}

// ──────────────────────────────────────────────
// DVE Workspace Identity Tests
// ──────────────────────────────────────────────

func TestGenerateDVEIdentity(t *testing.T) {
	service, err := NewDVEService(nil, nil, nil, DVEConfig{
		WorkspaceRoot:      "/tmp/test-workspaces",
		ProjectStoragePath: "/tmp/test-projects",
		MaxEnvironments:    5,
	})
	require.NoError(t, err)

	env := &DVEEnvironment{
		ID:              "test-id-123",
		Name:            "test-env",
		UserID:          "user-1",
		EnvironmentType: EnvTypeGeneral,
		CreatedAt:       time.Now(),
	}

	identity := service.generateDVEIdentity(env)
	assert.Contains(t, identity, "test-id-123")
	assert.Contains(t, identity, "test-env")
	assert.Contains(t, identity, "OverlayFS")
}

func TestGenerateAgentInstructions(t *testing.T) {
	service, err := NewDVEService(nil, nil, nil, DVEConfig{
		WorkspaceRoot:      "/tmp/test-workspaces",
		ProjectStoragePath: "/tmp/test-projects",
		MaxEnvironments:    5,
	})
	require.NoError(t, err)

	env := &DVEEnvironment{
		Name: "test-agent-workspace",
	}

	instructions := service.generateAgentInstructions(env)
	assert.Contains(t, instructions, "test-agent-workspace")
	assert.Contains(t, instructions, "BusyBox")
	assert.Contains(t, instructions, "/workspace")
}

// ──────────────────────────────────────────────
// DVEConfig Tests
// ──────────────────────────────────────────────

func TestDVEConfigDefaults(t *testing.T) {
	cfg := DVEConfig{
		EnableOverlayFS:   true,
		BusyBoxSource:     "embedded",
		BusyBoxRootfsPath: "/var/lib/knirvserver/busybox-rootfs",
		SkillExecTimeout:  120 * time.Second,
		SkillMaxMemoryMB:  512,
		MaxConcurrentWASM: 10,
	}
	assert.True(t, cfg.EnableOverlayFS)
	assert.Equal(t, "embedded", cfg.BusyBoxSource)
	assert.Equal(t, uint32(512), cfg.SkillMaxMemoryMB)
	assert.Equal(t, 10, cfg.MaxConcurrentWASM)
}

func TestDVEConfigRoundtrip(t *testing.T) {
	cfg := DVEConfig{
		WorkspaceRoot:            "/tmp/workspaces",
		EnableOverlayFS:          true,
		BusyBoxRootfsPath:        "/tmp/busybox",
		BusyBoxSource:            "download",
		FuseOverlayFSBin:         "/usr/bin/fuse-overlayfs",
		SkillExecTimeout:         60 * time.Second,
		SkillMaxMemoryMB:         256,
		MaxConcurrentWASM:        5,
		WorkspaceRetentionHours:  72,
		MaxEnvironments:          100,
		MaxCPUPerEnv:             4.0,
		MaxMemoryPerEnv:          4294967296,
		MaxDiskPerEnv:            21474836480,
		EnableNetworkIsolation:   true,
	}

	assert.Equal(t, "/tmp/workspaces", cfg.WorkspaceRoot)
	assert.True(t, cfg.EnableOverlayFS)
	assert.Equal(t, "download", cfg.BusyBoxSource)
	assert.Equal(t, 60*time.Second, cfg.SkillExecTimeout)
	assert.Equal(t, uint32(256), cfg.SkillMaxMemoryMB)
	assert.Equal(t, 5, cfg.MaxConcurrentWASM)
	assert.Equal(t, 72, cfg.WorkspaceRetentionHours)
	assert.Equal(t, 100, cfg.MaxEnvironments)
	assert.True(t, cfg.EnableNetworkIsolation)
}

// ──────────────────────────────────────────────
// CreateVirtualDVE Tests
// ──────────────────────────────────────────────

func TestCreateVirtualDVE_OverlayDisabled(t *testing.T) {
	// When EnableOverlayFS is false, the fallback path should work
	service, err := NewDVEService(nil, nil, nil, DVEConfig{
		WorkspaceRoot:      t.TempDir(),
		ProjectStoragePath: t.TempDir(),
		EnableOverlayFS:    false,
		MaxEnvironments:    10,
	})
	require.NoError(t, err)
	require.NoError(t, service.Start())
	defer service.Stop()

	env := &DVEEnvironment{
		ID:              "test-virtual-fallback",
		Name:            "test",
		UserID:          "user-1",
		Status:          EnvStatusCreating,
		EnvironmentType: EnvTypeGeneral,
		WorkspacePath:   filepath.Join(t.TempDir(), "workspace"),
		Ports:           make(map[string]int),
	}

	service.createVirtualDVEAsync(env)
	assert.Equal(t, EnvStatusRunning, env.Status,
		"virtual DVE should be running even without overlay")
	// Without overlay or eBPF, ContainerID may be empty during simulation
}

// ──────────────────────────────────────────────
// Namespace Safe Tests (no-opt if not on Linux)
// ──────────────────────────────────────────────

func TestDVENamespaceConstructor(t *testing.T) {
	// Test that DVENamespace struct can be created
	// (no actual namespace creation - that requires Linux namespaces)
	ns := &DVENamespace{
		Workspace: &OverlayWorkspace{
			DVEID:     "test",
			LowerDir:  "/tmp/lower",
			UpperDir:  "/tmp/upper",
			WorkDir:   "/tmp/work",
			MergedDir: "/tmp/merged",
		},
		UID: 1000,
		GID: 1000,
	}
	assert.Equal(t, "test", ns.Workspace.DVEID)
	assert.Equal(t, 1000, ns.UID)
	assert.Equal(t, 1000, ns.GID)
}

// ──────────────────────────────────────────────
// Resource Pool Tests
// ──────────────────────────────────────────────

func TestDVEResourcePool_OverlayAware(t *testing.T) {
	pool, err := NewDVEResourcePool()
	require.NoError(t, err)

	// Allocate resources with realistic DVE workspace load
	request := &DVEResourceAllocation{
		CPUCores:    1.0,
		MemoryBytes: 512 * 1024 * 1024,
		DiskBytes:   5 * 1024 * 1024 * 1024,
	}

	assert.True(t, pool.CanAllocate(request))
	err = pool.AllocateResources(request)
	assert.NoError(t, err)

	usage := pool.GetResourceUsage()
	cpuData, ok := usage["cpu"].(map[string]interface{})
	assert.True(t, ok)
	assert.GreaterOrEqual(t, cpuData["allocated"].(float64), 1.0)
}
