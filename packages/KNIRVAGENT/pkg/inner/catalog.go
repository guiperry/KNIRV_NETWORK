package inner

import (
	"os"
	"os/exec"
)

type Tool struct {
	Name        string
	Binary      string
	DefaultArgs []string
	InstallCmd  []string
}

func (t *Tool) IsInstalled() bool {
	_, err := exec.LookPath(t.Binary)
	return err == nil
}

func (t *Tool) Install(workdir string) error {
	if len(t.InstallCmd) == 0 {
		return nil
	}
	cmd := exec.Command(t.InstallCmd[0], t.InstallCmd[1:]...)
	cmd.Dir = workdir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func (t *Tool) BuildCmd(workdir string, extra []string) *exec.Cmd {
	cmd := exec.Command(t.Binary, t.DefaultArgs...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), extra...)
	return cmd
}

type ToolCatalog struct {
	tools map[string]*Tool
}

func (c *ToolCatalog) Lookup(name string) (*Tool, bool) {
	t, ok := c.tools[name]
	return t, ok
}

func (c *ToolCatalog) All() []Tool {
	out := make([]Tool, 0, len(c.tools))
	for _, t := range c.tools {
		out = append(out, *t)
	}
	return out
}

func DefaultCatalog() *ToolCatalog {
	return &ToolCatalog{tools: map[string]*Tool{
		"shell":      {Name: "shell", Binary: "bash", DefaultArgs: []string{"-i"}, InstallCmd: []string{}},
		"claude":     {Name: "claude", Binary: "claude", DefaultArgs: []string{"--no-auto-update"}, InstallCmd: []string{"npm", "install", "-g", "@anthropic-ai/claude-code"}},
		"aider":      {Name: "aider", Binary: "aider", InstallCmd: []string{"pip", "install", "aider-chat"}},
		"codex":      {Name: "codex", Binary: "codex", InstallCmd: []string{"npm", "install", "-g", "@openai/codex"}},
		"gh-copilot": {Name: "gh-copilot", Binary: "gh", DefaultArgs: []string{"copilot"}, InstallCmd: []string{"gh", "extension", "install", "github/gh-copilot"}},
	}}
}
