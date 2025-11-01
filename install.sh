#!/bin/bash
# Dexter CLI Build & Install Script

set -e

echo "=== Building Dexter CLI ==="

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Build the binary
echo "Building dex binary..."
go build -o dex

# Install to ~/Dexter/bin
echo "Installing to ~/Dexter/bin..."
mkdir -p ~/Dexter/bin
cp dex ~/Dexter/bin/dex
chmod +x ~/Dexter/bin/dex

echo ""
echo "✓ Installation complete!"
echo ""
echo "The 'dex' command has been installed to ~/Dexter/bin/dex"
echo ""
echo "To use 'dex' from anywhere, add this to your ~/.bashrc or ~/.zshrc:"
echo ""
echo "    export PATH=\"\$HOME/Dexter/bin:\$PATH\""
echo ""
echo "Then run: source ~/.bashrc (or ~/.zshrc)"
echo ""
echo "Usage:"
echo "  dex pull       # Clone/pull all Dexter services"
echo "  dex help       # Show help"
echo ""
