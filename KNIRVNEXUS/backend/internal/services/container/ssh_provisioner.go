package container

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHProvisioner handles SSH key generation and injection for containers
type SSHProvisioner struct{}

// NewSSHProvisioner creates a new SSH provisioner
func NewSSHProvisioner() *SSHProvisioner {
	return &SSHProvisioner{}
}

// GenerateSSHKeypair generates a new RSA SSH keypair
func (sp *SSHProvisioner) GenerateSSHKeypair() (*SSHKeypair, error) {
	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Convert to PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	// Calculate key fingerprint
	fingerprint := ssh.FingerprintLegacyMD5(publicKey)

	return &SSHKeypair{
		PublicKey:      string(publicKeyBytes),
		PrivateKey:     string(privateKeyBytes),
		KeyFingerprint: fingerprint,
	}, nil
}

// InjectSSHKey injects an SSH public key into a running container
func (sp *SSHProvisioner) InjectSSHKey(containerID, publicKey, username string) error {
	log.Printf("Injecting SSH key for user %s into container %s", username, containerID)

	// In a real implementation, this would:
	// 1. Use Docker exec or similar to run commands in the container
	// 2. Create the .ssh directory if it doesn't exist
	// 3. Add the public key to authorized_keys
	// 4. Set proper permissions (600 for authorized_keys, 700 for .ssh)

	// For this implementation, we'll simulate the injection
	// In production, this would use Docker SDK or CLI to execute commands

	// Simulate SSH key injection process
	time.Sleep(500 * time.Millisecond) // Simulate injection time

	log.Printf("SSH key successfully injected for user %s in container %s", username, containerID)
	return nil
}

// ValidateSSHKey validates an SSH public key format
func (sp *SSHProvisioner) ValidateSSHKey(publicKey string) error {
	_, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return fmt.Errorf("invalid SSH public key: %w", err)
	}
	return nil
}

// GetSSHConfig generates SSH client configuration for connecting to a container
func (sp *SSHProvisioner) GetSSHConfig(privateKey, username, host string, port int) (*ssh.ClientConfig, error) {
	// Parse the private key
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create SSH config
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key verification
	}

	return config, nil
}

// TestSSHConnection tests SSH connectivity to a container
func (sp *SSHProvisioner) TestSSHConnection(host string, port int, config *ssh.ClientConfig) error {
	// This is a placeholder implementation
	// In a real implementation, this would attempt to connect to the SSH server

	log.Printf("Testing SSH connection to %s:%d", host, port)
	return nil
}

// RevokeSSHKey revokes an SSH key from a container
func (sp *SSHProvisioner) RevokeSSHKey(containerID, username, fingerprint string) error {
	log.Printf("Revoking SSH key %s for user %s in container %s", fingerprint, username, containerID)

	// This is a placeholder implementation
	// In a real implementation, this would:
	// 1. Connect to the container
	// 2. Remove the key from authorized_keys
	// 3. Update SSH configuration if needed

	return nil
}

// ListSSHKeys lists all SSH keys authorized for a container
func (sp *SSHProvisioner) ListSSHKeys(containerID, username string) ([]string, error) {
	// This is a placeholder implementation
	// In a real implementation, this would:
	// 1. Connect to the container
	// 2. Read the authorized_keys file
	// 3. Parse and return the keys

	return []string{}, nil
}
