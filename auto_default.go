//go:build !rk3588

package goface

import "fmt"

func initAuto(basePath string) (string, error) {
	candidates := []string{"Megatron", "Pikachu"}
	for _, name := range candidates {
		modelPath := joinModelPath(basePath, name)
		if err := fallbackInit(modelPath); err == nil {
			return modelPath, nil
		}
		// Local model missing — try downloading from the built-in whitelist.
		if err := DownloadModel(basePath, name); err == nil {
			if err := fallbackInit(modelPath); err == nil {
				return modelPath, nil
			}
		}
	}
	return "", fmt.Errorf("all models failed")
}
