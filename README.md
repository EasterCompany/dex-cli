# dex-cli

Fast, simple command-line tool for managing Dexter services.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/eastercompany/dex-cli/main/install.sh | bash
```

**Requirements:** git, go

## Usage

```bash
dex pull       # Clone/pull all services
dex help       # Show help
dex version    # Show version
```

## What It Does

- Manages `~/Dexter` (binaries, configs, models)
- Manages `~/EasterCompany` (source code)
- Clones and updates all dex-* services
- Safe git operations (won't pull with uncommitted changes)

## Directory Structure

```
~/Dexter/           # Installation root
  ├── bin/          # Compiled binaries
  ├── config/       # Configuration files
  └── models/       # AI models

~/EasterCompany/    # Source code
  ├── dex-cli/
  ├── dex-*-service/
  └── ...
```

## Development

**Build from source:**
```bash
cd ~/EasterCompany/dex-cli
go build -o ~/Dexter/bin/dex
```

**Project structure:**
```
main.go       # Entry point
cmd/          # Commands
config/       # Config management
git/          # Git operations
```

## License

MIT
