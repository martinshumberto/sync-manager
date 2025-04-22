# Sync Manager

A robust, efficient cloud storage and synchronization platform designed as a private and customizable alternative to solutions like Google Drive and iCloud.

## Overview

Sync Manager focuses on security, efficiency, and seamless synchronization across multiple devices, utilizing S3-compatible storage providers as a backend. The platform operates with minimal resource usage, making it ideal for background operation without impacting system performance.

## Architecture

The Sync Manager system consists of two main components:

### CLI Component

The Command Line Interface (CLI) is a standalone tool that allows users to:
- Configure synchronization settings
- Add/remove folders to be synced
- Check sync status
- Manage storage provider settings
- Start and stop the agent process

The CLI interacts with a local configuration database to store user preferences and sync settings. You can use the CLI without having the agent running.

### Agent Component

The Agent is a background process that:
- Reads the configuration set by the CLI
- Performs the actual file synchronization
- Monitors folders for changes
- Uploads/downloads files to/from the configured storage provider
- Manages file versioning and conflict resolution

The agent can be started or stopped independently of the CLI. Once started, it will continue synchronizing based on the current configuration until stopped.

### Communication Between Components

When the CLI makes configuration changes while the agent is running, these changes are stored in the shared configuration. The agent will detect these changes and reload its configuration accordingly (for some changes, you may need to explicitly use the CLI to send a reload command to the agent).

## Key Features

- **Automatic Backup**: Schedule and automate backups of selected folders
- **Multi-device Synchronization**: Keep files in sync across devices with intelligent conflict resolution
- **File Versioning**: Track changes and restore previous versions when needed
- **Multiple Storage Backends**: Support for Amazon S3, Google Cloud Storage, MinIO, and more
- **Lightweight Client Agent**: Developed in Go for minimal resource usage
- **Powerful CLI**: Complete management via command line without GUI dependencies
- **Real-time Progress Monitoring**: Clear visibility into backup and sync operations

## Repository Structure

The Sync Manager project follows a monorepo approach with the following structure:

```
sync-manager/
├── agent/                 # Background sync agent process (Go)
├── cli/                   # Command-line interface for configuration (Go)
├── common/                # Shared libraries and utilities
├── docs/                  # Documentation
├── scripts/               # Development and deployment scripts
└── deployment/            # Deployment configurations
```

## Development Setup

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Access to S3-compatible storage (or MinIO for local development)

### Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/martinshumberto/sync-manager.git
   cd sync-manager
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Start development environment (MinIO for local storage):
   ```bash
   make dev-env
   ```

4. Build the components:
   ```bash
   make build
   ```

5. Run the CLI with hot reload (for development):
   ```bash
   make dev-cli
   ```

6. Start the agent separately:
   ```bash
   make run-agent
   ```

## Development Workflow

For rapid development, you can use the hot-reload capability of the CLI:

```bash
# In one terminal - run the CLI with hot reload
make dev-cli

# In another terminal - run the agent
make run-agent
```

Any changes to the CLI code will be automatically recompiled and the CLI will restart with those changes.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 