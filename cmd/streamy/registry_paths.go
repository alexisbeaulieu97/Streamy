package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func defaultRegistryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine user home directory: %w", err)
	}

	return filepath.Join(home, ".streamy", "registry.json"), nil
}

func defaultStatusCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine user home directory: %w", err)
	}

	return filepath.Join(home, ".streamy", "status-cache.json"), nil
}
