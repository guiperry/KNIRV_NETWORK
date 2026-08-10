package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/knirv/network-monitor/config"
)

func main() {
	tmpl := flag.String("template", "", "path to alertmanager template YAML")
	out := flag.String("output", "/etc/alertmanager/alertmanager.yml", "output path for resolved config")
	flag.Parse()

	if *tmpl == "" {
		fmt.Fprintln(os.Stderr, "usage: alertmanager-config --template=<path> [--output=<path>]")
		os.Exit(1)
	}

	if err := config.WriteResolvedConfig(*tmpl, *out); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("alertmanager config written to %s\n", *out)
}
