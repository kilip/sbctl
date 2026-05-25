package gitsync

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// ConfigureSSH generates a native ED25519 SSH key, uploads it to GitHub, and configures the git repo.
func ConfigureSSH(configDir, vaultDir, userEmail string) error {
	// 1. Check gh cli
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		return fmt.Errorf("gh cli not found or not authenticated. Please run 'gh auth login'")
	}

	if vaultDir == "" {
		return fmt.Errorf("vault directory is not configured")
	}
	if userEmail == "" {
		return fmt.Errorf("vault user_email is not configured")
	}

	// 2. Setup SSH directory and key path
	sshDir := filepath.Join(configDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create ssh directory: %w", err)
	}

	keyPath := filepath.Join(sshDir, "id_ed25519")
	pubKeyPath := keyPath + ".pub"

	// 3. Generate key natively if not exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		fmt.Printf("Generating new SSH key at %s...\n", keyPath)
		if err := generateEd25519Key(keyPath, pubKeyPath, userEmail); err != nil {
			return fmt.Errorf("failed to generate native ssh key: %w", err)
		}
	} else {
		fmt.Printf("SSH key already exists at %s\n", keyPath)
	}

	// 4. Upload to GitHub via gh cli
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	title := fmt.Sprintf("sbctl-%s", hostname)

	fmt.Printf("Uploading SSH key to GitHub with title '%s'...\n", title)
	ghCmd := exec.Command("gh", "ssh-key", "add", pubKeyPath, "--title", title)
	if out, err := ghCmd.CombinedOutput(); err != nil {
		outStr := string(out)
		if strings.Contains(outStr, "already in use") || strings.Contains(outStr, "already exists") {
			fmt.Println("Warning: SSH key already exists in GitHub account.")
		} else {
			return fmt.Errorf("failed to upload ssh key: %w\n%s", err, outStr)
		}
	} else {
		fmt.Println("Successfully uploaded SSH key to GitHub.")
	}

	fmt.Printf("Uploading SSH signing key to GitHub with title '%s'...\n", title+"-signing")
	ghSigningCmd := exec.Command("gh", "ssh-key", "add", pubKeyPath, "--title", title+"-signing", "--type", "signing")
	if out, err := ghSigningCmd.CombinedOutput(); err != nil {
		outStr := string(out)
		if strings.Contains(outStr, "already in use") || strings.Contains(outStr, "already exists") {
			fmt.Println("Warning: SSH signing key already exists in GitHub account.")
		} else {
			return fmt.Errorf("failed to upload ssh signing key: %w\n%s", err, outStr)
		}
	} else {
		fmt.Println("Successfully uploaded SSH signing key to GitHub.")
	}

	// 5. Configure local git repository
	fmt.Printf("Configuring local git repository in %s...\n", vaultDir)

	// Ensure git repo exists before configuring
	if _, err := os.Stat(filepath.Join(vaultDir, ".git")); os.IsNotExist(err) {
		fmt.Println("Git repository not found. Initializing...")
		initCmd := exec.Command("git", "init")
		initCmd.Dir = vaultDir
		if err := initCmd.Run(); err != nil {
			return fmt.Errorf("failed to init git repository: %w", err)
		}
	}

	sshCommand := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", keyPath)
	configs := [][]string{
		{"core.sshCommand", sshCommand},
		{"gpg.format", "ssh"},
		{"user.signingkey", pubKeyPath},
		{"commit.gpgsign", "true"},
	}

	for _, cfg := range configs {
		gitCfgCmd := exec.Command("git", "config", cfg[0], cfg[1])
		gitCfgCmd.Dir = vaultDir
		if out, err := gitCfgCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to configure git %s: %w\n%s", cfg[0], err, string(out))
		}
	}

	fmt.Println("Successfully configured vault repository to use custom SSH key and signing.")
	return nil
}

// generateEd25519Key creates a new ED25519 keypair and writes it to the specified paths.
func generateEd25519Key(privateKeyPath, publicKeyPath, comment string) error {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	// Encode Private Key to OpenSSH format
	privBlock, err := ssh.MarshalPrivateKey(privKey, comment)
	if err != nil {
		return err
	}

	privFile, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = privFile.Close() }()

	if err := pem.Encode(privFile, privBlock); err != nil {
		return err
	}

	// Encode Public Key
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)
	// Add comment to public key
	pubKeyStr := strings.TrimSpace(string(pubKeyBytes)) + " " + comment + "\n"

	if err := os.WriteFile(publicKeyPath, []byte(pubKeyStr), 0644); err != nil {
		return err
	}

	return nil
}
