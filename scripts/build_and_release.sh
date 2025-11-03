#!/bin/bash
#
# build_and_release.sh - Automated release process for dex-cli
#
# Usage: ./scripts/build_and_release.sh <version>
# Example: ./scripts/build_and_release.sh 1.0.0
#
# This script will:
# 1. Validate the version argument (semantic versioning: major.minor.patch)
# 2. Check that the version is not lower than existing tags
# 3. Checkout and pull main branch
# 4. Run formatting, linting, and tests
# 5. Build the binary with the new version
# 6. Update easter.company/tags/dex-cli.json and latest/dex binary
# 7. Create git tag and push
# 8. Display summary with release URL
#

set -e

# ANSI color codes (matching dex-cli ui/style.go)
COLOR_RED="\033[31m"
COLOR_GREEN="\033[32m"
COLOR_YELLOW="\033[33m"
COLOR_BLUE="\033[34m"
COLOR_CYAN="\033[36m"
COLOR_RESET="\033[0m"

# Paths
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEX_BINARY_PATH="$HOME/Dexter/bin/dex"
EASTER_COMPANY_REPO_PATH="$HOME/EasterCompany/easter.company"
TAGS_JSON_PATH="${EASTER_COMPANY_REPO_PATH}/tags/dex-cli.json"
LATEST_BINARY_PATH="${EASTER_COMPANY_REPO_PATH}/latest/dex"

# Ensure script is run from the repository root
cd "$REPO_ROOT"

# Check if we're in a git repository
if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    echo -e "${COLOR_RED}✗ Error: This script must be run from the root of the dex-cli git repository.${COLOR_RESET}"
    exit 1
fi

#############################################
# ARGUMENT VALIDATION
#############################################

if [ $# -eq 0 ]; then
    echo -e "${COLOR_RED}✗ Error: Version argument required${COLOR_RESET}"
    echo -e "${COLOR_BLUE}- Usage: $0 <version>${COLOR_RESET}"
    echo -e "${COLOR_BLUE}- Example: $0 1.0.0${COLOR_RESET}"
    exit 1
fi

NEW_VERSION="$1"

# Validate semantic version format (major.minor.patch)
if ! [[ "$NEW_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${COLOR_RED}✗ Error: Invalid version format${COLOR_RESET}"
    echo -e "${COLOR_BLUE}- Expected: <major>.<minor>.<patch>${COLOR_RESET}"
    echo -e "${COLOR_BLUE}- Example: 1.0.0 or 2.3.15${COLOR_RESET}"
    echo -e "${COLOR_BLUE}- Received: $NEW_VERSION${COLOR_RESET}"
    exit 1
fi

# Check against existing tags to prevent version downgrade
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
LATEST_TAG_CLEAN=$(echo "$LATEST_TAG" | sed 's/^v//')

# Simple version comparison (works for semantic versioning)
version_compare() {
    local ver1="$1"
    local ver2="$2"

    # Split versions into arrays
    IFS='.' read -ra VER1 <<< "$ver1"
    IFS='.' read -ra VER2 <<< "$ver2"

    # Compare major, minor, patch
    for i in 0 1 2; do
        local v1=${VER1[$i]:-0}
        local v2=${VER2[$i]:-0}

        if [ "$v1" -gt "$v2" ]; then
            return 0  # ver1 > ver2
        elif [ "$v1" -lt "$v2" ]; then
            return 1  # ver1 < ver2
        fi
    done

    return 2  # ver1 == ver2
}

if version_compare "$NEW_VERSION" "$LATEST_TAG_CLEAN"; then
    : # New version is greater, proceed
elif [ $? -eq 2 ]; then
    echo -e "${COLOR_YELLOW}⚠ Warning: Version $NEW_VERSION already exists as tag $LATEST_TAG${COLOR_RESET}"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborting."
        exit 1
    fi
else
    echo -e "${COLOR_RED}✗ Error: Version downgrade not allowed${COLOR_RESET}"
    echo -e "${COLOR_BLUE}- Latest tag: $LATEST_TAG${COLOR_RESET}"
    echo -e "${COLOR_BLUE}- Requested: v$NEW_VERSION${COLOR_RESET}"
    exit 1
fi

#############################################
# SECTION: Downloading
#############################################

echo -e "\n${COLOR_CYAN}=== Downloading ===${COLOR_RESET}"

# Capture old binary info before updating
OLD_VERSION=""
OLD_SIZE=0
if [ -f "$DEX_BINARY_PATH" ]; then
    OLD_VERSION=$($DEX_BINARY_PATH version 2>/dev/null | head -n 1 | awk '{print $1}' || echo "unknown")
    OLD_SIZE=$(stat -f%z "$DEX_BINARY_PATH" 2>/dev/null || stat -c%s "$DEX_BINARY_PATH" 2>/dev/null || echo "0")
fi

# Checkout main
git checkout main
echo -e "  Pulling latest changes..."
git pull --ff-only

#############################################
# SECTION: Building & Installing
#############################################

echo -e "\n${COLOR_CYAN}=== Building & Installing ===${COLOR_RESET}"

# Formatting
echo "Formatting..."
make format

# Linting
echo "Linting..."
make lint

# Testing
echo "Testing..."
make test

# Build with specific version
echo "Building..."
make build-for-release VERSION="v${NEW_VERSION}"

# Capture new binary info
if [ ! -f "$DEX_BINARY_PATH" ]; then
    echo -e "${COLOR_RED}✗ Error: Built binary not found at ${DEX_BINARY_PATH}${COLOR_RESET}"
    exit 1
fi

NEW_VERSION_FULL=$($DEX_BINARY_PATH version 2>/dev/null | head -n 1 | awk '{print $1}')
NEW_SIZE=$(stat -f%z "$DEX_BINARY_PATH" 2>/dev/null || stat -c%s "$DEX_BINARY_PATH" 2>/dev/null)

#############################################
# SECTION: Updating easter.company
#############################################

if [ ! -d "$EASTER_COMPANY_REPO_PATH" ]; then
    echo -e "${COLOR_RED}✗ Error: 'easter.company' repository not found at ${EASTER_COMPANY_REPO_PATH}${COLOR_RESET}"
    exit 1
fi

echo -e "\nUpdating latest version in ${TAGS_JSON_PATH}..."
jq --arg version "${NEW_VERSION_FULL}" '.["dex-cli"].latest = $version' "${TAGS_JSON_PATH}" > tmp.json && mv tmp.json "${TAGS_JSON_PATH}"

echo "Copying binary to ${LATEST_BINARY_PATH}..."
cp "${DEX_BINARY_PATH}" "${LATEST_BINARY_PATH}"

echo "Committing and pushing changes to easter.company..."
(cd "${EASTER_COMPANY_REPO_PATH}" && git add latest/dex tags/dex-cli.json && git commit -m "release: dex-cli ${NEW_VERSION_FULL}" && git push)

#############################################
# SECTION: Git Tag and Release
#############################################

echo -e "\nTagging the repository with v${NEW_VERSION}..."
git tag -a "v${NEW_VERSION}" -m "Release v${NEW_VERSION}"
git push origin "v${NEW_VERSION}"

#############################################
# SECTION: Complete
#############################################

echo -e "\n${COLOR_CYAN}=== Complete ===${COLOR_RESET}"

# Calculate size difference
SIZE_DIFF=$((NEW_SIZE - OLD_SIZE))
if [ $SIZE_DIFF -gt 0 ]; then
    SIZE_COLOR="$COLOR_RED"
    SIZE_INDICATOR="↑"
elif [ $SIZE_DIFF -lt 0 ]; then
    SIZE_COLOR="$COLOR_GREEN"
    SIZE_INDICATOR="↓"
    SIZE_DIFF=$((SIZE_DIFF * -1))
else
    SIZE_COLOR="$COLOR_YELLOW"
    SIZE_INDICATOR="="
fi

# Format bytes
format_bytes() {
    local bytes=$1
    if [ $bytes -lt 1024 ]; then
        echo "${bytes} B"
    elif [ $bytes -lt 1048576 ]; then
        echo "$(awk "BEGIN {printf \"%.1f\", $bytes/1024}") KB"
    elif [ $bytes -lt 1073741824 ]; then
        echo "$(awk "BEGIN {printf \"%.1f\", $bytes/1048576}") MB"
    else
        echo "$(awk "BEGIN {printf \"%.1f\", $bytes/1073741824}") GB"
    fi
}

OLD_SIZE_STR=$(format_bytes $OLD_SIZE)
NEW_SIZE_STR=$(format_bytes $NEW_SIZE)
DIFF_SIZE_STR=$(format_bytes $SIZE_DIFF)

# Get latest version from easter.company to check for trademark
LATEST_VERSION=$(curl -s https://easter.company/tags/dex-cli.json | jq -r '.["dex-cli"].latest' 2>/dev/null || echo "")

# Display version comparison
if [ -n "$OLD_VERSION" ] && [ "$OLD_VERSION" != "unknown" ]; then
    echo -e "${COLOR_BLUE}  Previous version: ${COLOR_RESET}${OLD_VERSION}"
fi

# Check if new version matches latest (it should, since we just updated it)
TRADEMARK=""
if [ "$NEW_VERSION_FULL" = "$LATEST_VERSION" ]; then
    BUILD_YEAR=$(date +"%Y")
    TRADEMARK=" \033[38;5;240m| Easter Company™ © ${BUILD_YEAR}\033[0m"
fi

echo -e "${COLOR_BLUE}  Current version:  ${COLOR_RESET}${NEW_VERSION_FULL}${TRADEMARK}"
echo -e "${COLOR_BLUE}  Latest version:   ${COLOR_RESET}${LATEST_VERSION}${TRADEMARK}"
echo -e "${COLOR_BLUE}  Binary size:      ${COLOR_RESET}${OLD_SIZE_STR} → ${NEW_SIZE_STR}${SIZE_COLOR}(${SIZE_INDICATOR} ${DIFF_SIZE_STR})${COLOR_RESET}"

# Display release URL
RELEASE_URL="https://github.com/EasterCompany/dex-cli/releases/new?tag=v${NEW_VERSION}"
echo -e "\n${COLOR_GREEN}✓ Release process complete!${COLOR_RESET}"
echo -e "${COLOR_BLUE}  Create release: ${COLOR_RESET}${RELEASE_URL}"
echo

