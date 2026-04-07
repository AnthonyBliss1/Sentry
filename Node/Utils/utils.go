package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

var (
	Green = color.New(color.FgGreen) // debug
	Blue  = color.New(color.FgBlue)  // actions
	Red   = color.New(color.FgRed)   // warnings
)

func ValidateOutputDir() (hlsDir string, err error) {
	ex, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to find binary path: %w", err)
	}

	cwd := filepath.Dir(ex)
	hlsDir = filepath.Join(cwd, "HLS")

	// Ensure HLS Dir exists
	if err := os.MkdirAll(hlsDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to make HLS Dir: %w", err)
	}

	return hlsDir, nil
}
