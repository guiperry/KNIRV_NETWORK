// encrypt_root_key.go
package main

import (
	"bufio"
	"flag"
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

	"backend_server/keyfile"
)

// loadEnvFile loads values from the .env file in the project root
func loadEnvFile(envFilePath string) map[string]string {
	values := make(map[string]string)
	// First, try to load from the standard project root .env file
	// This makes it easier for developers who have already set up their local env.
	projectRootEnvFile := ".env"
	if _, err := os.Stat(projectRootEnvFile); err == nil {
		log.Printf("Found project root .env file, loading defaults from it.")
		envFilePath = projectRootEnvFile
	}

	// Try to open the .env file
	file, err := os.Open(envFilePath)
	if err != nil {
		log.Printf("Warning: Could not open env file at '%s': %v", envFilePath, err)
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
		log.Printf("Warning: Error reading .env file: %v", err)
	}

	return values
}

func main() {
	// Allow specifying an env file to load defaults from
	defaultEnvFile := ".env"
	envFileFlag := flag.String("envfile", defaultEnvFile, "Path to a .env file to load default values from.")
	flag.Parse()

	a := app.New()
	w := a.NewWindow("KNIRVSERVER Root Key Encryptor")
	w.Resize(fyne.NewSize(700, 600))

	// Load values from .env file
	log.Printf("Loading default values from: %s", *envFileFlag)
	envValues := loadEnvFile(*envFileFlag)

	// Input Fields for Sensitive Data
	stripeSecretEntry := widget.NewEntry()
	stripeSecretEntry.SetPlaceHolder("Stripe Secret Key")
	stripeSecretEntry.SetText(envValues["STRIPE_SECRET_KEY"])

	stripeWebhookSecretEntry := widget.NewEntry()
	stripeWebhookSecretEntry.SetPlaceHolder("Stripe Webhook Secret")
	stripeWebhookSecretEntry.SetText(envValues["STRIPE_WEBHOOK_SECRET"]) // Note: typo in the original file

	coinbaseAPIKeyEntry := widget.NewEntry()
	coinbaseAPIKeyEntry.SetPlaceHolder("Coinbase API Key")
	coinbaseAPIKeyEntry.SetText(envValues["COINBASE_API_KEY"])

	coinbaseWebhookSecretEntry := widget.NewEntry()
	coinbaseWebhookSecretEntry.SetPlaceHolder("Coinbase Webhook Secret")
	coinbaseWebhookSecretEntry.SetText(envValues["COINBASE_WEBHOOK_SECRET"])

	rootPrivateKeyEntry := widget.NewEntry()
	rootPrivateKeyEntry.SetPlaceHolder("Root Private Key (Hex)")
	rootPrivateKeyEntry.Password = true // Mask the input
	rootPrivateKeyEntry.SetText(envValues["ROOT_PRIVATE_KEY"])

	cerebrasAPIKeyEntry := widget.NewEntry()
	cerebrasAPIKeyEntry.SetPlaceHolder("Cerebras API Key")
	cerebrasAPIKeyEntry.SetText(envValues["DEFAULT_CEREBRAS_API_KEY"])

	cerebrasBaseURLEntry := widget.NewEntry()
	cerebrasBaseURLEntry.SetPlaceHolder("Cerebras Base URL")
	cerebrasBaseURLEntry.SetText(envValues["DEFAULT_CEREBRAS_BASE_URL"])

	githubTokenEntry := widget.NewEntry()
	githubTokenEntry.SetPlaceHolder("GitHub Token")
	githubTokenEntry.SetText(envValues["DEFAULT_GITHUB_TOKEN"])

	githubPublicKeyEntry := widget.NewEntry()
	githubPublicKeyEntry.SetPlaceHolder("GitHub Public Key")
	githubPublicKeyEntry.SetText(envValues["DEFAULT_GITHUB_PUBLIC_KEY_FOR_UPDATES"])

	// Cloudflare credentials
	cloudflareAPITokenEntry := widget.NewEntry()
	cloudflareAPITokenEntry.SetPlaceHolder("Cloudflare API Token")
	cloudflareAPITokenEntry.SetText(envValues["CLOUDFLARE_API_TOKEN"])

	cloudflareZoneIDEntry := widget.NewEntry()
	cloudflareZoneIDEntry.SetPlaceHolder("Cloudflare Zone ID (optional)")
	cloudflareZoneIDEntry.SetText(envValues["CLOUDFLARE_ZONE_ID"])

	cloudflareAccountIDEntry := widget.NewEntry()
	cloudflareAccountIDEntry.SetPlaceHolder("Cloudflare Account ID (optional, for tunnel)")
	cloudflareAccountIDEntry.SetText(envValues["CLOUDFLARE_ACCOUNT_ID"])

	// Production secrets
	jwtSecretEntry := widget.NewEntry()
	jwtSecretEntry.SetPlaceHolder("JWT Secret")
	jwtSecretEntry.SetText(envValues["JWT_SECRET"])

	knirvJwtSecretEntry := widget.NewEntry()
	knirvJwtSecretEntry.SetPlaceHolder("KNIRV JWT Secret")
	knirvJwtSecretEntry.SetText(envValues["KNIRV_JWT_SECRET"])

	geminiApiKeyEntry := widget.NewEntry()
	geminiApiKeyEntry.SetPlaceHolder("Gemini API Key")
	geminiApiKeyEntry.SetText(envValues["GEMINI_API_KEY"])

	deepseekApiKeyEntry := widget.NewEntry()
	deepseekApiKeyEntry.SetPlaceHolder("DeepSeek API Key")
	deepseekApiKeyEntry.SetText(envValues["DEEPSEEK_API_KEY"])

	tlsCertEntry := widget.NewEntry()
	tlsCertEntry.SetPlaceHolder("TLS Certificate (PEM)")
	tlsCertEntry.SetText(envValues["TLS_CERT"])

	tlsKeyEntry := widget.NewEntry()
	tlsKeyEntry.SetPlaceHolder("TLS Private Key (PEM)")
	tlsKeyEntry.SetText(envValues["TLS_KEY"])

	// Password Input
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Enter Password to Encrypt Key File")

	confirmPasswordEntry := widget.NewPasswordEntry()
	confirmPasswordEntry.SetPlaceHolder("Confirm Password")

	// Output File Path
	outputFileEntry := widget.NewEntry()
	outputFileEntry.SetPlaceHolder("Output .key file path (e.g., root.key)")

	// Suggest a default path
	defaultKeyPath, err := keyfile.GetRootKeyPath()
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
		content := &keyfile.RootKeyFileContentProto{
			StripeSecretKey:       stripeSecretEntry.Text,
			StripeWebhookSecret:   stripeWebhookSecretEntry.Text,
			CoinbaseApiKey:        coinbaseAPIKeyEntry.Text,
			CoinbaseWebhookSecret: coinbaseWebhookSecretEntry.Text,
			RootPrivateKeyHex:     rootPrivateKeyEntry.Text,
			JwtSecret:             jwtSecretEntry.Text,
			KnirvJwtSecret:        knirvJwtSecretEntry.Text,
			GeminiApiKey:          geminiApiKeyEntry.Text,
			DeepseekApiKey:        deepseekApiKeyEntry.Text,
			TlsCert:               tlsCertEntry.Text,
			TlsKey:                tlsKeyEntry.Text,
			CerebrasApiKey:       cerebrasAPIKeyEntry.Text,
			CloudflareApiToken:    cloudflareAPITokenEntry.Text,
			CloudflareZoneId:      cloudflareZoneIDEntry.Text,
			CloudflareAccountId:   cloudflareAccountIDEntry.Text,
		}

		// Marshal to protobuf
		contentBytes, err := proto.Marshal(content)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to marshal content to protobuf: %v", err), w)
			return
		}

		// Generate Salt and Derive Key
		salt, err := keyfile.GenerateSalt(16)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to generate salt: %v", err), w)
			return
		}

		encryptionKey, err := keyfile.DeriveKeyFromPassword(password, salt, 32768, 8, 1, 32)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to derive encryption key: %v", err), w)
			return
		}

		// Encrypt Data
		encryptedData, err := keyfile.Encrypt(contentBytes, encryptionKey)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to encrypt data: %v", err), w)
			return
		}

		// Prepare File Content
		encryptedFileContent := &keyfile.EncryptedRootKeyFile{
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
		widget.NewLabel("Cloudflare Configuration:"),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Cloudflare API Token:"), cloudflareAPITokenEntry),
			container.NewVBox(widget.NewLabel("Cloudflare Zone ID:"), cloudflareZoneIDEntry),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Cloudflare Account ID:"), cloudflareAccountIDEntry),
		),
		widget.NewSeparator(),
		widget.NewLabel("Production Secrets:"),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("JWT Secret:"), jwtSecretEntry),
			container.NewVBox(widget.NewLabel("KNIRV JWT Secret:"), knirvJwtSecretEntry),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Gemini API Key:"), geminiApiKeyEntry),
			container.NewVBox(widget.NewLabel("DeepSeek API Key:"), deepseekApiKeyEntry),
		),
		widget.NewLabel("TLS Certificates (PEM format):"),
		container.NewVBox(widget.NewLabel("Certificate:"), tlsCertEntry),
		container.NewVBox(widget.NewLabel("Private Key:"), tlsKeyEntry),
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
