//go:build rk3588

package goface

import "fmt"

func initAuto(basePath string) (string, error) {
	candidates := []string{"Gundam_RK3588", "Megatron", "Pikachu"}
	for _, name := range candidates {
		modelPath := joinModelPath(basePath, name)
		if err := fallbackInit(modelPath); err == nil {
			fmt.Println("[InitAuto] Loaded model:", modelPath)
			return modelPath, nil
		}
		// Local model missing — try downloading from the built-in whitelist.
		if err := DownloadModel(basePath, name); err == nil {
			if err := fallbackInit(modelPath); err == nil {
				fmt.Println("[InitAuto] Loaded model:", modelPath)
				return modelPath, nil
			}
		}
		fmt.Printf("[InitAuto] %s not available locally or via download, trying next...\n", name)
	}
	return "", fmt.Errorf("all models failed")
}
