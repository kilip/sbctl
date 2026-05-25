# sbctl (Second Brain Controller)

[![Go Version](https://img.shields.io/github/go-mod/go-version/kilip/sbctl)](https://golang.org/doc/devel/release.html)
[![License](https://img.shields.io/github/license/kilip/sbctl)](LICENSE)

`sbctl` is a collection of tools designed to help you manage your Obsidian Vault as a long-term "Second Brain," specifically optimized for use with Personal AI Assistants. It provides automated background synchronization using Git, ensuring your knowledge base is always consistent and up-to-date.

## 🚀 Key Features

- **Automated Git Sync**: Real-time file system watching and synchronization (add, commit, pull, push) with remote repositories.
- **Daemon Mode**: Background process that ensures continuous synchronization without manual intervention.
- **Intelligent Debouncing**: Consolidates multiple rapid file changes into a single sync operation to minimize Git noise.
- **Config Reloading**: Automatically reloads and restarts workers when configuration changes are detected.
- **Cross-Platform Support**: Built in Go to run seamlessly across different operating systems.

## 🛠 Technology Stack

- **Go 1.26.3**: High-performance systems programming.
- **spf13/cobra**: Powerful CLI framework.
- **spf13/viper**: Flexible configuration management.
- **fsnotify/fsnotify**: Cross-platform file system notifications.
- **Task**: Modern task runner for development automation.
- **Lefthook**: Fast and lightweight Git hooks manager.

## 🏗 Project Architecture

The project follows a modular architecture:

- **CLI Layer (`cmd/sbctl`)**: Entry points for all commands, built with Cobra.
- **Daemon (`internal/daemon`)**: Manages background workers, handles signals, and monitors configuration changes.
- **Worker System**: A generic worker interface allowing for extensible background tasks.
- **GitSync Worker (`internal/gitsync`)**: Specialized worker that monitors a directory and performs automated Git operations.
- **Configuration (`internal/config`)**: Centralized management of application settings and state.

## 🚦 Getting Started

### Prerequisites

- **Go 1.26.3** or later.
- **Git** installed and available in your PATH.
- **Task** (optional, but recommended) for running development tasks.

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/kilip/sbctl.git
   cd sbctl
   ```

2. **Setup development environment**:
   ```bash
   task setup
   ```
   *This installs required tools like `lefthook`, `golangci-lint`, and `commitlint`.*

3. **Build the binary**:
   ```bash
   task build
   ```
   The binary will be available in the `bin/` directory.

## 💻 Development Workflow

We use `task` to automate common development operations:

- `task setup`: Install tools and setup git hooks.
- `task fmt`: Format the source code.
- `task lint`: Run static analysis with `golangci-lint`.
- `task test`: Run the test suite with race detection.
- `task build`: Compile the application.
- `task fix`: Run all formatting, tidying, and linting tasks.

### Coding Standards

- Adhere to standard Go idioms and formatting.
- All code must pass `golangci-lint`.
- Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages.

## 🧪 Testing

Run the full test suite to ensure everything is working correctly:

```bash
task test
```

## 📂 Project Structure

```text
.
├── cmd/sbctl/          # Application entry point and CLI commands
├── internal/
│   ├── config/         # Configuration logic and management
│   ├── daemon/         # Background service and worker management
│   ├── gitsync/        # Git-based synchronization logic
│   ├── shared/         # Shared utilities (logging, etc.)
│   └── service/        # Platform-specific service management
├── bin/                # Compiled binaries (generated)
├── skills/             # Agent skills and project memory
└── taskfile.yml        # Development task definitions
```

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. **Fork** the repository.
2. **Create a new branch** for your feature or bugfix.
3. **Write tests** for your changes.
4. **Ensure all checks pass** by running `task fix`.
5. **Submit a Pull Request** with a clear description of your changes.

Please use [Conventional Commits](https://www.conventionalcommits.org/) for your commit messages.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---
*Made by Anthonius with love ❤️ and caffeine ☕*
