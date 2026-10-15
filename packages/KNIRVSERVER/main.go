package main

import (
	"embed"

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
	launcher.Run(launcher.Assets{
		Frontend:      embeddedFiles,
		BackendBinary: backendBinary,
		ConfigFiles:   configFiles,
		Version:       Version,
		BuildTime:     BuildTime,
		GitCommit:     GitCommit,
	})
}
