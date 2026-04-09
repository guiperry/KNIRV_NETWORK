package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/runner"
	"gopkg.in/yaml.v3"
)

type AgentRecord struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Type        runner.AgentType  `yaml:"type"`
	Path        string            `yaml:"path,omitempty"`
	Args        []string          `yaml:"args,omitempty"`
	Env         []string          `yaml:"env,omitempty"`
	Image       string            `yaml:"image,omitempty"`
	Mounts      map[string]string `yaml:"mounts,omitempty"`
	Host        string            `yaml:"host,omitempty"`
	Port        int               `yaml:"port,omitempty"`
	User        string            `yaml:"user,omitempty"`
	KeyPath     string            `yaml:"key_path,omitempty"`
	RemoteCmd   string            `yaml:"remote_cmd,omitempty"`
	Tags        []string          `yaml:"tags,omitempty"`
	CreatedAt   time.Time         `yaml:"created_at,omitempty"`
	UpdatedAt   time.Time         `yaml:"updated_at,omitempty"`
}

type File struct {
	Agents []AgentRecord `yaml:"agents"`
}

type Store struct {
	path string
}

func DefaultPath() string {
	if envPath := os.Getenv("KNIRV_AGENT_REGISTRY"); envPath != "" {
		return envPath
	}

	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, "config", "agents.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "config/agents.yaml"
	}
	return filepath.Join(home, ".knirv", "agents.yaml")
}

func NewStore(path string) *Store {
	if path == "" {
		path = DefaultPath()
	}
	return &Store{path: path}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() (*File, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &File{}, nil
		}
		return nil, fmt.Errorf("read registry: %w", err)
	}

	var file File
	if err := yaml.Unmarshal(body, &file); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	return &file, nil
}

func (s *Store) List() ([]AgentRecord, error) {
	file, err := s.Load()
	if err != nil {
		return nil, err
	}
	return file.Agents, nil
}

func (s *Store) Get(name string) (*AgentRecord, error) {
	file, err := s.Load()
	if err != nil {
		return nil, err
	}

	for _, agent := range file.Agents {
		if agent.Name == name {
			copy := agent
			return &copy, nil
		}
	}

	return nil, fmt.Errorf("agent %q not found in %s", name, s.path)
}

func (s *Store) Upsert(agent AgentRecord) error {
	if agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}

	file, err := s.Load()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	agent.UpdatedAt = now

	replaced := false
	for i, existing := range file.Agents {
		if existing.Name == agent.Name {
			agent.CreatedAt = existing.CreatedAt
			file.Agents[i] = agent
			replaced = true
			break
		}
	}
	if !replaced {
		file.Agents = append(file.Agents, agent)
	}

	return s.write(file)
}

func (s *Store) write(file *File) error {
	body, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("create registry dir: %w", err)
	}

	if err := os.WriteFile(s.path, body, 0644); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	return nil
}

func (a AgentRecord) RunnerConfig() runner.AgentConfig {
	return runner.AgentConfig{
		Name:      a.Name,
		Type:      a.Type,
		Path:      a.Path,
		Args:      a.Args,
		Env:       a.Env,
		Image:     a.Image,
		Mounts:    a.Mounts,
		Host:      a.Host,
		Port:      a.Port,
		User:      a.User,
		KeyPath:   a.KeyPath,
		RemoteCmd: a.RemoteCmd,
	}
}
