#!/bin/bash
#
# build_and_release.sh - A script to automate the release process for dex-cli.
#
# This script will:
# 1. Ensure it's run from the git repository root.
# 2. Forcefully sync with the 'main' branch, discarding local changes.
# 3. Run all checks and build the project.
# 4. Guide the developer through creating a new release version tag.
# 5. Update the official release information in the 'easter.company' repository.
# 6. Commit and push the new binary and release information.
#

set -e

# --- Improvement: Ensure script is run from the repository root ---
if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    echo "Error: This script must be run from the root of the dex-cli git repository."
    exit 1
fi

echo "WARNING: This script will discard all uncommitted changes and reset your local 'main' branch to match 'origin/main'."
read -p "Are you sure you want to continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborting."
    exit 1
fi

echo "Fetching and resetting to the latest version of main..."
git fetch origin main
git checkout -f main
git reset --hard origin/main

echo "[EXISTING TAGS]:"
git tag --sort=-v:refname | head -n 5 # Show 5 most recent tags

echo ""
echo "Define the version numbers for this release:"
read -p "Enter major version: " MAJOR_VERSION
read -p "Enter minor version (${MAJOR_VERSION}.): " MINOR_VERSION
read -p "Enter patch version (${MAJOR_VERSION}.${MINOR_VERSION}.): " PATCH_VERSION

NEW_VERSION="v${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION}"
echo "Proposed new version tag: ${NEW_VERSION}"

echo "Tagging the repository with ${NEW_VERSION}..."
git tag -a "${NEW_VERSION}" -m "Release ${NEW_VERSION}"

echo "Running checks and building the project..."
make build-for-release VERSION=${NEW_VERSION}

echo "Build successful!"

echo "Successfully tagged ${NEW_VERSION}."
echo "Please visit https://github.com/EasterCompany/dex-cli/releases/new to publish this release."
echo "Use '${NEW_VERSION}' as the tag for the new release."

# --- Automate update of easter.company repository ---

DEX_BINARY_PATH="$HOME/Dexter/bin/dex"
EASTER_COMPANY_REPO_PATH="$HOME/EasterCompany/easter.company"
TAGS_JSON_PATH="${EASTER_COMPANY_REPO_PATH}/tags/dex-cli.json"
LATEST_BINARY_PATH="${EASTER_COMPANY_REPO_PATH}/latest/dex"

if [ ! -f "$DEX_BINARY_PATH" ]; then
    echo "Error: Built binary not found at ${DEX_BINARY_PATH}"
    exit 1
fi

if [ ! -d "$EASTER_COMPANY_REPO_PATH" ]; then
    echo "Error: 'easter.company' repository not found at ${EASTER_COMPANY_REPO_PATH}"
    exit 1
fi

echo "Capturing full version label from the new binary..."
FULL_VERSION_LABEL=$($DEX_BINARY_PATH version 2>/dev/null | head -n 1 | awk '{print $1}')
echo "Full version label: ${FULL_VERSION_LABEL}"

echo "Updating latest version in ${TAGS_JSON_PATH}..."
jq --arg version "${FULL_VERSION_LABEL}" '.latest = $version' "${TAGS_JSON_PATH}" > tmp.json && mv tmp.json "${TAGS_JSON_PATH}"

echo "Copying binary to ${LATEST_BINARY_PATH}..."
cp "${DEX_BINARY_PATH}" "${LATEST_BINARY_PATH}"

echo "The following changes will be committed and pushed to the 'easter.company' repository:"
(cd "${EASTER_COMPANY_REPO_PATH}" && git status --short)

read -p "Proceed with commit and push? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborting commit to 'easter.company'."
    exit 1
fi

echo "Committing and pushing changes to easter.company..."
(cd "${EASTER_COMPANY_REPO_PATH}" && git add latest/dex tags/dex-cli.json && git commit -m "release: dex-cli ${FULL_VERSION_LABEL}" && git push)

echo "Release process complete!"
