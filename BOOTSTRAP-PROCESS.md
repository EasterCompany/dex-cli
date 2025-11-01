# Dexter Bootstrap Process Documentation

This document explains the complete bootstrap installation process for new developers.

## Overview

The bootstrap script (`bootstrap.sh`) provides a **one-command setup** that takes a fresh system and configures the complete Dexter development environment.

## One-Line Install

```bash
curl -fsSL https://raw.githubusercontent.com/EasterCompany/dex-cli/main/bootstrap.sh | bash
```

## What Happens Behind the Scenes

### Phase 1: System Dependency Installation

The script detects your package manager and installs:

**Arch Linux (yay/pacman):**
```bash
yay -S --needed --noconfirm git go python python-pip redis
```

**Debian/Ubuntu (apt):**
```bash
sudo apt-get update
sudo apt-get install -y git golang-go python3 python3-pip python3-venv redis-server
```

**Fedora/RHEL (dnf):**
```bash
sudo dnf install -y git golang python3 python3-pip redis
```

### Phase 2: Tool Verification

Verifies installation and shows versions:
- `git --version`
- `go version`
- `python3 --version`

### Phase 3: GitHub SSH Setup

Checks if GitHub SSH access works:
```bash
ssh -T git@github.com
```

If not configured, provides step-by-step instructions:
1. Generate SSH key: `ssh-keygen -t ed25519 -C "your_email@example.com"`
2. Add to ssh-agent: `eval "$(ssh-agent -s)" && ssh-add ~/.ssh/id_ed25519`
3. Add to GitHub: Copy `~/.ssh/id_ed25519.pub` to github.com/settings/keys

The script waits for you to complete this before continuing.

### Phase 4: Directory Structure Creation

Creates the standard Dexter environment:

```
~/Dexter/
  ├── config/    # Configuration files
  ├── models/    # AI models
  └── bin/       # Installed binaries

~/EasterCompany/
  # (Service repositories will be cloned here)
```

### Phase 5: dex-cli Installation

1. **Clones the dex-cli repository:**
   ```bash
   git clone git@github.com:EasterCompany/dex-cli.git ~/Dexter/dex-cli
   ```

2. **Builds the Go binary:**
   ```bash
   cd ~/Dexter/dex-cli
   go build -o dex
   ```

3. **Installs to ~/Dexter/bin:**
   ```bash
   cp dex ~/Dexter/bin/dex
   chmod +x ~/Dexter/bin/dex
   ```

### Phase 6: Configuration File Setup

Creates `~/Dexter/config/service-map.json` with default service definitions:

```json
{
  "_doc": "Service registry - the source of truth for all Dexter services",
  "service_types": [...],
  "services": {
    "fe": [{"id": "easter.company", ...}],
    "cs": [{"id": "dex-event-service", ...}, ...],
    "be": [{"id": "dex-chat-service", ...}, ...],
    "th": [{"id": "dex-discord-service", ...}],
    "os": [{"id": "redis-cache", ...}]
  }
}
```

**Note:** You'll need to manually create `~/Dexter/config/options.json` with your credentials.

### Phase 7: Python Virtual Environment

Creates and configures Python virtual environment:

```bash
python3 -m venv ~/Dexter/python
source ~/Dexter/python/bin/activate
pip install --upgrade pip
```

Location: `~/Dexter/python/`

### Phase 8: Service Repository Cloning

Runs `dex pull` to clone all service repositories:

```bash
~/Dexter/bin/dex pull
```

This clones:
- `easter.company` (Frontend)
- `dex-event-service` (Core)
- `dex-model-service` (Core)
- `dex-chat-service` (Backend)
- `dex-stt-service` (Backend - Speech-to-Text)
- `dex-tts-service` (Backend - Text-to-Speech)
- `dex-web-service` (Backend - Web interface)
- `dex-discord-service` (Third-party - Discord integration)

All repositories are cloned to `~/EasterCompany/<service-name>`

### Phase 9: Shell Configuration

Adds `~/Dexter/bin` to your PATH by appending to `~/.bashrc` or `~/.zshrc`:

```bash
# Dexter CLI
export PATH="$HOME/Dexter/bin:$PATH"
```

**Important:** You must run `source ~/.bashrc` (or `~/.zshrc`) or restart your terminal for the PATH change to take effect.

## Post-Installation Steps

### 1. Configure Credentials

Create `~/Dexter/config/options.json`:

```json
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
```

### 2. Reload Shell

```bash
source ~/.bashrc  # or ~/.zshrc
```

### 3. Verify Installation

```bash
dex help
dex version
dex pull  # Update all repositories
```

### 4. Start Services

Each service has its own startup process. Refer to individual service READMEs.

## Troubleshooting

### "GitHub SSH not working"

Follow the SSH setup instructions provided by the script:
1. Generate key
2. Add to agent
3. Add public key to GitHub

### "Package manager not supported"

Manually install:
- git
- go (1.20+)
- python3 (3.10+)
- python3-pip
- redis

Then re-run the bootstrap script.

### "dex command not found"

Make sure you've:
1. Run `source ~/.bashrc` or `source ~/.zshrc`
2. Or restart your terminal

### "Permission denied" errors

Some package installations require sudo access. Make sure you have sudo privileges.

## System Requirements

- **OS:** Linux (Arch, Debian, Ubuntu, Fedora, or RHEL-based)
- **RAM:** 4GB minimum, 8GB+ recommended
- **Disk:** 10GB free space minimum
- **Network:** Internet connection for cloning repositories
- **Privileges:** sudo access for package installation

## Security Note

The one-line installer runs a script directly from GitHub. If you're security-conscious, you can:

1. Download and inspect the script first:
   ```bash
   curl -fsSL https://raw.githubusercontent.com/EasterCompany/dex-cli/main/bootstrap.sh > bootstrap.sh
   less bootstrap.sh  # Review the script
   chmod +x bootstrap.sh
   ./bootstrap.sh
   ```

2. Or use the manual installation method from the README.

## What Gets Installed

| Component | Location | Purpose |
|-----------|----------|---------|
| dex CLI | `~/Dexter/bin/dex` | Command-line tool |
| Config files | `~/Dexter/config/` | Service registry and options |
| Models | `~/Dexter/models/` | AI models (downloaded separately) |
| Python venv | `~/Dexter/python/` | Isolated Python environment |
| Service repos | `~/EasterCompany/` | All source code |
| System packages | System-wide | git, go, python, redis |

## Directory Layout After Installation

```
~/
├── Dexter/
│   ├── bin/
│   │   └── dex                    # CLI tool
│   ├── config/
│   │   ├── service-map.json       # Service registry
│   │   └── options.json           # Your credentials (create this)
│   ├── models/                    # AI models
│   ├── python/                    # Python virtual environment
│   │   ├── bin/
│   │   │   ├── python
│   │   │   └── pip
│   │   └── lib/
│   └── dex-cli/                   # dex CLI source code
│       ├── main.go
│       ├── cmd/
│       ├── config/
│       └── git/
│
└── EasterCompany/
    ├── easter.company/            # Frontend
    ├── dex-event-service/         # Core service
    ├── dex-model-service/         # Core service
    ├── dex-chat-service/          # Backend service
    ├── dex-stt-service/           # Backend service
    ├── dex-tts-service/           # Backend service
    ├── dex-web-service/           # Backend service
    └── dex-discord-service/       # Third-party integration
```

## Uninstallation

To completely remove Dexter:

```bash
rm -rf ~/Dexter ~/EasterCompany
```

Then remove the PATH export from your `~/.bashrc` or `~/.zshrc`.

## Contributing

If you encounter issues with the bootstrap script, please:
1. Check the troubleshooting section above
2. Open an issue at https://github.com/EasterCompany/dex-cli/issues
3. Include your OS, shell, and error messages
