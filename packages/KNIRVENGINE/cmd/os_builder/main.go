// Command os_builder creates the KNIRVENGINE Alpine runtime image.
//
// It is the KNIRVENGINE-specific counterpart to the server os_builder. A
// container cannot replace the host kernel, so eBPF and cgroup support are
// expressed both in the image and the documented runtime profile.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const defaultImage = "knirvengine-alpine"

func main() {
	action := flag.String("action", "0", "0=build Alpine image, 1=build and export image archive, validate=print runtime requirements")
	image := flag.String("image", defaultImage, "local Docker image name")
	tag := flag.String("tag", "latest", "local Docker image tag")
	contextDir := flag.String("context", "", "KNIRVENGINE package directory (defaults to the nearest parent)")
	output := flag.String("output", "", "archive output path for action 1 (default: artifacts/knirvengine-alpine.tar)")
	flag.Parse()

	if *action == "validate" {
		printRuntimeRequirements()
		return
	}
	if *action != "0" && *action != "1" {
		fatalf("invalid -action %q; use 0, 1, or validate", *action)
	}
	if err := requireDocker(); err != nil {
		fatalf("Docker prerequisite: %v", err)
	}

	root, err := findPackageRoot(*contextDir)
	if err != nil {
		fatalf("find KNIRVENGINE package: %v", err)
	}
	dockerfile := filepath.Join(root, "cmd", "os_builder", "alpine", "Dockerfile")
	ref := *image + ":" + *tag
	if err := run("docker", "build", "--file", dockerfile, "--tag", ref, root); err != nil {
		fatalf("build %s: %v", ref, err)
	}
	fmt.Printf("Built %s\n", ref)

	if *action == "1" {
		archive := *output
		if archive == "" {
			archive = filepath.Join(root, "artifacts", "knirvengine-alpine.tar")
		}
		if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
			fatalf("create artifact directory: %v", err)
		}
		if err := run("docker", "save", "--output", archive, ref); err != nil {
			fatalf("export %s: %v", ref, err)
		}
		fmt.Printf("Exported %s\n", archive)
	}
	printRuntimeRequirements()
}

func findPackageRoot(explicit string) (string, error) {
	dir := explicit
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		if exists(filepath.Join(dir, "go.mod")) && exists(filepath.Join(dir, "cmd", "os_builder", "alpine", "Dockerfile")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no KNIRVENGINE package above %s", explicit)
		}
		dir = parent
	}
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }

func requireDocker() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return errors.New("docker is not on PATH")
	}
	return nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}

func printRuntimeRequirements() {
	fmt.Println("For eBPF tracing and cgroup namespace control, run with docker compose -f cmd/container_deployer/compose.ebpf.yaml up.")
	fmt.Println("That profile is privileged by design; it grants host-level kernel visibility. Do not use it for untrusted workloads.")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "os_builder: "+format+"\n", args...)
	os.Exit(1)
}
