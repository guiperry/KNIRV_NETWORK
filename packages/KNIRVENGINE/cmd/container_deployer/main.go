// Command container_deployer builds, optionally runs, and optionally publishes
// the Alpine KNIRVENGINE image. It follows the server deployer's tag/push flow
// without embedding the engine source, which keeps the image context auditable.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	localImage  = "knirvengine-alpine"
	defaultRepo = "knirvcorp/knirvengine"
)

func main() {
	action := flag.String("action", "1", "1=build/deploy image")
	contextDir := flag.String("context", "", "KNIRVENGINE package directory (defaults to the nearest parent)")
	localTag := flag.String("tag", "latest", "local image tag")
	hubRepo := flag.String("hub-repo", defaultRepo, "Docker Hub repository (namespace/name)")
	push := flag.Bool("push", false, "tag and push to Docker Hub (requires docker login)")
	runLocal := flag.Bool("run", false, "start the eBPF/cgroup compose profile after building")
	flag.Parse()
	if *action != "1" {
		fatalf("invalid -action %q; only action 1 is supported", *action)
	}
	if _, err := exec.LookPath("docker"); err != nil {
		fatalf("docker is not on PATH")
	}

	root, err := findPackageRoot(*contextDir)
	if err != nil {
		fatalf("find KNIRVENGINE package: %v", err)
	}
	localRef := localImage + ":" + *localTag
	dockerfile := filepath.Join(root, "cmd", "os_builder", "alpine", "Dockerfile")
	if err := run("docker", "build", "--file", dockerfile, "--tag", localRef, root); err != nil {
		fatalf("build: %v", err)
	}

	if *push {
		hubRef := *hubRepo + ":alpine-latest"
		if err := run("docker", "tag", localRef, hubRef); err != nil {
			fatalf("tag: %v", err)
		}
		if err := run("docker", "push", hubRef); err != nil {
			fatalf("push %s: %v (run docker login first)", hubRef, err)
		}
		fmt.Printf("Pushed %s\n", hubRef)
	}
	if *runLocal {
		compose := filepath.Join(root, "cmd", "container_deployer", "compose.ebpf.yaml")
		if err := run("docker", "compose", "--file", compose, "up", "--detach"); err != nil {
			fatalf("start compose profile: %v", err)
		}
	}
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
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "cmd", "os_builder", "alpine", "Dockerfile")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no KNIRVENGINE package above %s", explicit)
		}
		dir = parent
	}
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "container_deployer: "+format+"\n", args...)
	os.Exit(1)
}
