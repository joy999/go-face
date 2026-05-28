package goface

import (
	"testing"
)

// TestDefaultSessionOption sanity-checks the defaults.
func TestDefaultSessionOption(t *testing.T) {
	opt := DefaultSessionOption()
	if opt.MaxFaces != 10 {
		t.Errorf("expected MaxFaces=10, got %d", opt.MaxFaces)
	}
	if opt.DetectPixelLevel != -1 {
		t.Errorf("expected DetectPixelLevel=-1, got %d", opt.DetectPixelLevel)
	}
	if opt.EnableRecognition != 1 {
		t.Errorf("expected EnableRecognition=1, got %d", opt.EnableRecognition)
	}
}

// TestInitWithoutModel expects an error when model path is invalid.
func TestInitWithoutModel(t *testing.T) {
	err := Init(InitOption{
		ModelPath: "/nonexistent/path",
		Backend:   BackendCPU,
		LogLevel:  0,
	})
	if err == nil {
		t.Error("expected error for invalid model path, got nil")
	}
	// No Deinit needed on failed init, but call to be safe.
	Deinit()
}

// TestCompareFeatures verifies cosine similarity logic.
func TestCompareFeatures(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	if sim := CompareFeatures(a, b); sim < 0.99 {
		t.Errorf("expected similarity ~1.0 for identical vectors, got %f", sim)
	}

	c := []float32{1, 0, 0}
	d := []float32{0, 1, 0}
	if sim := CompareFeatures(c, d); sim > 0.01 || sim < -0.01 {
		t.Errorf("expected similarity ~0.0 for orthogonal vectors, got %f", sim)
	}
}

// TestImageFormatValues ensures enums stay in sync with C.
func TestImageFormatValues(t *testing.T) {
	if FormatRGB != 0 || FormatBGR != 1 || FormatNV12 != 4 || FormatGRAY != 7 {
		t.Error("ImageFormat enum values changed")
	}
}
