// Command installer prepares and starts the containerized KNIRVENGINE runtime.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
)

const (
	imageRef         = "knirvcorp/knirvengine:alpine-latest"
	preparedImageRef = "knirvengine-alpine:runtime"
	provisionerName  = "knirvengine-alpine-provisioner"
	runtimeName      = "knirvengine-alpine"
	toolsArchiveURL  = "https://releases.knirv.com/engine/tools/tools.tar.gz"
)

type commandRunner func(context.Context, string, ...string) error

func main() {
	if err := install(context.Background(), runHostCommand); err != nil {
		fatalf("KNIRVENGINE installation failed: %v", err)
	}
	fmt.Printf("KNIRVENGINE is running in Docker container %q\n", runtimeName)
}

// install performs the complete container-only setup. The tools archive is
// downloaded and expanded inside the Alpine container, then docker commit
// persists that Alpine filesystem as the image that runs KNIRVENGINE.
func install(ctx context.Context, run commandRunner) error {
	if err := ensureDocker(ctx, run); err != nil {
		return err
	}
	if err := run(ctx, "docker", "pull", imageRef); err != nil {
		return fmt.Errorf("pull %s: %w", imageRef, err)
	}

	_ = run(ctx, "docker", "rm", "--force", provisionerName)
	if err := run(ctx, "docker", "run", "--detach", "--name", provisionerName,
		"--entrypoint", "/bin/sh", imageRef, "-c", "while :; do sleep 3600; done"); err != nil {
		return fmt.Errorf("start Alpine tool provisioner: %w", err)
	}
	provisionerStarted := true
	defer func() {
		if provisionerStarted {
			_ = run(context.Background(), "docker", "rm", "--force", provisionerName)
		}
	}()

	if err := run(ctx, "docker", "exec", provisionerName, "/bin/sh", "-ec", toolInstallScript()); err != nil {
		return fmt.Errorf("install tools inside Alpine container: %w", err)
	}
	// The provisioner overrides the image entrypoint with a shell. Restore the
	// public image's precompiled-engine entrypoint in the committed runtime.
	if err := run(ctx, "docker", "commit",
		"--change", `ENTRYPOINT ["/app/knirv-engine"]`,
		"--change", `CMD ["--production", "--browser"]`,
		provisionerName, preparedImageRef); err != nil {
		return fmt.Errorf("persist prepared Alpine filesystem: %w", err)
	}
	if err := run(ctx, "docker", "rm", "--force", provisionerName); err != nil {
		return fmt.Errorf("remove tool provisioner: %w", err)
	}
	provisionerStarted = false

	_ = run(ctx, "docker", "rm", "--force", runtimeName)
	if err := run(ctx, "docker", "run", "--detach", "--name", runtimeName,
		"--publish", "8080:8080", "--publish", "8081:8081",
		"--privileged", "--pid", "host", "--cgroupns", "host",
		"--security-opt", "seccomp=unconfined", preparedImageRef); err != nil {
		return fmt.Errorf("launch prepared KNIRVENGINE: %w", err)
	}
	return nil
}

// toolInstallScript runs only inside the pulled Alpine image. It does not use
// a bind mount: after docker commit, this path is part of the image's own
// Alpine filesystem and is exactly the app-data directory resolved by Go.
func toolInstallScript() string {
	return strings.Join([]string{
		"tools_dir=\"${XDG_CONFIG_HOME:-$HOME/.config}/KNIRV-Engine/data/bin\"",
		"tmp_dir=$(mktemp -d)",
		"trap 'rm -rf \"$tmp_dir\"' EXIT",
		"curl -fsSL \"" + toolsArchiveURL + "\" -o \"$tmp_dir/tools.tar.gz\"",
		"tar -xzf \"$tmp_dir/tools.tar.gz\" -C \"$tmp_dir\"",
		"test -d \"$tmp_dir/tools\"",
		"rm -rf \"$tools_dir\"",
		"mkdir -p \"$(dirname \"$tools_dir\")\"",
		"mv \"$tmp_dir/tools\" \"$tools_dir\"",
		"find \"$tools_dir\" -type f -exec chmod a+rx {} +",
	}, "\n")
}

func fatalf(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...); os.Exit(1) }
