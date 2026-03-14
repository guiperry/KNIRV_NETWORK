// encrypt_root_key.go
package main

import (
	"backend_server/internal/config"
	pb "backend_server/internal/proto"
	"backend_server/internal/utils"
	"flag"
	"bufio"
	"fmt"
	
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"google.golang.org/protobuf/proto"
)

// loadDotKeyFile loads values from the .key file in the project root
func loadDotKeyFile(keyFilePath string) map[string]string {
	values := make(map[string]string)
	// First, try to load from the standard project root .key file
	// This makes it easier for developers who have already set up their local env.
	projectRootKeyFile := ".key"
	if _, err := os.Stat(projectRootKeyFile); err == nil {
		log.Printf("Found project root .key file, loading defaults from it.")
		keyFilePath = projectRootKeyFile
	}

	// Try to open the .key file
	file, err := os.Open(keyFilePath)
	if err != nil {
		log.Printf("Warning: Could not open key file at '%s': %v", keyFilePath, err)
		return values
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			values[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Warning: Error reading .key file: %v", err)
	}

	return values
}

func main() {
	// Allow specifying a key file to load defaults from
	defaultKeyFile := filepath.Join("config", "embedded", "default_root.key")
	keyFileFlag := flag.String("keyfile", defaultKeyFile, "Path to a .key file to load default values from.")
	flag.Parse()

	a := app.New()
	w := a.NewWindow("KNIRVSERVER Root Key Encryptor")
	w.Resize(fyne.NewSize(700, 600))

	// Load values from .key file
	log.Printf("Loading default values from: %s", *keyFileFlag)
	dotKeyValues := loadDotKeyFile(*keyFileFlag)

	// Input Fields for Sensitive Data
	stripeSecretEntry := widget.NewEntry()
	stripeSecretEntry.SetPlaceHolder("Stripe Secret Key")
	stripeSecretEntry.SetText(dotKeyValues["STRIPE_SECRET_KEY"])

	stripeWebhookSecretEntry := widget.NewEntry()
	stripeWebhookSecretEntry.SetPlaceHolder("Stripe Webhook Secret")
	stripeWebhookSecretEntry.SetText(dotKeyValues["STRIPE_WEBHOOK_SECRET"]) // Note: typo in the original file

	coinbaseAPIKeyEntry := widget.NewEntry()
	coinbaseAPIKeyEntry.SetPlaceHolder("Coinbase API Key")
	coinbaseAPIKeyEntry.SetText(dotKeyValues["COINBASE_API_KEY"])

	coinbaseWebhookSecretEntry := widget.NewEntry()
	coinbaseWebhookSecretEntry.SetPlaceHolder("Coinbase Webhook Secret")
	coinbaseWebhookSecretEntry.SetText(dotKeyValues["COINBASE_WEBHOOK_SECRET"])

	rootPrivateKeyEntry := widget.NewEntry()
	rootPrivateKeyEntry.SetPlaceHolder("Root Private Key (Hex)")
	rootPrivateKeyEntry.Password = true // Mask the input
	rootPrivateKeyEntry.SetText(dotKeyValues["ROOT_PRIVATE_KEY"])

	cerebrasAPIKeyEntry := widget.NewEntry()
	cerebrasAPIKeyEntry.SetPlaceHolder("Cerebras API Key")
	cerebrasAPIKeyEntry.SetText(dotKeyValues["DEFAULT_CEREBRAS_API_KEY"])

	cerebrasBaseURLEntry := widget.NewEntry()
	cerebrasBaseURLEntry.SetPlaceHolder("Cerebras Base URL")
	cerebrasBaseURLEntry.SetText(dotKeyValues["DEFAULT_CEREBRAS_BASE_URL"])

	githubTokenEntry := widget.NewEntry()
	githubTokenEntry.SetPlaceHolder("GitHub Token")
	githubTokenEntry.SetText(dotKeyValues["DEFAULT_GITHUB_TOKEN"])

	githubPublicKeyEntry := widget.NewEntry()
	githubPublicKeyEntry.SetPlaceHolder("GitHub Public Key")
	githubPublicKeyEntry.SetText(dotKeyValues["DEFAULT_GITHUB_PUBLIC_KEY_FOR_UPDATES"])

	// Password Input
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Enter Password to Encrypt Key File")

	confirmPasswordEntry := widget.NewPasswordEntry()
	confirmPasswordEntry.SetPlaceHolder("Confirm Password")

	// Output File Path
	outputFileEntry := widget.NewEntry()
	outputFileEntry.SetPlaceHolder("Output .key file path (e.g., root.key)")

	// Suggest a default path
	defaultKeyPath, err := config.GetRootKeyPath()
	if err == nil {
		outputFileEntry.SetText(defaultKeyPath)
	} else {
		log.Printf("Warning: Could not determine default key path: %v", err)
		outputFileEntry.SetText("root.key")
	}

	// Encrypt Button
	encryptButton := widget.NewButton("Encrypt Key File", func() {
		password := []byte(passwordEntry.Text)
		confirmPassword := []byte(confirmPasswordEntry.Text)
		outputPath := outputFileEntry.Text

		if len(password) == 0 {
			dialog.ShowError(fmt.Errorf("password cannot be empty"), w) // #nosec G104
			return
		}
		if string(password) != string(confirmPassword) { // Compare strings for passwords
			dialog.ShowError(fmt.Errorf("passwords do not match"), w) // #nosec G104
			return
		}
		if outputPath == "" {
			dialog.ShowError(fmt.Errorf("output file path cannot be empty"), w)
			return
		}

		// Gather Sensitive Data
		content := &pb.RootKeyFileContentProto{
			StripeSecretKey:       stripeSecretEntry.Text,
			StripeWebhookSecret:   stripeWebhookSecretEntry.Text,
			CoinbaseApiKey:        coinbaseAPIKeyEntry.Text,
			CoinbaseWebhookSecret: coinbaseWebhookSecretEntry.Text,
			RootPrivateKeyHex:     rootPrivateKeyEntry.Text,
	
		}

		// Marshal to protobuf
		contentBytes, err := proto.Marshal(content)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to marshal content to protobuf: %v", err), w)
			return
		}

		// Generate Salt and Derive Key
		salt, err := utils.GenerateSalt(utils.SaltLen)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to generate salt: %v", err), w)
			return
		}

		encryptionKey, err := utils.DeriveKeyFromPassword(password, salt, utils.ScryptN, utils.ScryptR, utils.ScryptP, utils.KeyLen)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to derive encryption key: %v", err), w)
			return
		}

		// Encrypt Data
		encryptedData, err := utils.Encrypt(contentBytes, encryptionKey)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to encrypt data: %v", err), w)
			return
		}

		// Prepare File Content
		encryptedFileContent := &pb.EncryptedRootKeyFile{
			EncryptedContent: encryptedData,
			Salt:             salt,
		}

		// Marshal the outer envelope using Protobuf for consistency
		fileBytes, err := proto.Marshal(encryptedFileContent)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to marshal file content to protobuf: %v", err), w)
			return
		}

		// Save File
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0700); err != nil {
			dialog.ShowError(fmt.Errorf("failed to create directory %s: %v", dir, err), w)
			return
		}
		if err := os.WriteFile(outputPath, fileBytes, 0600); err != nil {
			dialog.ShowError(fmt.Errorf("failed to write key file %s: %v", outputPath, err), w)
			return
		}

		dialog.ShowInformation("Success", fmt.Sprintf("Root key file encrypted and saved to:\n%s\n\nIMPORTANT: Securely back up this file and remember your password!", outputPath), w)
	})

	// Layout
	contentForm := container.NewVBox(
		widget.NewLabel("Enter Sensitive Root Node Configuration:"),
		widget.NewLabel("Payment Processor Keys:"),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Stripe Secret Key:"), stripeSecretEntry),
			container.NewVBox(widget.NewLabel("Stripe Webhook Secret:"), stripeWebhookSecretEntry),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Coinbase API Key:"), coinbaseAPIKeyEntry),
			container.NewVBox(widget.NewLabel("Coinbase Webhook Secret:"), coinbaseWebhookSecretEntry),
		),
		widget.NewSeparator(),
		widget.NewLabel("Root Node Private Key:"),
		rootPrivateKeyEntry,
		widget.NewSeparator(),
		widget.NewLabel("Cerebras Configuration:"),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Cerebras API Key:"), cerebrasAPIKeyEntry),
			container.NewVBox(widget.NewLabel("Cerebras Base URL:"), cerebrasBaseURLEntry),
		),
		widget.NewSeparator(),
		widget.NewLabel("GitHub Configuration:"),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("GitHub Token:"), githubTokenEntry),
			container.NewVBox(widget.NewLabel("GitHub Public Key:"), githubPublicKeyEntry),
		),
		widget.NewSeparator(),
		widget.NewLabel("Set a Password for Encryption:"),
		passwordEntry,
		confirmPasswordEntry,
		widget.NewSeparator(),
		widget.NewLabel("Output File Path:"),
		outputFileEntry,
		encryptButton,
	)

	w.SetContent(container.NewVScroll(contentForm))
	w.ShowAndRun()
}
