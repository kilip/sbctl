package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/kilip/sbctl/internal/daemon"
	"github.com/spf13/viper"
)

var (
	// Gem Palette
	colorGem   = lipgloss.Color("#8839ef") // Mauve/Purple
	colorCyan  = lipgloss.Color("#11111b") // Crust (Background)
	colorGreen = lipgloss.Color("#40a02b") // Green
	colorGold  = lipgloss.Color("#df8e1d") // Yellow/Gold

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#eff1f5")).
			Background(colorGem).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)

	styleDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4c4f69")).
			Italic(true)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			MarginTop(1)

	styleWarning = lipgloss.NewStyle().
			Foreground(colorGold).
			Bold(true).
			Padding(0, 1)
)

// RunWizard starts the interactive setup wizard.
func RunWizard() error {
	cfg := GetConfig()

	var (
		vaultDir       = cfg.Vault.Dir
		userName       = cfg.Vault.UserName
		userEmail      = cfg.Vault.UserEmail
		gitRepository  = cfg.Vault.GitRepository
		gitEnabled     = cfg.GitSync.Enabled
		installService bool
	)

	// If vaultDir is empty, suggest home/Documents/Obsidian
	if vaultDir == "" {
		home, _ := os.UserHomeDir()
		vaultDir = home + "/Documents/Obsidian"
	}

	fmt.Println(styleTitle.Render(" ✨ SBCTL SETUP WIZARD "))
	fmt.Println(styleDesc.Render("Let's get your Second Brain ready, Pak Bos!"))

	theme := huh.ThemeCharm()

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Vault Path").
				Description("Where is your Obsidian Vault located?").
				Value(&vaultDir).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("vault path cannot be empty")
					}
					return nil
				}),
			huh.NewInput().
				Title("User Name").
				Description("Your name for Git commits.").
				Value(&userName),
			huh.NewInput().
				Title("User Email").
				Description("Your email for Git commits/SSH.").
				Value(&userEmail),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable Git Sync").
				Description("Do you want to enable automatic Git synchronization?").
				Value(&gitEnabled),
			huh.NewInput().
				Title("Git Repository").
				Description("Remote Git repository URL (optional).").
				Value(&gitRepository),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Install Service").
				Description("Do you want to install sbctl as a background service?").
				Value(&installService),
		),
	).WithTheme(theme)

	if err := form.Run(); err != nil {
		return err
	}

	// Update config
	cfg.Vault.Dir = vaultDir
	cfg.Vault.UserName = userName
	cfg.Vault.UserEmail = userEmail
	cfg.Vault.GitRepository = gitRepository
	cfg.GitSync.Enabled = gitEnabled
	cfg.GitSync.Dir = vaultDir
	cfg.GitSync.UserName = userName
	cfg.GitSync.UserEmail = userEmail
	cfg.GitSync.GitRepository = gitRepository

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("\n" + styleSuccess.Render("✨ Config saved successfully!"))

	// SSH Setup
	if gitEnabled && userEmail != "" {
		fmt.Printf("\n🔑 %s\n", lipgloss.NewStyle().Foreground(colorGem).Bold(true).Render("Configuring SSH..."))
		if err := cfg.GetGitSyncSSH(); err != nil {
			fmt.Printf("   %s %v\n", styleWarning.Render("⚠️  Warning:"), err)
		} else {
			fmt.Println("   ✅ SSH configured successfully.")
		}
	}

	// Service Install
	if installService {
		fmt.Printf("\n⚙️  %s\n", lipgloss.NewStyle().Foreground(colorGem).Bold(true).Render("Installing service..."))
		configFile := viper.ConfigFileUsed()
		m, err := daemon.NewManager(cfg.ConfigDir, configFile)
		if err != nil {
			return err
		}
		if err := m.Install(); err != nil {
			fmt.Printf("   %s %v\n", styleWarning.Render("⚠️  Warning:"), err)
		} else {
			fmt.Println("   ✅ Service installed successfully.")
			fmt.Printf("   🚀 You can start it with: %s\n", lipgloss.NewStyle().Foreground(colorCyan).Background(colorGem).Padding(0, 1).Render("sbctl service start"))
		}
	}

	completeMsg := strings.Builder{}
	completeMsg.WriteString("🎉 Setup complete! Happy note-taking.\n")
	completeMsg.WriteString("   Your Second Brain is now being managed by Gem.")

	fmt.Println("\n" + lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(colorGem).
		Padding(1, 2).
		MarginBottom(1).
		Render(completeMsg.String()))

	return nil
}
