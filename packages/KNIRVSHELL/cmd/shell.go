package cmd

import (
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/ui/screens"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Launch the KNIRVSHELL supervisor TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		model := screens.NewSupervisorModel(theme, registryPath)
		program := tea.NewProgram(model, tea.WithAltScreen())
		_, err := program.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
	shellCmd.Flags().StringVar(&registryPath, "registry-file", "", "Path to the agent registry file")
}
