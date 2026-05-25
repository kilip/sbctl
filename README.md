# sbctl (Second Brain Controller) 🧠

[![Go Version](https://img.shields.io/github/go-mod/go-version/kilip/sbctl)](https://golang.org/doc/devel/release.html)
[![License](https://img.shields.io/github/license/kilip/sbctl)](LICENSE)
[![codecov](https://img.shields.io/codecov/c/github/kilip/sbctl)](https://codecov.io/gh/kilip/sbctl)

**sbctl** is a collection of tools to help your Personal AI Assistant to manage your obsidian long term `Second Brain` vault and projects.

---

## ✨ Why use sbctl?

- **Zero-Friction Automation**: Your notes are synced to Git whenever changes are detected. No more manual `git commit` or `git push`.
- **Lightweight & Efficient**: Built with Go, it runs silently in the background with minimal impact on your battery and CPU.
- **Easy Setup**: Includes an interactive wizard to get you up and running in minutes.
- **AI-Ready**: Specifically designed to keep your knowledge base consistent for indexing by AI Assistants.

---

## 🚀 Quick Installation

Open your Terminal (or PowerShell on Windows) and run the command below:

### Linux / macOS
```bash
curl -fsSL https://raw.githubusercontent.com/kilip/sbctl/main/scripts/install.sh | bash
```

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/kilip/sbctl/main/scripts/install.ps1 | iex
```

---

## 🛠 Getting Started

Once installed, follow these simple steps:

1.  **Initial Setup**:
    Connect your local notes folder to GitHub:
    ```bash
    sbctl setup
    ```
    *Just follow the on-screen instructions.*

2.  **Health Check**:
    Ensure everything is configured correctly:
    ```bash
    sbctl doctor
    ```

3.  **Start Syncing**:
    Install and start the background service:
    ```bash
    sbctl service install
    ```

4.  **Monitor Status**:
    Check what the sync service is doing:
    ```bash
    sbctl status
    ```

---

## 💡 Key Commands

- `sbctl info`: View detailed configuration and worker status.
- `sbctl restart`: Restart the synchronization service.
- `sbctl sync`: Force a manual synchronization right now.

---

## 🛡 Security

Your GitHub tokens and SSH keys are stored securely using your operating system's native credential store. **sbctl** never stores your passwords in plain text.

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---
*Built by Anthonius with ❤️ and ☕*
