# Dexter CLI

Master command-line tool for managing all Dexter services.

## Installation

```bash
cd ~/Dexter/dex-cli
./install.sh
```

Add to your shell configuration (`~/.bashrc` or `~/.zshrc`):

```bash
export PATH="$HOME/Dexter/bin:$PATH"
```

Then reload your shell:

```bash
source ~/.bashrc  # or ~/.zshrc
```

## Usage

### Pull Command

Clones or pulls all Dexter services from GitHub:

```bash
dex pull
```

**Features:**
- Ensures `~/Dexter` and `~/EasterCompany` directory structure exists
- Reads service definitions from `~/Dexter/config/service-map.json`
- Clones repositories that don't exist
- Pulls updates for existing repositories (only if safe)
- Skips repositories with uncommitted changes
- Skips repositories with unpushed commits
- Provides detailed status for each service

**Safety Features:**
- Never pulls if there are uncommitted changes
- Never pulls if local branch is ahead of remote
- Uses `--ff-only` flag to prevent merge conflicts
- Reports issues clearly so you can fix them manually

### Other Commands

```bash
dex help       # Show help
dex version    # Show version
```

## Environment

The CLI expects and enforces this directory layout:

```
~/Dexter/               # Dexter installation root
  ├── bin/              # Installed binaries
  ├── config/           # Centralized configuration
  │   ├── options.json
  │   └── service-map.json
  └── models/           # AI models

~/EasterCompany/        # EasterCompany source code
  ├── dex-event-service/
  ├── dex-model-service/
  ├── dex-chat-service/
  ├── dex-stt-service/
  ├── dex-tts-service/
  ├── dex-web-service/
  ├── dex-discord-service/
  └── easter.company/
```

## Development

### Building

```bash
go build -o dex
```

### Project Structure

```
dex-cli/
├── main.go           # CLI entry point
├── cmd/              # Commands
│   └── pull.go       # Pull command implementation
├── config/           # Configuration management
│   └── config.go     # Config loading and validation
└── git/              # Git operations
    └── git.go        # Git status, clone, pull operations
```

### Adding New Commands

1. Create a new file in `cmd/` (e.g., `cmd/build.go`)
2. Implement the command function
3. Add the command to the switch statement in `main.go`
4. Update this README

## Configuration

All configuration is centralized in `~/Dexter/config/`:

- `service-map.json` - Service registry with repo URLs and ports
- `options.json` - Shared configuration values

The CLI automatically reads these files and uses them to manage services.
