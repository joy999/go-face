package goface

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// modelDownloadURLs maps known model pack names to their official release URLs.
var modelDownloadURLs = map[string]string{
	"Pikachu":       "https://github.com/joy999/go-face/releases/download/v1.0.0/Pikachu",
	"Megatron":      "https://github.com/joy999/go-face/releases/download/v1.0.0/Megatron",
	"Megatron_TRT":  "https://github.com/joy999/go-face/releases/download/v1.0.0/Megatron_TRT",
	"Gundam_RK3588": "https://github.com/joy999/go-face/releases/download/v1.0.0/Gundam_RK3588",
}

// DownloadModel downloads the named model pack to basePath if it does not already exist.
//
// modelName must be one of the built-in model packs (Pikachu, Megatron, Megatron_TRT, Gundam_RK3588).
// The downloaded file is saved as {basePath}/{modelName}.tar.
func DownloadModel(basePath, modelName string) error {
	url, ok := modelDownloadURLs[modelName]
	if !ok {
		return fmt.Errorf("unknown model %q; supported models: Pikachu, Megatron, Megatron_TRT, Gundam_RK3588", modelName)
	}

	tarPath := filepath.Join(basePath, modelName+".tar")
	if _, err := os.Stat(tarPath); err == nil {
		return nil // already exists
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", modelName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", modelName, resp.StatusCode)
	}

	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return fmt.Errorf("create base path %s: %w", basePath, err)
	}

	tmpPath := tarPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file %s: %w", tmpPath, err)
	}

	_, copyErr := io.Copy(f, resp.Body)
	f.Close()
	if copyErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("download %s: %w", modelName, copyErr)
	}

	if err := os.Rename(tmpPath, tarPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename %s -> %s: %w", tmpPath, tarPath, err)
	}

	return nil
}
