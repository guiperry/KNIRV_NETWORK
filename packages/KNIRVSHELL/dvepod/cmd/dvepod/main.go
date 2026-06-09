package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/dvepod/internal/agent"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/dvepod/internal/shell"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/dvepod/internal/tee"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/dvepod/internal/tools"
)

func main() {
	ctx, err := tee.NewContext()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dvepod: failed to initialize TEE context: %v\n", err)
		os.Exit(1)
	}

	bb := tools.NewBusyBox(ctx)
	ag := agent.New(ctx)
	sh := shell.New(ctx, bb, ag)

	err = sh.Boot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dvepod: boot failed: %v\n", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Fprint(os.Stdout, sh.Prompt())
		if !scanner.Scan() {
			break
		}
		sh.Exec(scanner.Text())
	}

	if scanErr := scanner.Err(); scanErr != nil {
		fmt.Fprintf(os.Stderr, "dvepod: read error: %v\n", scanErr)
		os.Exit(1)
	}
}
