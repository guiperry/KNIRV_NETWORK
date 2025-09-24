#!/bin/bash
set -e

# KNIRV Network Release Script
#
# This script automates building binaries, creating GitHub releases,
# and updating portal configuration.
#
# USAGE:
# ./release.sh <command> <version>
#
# COMMANDS:
#   build        - Build all binaries.
#   upload       - Create a GitHub release and upload binaries.
#   update-links - Update portal-links.yaml with the new release URLs.
#   all          - Run all steps: build, upload, update-links.
#
# ARGUMENTS:
#   version      - The git tag for the release (e.g., v1.2.3).

# --- Configuration ---
# Ensure you have the GitHub CLI `gh` installed and authenticated: `gh auth login`
# Ensure you have `yq` installed for updating YAML files: https://github.com/mikefarah/yq
# For cross-compilation, you may need tools like `xgo` or specific toolchains.

COMMAND=$1
VERSION=$2
REPO="knirv-network/cloud-equities" # IMPORTANT: Replace with your actual GitHub repo: owner/repo
PROJECT_ROOT=$(git rev-parse --show-toplevel)
RELEASE_DIR="${PROJECT_ROOT}/dist/releases/${VERSION}"
PORTAL_CONFIG_FILE="${PROJECT_ROOT}/KNIRVGATEWAY/network-website/public/config/portal-links.yaml"

# List of applications to build and release.
# Format: <app_name>:<source_directory>
APPS=(
    "knirvrouter:KNIRVROUTER"
    "knirvana:KNIRVANA"
    "knirvoracle:KNIRVORACLE"
    "knirvwallet:KNIRVWALLET"
    "knirvcontroller:KNIRVCONTROLLER"
    "knirvcli:KNIRVCLI"
)

# --- Helper Functions ---

log() {
    echo "➡️  $1"
}

build_app() {
    local app_name=$1
    local app_dir=$2
    local output_base="${RELEASE_DIR}/${app_name}"

    log "Building ${app_name} from ${app_dir}..."
    cd "${PROJECT_ROOT}/${app_dir}"

    # --- !!! IMPORTANT: REPLACE WITH YOUR BUILD COMMANDS !!! ---
    # This is a placeholder. Your actual build process for Go, Rust, Node.js (pkg),
    # or other languages will go here.
    log "Placeholder build for ${app_name}. Creating dummy files."
    touch "${output_base}-setup.exe"
    touch "${output_base}.dmg"
    touch "${output_base}.AppImage"
    touch "${RELEASE_DIR}/${app_name}_latest.tar.gz" # For knirvoracle/knirvcli
    touch "${RELEASE_DIR}/${app_name}_latest.zip"   # For knirvcli

    if [ "$app_name" == "knirvcontroller" ]; then
        touch "${RELEASE_DIR}/knirvcontroller-android-pwa.zip"
        touch "${RELEASE_DIR}/knirvcontroller-ios-pwa.zip"
    fi

    cd "${PROJECT_ROOT}"
    log "✅ Finished building ${app_name}."
}

# --- Main Commands ---

do_build() {
    log "Starting build for version ${VERSION}..."
    rm -rf "${RELEASE_DIR}"
    mkdir -p "${RELEASE_DIR}"

    for app_info in "${APPS[@]}"; do
        IFS=':' read -r app_name app_dir <<< "$app_info"
        build_app "$app_name" "$app_dir"
    done

    log "🎉 All binaries built in ${RELEASE_DIR}"
}

do_upload() {
    log "Creating GitHub release ${VERSION}..."
    if gh release view "${VERSION}" >/dev/null 2>&1; then
        log "Release ${VERSION} already exists. Skipping creation."
    else
        gh release create "${VERSION}" --repo "$REPO" --title "Release ${VERSION}" --notes "Release notes for version ${VERSION}."
        log "✅ GitHub release ${VERSION} created."
    fi

    log "Uploading assets to release ${VERSION}..."
    gh release upload "${VERSION}" --repo "$REPO" "${RELEASE_DIR}"/* --clobber
    log "🎉 All assets uploaded to release ${VERSION}."
}

do_update_links() {
    log "Updating download links in ${PORTAL_CONFIG_FILE} for version ${VERSION}..."
    if ! command -v yq &> /dev/null; then
        echo "❌ 'yq' is not installed. Please install it from https://github.com/mikefarah/yq"
        exit 1
    fi

    BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

    yq e -i ".downloads.knirvrouter.windows = \"${BASE_URL}/knirvrouter-setup.exe\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvrouter.mac = \"${BASE_URL}/knirvrouter.dmg\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvrouter.linux = \"${BASE_URL}/knirvrouter.AppImage\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvana.windows = \"${BASE_URL}/knirvana-setup.exe\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvana.mac = \"${BASE_URL}/knirvana.dmg\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvana.linux = \"${BASE_URL}/knirvana.AppImage\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvoracle.linux = \"${BASE_URL}/knirvoracle_latest.tar.gz\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvwallet.windows = \"${BASE_URL}/knirvwallet-setup.exe\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvwallet.mac = \"${BASE_URL}/knirvwallet.dmg\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvwallet.linux = \"${BASE_URL}/knirvwallet.AppImage\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvcontroller.windows = \"${BASE_URL}/knirvcontroller-setup.exe\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvcontroller.mac = \"${BASE_URL}/knirvcontroller.dmg\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvcontroller.linux = \"${BASE_URL}/knirvcontroller.AppImage\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvcontroller.android = \"${BASE_URL}/knirvcontroller-android-pwa.zip\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvcontroller.ios = \"${BASE_URL}/knirvcontroller-ios-pwa.zip\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvcli.windows = \"${BASE_URL}/knirvcli_latest.zip\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvcli.mac = \"${BASE_URL}/knirvcli_latest.tar.gz\"" "$PORTAL_CONFIG_FILE"
    yq e -i ".downloads.knirvcli.linux = \"${BASE_URL}/knirvcli_latest.tar.gz\"" "$PORTAL_CONFIG_FILE"

    log "✅ portal-links.yaml has been updated."
}

if [ -z "$COMMAND" ] || [ -z "$VERSION" ]; then echo "Usage: $0 <command> <version>" && exit 1; fi

case "$COMMAND" in
    build) do_build ;;
    upload) do_upload ;;
    update-links) do_update_links ;;
    all) do_build; do_upload; do_update_links ;;
    *) echo "Unknown command: $COMMAND" && exit 1 ;;
esac

log "🚀 Release process for ${COMMAND} completed successfully!"