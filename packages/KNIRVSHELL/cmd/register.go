package cmd

import (
	"fmt"
	"strings"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/registry"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/runner"
	"github.com/spf13/cobra"
)

var (
	registryPath string

	registerCmd = &cobra.Command{
		Use:   "register [name]",
		Short: "Register or update an agent in the KNIRVSHELL registry",
		Args:  cobra.ExactArgs(1),
		RunE:  runRegisterAgent,
	}
)

func init() {
	rootCmd.AddCommand(registerCmd)

	registerCmd.Flags().StringVar(&registryPath, "registry-file", "", "Path to the agent registry file")
	registerCmd.Flags().String("description", "", "Agent description")
	registerCmd.Flags().String("type", string(runner.AgentTypeLocal), "Runner type: local, docker, ssh")
	registerCmd.Flags().String("path", "", "Local executable path")
	registerCmd.Flags().StringSlice("arg", nil, "Agent arguments")
	registerCmd.Flags().StringSlice("env", nil, "Environment variables (KEY=VALUE)")
	registerCmd.Flags().String("image", "", "Docker image")
	registerCmd.Flags().StringSlice("mount", nil, "Docker mounts in host=container form")
	registerCmd.Flags().String("host", "", "SSH host")
	registerCmd.Flags().Int("port", 22, "SSH port")
	registerCmd.Flags().String("user", "", "SSH user")
	registerCmd.Flags().String("key-path", "", "SSH private key path")
	registerCmd.Flags().String("remote-cmd", "", "SSH remote command")
	registerCmd.Flags().StringSlice("tag", nil, "Registry tags")
}

func runRegisterAgent(cmd *cobra.Command, args []string) error {
	typ, _ := cmd.Flags().GetString("type")
	mounts, err := parseMounts(cmd)
	if err != nil {
		return err
	}

	agent := registry.AgentRecord{
		Name:        args[0],
		Description: mustStringFlag(cmd, "description"),
		Type:        runner.AgentType(strings.ToLower(typ)),
		Path:        mustStringFlag(cmd, "path"),
		Args:        mustStringSliceFlag(cmd, "arg"),
		Env:         mustStringSliceFlag(cmd, "env"),
		Image:       mustStringFlag(cmd, "image"),
		Mounts:      mounts,
		Host:        mustStringFlag(cmd, "host"),
		Port:        mustIntFlag(cmd, "port"),
		User:        mustStringFlag(cmd, "user"),
		KeyPath:     mustStringFlag(cmd, "key-path"),
		RemoteCmd:   mustStringFlag(cmd, "remote-cmd"),
		Tags:        mustStringSliceFlag(cmd, "tag"),
	}

	if err := validateAgentRecord(agent); err != nil {
		return err
	}

	store := registry.NewStore(registryPath)
	if err := store.Upsert(agent); err != nil {
		return err
	}

	fmt.Printf("Registered agent %q in %s\n", agent.Name, store.Path())
	return nil
}

func validateAgentRecord(agent registry.AgentRecord) error {
	switch agent.Type {
	case runner.AgentTypeLocal:
		if agent.Path == "" {
			return fmt.Errorf("local agents require --path")
		}
	case runner.AgentTypeDocker:
		if agent.Image == "" {
			return fmt.Errorf("docker agents require --image")
		}
	case runner.AgentTypeSSH:
		if agent.Host == "" || agent.User == "" || agent.KeyPath == "" || agent.RemoteCmd == "" {
			return fmt.Errorf("ssh agents require --host, --user, --key-path, and --remote-cmd")
		}
	default:
		return fmt.Errorf("unsupported agent type %q", agent.Type)
	}

	return nil
}

func parseMounts(cmd *cobra.Command) (map[string]string, error) {
	values, _ := cmd.Flags().GetStringSlice("mount")
	if len(values) == 0 {
		return nil, nil
	}

	mounts := make(map[string]string, len(values))
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid mount %q; expected host=container", value)
		}
		mounts[parts[0]] = parts[1]
	}

	return mounts, nil
}

func mustStringFlag(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return value
}

func mustStringSliceFlag(cmd *cobra.Command, name string) []string {
	value, _ := cmd.Flags().GetStringSlice(name)
	return value
}

func mustIntFlag(cmd *cobra.Command, name string) int {
	value, _ := cmd.Flags().GetInt(name)
	return value
}
