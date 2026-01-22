package starter

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	// For macOS bundle path finding (example using a helper)
	// "github.com/your_org/your_utils" // Or implement directly
)

// URIScheme represents a URI scheme to register
type URIScheme struct {
	Name        string // The scheme name (e.g., "nrn", "chain")
	Description string // Description of the scheme
}

// registerURISchemes registers multiple URI schemes on the current operating system
func registerURISchemes(schemes []URIScheme) error {
	log.Printf("Registering URI schemes: %s", getSchemeList(schemes))

	osType := runtime.GOOS
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	switch osType {
	case "windows":
		// Check privileges *before* attempting registration
		if !CheckAdminPrivileges() {
			return fmt.Errorf("administrator privileges are required to register URI schemes on Windows")
		}
		return registerURISchemesWindows(schemes, executablePath)
	case "darwin":
		// Pass executable path to find bundle
		return registerURISchemesMacOS(schemes, executablePath)
	case "linux":
		return registerURISchemesLinux(schemes, executablePath)
	default:
		return fmt.Errorf("unsupported operating system: %s", osType)
	}
}

// registerURISchemesWindows registers multiple URI schemes on Windows
// Takes executablePath as argument
func registerURISchemesWindows(schemes []URIScheme, executablePath string) error {
	// Privilege check should happen *before* calling this function in registerURISchemes

	// Build and execute commands (combine if possible, but separate is fine)
	for _, scheme := range schemes {
		// Use strings.Builder for potentially better performance if many schemes
		var regCommands strings.Builder
		fmt.Fprintf(&regCommands, `reg add HKCR\%s /ve /d "URL:%s Protocol" /f && `, scheme.Name, scheme.Name)
		fmt.Fprintf(&regCommands, `reg add HKCR\%s /v "URL Protocol" /d "" /f && `, scheme.Name)
		fmt.Fprintf(&regCommands, `reg add HKCR\%s\DefaultIcon /ve /d "\"%s\",1" /f && `, scheme.Name, executablePath)
		fmt.Fprintf(&regCommands, `reg add HKCR\%s\shell\open\command /ve /d "\"%s\" \"%%1\"" /f`, scheme.Name, executablePath)

		cmd := exec.Command("cmd", "/c", regCommands.String())
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Provide more specific error context
			return fmt.Errorf("failed to register URI scheme '%s' (Windows): %w. Output: %s. Ensure you are running with Administrator privileges",
				scheme.Name, err, string(output))
		}
		fmt.Printf("URI scheme '%s://' registered successfully on Windows.\n", scheme.Name)
	}
	return nil
}

// findInfoPlist locates Info.plist relative to the executable
func findInfoPlist(executablePath string) (string, error) {
	// IMPORTANT: This is a simplified example. Robust bundle finding can be complex.
	// Consider edge cases like symlinks, different bundle structures.
	baseDir := filepath.Dir(executablePath)
	// Assume standard structure: /path/to/YourApp.app/Contents/MacOS/yourexecutable
	contentsDir := filepath.Dir(baseDir) // Should be Contents/MacOS
	if filepath.Base(contentsDir) != "MacOS" {
		// Maybe it's not in a standard bundle? Or path is different.
		// Fallback or more sophisticated search needed.
		// For now, try a simpler assumption if not found directly:
		// Maybe executable is directly in Contents?
		contentsDir = baseDir
	}

	// Check if we are likely inside Contents
	if filepath.Base(contentsDir) == "Contents" || filepath.Base(filepath.Dir(contentsDir)) == "Contents" {
		bundleDir := filepath.Dir(contentsDir) // Should be YourApp.app
		infoPlistPath := filepath.Join(bundleDir, "Contents", "Info.plist")
		if _, err := os.Stat(infoPlistPath); err == nil {
			return infoPlistPath, nil
		}
	}

	// Fallback: Check near executable (less likely for installed apps)
	altPath := filepath.Join(baseDir, "Info.plist")
	if _, err := os.Stat(altPath); err == nil {
		log.Printf("Warning: Found Info.plist next to executable (%s), not in standard bundle structure.", altPath)
		return altPath, nil
	}

	// Final fallback: Check current dir (original behavior, but unreliable)
	cwdPath := "Info.plist"
	if _, err := os.Stat(cwdPath); err == nil {
		log.Printf("Warning: Found Info.plist in current working directory (%s). This is unreliable for installed applications.", cwdPath)
		return cwdPath, nil
	}

	return "", fmt.Errorf("Info.plist not found relative to executable %s or in CWD. macOS URI registration requires a bundled application with Info.plist", executablePath)
}

// registerURISchemesMacOS registers multiple URI schemes on macOS
// Takes executablePath as argument
func registerURISchemesMacOS(schemes []URIScheme, executablePath string) error {
	// Find the Info.plist file relative to the executable
	infoPlistPath, err := findInfoPlist(executablePath)
	if err != nil {
		return err // Error already explains the issue
	}
	fmt.Printf("Attempting to modify Info.plist at: %s\n", infoPlistPath)

	// TODO: Read existing BundleIdentifier instead of hardcoding?
	// bundleIdentifier, err := readBundleIdentifier(infoPlistPath)
	// if err != nil {
	//     return fmt.Errorf("failed to read bundle identifier from %s: %w", infoPlistPath, err)
	// }
	bundleIdentifier := "com.knirv.verifyer" // Replace with dynamic reading if possible

	// Check if CFBundleURLTypes array exists, add if not
	checkCmd := fmt.Sprintf(`/usr/libexec/PlistBuddy -c "Print :CFBundleURLTypes" "%s" > /dev/null 2>&1 || /usr/libexec/PlistBuddy -c "Add :CFBundleURLTypes array" "%s"`, infoPlistPath, infoPlistPath)
	cmd := exec.Command("bash", "-c", checkCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Warning: Failed to ensure CFBundleURLTypes array exists (may already exist): %v, output: %s", err, string(output))
		// Continue cautiously, subsequent adds might fail if the array wasn't added correctly
	}

	// Build PlistBuddy commands for all schemes
	// Determine the next available index for CFBundleURLTypes
	// Note: This simple index 'i' assumes we are overwriting/adding fresh each time.
	// A more robust approach would read the existing array size.
	for i, scheme := range schemes {
		// Use a single PlistBuddy invocation with multiple commands for efficiency
		plistCommand := fmt.Sprintf(`
            /usr/libexec/PlistBuddy \
                -c "Add :CFBundleURLTypes:%d dict" \
                -c "Add :CFBundleURLTypes:%d:CFBundleURLName string '%s'" \
                -c "Add :CFBundleURLTypes:%d:CFBundleURLSchemes array" \
                -c "Add :CFBundleURLTypes:%d:CFBundleURLSchemes:0 string '%s'" \
                "%s"
        `, i, i, bundleIdentifier, i, i, scheme.Name, infoPlistPath)

		cmd := exec.Command("bash", "-c", plistCommand)
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Check if error is because entry already exists? PlistBuddy might return non-zero.
			// More robust error handling could check output for specific messages.
			log.Printf("Warning/Error executing PlistBuddy for scheme '%s': %v, output: %s", scheme.Name, err, string(output))
			// Decide whether to return error or just warn
			// return fmt.Errorf("failed to register URI scheme '%s' (macOS PlistBuddy): %w, output: %s", scheme.Name, err, string(output))
		} else {
			fmt.Printf("URI scheme '%s://' registration command executed on macOS.\n", scheme.Name)
		}
	}

	fmt.Println("\nIMPORTANT: Info.plist modified. You may need to codesign your application again.")
	fmt.Println("           macOS may require the application to be moved to the /Applications folder for changes to take effect.")
	return nil
}

// registerURISchemesLinux registers multiple URI schemes on Linux
// Takes executablePath as argument
func registerURISchemesLinux(schemes []URIScheme, executablePath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user's home directory: %w", err)
	}

	applicationsDir := filepath.Join(homeDir, ".local/share/applications")
	mimeDir := filepath.Join(homeDir, ".local/share/mime/packages")
	mimeInfoDir := filepath.Join(homeDir, ".local/share/mime") // For update-mime-database

	if err := os.MkdirAll(applicationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create applications directory '%s': %w", applicationsDir, err)
	}
	if err := os.MkdirAll(mimeDir, 0755); err != nil {
		return fmt.Errorf("failed to create mime directory '%s': %w", mimeDir, err)
	}

	// Register each scheme
	for _, scheme := range schemes {
		// 1. Create the .desktop file
		desktopFileName := fmt.Sprintf("knirvchain-verifyer-%s.desktop", scheme.Name) // Unique name per scheme
		desktopFilePath := filepath.Join(applicationsDir, desktopFileName)
		desktopFileContent := fmt.Sprintf(`[Desktop Entry]
Name=KNIRVROUTER Verifyer (%s)
Comment=%s
Exec=%s %%u
Type=Application
NoDisplay=true
MimeType=x-scheme-handler/%s;
`, scheme.Name, scheme.Description, executablePath, scheme.Name) // Added NoDisplay=true as it's a handler

		if err := os.WriteFile(desktopFilePath, []byte(desktopFileContent), 0644); err != nil {
			return fmt.Errorf("failed to create .desktop file '%s' for %s: %w", desktopFilePath, scheme.Name, err)
		}

		// 2. Create the MIME type file
		mimeFileName := fmt.Sprintf("knirvchain-verifyer-%s.xml", scheme.Name) // Unique name per scheme
		mimeFilePath := filepath.Join(mimeDir, mimeFileName)
		mimeFileContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<mime-info xmlns="http://www.freedesktop.org/standards/shared-mime-info">
  <mime-type type="x-scheme-handler/%s">
    <comment>%s URI Handler</comment>
    <glob pattern="%s:*" /> <!-- Use glob pattern -->
  </mime-type>
</mime-info>
`, scheme.Name, scheme.Description, scheme.Name)

		if err := os.WriteFile(mimeFilePath, []byte(mimeFileContent), 0644); err != nil {
			return fmt.Errorf("failed to create MIME file '%s' for %s: %w", mimeFilePath, scheme.Name, err)
		}
		fmt.Printf("URI scheme '%s://' definition files created successfully on Linux.\n", scheme.Name)
	}

	// 3. Update the MIME database
	fmt.Println("Updating MIME database...")
	cmdMime := exec.Command("update-mime-database", mimeInfoDir)
	outputMime, errMime := cmdMime.CombinedOutput()
	if errMime != nil {
		// Don't fail installation, but warn user they might need to run it manually
		log.Printf("Warning: Failed to update MIME database automatically: %v. Output: %s", errMime, string(outputMime))
		log.Printf("         You may need to run 'update-mime-database %s' manually.", mimeInfoDir)
	} else {
		fmt.Println("MIME database updated successfully.")
	}

	// 4. Update desktop database
	fmt.Println("Updating desktop database...")
	cmdDesktop := exec.Command("update-desktop-database", "-q", applicationsDir) // Add -q for quiet
	outputDesktop, errDesktop := cmdDesktop.CombinedOutput()
	if errDesktop != nil {
		// Don't fail installation, but warn user they might need to run it manually
		log.Printf("Warning: Failed to update desktop database automatically: %v. Output: %s", errDesktop, string(outputDesktop))
		log.Printf("         You may need to run 'update-desktop-database %s' manually.", applicationsDir)
	} else {
		fmt.Println("Desktop database updated successfully.")
	}

	return nil // Return nil even if updates failed, but warnings were logged
}

// CheckAdminPrivileges checks if the program is running with admin/root privileges
func CheckAdminPrivileges() bool {
	switch runtime.GOOS {
	case "windows":
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		// More reliable check: attempt to open a restricted resource.
		// net session requires network service running and might fail for other reasons.
		// Alternative: Check if the process token has the Administrators SID. Requires windows-specific API calls.
		return err == nil // If opening succeeds, likely admin. Crude but often works.
	case "linux", "darwin":
		return os.Geteuid() == 0
	default:
		return false
	}
}

// getSchemeList returns a formatted list of schemes for display
func getSchemeList(schemes []URIScheme) string {
	var schemeNames []string
	for _, scheme := range schemes {
		schemeNames = append(schemeNames, scheme.Name+"://")
	}
	return strings.Join(schemeNames, ", ")
}
