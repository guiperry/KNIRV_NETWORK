#!/bin/bash
set -e

echo "🚀 Building with Expo Application Services (EAS)..."

# Color codes for output
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Default values
PLATFORM="all"
PROFILE="production"

# Parse command line arguments
for arg in "$@"
do
    case $arg in
        --platform=*)
        PLATFORM="${arg#*=}"
        shift
        ;;
        --profile=*)
        PROFILE="${arg#*=}"
        shift
        ;;
    esac
done

print_status "Building web assets for the native wrapper..."
npm run build

print_status "Syncing web assets to Capacitor (iOS and Android)..."
npx cap sync

print_status "Starting EAS Build for platform: ${YELLOW}$PLATFORM${NC} and profile: ${YELLOW}$PROFILE${NC}"
echo ""

print_status "Executing EAS Build..."
npx eas build --platform "$PLATFORM" --profile "$PROFILE"

print_success "✅ EAS Build process initiated successfully!"
echo "Monitor the build progress in your EAS dashboard: https://expo.dev/builds"