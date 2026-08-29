package main

import (
	"context"
	"strings"
	"testing"
)

func TestToolInstallScriptUsesContainerArchiveAndAppData(t *testing.T) {
	script := toolInstallScript()
	for _, want := range []string{
		toolsArchiveURL,
		"XDG_CONFIG_HOME",
		"KNIRV-Engine/data/bin",
		"tar -xzf",
		"mv \"$tmp_dir/tools\" \"$tools_dir\"",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("tool install script does not contain %q", want)
		}
	}
}

func TestInstallerUsesNoBindMounts(t *testing.T) {
	var commands []string
	run := func(_ context.Context, name string, args ...string) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil
	}
	if err := install(context.Background(), run); err != nil {
		t.Fatalf("install() error = %v", err)
	}
	joined := strings.Join(commands, "\n")
	for _, want := range []string{
		"docker pull " + imageRef,
		"docker exec " + provisionerName,
		"docker commit --change",
		"docker run --detach --name " + runtimeName,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("installer commands do not contain %q", want)
		}
	}
	if strings.Contains(joined, "--volume") || strings.Contains(joined, "--mount") {
		t.Error("installer must not use host bind mounts")
	}
}
