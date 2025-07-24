#!/bin/bash

# Release script for Agentic Engine Desktop Application
# This script handles version bumping, building, and releasing

set -e

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to get current version from package.json
get_current_version() {
    node -p "require('$PROJECT_ROOT/electron/package.json').version"
}

# Function to update version in package.json files
update_version() {
    local new_version=$1
    
    print_status "Updating version to $new_version..."
    
    # Update electron package.json
    cd "$PROJECT_ROOT/electron"
    npm version "$new_version" --no-git-tag-version
    
    # Update gui package.json
    cd "$PROJECT_ROOT/gui"
    npm version "$new_version" --no-git-tag-version
    
    print_success "Version updated to $new_version"
}

# Function to build all platforms
build_all_platforms() {
    print_status "Building for all platforms..."
    
    cd "$PROJECT_ROOT"
    
    # Build backend
    print_status "Building Go backend..."
    go build -o agentic-engine .
    
    # Build frontend
    print_status "Building React frontend..."
    cd gui
    npm run build
    
    # Build Electron apps
    print_status "Building Electron applications..."
    cd ../electron
    
    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        npm install
    fi
    
    # Build for all platforms
    npm run build:all
    
    print_success "All platforms built successfully"
}

# Function to create git tag and commit
create_git_tag() {
    local version=$1
    
    print_status "Creating git tag v$version..."
    
    cd "$PROJECT_ROOT"
    
    # Add changed files
    git add electron/package.json gui/package.json
    
    # Commit version bump
    git commit -m "Bump version to $version"
    
    # Create tag
    git tag -a "v$version" -m "Release version $version"
    
    print_success "Git tag v$version created"
}

# Function to push to remote
push_to_remote() {
    local version=$1
    
    print_status "Pushing to remote repository..."
    
    cd "$PROJECT_ROOT"
    
    # Push commits and tags
    git push origin main
    git push origin "v$version"
    
    print_success "Pushed to remote repository"
}

# Function to create GitHub release
create_github_release() {
    local version=$1
    
    if ! command_exists gh; then
        print_warning "GitHub CLI not found. Skipping GitHub release creation."
        print_warning "You can create the release manually at: https://github.com/your-username/agentic-engine/releases"
        return
    fi
    
    print_status "Creating GitHub release..."
    
    cd "$PROJECT_ROOT"
    
    # Create release
    gh release create "v$version" \
        --title "Agentic Engine v$version" \
        --notes "Release notes for version $version" \
        --draft
    
    # Upload artifacts
    print_status "Uploading release artifacts..."
    
    cd electron/dist
    
    # Upload all distribution files
    for file in *.exe *.dmg *.AppImage *.deb *.rpm *.tar.gz; do
        if [ -f "$file" ]; then
            print_status "Uploading $file..."
            gh release upload "v$version" "$file"
        fi
    done
    
    # Upload checksums and build info
    if [ -f "checksums.json" ]; then
        gh release upload "v$version" "checksums.json"
    fi
    
    if [ -f "build-info.json" ]; then
        gh release upload "v$version" "build-info.json"
    fi
    
    print_success "GitHub release created: https://github.com/your-username/agentic-engine/releases/tag/v$version"
}

# Function to validate environment
validate_environment() {
    print_status "Validating environment..."
    
    # Check required tools
    local required_tools=("node" "npm" "go" "git")
    
    for tool in "${required_tools[@]}"; do
        if ! command_exists "$tool"; then
            print_error "$tool is not installed or not in PATH"
            exit 1
        fi
    done
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "Not in a git repository"
        exit 1
    fi
    
    # Check if working directory is clean
    if ! git diff-index --quiet HEAD --; then
        print_error "Working directory is not clean. Please commit or stash changes."
        exit 1
    fi
    
    print_success "Environment validation passed"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [command] [version]"
    echo ""
    echo "Commands:"
    echo "  patch     - Bump patch version (1.0.0 -> 1.0.1)"
    echo "  minor     - Bump minor version (1.0.0 -> 1.1.0)"
    echo "  major     - Bump major version (1.0.0 -> 2.0.0)"
    echo "  version   - Set specific version (e.g., 1.2.3)"
    echo "  build     - Build without version bump"
    echo "  help      - Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 patch              # Bump patch version and release"
    echo "  $0 minor              # Bump minor version and release"
    echo "  $0 version 1.2.3      # Set version to 1.2.3 and release"
    echo "  $0 build              # Build current version without releasing"
}

# Main function
main() {
    local command=${1:-help}
    local version_arg=$2
    
    case $command in
        patch|minor|major)
            validate_environment
            
            current_version=$(get_current_version)
            print_status "Current version: $current_version"
            
            cd "$PROJECT_ROOT/electron"
            new_version=$(npm version "$command" --no-git-tag-version | sed 's/^v//')
            
            # Reset the version change in electron package.json
            git checkout -- package.json
            
            # Update both package.json files properly
            update_version "$new_version"
            
            build_all_platforms
            create_git_tag "$new_version"
            push_to_remote "$new_version"
            create_github_release "$new_version"
            
            print_success "Release $new_version completed successfully!"
            ;;
            
        version)
            if [ -z "$version_arg" ]; then
                print_error "Version argument required for 'version' command"
                show_usage
                exit 1
            fi
            
            validate_environment
            
            update_version "$version_arg"
            build_all_platforms
            create_git_tag "$version_arg"
            push_to_remote "$version_arg"
            create_github_release "$version_arg"
            
            print_success "Release $version_arg completed successfully!"
            ;;
            
        build)
            validate_environment
            build_all_platforms
            print_success "Build completed successfully!"
            ;;
            
        help|--help|-h)
            show_usage
            ;;
            
        *)
            print_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
