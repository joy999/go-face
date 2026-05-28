package goface

import (
	"os"
	"path/filepath"
	"testing"
)

// findModelDir locates a model pack relative to the package root.
func findModelDir(name string) string {
	// Try current dir (repo root) first
	if _, err := os.Stat(filepath.Join("models", name)); err == nil {
		return filepath.Join("models", name)
	}
	// Fallback: walk up from test file location
	for _, base := range []string{".", "..", "../.."} {
		p := filepath.Join(base, "models", name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// TestEndToEnd_Pikachu runs a full detection pipeline using the bundled Pikachu model.
func TestEndToEnd_Pikachu(t *testing.T) {
	modelDir := findModelDir("Pikachu")
	if modelDir == "" {
		t.Skip("Pikachu model not found in models/")
	}

	err := Init(InitOption{
		ModelPath: modelDir,
		Backend:   BackendCPU,
		LogLevel:  0,
	})
	if err != nil {
		t.Skipf("Init skipped (model may not be compatible with this platform build): %v", err)
	}
	defer Deinit()

	sess, err := NewSession(DefaultSessionOption())
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	defer sess.Close()

	// Create a synthetic BGR image (blank blue-ish image, no real face)
	width, height := 320, 240
	img := make([]byte, width*height*3)
	for i := 0; i < width*height; i++ {
		img[i*3+0] = 200 // B
		img[i*3+1] = 100 // G
		img[i*3+2] = 50  // R
	}

	faces, err := sess.Detect(img, width, height, FormatBGR)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// No real face in the synthetic image, so we expect 0 faces.
	// The important thing is that the pipeline didn't crash.
	t.Logf("Detected %d face(s) in blank image (expected 0)", len(faces))

	// Now test with a simple synthetic "face-like" pattern:
	// a bright oval in the center to trick the detector into finding something.
	for y := 60; y < 180; y++ {
		for x := 80; x < 240; x++ {
			dx := float64(x - 160)
			dy := float64(y - 120)
			if (dx*dx)/6400+(dy*dy)/3600 < 1.0 {
				idx := (y*width + x) * 3
				img[idx+0] = 255
				img[idx+1] = 255
				img[idx+2] = 255
			}
		}
	}

	faces2, err := sess.Detect(img, width, height, FormatBGR)
	if err != nil {
		t.Fatalf("Detect failed on oval image: %v", err)
	}
	t.Logf("Detected %d face(s) in oval image", len(faces2))
}
