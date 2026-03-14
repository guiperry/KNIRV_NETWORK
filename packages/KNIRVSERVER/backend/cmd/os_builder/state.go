package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// BuildState represents the state of a build operation
type BuildState struct {
	BuildType   string `json:"build_type"`
	BuildID     string `json:"build_id"`
	AMIID       string `json:"ami_id,omitempty"`
	AMIName     string `json:"ami_name,omitempty"`
	Region      string `json:"region"`
	BuildDate   string `json:"build_date"`
	Status      string `json:"status"`
	OutputPath  string `json:"output_path"`
}

// DefaultStatePath returns the default path for build state files
func DefaultStatePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".knirvserver", "builds")
}

// SaveBuildState saves the build state to a file
func SaveBuildState(state *BuildState) error {
	stateDir := DefaultStatePath()
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return err
	}
	statePath := filepath.Join(stateDir, state.BuildID+".json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, data, 0644)
}

// LoadBuildState loads build state from a file
func LoadBuildState(buildID string) (*BuildState, error) {
	statePath := filepath.Join(DefaultStatePath(), buildID+".json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}
	var state BuildState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// FindLatestAMIBuild finds the most recent AMI build state
func FindLatestAMIBuild() (*BuildState, error) {
	stateDir := DefaultStatePath()
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return nil, err
	}

	var latest *BuildState
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		state, err := LoadBuildState(entry.Name()[:len(entry.Name())-5])
		if err != nil {
			continue
		}
		if state.BuildType == "ami" && state.Status == "completed" {
			if latest == nil || state.BuildDate > latest.BuildDate {
				latest = state
			}
		}
	}
	return latest, nil
}

// GetAMIIDFromLatestBuild returns the AMI ID from the latest successful AMI build
func GetAMIIDFromLatestBuild() (string, string, error) {
	state, err := FindLatestAMIBuild()
	if err != nil {
		return "", "", err
	}
	if state == nil {
		return "", "", nil
	}
	return state.AMIID, state.Region, nil
}
