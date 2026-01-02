
# Dexter CLI


**A unified command-line interface for managing Dexter services and development workflows**

Dexter CLI (`dex`) is the central management tool for the Dexter ecosystem. It provides developers and users with a streamlined interface for building, deploying, monitoring, and interacting with all Dexter services. Whether you're developing new features, deploying updates, or checking service health, `dex` handles the complexity of multi-service orchestration with simple, intuitive commands.

The CLI manages the complete lifecycle of Dexter services—from source code compilation and testing to systemd integration and version control—ensuring consistent, reproducible deployments across development and production environments.

**Platform Support:** Linux (Arch/Debian/Ubuntu)

## Installation

### Install Pre-built Binary (Recommended)

You can install `dex-cli` by downloading the pre-built binary and running the installation script:

```bash
/bin/bash -c "$(curl -fsSL https://easter.company/scripts/dex.sh)"
```

This method is **recommended for most users** as it's faster and doesn't require a Go development environment.

### Install from Source (Advanced)

You can install `dex-cli` from source by running the following script:

```bash
/bin/bash -c "$(curl -fsSL https://easter.company/scripts/install_from_source_dex-cli.sh)"
```

This method is **not recommended** for most users, _unless you intend of modifying and/or contributing to the dex-cli project_.

## Commands & Usage

The `dex` command provides a comprehensive set of tools for managing services, development workflows, and system operations.

### Version & Help

```bash
dex version                 # Display CLI version information
dex help                    # Show available commands and usage
```

### Service Management

```bash
dex status                  # Check status of all services
dex status <service>        # Check status of specific service
dex start                   # Start all manageable services
dex stop                    # Stop all manageable services
dex restart                 # Restart all manageable services
dex logs <service>          # View service logs
dex logs <service> -f       # Follow service logs in real-time
```

### Development Commands

```bash
dex build                   # Build services with uncommitted changes (patch increment)
dex build patch             # Build services with patch version increment
dex build minor             # Build all services with minor version increment
dex build major             # Build all services with major version increment
dex build -f                # Force rebuild all services without version increment
dex test                    # Run tests for all services
```

### Service Installation

```bash
dex add <service>           # Install a service from easter.company
dex remove <service>        # Uninstall a service
dex update                  # Update all services to latest versions
```

### System Utilities

```bash
dex system                  # Show system info and manage packages
dex config <service>        # Show service configuration
dex cache                   # Manage local cache
dex cache clear             # Clear local cache
dex cache list              # List cache contents
```

### Proxy Commands

Direct access to underlying tools:

```bash
dex python <args>           # Access Python virtual environment
dex bun <args>              # Access system Bun executable
dex bunx <args>             # Access system Bunx executable
dex ollama <args>           # Access system Ollama executable
```

### Service-Specific Commands

```bash
dex event <args>            # Interact with event service
dex discord <args>          # Interact with discord service
```

## Additional Resources

For additional, up-to-date information and documentation about **Dexter** and **Dex CLI**, visit [easter.company/dexter](https://easter.company/dexter).
