package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kilip/sbctl/internal/config"
	"github.com/kilip/sbctl/internal/daemon"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the health of your sbctl environment",
	Long:  `Check dependencies, configuration, and vault integrity to ensure sbctl can run correctly.`,
	Run: func(cmd *cobra.Command, args []string) {
		runDoctor()
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor() {
	fmt.Println("🩺 Running sbctl doctor...")
	fmt.Println()

	allPassed := true

	// 1. Check Dependencies
	fmt.Println("Checking Dependencies:")
	if !checkDependency("git", "--version") {
		allPassed = false
	}
	if !checkDependency("gh", "--version") {
		allPassed = false
	}
	fmt.Println()

	// 2. Check Configuration
	cfg := config.GetConfig()
	fmt.Println("Checking Configuration:")
	if cfg.Vault.Dir == "" {
		fmt.Println("  ❌ vault.dir is not set")
		allPassed = false
	} else {
		fmt.Printf("  ✅ vault.dir: %s\n", cfg.Vault.Dir)
	}

	if cfg.Vault.UserName == "" {
		fmt.Println("  ❌ vault.user_name is not set")
		allPassed = false
	}
	if cfg.Vault.UserEmail == "" {
		fmt.Println("  ❌ vault.user_email is not set")
		allPassed = false
	}
	fmt.Println()

	// 3. Check Vault Integrity
	if cfg.Vault.Dir != "" {
		fmt.Println("Checking Vault Integrity:")
		info, err := os.Stat(cfg.Vault.Dir)
		if err != nil {
			fmt.Printf("  ❌ vault directory not found: %v\n", err)
			allPassed = false
		} else if !info.IsDir() {
			fmt.Println("  ❌ vault.dir path exists but is not a directory")
			allPassed = false
		} else {
			fmt.Println("  ✅ vault directory exists")

			// Check .obsidian
			if _, err := os.Stat(filepath.Join(cfg.Vault.Dir, ".obsidian")); err != nil {
				fmt.Println("  ⚠️  .obsidian folder not found (is this an Obsidian vault?)")
			} else {
				fmt.Println("  ✅ .obsidian folder found")
			}

			// Check git repo
			if _, err := os.Stat(filepath.Join(cfg.Vault.Dir, ".git")); err != nil {
				fmt.Println("  ❌ git repository not initialized in vault directory")
				allPassed = false
			} else {
				fmt.Println("  ✅ git repository initialized")

				// Check remote if configured
				if cfg.Vault.GitRepository != "" {
					cmd := exec.Command("git", "remote", "get-url", "origin")
					cmd.Dir = cfg.Vault.Dir
					out, err := cmd.CombinedOutput()
					if err != nil {
						fmt.Printf("  ❌ remote 'origin' not found: %v\n", err)
						allPassed = false
					} else {
						remote := strings.TrimSpace(string(out))
						if remote != cfg.Vault.GitRepository {
							fmt.Printf("  ❌ remote 'origin' mismatch: expected %s, got %s\n", cfg.Vault.GitRepository, remote)
							allPassed = false
						} else {
							fmt.Printf("  ✅ remote 'origin' matches configuration: %s\n", remote)
						}
					}
				}
			}
		}
		fmt.Println()
	}

	// 4. Check GitHub Auth
	fmt.Println("Checking GitHub Authentication:")
	cmdGh := exec.Command("gh", "auth", "status")
	if out, err := cmdGh.CombinedOutput(); err != nil {
		fmt.Printf("  ❌ github auth status check failed: %v\n", err)
		fmt.Println("     Run 'gh auth login' to authenticate.")
		allPassed = false
	} else {
		fmt.Println("  ✅ github authentication OK")
		// Log a bit of the output for clarity
		lines := strings.Split(string(out), "\n")
		if len(lines) > 0 {
			fmt.Printf("     %s\n", strings.TrimSpace(lines[0]))
		}
	}
	fmt.Println()

	// 5. Check SSH and Signing
	fmt.Println("Checking SSH and Git Signing:")
	checkSSHAndSigning(cfg, cfg.Vault.Dir)
	fmt.Println()
	// 6. Check Service Status
	fmt.Println("Checking Background Service:")
	configFile := viper.ConfigFileUsed()
	configDir := filepath.Dir(configFile)
	m, err := daemon.NewManager(configDir, configFile)
	if err != nil {
		fmt.Printf("  ❌ failed to initialize service manager: %v\n", err)
	} else {
		pidPath := filepath.Join(configDir, "sbctl.pid")
		if _, err := os.Stat(pidPath); err != nil {
			fmt.Println("  ⚠️  Background service is not running (stopped)")
		} else {
			// Try to see if it's actually running
			fmt.Print("  ")
			_ = m.Status()
		}
	}
	fmt.Println()

	if allPassed {
		fmt.Println("✨ Everything looks good! Your Second Brain is ready.")
	} else {
		fmt.Println("❌ Some issues were found. Please fix them to ensure sbctl works correctly.")
		os.Exit(1)
	}
}

func checkSSHAndSigning(cfg *config.Config, vaultDir string) bool {
	passed := true
	sshDir := filepath.Join(cfg.ConfigDir, ".ssh")
	keyPath := filepath.Join(sshDir, "id_ed25519")

	// 1. Check SSH Key
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		fmt.Println("  ⚠️  Custom SSH key not found at .sbctl/.ssh/id_ed25519")
		fmt.Println("     Run 'sbctl sync ssh' to configure it.")
	} else {
		fmt.Println("  ✅ Custom SSH key found")
	}

	// 2. Check Git Config for SSH and Signing
	if vaultDir != "" {
		if _, err := os.Stat(filepath.Join(vaultDir, ".git")); err == nil {
			// Check core.sshCommand
			out, err := runGitInDir(vaultDir, "config", "core.sshCommand")
			if err != nil || out == "" {
				fmt.Println("  ⚠️  Git core.sshCommand is not configured for this vault")
			} else {
				fmt.Printf("  ✅ Git core.sshCommand is configured: %s\n", out)
			}

			// Check gpg.format
			out, err = runGitInDir(vaultDir, "config", "gpg.format")
			if err == nil && out == "ssh" {
				fmt.Println("  ✅ Git gpg.format is set to 'ssh'")
			} else {
				fmt.Println("  ⚠️  Git gpg.format is not set to 'ssh'")
			}

			// Check commit.gpgsign
			out, err = runGitInDir(vaultDir, "config", "commit.gpgsign")
			if err == nil && out == "true" {
				fmt.Println("  ✅ Git commit.gpgsign is enabled")
			} else {
				fmt.Println("  ⚠️  Git commit.gpgsign is not enabled")
			}
		}
	}

	return passed
}

func runGitInDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func checkDependency(name string, args ...string) bool {
	cmd := exec.Command(name, args...)
	if err := cmd.Run(); err != nil {
		fmt.Printf("  ❌ %s is not installed or not in PATH\n", name)
		return false
	}
	fmt.Printf("  ✅ %s is installed\n", name)
	return true
}
