package core

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

const (
	repoName = "kilip/sbctl"
)

var apiURL = "https://api.github.com/repos/" + repoName + "/releases/latest"
var osExecutable = os.Executable

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func Upgrade() error {
	fmt.Println("Checking for updates...")

	// Fetch latest release
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github API returned status: %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to decode github API response: %w", err)
	}

	latestVersion := release.TagName
	vLatest := latestVersion
	if !strings.HasPrefix(vLatest, "v") {
		vLatest = "v" + vLatest
	}

	vCurrent := Version
	if !strings.HasPrefix(vCurrent, "v") {
		vCurrent = "v" + vCurrent
	}

	if Version == "dev" {
		fmt.Printf("Running development version (dev). Latest is %s. Proceeding with upgrade anyway.\n", latestVersion)
	} else if semver.Compare(vLatest, vCurrent) <= 0 {
		fmt.Printf("You are already using the latest version (%s)\n", Version)
		return nil
	}

	fmt.Printf("Upgrading from %s to %s...\n", Version, latestVersion)

	// Determine asset name
	// Version in goreleaser usually strips the 'v' prefix for {{.Version}}
	versionNum := strings.TrimPrefix(latestVersion, "v")
	expectedNamePrefix := fmt.Sprintf("sbctl-%s-%s-%s", versionNum, runtime.GOOS, runtime.GOARCH)

	var downloadURL string
	var isZip bool

	for _, asset := range release.Assets {
		if strings.HasPrefix(asset.Name, expectedNamePrefix) {
			downloadURL = asset.BrowserDownloadURL
			isZip = strings.HasSuffix(asset.Name, ".zip")
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no suitable asset found for %s %s in release %s", runtime.GOOS, runtime.GOARCH, latestVersion)
	}

	// Download asset
	fmt.Printf("Downloading %s...\n", downloadURL)
	tmpFile, err := os.CreateTemp("", "sbctl-release-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	downloadResp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download release: %w", err)
	}
	defer func() { _ = downloadResp.Body.Close() }()

	if downloadResp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status: %d", downloadResp.StatusCode)
	}

	if _, err := io.Copy(tmpFile, downloadResp.Body); err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}
	_ = tmpFile.Close()

	// Extract binary
	fmt.Println("Extracting binary...")
	tmpBin, err := os.CreateTemp("", "sbctl-bin-*")
	if err != nil {
		return fmt.Errorf("failed to create temp binary file: %w", err)
	}
	tmpBinPath := tmpBin.Name()
	_ = tmpBin.Close() // Will be opened by extraction
	defer func() { _ = os.Remove(tmpBinPath) }()

	if isZip {
		if err := extractZip(tmpFile.Name(), tmpBinPath); err != nil {
			return err
		}
	} else {
		if err := extractTarGz(tmpFile.Name(), tmpBinPath); err != nil {
			return err
		}
	}

	// Replace binary
	executable, err := osExecutable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return fmt.Errorf("failed to eval symlinks for executable: %w", err)
	}

	fmt.Println("Installing new version...")

	// Rename old binary
	oldBin := executable + ".old"
	_ = os.Remove(oldBin) // Ignore error if it doesn't exist
	if err := os.Rename(executable, oldBin); err != nil {
		return fmt.Errorf("failed to rename old binary (you may need to run as administrator/root): %w", err)
	}
	defer func() { _ = os.Remove(oldBin) }() // Clean up old binary

	// Move new binary into place
	if err := copyFile(tmpBinPath, executable); err != nil {
		// Rollback
		_ = os.Rename(oldBin, executable)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Set executable permission
	if err := os.Chmod(executable, 0755); err != nil {
		return fmt.Errorf("failed to set executable permission: %w", err)
	}

	fmt.Printf("Successfully upgraded to %s!\n", latestVersion)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func extractZip(archivePath, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		if f.Name == "sbctl" || f.Name == "sbctl.exe" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer func() { _ = rc.Close() }()

			dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() { _ = dst.Close() }()

			_, err = io.Copy(dst, rc)
			return err
		}
	}
	return fmt.Errorf("binary not found in zip archive")
}

func extractTarGz(archivePath, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Name == "sbctl" {
			dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer func() { _ = dst.Close() }()

			_, err = io.Copy(dst, tr)
			return err
		}
	}
	return fmt.Errorf("binary not found in tar.gz archive")
}
