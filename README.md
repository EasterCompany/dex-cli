# Project: ~/EasterCompany/dex-cli/

The Dexter `dex-cli` Command Line Interface tool is an essential requirement
for all Dexter related services and products running on any system.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/eastercompany/dex-cli/main/install.sh | bash
```

**Requirements:** git, go

## Development Notes

This applications source code must be located within `~/EasterCompany/dex-cli`,
also this application must be able to...
Build to `~/Dexter/bin/dex`,
Read configuration files directly within `~/Dexter/config`,
Read and append log files directly within `~/Dexter/logs`,
Read and list model files directly within or nested (within sub-directories) `~/Dexter/models`.
