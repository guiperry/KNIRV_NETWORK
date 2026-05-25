// bootnode_key_encryptor.go
//
// Bootnode Key Encryptor — a clone of root_key_encryptor with all
// static infrastructure secrets hard-coded so the user can only
// override a limited set of runtime configuration fields.
//
// Unlike the general-purpose root_key_encryptor this version does
// NOT read a .env file.  Secrets that are fixed per-bootnode
// (Stripe, Coinbase, Cloudflare, TLS, GitHub, SMTP, etc.) are
// compiled directly into the binary.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"google.golang.org/protobuf/proto"

	"backend_server/keyfile"
)

// ─────────────────────────────────────────────────────────────────
//  BOOTNODE FIXED SECRETS — hard-coded, not user-visible
// ─────────────────────────────────────────────────────────────────

const (
	// Stripe
	defaultStripeSecretKey     = "sk_test_51RNeHkQvOnycmPYONL301eac0qhAIL7UBSYcP7IIsWGMhiwuViEzQezB1aMUrlqCqR9Fn0a1HUyjzCLYubUAX0Eg00nMZN93Tp"
	defaultStripeWebhookSecret = ""

	// Coinbase
	defaultCoinbaseAPIKey        = ""
	defaultCoinbaseWebhookSecret = ""

	// Root node
	defaultRootPrivateKeyHex = "0x1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a"

	// JWT
	defaultJwtSecret      = "knirv100"
	defaultKnirvJwtSecret = "knirv100"

	// GitHub
	defaultGithubToken     = "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"
	defaultGithubPublicKey = "" // not carried in protobuf but kept for symmetry

	// Cloudflare
	defaultCloudflareAPIToken      = "cfut_5dyAlA040UoVfrOMTzdmvk54BIKLQibOGnQOBcxla7951334"
	defaultCloudflareZoneID        = "6ec647f95e75c97a504c40a5e07e4e52"
	defaultCloudflareAccountID     = "a4e323be05e19697704b8bd8a0faa435"
	defaultCloudflareEmbeddingsURL = "https://embeddings.knirv.com"

	// TLS (not supplied in original .env — placeholder)
	defaultTLSCert = ""
	defaultTLSKey  = ""

	// SMTP (ORACLE_KEY_PASSWORD from .env mapped to SMTP field)
	defaultSmtpPassword = "marching100"
)

func main() {
	// ── CLI flags (only output path is configurable) ──────────
	defaultKeyPath, err := keyfile.GetRootKeyPath()
	if err != nil {
		log.Printf("Warning: could not determine default key path: %v", err)
		defaultKeyPath = "root.key"
	}
	outputPathFlag := flag.String("output", defaultKeyPath, "Output .key file path")
	flag.Parse()

	// ── Fyne UI ───────────────────────────────────────────────
	a := app.New()
	w := a.NewWindow("KNIRVSERVER Bootnode Key Encryptor")
	w.Resize(fyne.NewSize(700, 700))

	// ---- Editable: LLM Provider Configuration (Gemini) ----
	geminiAPIKeyEntry := widget.NewEntry()
	geminiAPIKeyEntry.SetPlaceHolder("Gemini API Key")

	geminiAPIEndpointEntry := widget.NewEntry()
	geminiAPIEndpointEntry.SetPlaceHolder("Gemini API Endpoint")

	geminiModelEntry := widget.NewEntry()
	geminiModelEntry.SetPlaceHolder("Gemini Model")

	modelNameEntry := widget.NewEntry()
	modelNameEntry.SetPlaceHolder("Model Name")

	// ---- Editable: LLM Provider Configuration (DeepSeek) ----
	deepseekAPIKeyEntry := widget.NewEntry()
	deepseekAPIKeyEntry.SetPlaceHolder("DeepSeek API Key")

	deepseekAPIEndpointEntry := widget.NewEntry()
	deepseekAPIEndpointEntry.SetPlaceHolder("DeepSeek API Endpoint")

	deepseekBaseURLEntry := widget.NewEntry()
	deepseekBaseURLEntry.SetPlaceHolder("DeepSeek Base URL")

	deepseekModelEntry := widget.NewEntry()
	deepseekModelEntry.SetPlaceHolder("DeepSeek Model")

	// ---- Editable: Cerebras ----
	cerebrasAPIKeyEntry := widget.NewEntry()
	cerebrasAPIKeyEntry.SetPlaceHolder("Cerebras API Key")

	cerebrasBaseURLEntry := widget.NewEntry()
	cerebrasBaseURLEntry.SetPlaceHolder("Cerebras Base URL")

	// ---- Editable: Device Credentials (Hasher) ----
	deviceIPEntry := widget.NewEntry()
	deviceIPEntry.SetPlaceHolder("Device IP (e.g. 192.168.12.151)")

	devicePasswordEntry := widget.NewEntry()
	devicePasswordEntry.Password = true
	devicePasswordEntry.SetPlaceHolder("Device Password")

	deviceUsernameEntry := widget.NewEntry()
	deviceUsernameEntry.SetPlaceHolder("Device Username (e.g. root)")

	// ---- Password & output ----
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Enter Password to Encrypt Key File")

	confirmPasswordEntry := widget.NewPasswordEntry()
	confirmPasswordEntry.SetPlaceHolder("Confirm Password")

	outputFileEntry := widget.NewEntry()
	outputFileEntry.SetPlaceHolder("Output .key file path (e.g. root.key)")
	outputFileEntry.SetText(*outputPathFlag)

	// ---- Encrypt button ----
	encryptButton := widget.NewButton("Encrypt Key File", func() {
		password := []byte(passwordEntry.Text)
		confirmPassword := []byte(confirmPasswordEntry.Text)
		outputPath := outputFileEntry.Text

		if len(password) == 0 {
			dialog.ShowError(fmt.Errorf("password cannot be empty"), w)
			return
		}
		if string(password) != string(confirmPassword) {
			dialog.ShowError(fmt.Errorf("passwords do not match"), w)
			return
		}
		if outputPath == "" {
			dialog.ShowError(fmt.Errorf("output file path cannot be empty"), w)
			return
		}

		// Build the protobuf: fixed bootnode secrets + user-supplied values
		content := &keyfile.RootKeyFileContentProto{
			StripeSecretKey:          defaultStripeSecretKey,
			StripeWebhookSecret:      defaultStripeWebhookSecret,
			CoinbaseApiKey:           defaultCoinbaseAPIKey,
			CoinbaseWebhookSecret:    defaultCoinbaseWebhookSecret,
			RootPrivateKeyHex:        defaultRootPrivateKeyHex,
			JwtSecret:                defaultJwtSecret,
			KnirvJwtSecret:           defaultKnirvJwtSecret,
			GeminiApiKey:             geminiAPIKeyEntry.Text,
			DeepseekApiKey:           deepseekAPIKeyEntry.Text,
			CerebrasApiKey:           cerebrasAPIKeyEntry.Text,
			TlsCert:                  defaultTLSCert,
			TlsKey:                   defaultTLSKey,
			CloudflareApiToken:       defaultCloudflareAPIToken,
			CloudflareZoneId:         defaultCloudflareZoneID,
			CloudflareAccountId:      defaultCloudflareAccountID,
			CloudflareEmbeddingsUrl:  defaultCloudflareEmbeddingsURL,
			SmtpPassword:             defaultSmtpPassword,
			DefaultGithubToken:       defaultGithubToken,
			DeviceIp:                 deviceIPEntry.Text,
			DevicePassword:           devicePasswordEntry.Text,
			DeviceUsername:           deviceUsernameEntry.Text,
		}

		// Marshal to protobuf
		contentBytes, err := proto.Marshal(content)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to marshal content: %v", err), w)
			return
		}

		// Generate salt & derive encryption key
		salt, err := keyfile.GenerateSalt(16)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to generate salt: %v", err), w)
			return
		}

		encryptionKey, err := keyfile.DeriveKeyFromPassword(password, salt, 32768, 8, 1, 32)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to derive key: %v", err), w)
			return
		}

		// Encrypt
		encryptedData, err := keyfile.Encrypt(contentBytes, encryptionKey)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to encrypt: %v", err), w)
			return
		}

		// Outer envelope
		fileBytes, err := proto.Marshal(&keyfile.EncryptedRootKeyFile{
			EncryptedContent: encryptedData,
			Salt:             salt,
		})
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to marshal envelope: %v", err), w)
			return
		}

		// Write to disk
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0700); err != nil {
			dialog.ShowError(fmt.Errorf("failed to create directory %s: %v", dir, err), w)
			return
		}
		if err := os.WriteFile(outputPath, fileBytes, 0600); err != nil {
			dialog.ShowError(fmt.Errorf("failed to write %s: %v", outputPath, err), w)
			return
		}

		dialog.ShowInformation("Success",
			fmt.Sprintf("Bootnode key file encrypted and saved to:\n%s\n\nIMPORTANT: Securely back up this file and remember your password!", outputPath),
			w,
		)
	})

	// ── Layout ─────────────────────────────────────────────────
	form := container.NewVBox(
		widget.NewLabelWithStyle(
			"Bootnode Key Encryptor — Configuration",
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		),
		widget.NewLabel("All other secrets (Stripe, Coinbase, Cloudflare, TLS, JWT,\nGitHub, SMTP, etc.) are compiled into this binary and are\nnot user-configurable."),
		widget.NewSeparator(),

		widget.NewLabelWithStyle("Google Gemini API", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("API Key:"), geminiAPIKeyEntry),
			container.NewVBox(widget.NewLabel("API Endpoint:"), geminiAPIEndpointEntry),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Model:"), geminiModelEntry),
			container.NewVBox(widget.NewLabel("Model Name:"), modelNameEntry),
		),
		widget.NewSeparator(),

		widget.NewLabelWithStyle("DeepSeek API", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("API Key:"), deepseekAPIKeyEntry),
			container.NewVBox(widget.NewLabel("API Endpoint:"), deepseekAPIEndpointEntry),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Base URL:"), deepseekBaseURLEntry),
			container.NewVBox(widget.NewLabel("Model:"), deepseekModelEntry),
		),
		widget.NewSeparator(),

		widget.NewLabelWithStyle("Cerebras", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("API Key:"), cerebrasAPIKeyEntry),
			container.NewVBox(widget.NewLabel("Base URL:"), cerebrasBaseURLEntry),
		),
		widget.NewSeparator(),

		widget.NewLabelWithStyle("Hasher Device Credentials", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Device IP:"), deviceIPEntry),
			container.NewVBox(widget.NewLabel("Device Password:"), devicePasswordEntry),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Device Username:"), deviceUsernameEntry),
		),
		widget.NewSeparator(),

		widget.NewLabelWithStyle("Encryption Password", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		passwordEntry,
		confirmPasswordEntry,
		widget.NewSeparator(),

		widget.NewLabelWithStyle("Output", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		outputFileEntry,
		encryptButton,
	)

	w.SetContent(container.NewVScroll(form))
	w.ShowAndRun()
}
