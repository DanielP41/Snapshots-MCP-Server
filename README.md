# Dev Environment Snapshots MCP Server

A Model Context Protocol (MCP) server that enables AI assistants to capture, restore, and analyze your local development environment.

It allows an AI to understand the context of your work by inspecting open windows, terminals, git branches, and IDE states, persisting this information into snapshots for later comparison or restoration.

## Features

- **Snapshot Capture**: Records the state of:
  - **Windows**: Position, size, title, and application name.
  - **Git Context**: Branch, repository root, dirty status, and HEAD hash.
  - **Terminals**: Identifies active terminal emulators (PowerShell, CMD, Windows Terminal).
  - **IDEs**: Detects VS Code and JetBrains IDEs, extracting the active project name.
  - **Browsers**: Logs active browser windows (Chrome, Edge, Firefox).
- **Windows Support**: Native, dependency-free implementation using the Win32 API (no CGO required).
- **Persistence**: Stores all metadata in a local SQLite database (`~/.dev-env-snapshots/snapshots.db`).
- **Comparison (Diff)**: Analyzes changes between two snapshots (window differences, context switches).
- **Restoration**: Attempts to move and resize windows back to their captured positions.

## Installation

### Prerequisites

- Go 1.21 or higher.
- Windows (for full functionality).

### Build

Clone the repository and build the server:

```bash
git clone https://github.com/DanielP41/Snapshots-MCP-Server.git
cd Snapshots-MCP-Server
go mod tidy
go build -o dev-env-snapshots.exe ./cmd/server
```

## Usage

### Configuration with Claude Desktop

Add the server to your Claude Desktop configuration file (`%APPDATA%\Claude\claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "dev-snapshots": {
      "command": "c:/path/to/dev-env-snapshots.exe"
    }
  }
}
```

### Available Tools

| Tool               | Description                                    |
|               :--- |                                           :--- |
| `capture_snapshot` | Captures the current state of the environment. |
| `restore_snapshot` | Restores windows to a previous state.          |
| `list_snapshots`   | Lists all saved snapshots.                     |
| `delete_snapshot`  | Deletes a snapshot by ID.                      |
| `diff_snapshots`   | Compares two snapshots.                        |

## Security Note

This server runs locally and inspects your window titles and process names. It does **not** upload data to the cloud; all data is stored locally in your SQLite database.

