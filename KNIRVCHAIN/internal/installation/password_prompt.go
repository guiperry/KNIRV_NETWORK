package installation

import (
	"KNIRVCHAIN/config"
	pb "KNIRVCHAIN/internal/protocol/proto"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"KNIRVCHAIN/internal/utils"

	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
)

// PromptForPassword prompts the user for a password in the terminal.
func PromptForPassword(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add a newline after the password input
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}
	return password, nil
}

// LoadRootKeyFile loads and decrypts the Root key file using the provided password.
func LoadRootKeyFile(keyFilePath string, password []byte) (*pb.RootKeyFileContentProto, error) {
	// Load encrypted key file
	encryptedFile, err := config.LoadEncryptedRootKeyFile(keyFilePath)
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

// CreateRootKeyFile creates a new encrypted key file with the provided content and password.
func CreateRootKeyFile(content *pb.RootKeyFileContentProto, password []byte, outputPath string) error {
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

// PromptForRootKeyCreation prompts the user to create a new Root key file if one doesn't exist.
func PromptForRootKeyCreation(keyFilePath string) error {
	fmt.Println("No Root key file found. You need to create one to continue.")
	fmt.Println("This will store your sensitive Root configuration securely.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Prompt for sensitive data
	fmt.Print("Enter Stripe Secret Key (or leave empty): ")
	stripeSecretKey, _ := reader.ReadString('\n')
	stripeSecretKey = strings.TrimSpace(stripeSecretKey)

	fmt.Print("Enter Stripe Webhook Secret (optional): ")
	stripeWebhookSecret, _ := reader.ReadString('\n')
	stripeWebhookSecret = strings.TrimSpace(stripeWebhookSecret)

	fmt.Print("Enter Coinbase API Key (optional): ")
	coinbaseAPIKey, _ := reader.ReadString('\n')
	coinbaseAPIKey = strings.TrimSpace(coinbaseAPIKey)

	fmt.Print("Enter Coinbase Webhook Secret (optional): ")
	coinbaseWebhookSecret, _ := reader.ReadString('\n')
	coinbaseWebhookSecret = strings.TrimSpace(coinbaseWebhookSecret)

	fmt.Print("Enter Root Private Key (hex, required): ")
	rootPrivateKeyHex, _ := reader.ReadString('\n')
	rootPrivateKeyHex = strings.TrimSpace(rootPrivateKeyHex)

	if rootPrivateKeyHex == "" {
		return fmt.Errorf("root private key is required")
	}

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
		StripeSecretKey:       stripeSecretKey,
		StripeWebhookSecret:   stripeWebhookSecret,
		CoinbaseApiKey:        coinbaseAPIKey,
		CoinbaseWebhookSecret: coinbaseWebhookSecret,
		RootPrivateKeyHex:     rootPrivateKeyHex,
	}

	// Create key file
	if err := CreateRootKeyFile(content, password, keyFilePath); err != nil {
		return err
	}

	fmt.Printf("Root key file created successfully at %s\n", keyFilePath)
	fmt.Println("IMPORTANT: Securely back up this file and remember your password!")

	return nil
}
