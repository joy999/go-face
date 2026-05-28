package goface

import "path/filepath"

// InitAuto tries to initialize with the best available model.
//
// On RK3588 (build tag rk3588) it first attempts the NPU-optimized model
// (Gundam_RK3588). If that fails (e.g. NPU unavailable or incompatible),
// it automatically falls back to the CPU model (Pikachu).
//
// On non-RK3588 platforms it directly uses the CPU model (Pikachu).
//
// If a model is not present locally, InitAuto will automatically download it
// from the built-in whitelist (GitHub releases) before retrying.
//
// Returns the actual model path that was successfully loaded.
func InitAuto(basePath string) (string, error) {
	return initAuto(basePath)
}

// fallbackInit attempts to init with the given model path.
// On failure it calls Deinit to leave a clean state before the caller retries.
func fallbackInit(modelPath string) error {
	opt := InitOption{
		ModelPath: modelPath,
		Backend:   BackendAuto,
		LogLevel:  1,
	}
	if err := Init(opt); err != nil {
		// Best-effort cleanup so the next Init starts from a clean slate.
		Deinit()
		return err
	}
	return nil
}

func joinModelPath(base, name string) string {
	if base == "" {
		return name
	}
	return filepath.Join(base, name)
}
