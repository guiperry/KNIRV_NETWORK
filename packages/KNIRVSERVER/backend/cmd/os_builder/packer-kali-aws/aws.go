// Package aws provides AWS AMI build functionality for the os_builder.
package aws

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BuildConfig holds configuration for AWS AMI builds.
type BuildConfig struct {
	Region          string
	AMIName         string
	InstanceType    string
	SubnetID        string
	SecurityGroupID string
	Description     string
	WorkDir         string
}

// Result contains the result of an AMI build operation.
type Result struct {
	AMIID      string
	AMIName    string
	Region     string
	BuildTime  time.Duration
}

// BuildAWSAMI builds a Kali Linux AMI for AWS using Packer.
func BuildAWSAMI(config *BuildConfig) (*Result, error) {
	log.Printf("Starting AWS AMI build in region: %s", config.Region)

	startTime := time.Now()

	// Initialize Packer with Amazon plugin
	log.Println("Initializing Packer with Amazon plugin...")
	initCmd := exec.Command("packer", "init", ".")
	initCmd.Dir = config.WorkDir
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr

	if err := initCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to initialize Packer: %w", err)
	}

	// Build the AMI
	log.Println("Building AWS AMI with Packer...")

	args := []string{
		"build",
		"-force",
		"-var", fmt.Sprintf("aws_region=%s", config.Region),
		"-var", fmt.Sprintf("aws_ami_name=%s", config.AMIName),
		"-var", fmt.Sprintf("aws_instance_type=%s", config.InstanceType),
		"-var", fmt.Sprintf("aws_ami_description=%s", config.Description),
	}

	if config.SubnetID != "" {
		args = append(args, "-var", fmt.Sprintf("aws_subnet_id=%s", config.SubnetID))
	}

	if config.SecurityGroupID != "" {
		args = append(args, "-var", fmt.Sprintf("aws_security_group_id=%s", config.SecurityGroupID))
	}

	args = append(args, "kali-aws-ami.pkr.hcl")

	buildCmd := exec.Command("packer", args...)
	buildCmd.Dir = config.WorkDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to build AMI: %w", err)
	}

	buildDuration := time.Since(startTime)
	log.Printf("AWS AMI build completed in %v", buildDuration)

	// Return the result (AMI ID would be parsed from output in a real implementation)
	result := &Result{
		AMIName:   config.AMIName,
		Region:    config.Region,
		BuildTime: buildDuration,
	}

	return result, nil
}

// GetLatestKaliAMI returns information about the latest official Kali Linux AMI.
func GetLatestKaliAMI(region string) (string, error) {
	// In a real implementation, this would query AWS EC2 describe-images
	// For now, return the expected owner ID for Kali Linux
	log.Printf("Kali Linux AMI owner ID: 679593333241 (official Kali AWS account)")
	return "679593333241", nil
}

// ValidateAWSConfig validates the AWS configuration for AMI builds.
func ValidateAWSConfig(config *BuildConfig) error {
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

// AMIBuildLog contains detailed logs for AMI builds.
type AMIBuildLog struct {
	Steps     []string
	Errors    []string
	StartTime time.Time
	EndTime   time.Time
}

// NewAMIBuildLog creates a new AMI build log.
func NewAMIBuildLog() *AMIBuildLog {
	return &AMIBuildLog{
		Steps:     make([]string, 0),
		Errors:    make([]string, 0),
		StartTime: time.Now(),
	}
}

// AddStep adds a build step to the log.
func (l *AMIBuildLog) AddStep(step string) {
	l.Steps = append(l.Steps, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), step))
}

// AddError adds an error to the log.
func (l *AMIBuildLog) AddError(err string) {
	l.Errors = append(l.Errors, fmt.Sprintf("[%s] ERROR: %s", time.Now().Format("15:04:05"), err))
}

// GetDuration returns the build duration.
func (l *AMIBuildLog) GetDuration() time.Duration {
	return l.EndTime.Sub(l.StartTime)
}

// Save saves the build log to a file.
func (l *AMIBuildLog) Save(logPath string) error {
	l.EndTime = time.Now()

	content := fmt.Sprintf("AMI Build Log\n")
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

// FindAMIID searches for the AMI ID in Packer build output.
func FindAMIID(output string) (string, error) {
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

// GetDefaultBuildConfig returns the default configuration for AMI builds.
func GetDefaultBuildConfig(workDir string) *BuildConfig {
	return &BuildConfig{
		Region:          "us-east-1",
		AMIName:         fmt.Sprintf("knirvserver-kali-%s", time.Now().Format("2006-01-02")),
		InstanceType:    "t3.medium",
		SubnetID:        "",
		SecurityGroupID: "",
		Description:     "KNIRVSERVER Kali Linux - Native deployment ready",
		WorkDir:         workDir,
	}
}

// EnsureAWSTools checks and installs required AWS tools.
func EnsureAWSTools() error {
	// Check for Packer
	if _, err := exec.LookPath("packer"); err != nil {
		log.Println("Installing HashiCorp Packer...")
		if err := installPacker(); err != nil {
			return fmt.Errorf("failed to install Packer: %w", err)
		}
	}

	// Check for AWS CLI
	if _, err := exec.LookPath("aws"); err != nil {
		log.Println("AWS CLI not found - AMI builds require AWS CLI")
		log.Println("Please install AWS CLI: https://aws.amazon.com/cli/")
	}

	return nil
}

// installPacker installs Packer on the system.
func installPacker() error {
	installCmd := exec.Command("bash", "-c", `
		curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
		echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
		sudo apt-get update && sudo apt-get install -y packer
	`)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	return installCmd.Run()
}

// GetAWSCredentialsStatus checks AWS credentials configuration.
func GetAWSCredentialsStatus() (map[string]bool, error) {
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

// IsAWSConfigured returns true if AWS is properly configured.
func IsAWSConfigured() bool {
	status, err := GetAWSCredentialsStatus()
	if err != nil {
		return false
	}

	// Need access key and secret key
	return status["AWS_ACCESS_KEY_ID"] && status["AWS_SECRET_ACCESS_KEY"]
}
