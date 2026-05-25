package core

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.txt")
	dstPath := filepath.Join(tmpDir, "dst.txt")

	content := []byte("hello sbctl copy")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}

	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read dst file: %v", err)
	}

	if string(got) != string(content) {
		t.Errorf("expected %q, got %q", string(content), string(got))
	}
}

func TestExtractZip(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")
	destPath := filepath.Join(tmpDir, "extracted")

	// Create a dummy zip file containing "sbctl"
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}
	zw := zip.NewWriter(f)

	w, err := zw.Create("sbctl")
	if err != nil {
		t.Fatalf("failed to create file in zip: %v", err)
	}
	_, _ = w.Write([]byte("binary content"))
	_ = zw.Close()
	_ = f.Close()

	if err := extractZip(zipPath, destPath); err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(got) != "binary content" {
		t.Errorf("expected 'binary content', got %q", string(got))
	}
}

func TestExtractZip_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")
	destPath := filepath.Join(tmpDir, "extracted")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("not-sbctl")
	if err != nil {
		t.Fatalf("failed to create file in zip: %v", err)
	}
	_, _ = w.Write([]byte("other content"))
	_ = zw.Close()
	_ = f.Close()

	err = extractZip(zipPath, destPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExtractTarGz(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "test.tar.gz")
	destPath := filepath.Join(tmpDir, "extracted")

	f, err := os.Create(tarPath)
	if err != nil {
		t.Fatalf("failed to create tar file: %v", err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	header := &tar.Header{
		Name: "sbctl",
		Mode: 0755,
		Size: int64(len("tar content")),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	_, _ = tw.Write([]byte("tar content"))
	_ = tw.Close()
	_ = gw.Close()
	_ = f.Close()

	if err := extractTarGz(tarPath, destPath); err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(got) != "tar content" {
		t.Errorf("expected 'tar content', got %q", string(got))
	}
}

func TestExtractTarGz_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "test.tar.gz")
	destPath := filepath.Join(tmpDir, "extracted")

	f, err := os.Create(tarPath)
	if err != nil {
		t.Fatalf("failed to create tar file: %v", err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	header := &tar.Header{
		Name: "not-sbctl",
		Mode: 0755,
		Size: int64(len("tar content")),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	_, _ = tw.Write([]byte("tar content"))
	_ = tw.Close()
	_ = gw.Close()
	_ = f.Close()

	err = extractTarGz(tarPath, destPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpgrade_LatestAlready(t *testing.T) {
	originalURL := apiURL
	defer func() { apiURL = originalURL }()

	originalVersion := Version
	Version = "v1.1.0"
	defer func() { Version = originalVersion }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"tag_name": "v1.0.0", "assets": []}`)
	}))
	defer server.Close()

	apiURL = server.URL

	err := Upgrade()
	if err != nil {
		t.Fatalf("Upgrade failed: %v", err)
	}
}

func TestUpgrade_APIErrors(t *testing.T) {
	originalURL := apiURL
	defer func() { apiURL = originalURL }()

	// Test HTTP request failure
	apiURL = "http://invalid-localhost-domain.local/nonexistent"
	err := Upgrade()
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}

	// Test non-200 Status code
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	apiURL = server.URL
	err = Upgrade()
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}

	// Test bad JSON response
	serverJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{bad-json`)
	}))
	defer serverJSON.Close()
	apiURL = serverJSON.URL
	err = Upgrade()
	if err == nil {
		t.Error("expected error for bad JSON response, got nil")
	}
}

func TestUpgrade_NoSuitableAsset(t *testing.T) {
	originalURL := apiURL
	defer func() { apiURL = originalURL }()

	originalVersion := Version
	Version = "v1.0.0"
	defer func() { Version = originalVersion }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"tag_name": "v1.2.0", "assets": [{"name": "sbctl-1.2.0-otheros-otherarch.tar.gz", "browser_download_url": "http://example.com"}]}`)
	}))
	defer server.Close()

	apiURL = server.URL

	err := Upgrade()
	if err == nil {
		t.Error("expected error due to lack of suitable asset, got nil")
	}
}

func TestUpgrade_Success(t *testing.T) {
	originalURL := apiURL
	originalOsExecutable := osExecutable
	originalVersion := Version
	defer func() {
		apiURL = originalURL
		osExecutable = originalOsExecutable
		Version = originalVersion
	}()

	Version = "v1.0.0"

	tmpDir := t.TempDir()
	dummyExec := filepath.Join(tmpDir, "dummy-sbctl")
	if err := os.WriteFile(dummyExec, []byte("original executable"), 0755); err != nil {
		t.Fatalf("failed to write dummy exec: %v", err)
	}

	osExecutable = func() (string, error) {
		return dummyExec, nil
	}

	// Create a mock zip archive to serve
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	w, err := zw.Create("sbctl")
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}
	_, _ = w.Write([]byte("new upgraded binary"))
	_ = zw.Close()

	// Start mock server
	// Yes! `http://` + r.Host + `/download` is perfect and clean.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/download" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(zipBuf.Bytes())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"tag_name": "v1.2.0", "assets": [{"name": "sbctl-1.2.0-%s-%s.zip", "browser_download_url": "http://%s/download"}]}`,
			runtime.GOOS, runtime.GOARCH, r.Host)
	}))
	defer server.Close()

	apiURL = server.URL

	if err := Upgrade(); err != nil {
		t.Fatalf("Upgrade failed: %v", err)
	}

	got, err := os.ReadFile(dummyExec)
	if err != nil {
		t.Fatalf("failed to read upgraded executable: %v", err)
	}

	if string(got) != "new upgraded binary" {
		t.Errorf("expected 'new upgraded binary', got %q", string(got))
	}
}
