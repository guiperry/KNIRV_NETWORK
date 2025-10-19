**Registering the URI Scheme (Operating System Level)**

To make `nrn://` links clickable and automatically open your application, you need to register the URI scheme with the operating system. This is OS-specific:

*   **Windows:** You need to modify the Windows Registry. See the Microsoft documentation for "Registering an Application to a URI Scheme" (search for it online). You'll create a registry key under `HKEY_CLASSES_ROOT` named `nrn`.

    ```regedit
    Windows Registry Editor Version 5.00

    [HKEY_CLASSES_ROOT\nrn]
    @="URL:nrn Protocol"
    "URL Protocol"=""

    [HKEY_CLASSES_ROOT\nrn\DefaultIcon]
    @="\"C:\\Path\\To\\Your\\Application.exe\",1"  ; Replace with your app's path

    [HKEY_CLASSES_ROOT\nrn\shell]

    [HKEY_CLASSES_ROOT\nrn\shell\open]

    [HKEY_CLASSES_ROOT\nrn\shell\open\command]
    @="\"C:\\Path\\To\\Your\\Application.exe\" \"%1\""  ; Replace with your app's path
    ```

*   **macOS:** You need to modify your application's `Info.plist` file.  Add a `CFBundleURLTypes` array with a dictionary that defines the `nrn` scheme.  See Apple's documentation for "Defining a Custom URL Scheme for Your App" (search for it online).

    ```xml
    <key>CFBundleURLTypes</key>
    <array>
        <dict>
            <key>CFBundleURLName</key>
            <string>com.yourcompany.yourapp</string>  <!-- Replace with your bundle identifier -->
            <key>CFBundleURLSchemes</key>
            <array>
                <string>nrn</string>
            </array>
        </dict>
    </array>
    ```

*   **Linux:**  The process depends on the desktop environment (e.g., GNOME, KDE).  You'll typically need to create a `.desktop` file that defines the URI scheme and associates it with your application.  Place the `.desktop` file in `~/.local/share/applications/`.

    ```ini
    [Desktop Entry]
    Name=Your Application
    Exec=/path/to/your/application %u  ; Replace with your app's path
    Type=Application
    MimeType=x-scheme-handler/nrn;
    ```

    You'll also need to register the MIME type:

    ```xml
    <?xml version="1.0"?>
    <mime-info xmlns="http://www.freedesktop.org/standards/shared-mime-info">
      <mime-type type="x-scheme-handler/nrn">
        <comment>nrn URI</comment>
        <glob pattern="nrn:"/>
      </mime-type>
    </mime-info>
    ```

    Save this as `~/.local/share/mime/packages/nrn.xml` and then run:

    ```bash
    update-mime-database ~/.local/share/mime
    ```


==========================================

A go script that registers the URI (chain://LinkContent.knirv.com) on every device mentioned?

```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// registerURI registers the chain://LinkContent.knirv.com URI scheme on the current operating system.
func registerURI() error {
	osType := runtime.GOOS

	switch osType {
	case "windows":
		return registerURIWindows()
	case "darwin":
		return registerURIMacOS()
	case "linux":
		return registerURILinux()
	default:
		return fmt.Errorf("unsupported operating system: %s", osType)
	}
}

// registerURIWindows registers the URI scheme on Windows.
func registerURIWindows() error {
	// Requires elevated privileges (Run as Administrator)
	// This is a simplified example. You may need to adjust the paths and parameters.

	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	regCommand := fmt.Sprintf(`reg add HKCR\chain /ve /d "URL:chain Protocol" /f && `+
		`reg add HKCR\chain /v "URL Protocol" /d "" /f && `+
		`reg add HKCR\chain\DefaultIcon /ve /d "\"%s\",1" /f && `+
		`reg add HKCR\chain\shell\open\command /ve /d "\"%s\" \"%%1\"" /f`, executablePath, executablePath)

	cmd := exec.Command("cmd", "/c", regCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to register URI scheme (Windows): %w, output: %s", err, string(output))
	}

	fmt.Println("URI scheme registered successfully on Windows.")
	return nil
}

// registerURIMacOS registers the URI scheme on macOS.
func registerURIMacOS() error {
	// This requires creating or modifying the Info.plist file inside the .app bundle.
	// It's more complex and usually handled during the application build process.
	// This example provides a conceptual outline.

	// 1. Get the application's bundle identifier (e.g., com.example.YourApp)
	// You would typically read this from a configuration file or environment variable.
	bundleIdentifier := "com.example.yourapp" // Replace with your app's bundle identifier

	// 2. Construct the plistbuddy command to add the URI scheme to Info.plist.
	// **Important:** This assumes you have plistbuddy installed (it's part of Xcode).
	// This command needs to be adapted to your Info.plist structure.

	plistCommand := fmt.Sprintf(`
		/usr/libexec/PlistBuddy -c "Add :CFBundleURLTypes:0 dict" Info.plist &&
		/usr/libexec/PlistBuddy -c "Add :CFBundleURLTypes:0:CFBundleURLName string '%s'" Info.plist &&
		/usr/libexec/PlistBuddy -c "Add :CFBundleURLTypes:0:CFBundleURLSchemes array" Info.plist &&
		/usr/libexec/PlistBuddy -c "Add :CFBundleURLTypes:0:CFBundleURLSchemes:0 string 'chain'" Info.plist
	`, bundleIdentifier)

	// 3. Execute the plistbuddy command
	cmd := exec.Command("bash", "-c", plistCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to register URI scheme (macOS): %w, output: %s", err, string(output))
	}

	fmt.Println("URI scheme registration command executed on macOS (Info.plist modification required).")

	// Important: You'll likely need to codesign your application after modifying Info.plist.

	return nil
}

// registerURILinux registers the URI scheme on Linux.
func registerURILinux() error {
	// This is a simplified example and may require adjustments depending on the desktop environment.
	// It assumes the user has a ~/.local/share/applications directory and update-mime-database.

	// 1. Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user's home directory: %w", err)
	}

	// 2. Create the .desktop file
	desktopFileContent := fmt.Sprintf(`
[Desktop Entry]
Name=Your Application
Exec=/path/to/your/application %%u  ; Replace with your app's path
Type=Application
MimeType=x-scheme-handler/chain;
	`) // Replace with your Exec path

	desktopFilePath := homeDir + "/.local/share/applications/chain.desktop"
	err = os.WriteFile(desktopFilePath, []byte(desktopFileContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to create .desktop file: %w", err)
	}

	// 3. Create the MIME type file
	mimeFileContent := `<?xml version="1.0"?>
<mime-info xmlns="http://www.freedesktop.org/standards/shared-mime-info">
  <mime-type type="x-scheme-handler/chain">
    <comment>chain URI</comment>
    <glob pattern="chain:"/>
  </mime-type>
</mime-info>
	`

	mimeFilePath := homeDir + "/.local/share/mime/packages/chain.xml"
	err = os.WriteFile(mimeFilePath, []byte(mimeFileContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to create MIME file: %w", err)
	}

	// 4. Update the MIME database
	cmd := exec.Command("update-mime-database", homeDir+"/.local/share/mime")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update MIME database: %w, output: %s", err, string(output))
	}

	fmt.Println("URI scheme registered successfully on Linux (requires manual verification).")

	return nil
}

func main() {
	err := registerURI()
	if err != nil {
		log.Fatalf("Failed to register URI scheme: %v", err)
	}

	fmt.Println("URI registration process completed.")
}
```

**Key Considerations and Improvements:**

*   **Elevated Privileges:**  The Windows registration requires elevated privileges (Run as Administrator).  You'll need to handle this in your application's installation process.  The script will likely fail if not run as administrator.
*   **macOS: Info.plist Modification:**  Modifying the `Info.plist` file on macOS is complex and should ideally be done during the application build process. The provided `plistbuddy` command may need adjustments depending on your `Info.plist` structure.  After modifying `Info.plist`, you *must* codesign your application again.  This example is a *conceptual* outline, not a complete solution.
*   **Linux: Desktop Environment Differences:**  The Linux registration is a simplified example and may require adjustments depending on the desktop environment (GNOME, KDE, etc.). The paths and commands may vary.  The script also requires `update-mime-database` to be available.
*   **Error Handling:** The code includes basic error handling, but you should add more robust error logging and user feedback.
*   **Paths:**  The paths to the application executable are hardcoded. You'll need to make them dynamic so the script can find the executable regardless of where the application is installed.
*   **Security:** Be extremely careful when executing external commands.  Validate all inputs to prevent command injection vulnerabilities.
*   **Build Process Integration:**  Ideally, URI scheme registration should be part of your application's build and installation process, not a separate script.  Use platform-specific tools (e.g., installers, package managers) to handle this.
*   **Testing:**  Thoroughly test the URI scheme registration on all supported operating systems.
*   **User Prompts:**  For a better user experience, consider prompting the user for confirmation before attempting to register the URI scheme.
*   **Idempotency:** The script is not idempotent. Running it multiple times may cause issues. Check if the scheme is already registered before attempting to register it.
*   **Executable Path:** The executable path on Linux is a placeholder. Find a proper method of finding the executable's full path.

**Important: This script is a starting point and requires significant adaptation and testing to work correctly in a real-world application.  It's strongly recommended to integrate URI scheme registration into your application's build and installation process using platform-specific tools and techniques.**  Directly manipulating the registry or system files with a Go script is generally not the best approach for deployment.
