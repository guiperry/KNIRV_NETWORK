package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func resolvePassword(cmd *cobra.Command, flagName, envName, prompt string) (string, error) {
	password, err := cmd.Flags().GetString(flagName)
	if err != nil {
		return "", err
	}
	if password != "" {
		return password, nil
	}

	if envName != "" {
		if envPassword := strings.TrimSpace(os.Getenv(envName)); envPassword != "" {
			return envPassword, nil
		}
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", nil
	}

	fmt.Fprint(os.Stderr, prompt)
	secret, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(secret)), nil
}
