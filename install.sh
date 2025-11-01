#!/bin/bash
# dex-cli installer
set -e

echo "Installing dex-cli..."
echo ""

# Ensure directories exist
mkdir -p ~/Dexter/bin ~/EasterCompany

# Check for existing config and prompt user
if [ -d ~/Dexter/config ] && [ -f ~/Dexter/config/service-map.json ]; then
    echo "⚠ Existing config found at ~/Dexter/config/"
    echo ""
    read -p "Keep existing config? (y/n): " -n 1 -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "→ Backing up old config to ~/Dexter/config.backup..."
        rm -rf ~/Dexter/config.backup
        mv ~/Dexter/config ~/Dexter/config.backup
        echo "→ Will create fresh config"
        CREATE_CONFIG=true
    else
        echo "→ Keeping existing config"
        CREATE_CONFIG=false
    fi
else
    echo "→ No existing config found"
    CREATE_CONFIG=true
fi

mkdir -p ~/Dexter/config

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

# Create default config if needed
if [ "$CREATE_CONFIG" = true ]; then
    echo "→ Creating default config files..."

    # Create service-map.json
    cat > ~/Dexter/config/service-map.json <<'EOF'
{
  "_doc": "Service registry - the source of truth for all Dexter services",
  "service_types": [
    {
      "type": "fe",
      "label": "Frontend Application",
      "description": "HTML/CSS/JS/TS applications",
      "min_port": 8000,
      "max_port": 8099
    },
    {
      "type": "cs",
      "label": "Core Service",
      "description": "Foundational resource intensive services",
      "min_port": 8100,
      "max_port": 8199
    },
    {
      "type": "be",
      "label": "Backend Service",
      "description": "Backend services that can drop off and reconnect",
      "min_port": 8200,
      "max_port": 8299
    },
    {
      "type": "th",
      "label": "Third party integrations",
      "description": "Third party application interfaces",
      "min_port": 8300,
      "max_port": 8399
    },
    {
      "type": "os",
      "label": "Other Service",
      "description": "Other services (redis, etc)",
      "min_port": -1,
      "max_port": -1
    }
  ],
  "services": {
    "fe": [
      {
        "id": "easter.company",
        "source": "~/EasterCompany/easter.company",
        "repo": "git@github.com:eastercompany/eastercompany.github.io",
        "addr": "https://easter.company",
        "socket": "wss://easter.company"
      }
    ],
    "cs": [
      {
        "id": "dex-event-service",
        "source": "~/EasterCompany/dex-event-service",
        "repo": "git@github.com:eastercompany/dex-event-service",
        "addr": "http://127.0.0.1:8100/",
        "socket": "ws://127.0.0.1:8100/"
      },
      {
        "id": "dex-model-service",
        "source": "~/EasterCompany/dex-model-service",
        "repo": "git@github.com:eastercompany/dex-model-service",
        "addr": "http://127.0.0.1:8101/",
        "socket": "ws://127.0.0.1:8101/"
      }
    ],
    "be": [
      {
        "id": "dex-chat-service",
        "source": "~/EasterCompany/dex-chat-service",
        "repo": "git@github.com:eastercompany/dex-chat-service",
        "addr": "http://127.0.0.1:8200/",
        "socket": "ws://127.0.0.1:8200/"
      },
      {
        "id": "dex-stt-service",
        "source": "~/EasterCompany/dex-stt-service",
        "repo": "git@github.com:eastercompany/dex-stt-service",
        "addr": "http://127.0.0.1:8201/",
        "socket": "ws://127.0.0.1:8201/"
      },
      {
        "id": "dex-tts-service",
        "source": "~/EasterCompany/dex-tts-service",
        "repo": "git@github.com:eastercompany/dex-tts-service",
        "addr": "http://127.0.0.1:8202/",
        "socket": "ws://127.0.0.1:8202/"
      },
      {
        "id": "dex-web-service",
        "source": "~/EasterCompany/dex-web-service",
        "repo": "git@github.com:eastercompany/dex-web-service",
        "addr": "http://127.0.0.1:8203/",
        "socket": "ws://127.0.0.1:8203/"
      }
    ],
    "th": [
      {
        "id": "dex-discord-service",
        "source": "~/EasterCompany/dex-discord-service",
        "repo": "git@github.com:eastercompany/dex-discord-service",
        "addr": "http://127.0.0.1:8300/",
        "socket": "ws://127.0.0.1:8300/"
      }
    ],
    "os": [
      {
        "id": "redis-cache",
        "source": "",
        "repo": "",
        "addr": "http://127.0.0.1:6379/",
        "socket": "ws://127.0.0.1:6379/"
      }
    ]
  }
}
EOF

    # Create options.json template
    cat > ~/Dexter/config/options.json <<'EOF'
{
  "python": {
    "version": 3.13,
    "venv": "~/Dexter/python",
    "bin": "~/Dexter/python/bin/python",
    "pip": "~/Dexter/python/bin/python -m pip"
  },
  "discord": {
    "token": "YOUR_DISCORD_BOT_TOKEN",
    "server_id": "YOUR_SERVER_ID",
    "debug_channel_id": "YOUR_DEBUG_CHANNEL_ID"
  },
  "redis": {
    "password": "",
    "db": 0
  },
  "command_permissions": {
    "default_level": 0,
    "allowed_roles": [],
    "user_whitelist": []
  }
}
EOF

    echo "✓ Created default configs"
    echo "  → Edit ~/Dexter/config/options.json with your credentials"
fi

echo ""
echo "✓ Installed!"
echo ""
echo "Run: source ~/.bashrc (or restart terminal)"
echo "Then: dex help"
echo ""
