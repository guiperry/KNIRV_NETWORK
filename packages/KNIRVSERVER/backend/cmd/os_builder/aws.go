package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// awsBuildConfig holds configuration for AWS AMI builds
type awsBuildConfig struct {
	Region          string
	AMIName         string
	InstanceType    string
	SubnetID        string
	SecurityGroupID string
	Description     string
	WorkDir         string
}

// validateAWSConfig validates the AWS configuration for AMI builds
func validateAWSConfig(config *awsBuildConfig) error {
	if config.Region == "" {
		return fmt.Errorf("aws_region is required")
	}

	if config.AMIName == "" {
		return fmt.Errorf("aws_ami_name is required")
	}

	if config.InstanceType == "" {
		config.InstanceType = "t3.medium"
	}

	if config.Description == "" {
		config.Description = "KNIRVSERVER Kali Linux - Native deployment ready"
	}

	// Check if AWS credentials are available
	_, err := exec.LookPath("aws")
	if err != nil {
		return fmt.Errorf("AWS CLI not found - please install AWS CLI for AMI builds")
	}

	return nil
}

// getDefaultBuildConfig returns the default configuration for AMI builds
func getDefaultBuildConfig(workDir string) *awsBuildConfig {
	return &awsBuildConfig{
		Region:          "us-east-1",
		AMIName:         fmt.Sprintf("knirvserver-kali-%s", time.Now().Format("2006-01-02")),
		InstanceType:    "t3.medium",
		SubnetID:        "",
		SecurityGroupID: "",
		Description:     "KNIRVSERVER Kali Linux - Native deployment ready",
		WorkDir:         workDir,
	}
}

// findAMIID searches for the AMI ID in Packer build output
func findAMIID(output string) (string, error) {
	// Look for pattern like "ami-12345678" in the output
	for i := 0; i < len(output); i++ {
		if i+3 <= len(output) && output[i:i+3] == "ami" {
			// Found potential AMI ID
			amiID := output[i:]
			// Extract until whitespace or comma
			end := 0
			for j := 0; j < len(amiID) && j < 25; j++ {
				if amiID[j] == ' ' || amiID[j] == '\n' || amiID[j] == ',' {
					end = j
					break
				}
				end = j + 1
			}
			if end > 0 {
				return amiID[:end], nil
			}
		}
	}
	return "", fmt.Errorf("AMI ID not found in output")
}

// amiBuildLog contains detailed logs for AMI builds
type amiBuildLog struct {
	Steps     []string
	Errors    []string
	StartTime time.Time
	EndTime   time.Time
}

// newAMIBuildLog creates a new AMI build log
func newAMIBuildLog() *amiBuildLog {
	return &amiBuildLog{
		Steps:     make([]string, 0),
		Errors:    make([]string, 0),
		StartTime: time.Now(),
	}
}

// AddStep adds a build step to the log
func (l *amiBuildLog) AddStep(step string) {
	l.Steps = append(l.Steps, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), step))
}

// AddError adds an error to the log
func (l *amiBuildLog) AddError(err string) {
	l.Errors = append(l.Errors, fmt.Sprintf("[%s] ERROR: %s", time.Now().Format("15:04:05"), err))
}

// GetDuration returns the build duration
func (l *amiBuildLog) GetDuration() time.Duration {
	return l.EndTime.Sub(l.StartTime)
}

// Save saves the build log to a file
func (l *amiBuildLog) Save(logPath string) error {
	l.EndTime = time.Now()

	content := "AMI Build Log\n"
	content += fmt.Sprintf("Start Time: %s\n", l.StartTime.Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("End Time: %s\n", l.EndTime.Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("Duration: %v\n\n", l.GetDuration())
	content += "Steps:\n"
	for _, step := range l.Steps {
		content += fmt.Sprintf("  %s\n", step)
	}
	if len(l.Errors) > 0 {
		content += "\nErrors:\n"
		for _, err := range l.Errors {
			content += fmt.Sprintf("  %s\n", err)
		}
	}

	return os.WriteFile(logPath, []byte(content), 0644)
}

// getAWSCredentialsStatus checks AWS credentials configuration
func getAWSCredentialsStatus() (map[string]bool, error) {
	status := make(map[string]bool)

	// Check for AWS_ACCESS_KEY_ID
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		status["AWS_ACCESS_KEY_ID"] = true
	}

	// Check for AWS_SECRET_ACCESS_KEY
	if os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		status["AWS_SECRET_ACCESS_KEY"] = true
	}

	// Check for AWS_SESSION_TOKEN (optional)
	if os.Getenv("AWS_SESSION_TOKEN") != "" {
		status["AWS_SESSION_TOKEN"] = true
	}

	// Check for shared credentials file
	homeDir, err := os.UserHomeDir()
	if err == nil {
		credentialsPath := filepath.Join(homeDir, ".aws", "credentials")
		if _, err := os.Stat(credentialsPath); err == nil {
			status["credentials_file"] = true
		}
	}

	return status, nil
}
