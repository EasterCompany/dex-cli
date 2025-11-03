# dex-cli

`dex-cli` is a command-line interface for managing Dexter services and the Dexter ecosystem. It provides a unified tool for building, updating, and controlling your Dexter projects.

## Installation

You can install `dex-cli` by running the following script:

```bash
/bin/bash -c "$(curl -fsSL https://easter.company/scripts/install_from_source_dex-cli.sh)"
```

## Usage

Here's a quick overview of the available commands:

*   **`dex update`**: Updates `dex-cli` to the latest version.
*   **`dex build <service|all>`**: Builds a specific Dexter service or all services.
*   **`dex status [service]`**: Checks the health and status of a specific Dexter service or all services.
*   **`dex start <service>`**: Starts a specified Dexter service.
*   **`dex stop <service>`**: Stops a specified Dexter service.
*   **`dex restart <service>`**: Restarts a specified Dexter service.
*   **`dex python [<subcommand>] [args...]`**: Manages Dexter's Python environment or runs Python commands.
*   **`dex bun [args...]`**: Proxies commands to the system's `bun` executable.
*   **`dex bunx [args...]`**: Proxies commands to the system's `bunx` executable.

## Contributing

`dex-cli` is an open-source project, and contributions are welcome! If you have ideas for new features, bug fixes, or improvements, please feel free to contribute.

**Main Contact/Contributor:**
*   GitHub: [github.com/oceaster](https://github.com/oceaster)
*   Email: owen@easter.company