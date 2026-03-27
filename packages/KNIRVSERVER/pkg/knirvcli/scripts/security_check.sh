#!/bin/bash

# Security check script for KNIRVCHAIN CLI
# This script performs security checks on the codebase

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Print header
print_header() {
    echo -e "\n${YELLOW}=======================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}=======================================${NC}\n"
}

# Check if a command exists
check_command() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}Error: $1 is not installed or not in PATH${NC}"
        echo -e "Please install $1 to run this check"
        return 1
    fi
    return 0
}

# Get the project root directory
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$PROJECT_ROOT"

print_header "Running security checks for KNIRVCHAIN CLI"

# Check for Go vulnerabilities
print_header "Checking for Go vulnerabilities"
if check_command govulncheck; then
    govulncheck ./...
else
    echo -e "${YELLOW}Skipping Go vulnerability check (govulncheck not installed)${NC}"
    echo -e "Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

# Run static code analysis
print_header "Running static code analysis"
if check_command staticcheck; then
    staticcheck ./...
else
    echo -e "${YELLOW}Skipping static code analysis (staticcheck not installed)${NC}"
    echo -e "Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"
fi

# Check for hardcoded secrets
print_header "Checking for hardcoded secrets"
if check_command gitleaks; then
    gitleaks detect --no-git
else
    echo -e "${YELLOW}Skipping hardcoded secrets check (gitleaks not installed)${NC}"
    echo -e "Install gitleaks from: https://github.com/zricethezav/gitleaks"
    
    # Fallback to grep for basic secret detection
    echo -e "Performing basic secret detection with grep..."
    
    # Look for potential API keys, tokens, passwords
    grep -r -E "(api[_-]?key|token|password|secret|credential)[\"']?\s*[:=]\s*[\"'][A-Za-z0-9_\-]{16,}[\"']" \
        --include="*.go" --include="*.json" --include="*.yaml" --include="*.yml" \
        . || echo -e "${GREEN}No obvious hardcoded secrets found${NC}"
fi

# Check for insecure cryptographic algorithms
print_header "Checking for insecure cryptographic algorithms"
echo "Searching for MD5 usage..."
grep -r --include="*.go" "md5\." . || echo -e "${GREEN}No MD5 usage found${NC}"

echo "Searching for SHA1 usage..."
grep -r --include="*.go" "sha1\." . || echo -e "${GREEN}No SHA1 usage found${NC}"

echo "Searching for weak ciphers..."
grep -r --include="*.go" -E "(des\.New|rc4\.New)" . || echo -e "${GREEN}No weak cipher usage found${NC}"

# Check for proper error handling
print_header "Checking for proper error handling"
echo "Searching for ignored errors..."
grep -r --include="*.go" -E "[^_] = .*\(.*\)$" . | grep -v "_ = " | grep -v "if err " || echo -e "${GREEN}No obvious ignored errors found${NC}"

# Check for proper input validation
print_header "Checking for proper input validation"
echo "Searching for potential SQL injection points..."
grep -r --include="*.go" -E "db\.(Query|Exec)\(.*\+.*\)" . || echo -e "${GREEN}No obvious SQL injection points found${NC}"

echo "Searching for potential command injection points..."
grep -r --include="*.go" -E "exec\.Command\(.*\+.*\)" . || echo -e "${GREEN}No obvious command injection points found${NC}"

# Check for proper file permissions
print_header "Checking for proper file permissions"
echo "Searching for insecure file permission settings..."
grep -r --include="*.go" -E "os\.(Create|OpenFile|Mkdir|MkdirAll).*0[0-7]{3}" . | grep -v "0[6-7][0-4][0-4]" || echo -e "${GREEN}No obviously insecure file permissions found${NC}"

print_header "Security checks completed!"