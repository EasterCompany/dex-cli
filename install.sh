#!/bin/bash
# dex-cli installer
set -e

echo "Installing dex-cli..."
echo ""

# Ensure directories exist
mkdir -p ~/Dexter/bin ~/EasterCompany

# Clone or update repo
if [ -d ~/EasterCompany/dex-cli/.git ]; then
    echo "→ Updating dex-cli..."
    cd ~/EasterCompany/dex-cli
    git pull --ff-only
else
    echo "→ Cloning dex-cli..."
    git clone https://github.com/eastercompany/dex-cli.git ~/EasterCompany/dex-cli
    cd ~/EasterCompany/dex-cli
fi

# Build and install
echo "→ Building..."
go build -o ~/Dexter/bin/dex
chmod +x ~/Dexter/bin/dex

# Add to PATH if not already there
SHELL_RC=""
if [ -n "$BASH_VERSION" ]; then
    SHELL_RC="$HOME/.bashrc"
elif [ -n "$ZSH_VERSION" ]; then
    SHELL_RC="$HOME/.zshrc"
fi

if [ -n "$SHELL_RC" ]; then
    if ! grep -q 'export PATH="$HOME/Dexter/bin:$PATH"' "$SHELL_RC" 2>/dev/null; then
        echo '' >> "$SHELL_RC"
        echo '# dex-cli' >> "$SHELL_RC"
        echo 'export PATH="$HOME/Dexter/bin:$PATH"' >> "$SHELL_RC"
        echo "→ Added ~/Dexter/bin to PATH"
    fi
fi

echo ""
echo "✓ Installed!"
echo ""
echo "Run: source ~/.bashrc (or restart terminal)"
echo "Then: dex help"
echo ""
