package main

import (
	"embed"
	"fmt"
	"os"

	"knirv-server/internal/cli"
	"knirv-server/internal/launcher"
)

//go:embed all:frontend/out/*
var embeddedFiles embed.FS

//go:embed bin/backend_server
var backendBinary []byte

//go:embed all:config/*
var configFiles embed.FS

// Version information is set by build flags.
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// The `enroll` subcommand performs a DEC-2 boot-node enrollment without
	// starting the full server. All other invocations run the launcher as before.
	if len(os.Args) > 1 && os.Args[1] == "enroll" {
		if err := cli.RunEnroll(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "enroll:", err)
			os.Exit(1)
		}
		return
	}

	launcher.Run(launcher.Assets{
		Frontend:      embeddedFiles,
		BackendBinary: backendBinary,
		ConfigFiles:   configFiles,
		Version:       Version,
		BuildTime:     BuildTime,
		GitCommit:     GitCommit,
	})
}
