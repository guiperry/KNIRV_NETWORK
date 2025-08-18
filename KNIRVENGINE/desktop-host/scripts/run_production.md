# Rebuild frontend
cd gui && npm run build

# Rebuild backend (if needed)
cd .. && go build -o knirv-engine .

# Repackage Electron app
cd electron && npm run pack:linux

# Then run
./dist/linux-unpacked/knirv-engine-desktop

# Note: Vulkan warnings are disabled via app.disableHardwareAcceleration() in main.js
# If you encounter graphics issues, you can re-enable hardware acceleration and use
# command line flags instead (see comments in main.js)