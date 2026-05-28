package goface

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveModelPathTarOnly verifies that resolveModelPath returns the
// .tar file when the directory does not exist but the archive does.
func TestResolveModelPathTarOnly(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "Pikachu.tar")
	if err := os.WriteFile(tarPath, []byte("fake tar"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pass the directory path — it does not exist, but .tar does.
	dirPath := filepath.Join(tmpDir, "Pikachu")
	got, err := resolveModelPath(dirPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != tarPath {
		t.Errorf("expected %q, got %q", tarPath, got)
	}
}

// TestResolveModelPathDirectoryWithTar verifies the original behaviour:
// when a directory exists alongside its .tar, the .tar path is returned.
func TestResolveModelPathDirectoryWithTar(t *testing.T) {
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "Megatron")
	tarPath := dirPath + ".tar"

	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tarPath, []byte("tar"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveModelPath(dirPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != tarPath {
		t.Errorf("expected %q, got %q", tarPath, got)
	}
}

// TestResolveModelPathMissing verifies an error when neither directory nor
// .tar exists.
func TestResolveModelPathMissing(t *testing.T) {
	tmpDir := t.TempDir()
	missing := filepath.Join(tmpDir, "Ghost")
	_, err := resolveModelPath(missing)
	if err == nil {
		t.Error("expected error for missing path, got nil")
	}
}

// TestJoinModelPath covers the helper.
func TestJoinModelPath(t *testing.T) {
	if got := joinModelPath("/models", "Pikachu"); got != filepath.Join("/models", "Pikachu") {
		t.Errorf("unexpected result: %q", got)
	}
	if got := joinModelPath("", "Pikachu"); got != "Pikachu" {
		t.Errorf("unexpected result: %q", got)
	}
}

// TestDownloadModelSuccess exercises the full download path using a mock server.
func TestDownloadModelSuccess(t *testing.T) {
	content := []byte("fake model tar bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	// Override the URL for "Megatron" to point at our mock server.
	origURL := modelDownloadURLs["Megatron"]
	modelDownloadURLs["Megatron"] = server.URL
	defer func() { modelDownloadURLs["Megatron"] = origURL }()

	tmpDir := t.TempDir()
	if err := DownloadModel(tmpDir, "Megatron"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tarPath := filepath.Join(tmpDir, "Megatron.tar")
	data, err := os.ReadFile(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(content) {
		t.Errorf("downloaded content mismatch")
	}
}

// TestDownloadModelSkipExisting ensures DownloadModel is a no-op when the
// .tar file already exists.
func TestDownloadModelSkipExisting(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "Pikachu.tar")
	if err := os.WriteFile(tarPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should succeed immediately without network access.
	err := DownloadModel(tmpDir, "Pikachu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing" {
		t.Error("existing file was overwritten")
	}
}

// TestDownloadModel404 verifies error handling when the release asset is missing.
func TestDownloadModel404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	origURL := modelDownloadURLs["Pikachu"]
	modelDownloadURLs["Pikachu"] = server.URL
	defer func() { modelDownloadURLs["Pikachu"] = origURL }()

	err := DownloadModel(t.TempDir(), "Pikachu")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

// TestDownloadModelUnknown verifies that an unknown model name is rejected.
func TestDownloadModelUnknown(t *testing.T) {
	err := DownloadModel(t.TempDir(), "NotInWhitelist")
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
}
