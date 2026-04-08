package utils

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

var (
	Green = color.New(color.FgGreen) // debug
	Blue  = color.New(color.FgBlue)  // actions
	Red   = color.New(color.FgRed)   // warnings
)

func LANIPv4() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

func ValidateSaveDir() (hlsDir string, err error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to find User Config Dir: %w", err)
	}

	hlsDir = filepath.Join(base, "Sentry")

	// ensuring 'HLS' exists
	if err := os.MkdirAll(hlsDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to make HLS dir: %w", err)
	}

	return hlsDir, nil
}

func SaveFile(hlsDir string, deviceID string, fileName string, data []byte) error {
	deviceDir := filepath.Join(hlsDir, deviceID)
	filePath := filepath.Join(deviceDir, fileName)

	if err := os.MkdirAll(deviceDir, 0o755); err != nil {
		return fmt.Errorf("failed to make device directory: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write file to server: %w", err)
	}

	return nil
}
