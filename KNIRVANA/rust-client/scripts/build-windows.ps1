# PowerShell script to build the KNIRVANA Rust client for Windows

Write-Host "Packaging for Windows..."

# Copy the built binary to the packaging directory
Copy-Item "target\release\knirvana_game.exe" "packaging\windows\"

Write-Host "Windows packaging complete."