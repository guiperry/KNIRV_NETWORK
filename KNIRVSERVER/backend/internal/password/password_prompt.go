package password

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pb "backend_server/internal/proto"
	"backend_server/internal/utils"

	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
)

// PromptForPassword prompts the user for a password in the terminal.
func PromptForPassword(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Add a newline after the password input
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}
	return password, nil
}

// PromptForKeyCreation prompts the user to create a new encrypted key file.
func PromptForKeyCreation(keyFilePath string) error {
	fmt.Println("No encrypted key file found. You need to create one to continue.")
	fmt.Println("This will store your sensitive configuration securely.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Prompt for sensitive data
	fmt.Print("Enter API Key (or leave empty): ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	fmt.Print("Enter Secret Key (optional): ")
	secretKey, _ := reader.ReadString('\n')
	secretKey = strings.TrimSpace(secretKey)

	// Prompt for password
	password, err := PromptForPassword("Enter password to encrypt key file: ")
	if err != nil {
		return err
	}

	confirmPassword, err := PromptForPassword("Confirm password: ")
	if err != nil {
		return err
	}

	if string(password) != string(confirmPassword) {
		return fmt.Errorf("passwords do not match")
	}

	// Create content
	content := &pb.RootKeyFileContentProto{
		StripeSecretKey:       "", // Not used in this simplified version
		StripeWebhookSecret:   "",
		CoinbaseApiKey:        apiKey,
		CoinbaseWebhookSecret: "",
		RootPrivateKeyHex:     secretKey, // Using secret_key as root private key for simplicity
	}

	// Create key file
	if err := CreateEncryptedKeyFile(content, password, keyFilePath); err != nil {
		return err
	}

	fmt.Printf("Encrypted key file created successfully at %s\n", keyFilePath)
	fmt.Println("IMPORTANT: Securely back up this file and remember your password!")

	return nil
}

// CreateEncryptedKeyFile creates a new encrypted key file with the provided content and password.
func CreateEncryptedKeyFile(content *pb.RootKeyFileContentProto, password []byte, outputPath string) error {
	// Marshal content to protobuf
	contentBytes, err := proto.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal content: %w", err)
	}

	// Generate salt
	salt, err := utils.GenerateSalt(utils.SaltLen)
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive encryption key
	key, err := utils.DeriveKeyFromPassword(password, salt, utils.ScryptN, utils.ScryptR, utils.ScryptP, utils.KeyLen)
	if err != nil {
		return fmt.Errorf("failed to derive key: %w", err)
	}

	// Encrypt data
	encryptedData, err := utils.Encrypt(contentBytes, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Create encrypted file structure
	encryptedFile := &pb.EncryptedRootKeyFile{
		EncryptedContent: encryptedData,
		Salt:             salt,
	}

	// Marshal to protobuf
	fileBytes, err := proto.Marshal(encryptedFile)
	if err != nil {
		return fmt.Errorf("failed to marshal file content: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(outputPath, fileBytes, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// LoadEncryptedKeyFile loads and decrypts the key file using the provided password.
func LoadEncryptedKeyFile(keyFilePath string, password []byte) (*pb.RootKeyFileContentProto, error) {
	// Load encrypted key file
	encryptedFile, err := loadEncryptedKeyFile(keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load key file: %w", err)
	}

	// Derive encryption key from password
	key, err := utils.DeriveKeyFromPassword(
		password,
		encryptedFile.Salt,
		utils.ScryptN,
		utils.ScryptR,
		utils.ScryptP,
		utils.KeyLen,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key from password: %w", err)
	}

	// Decrypt data
	decryptedData, err := utils.Decrypt(encryptedFile.EncryptedContent, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key file (incorrect password?): %w", err)
	}

	// Unmarshal content
	var content pb.RootKeyFileContentProto
	if err := proto.Unmarshal(decryptedData, &content); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key file content: %w", err)
	}

	return &content, nil
}


// loadEncryptedKeyFile loads the encrypted key file from disk
func loadEncryptedKeyFile(keyFilePath string) (*pb.EncryptedRootKeyFile, error) {
	data, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	var encryptedFile pb.EncryptedRootKeyFile
	if err := proto.Unmarshal(data, &encryptedFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal encrypted key file: %w", err)
	}

	return &encryptedFile, nil
}