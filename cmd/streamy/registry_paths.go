package main

import (
	"os"
	"path/filepath"
)

func defaultRegistryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".streamy", "registry.json"), nil
}

func defaultStatusCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".streamy", "status.json"), nil
}
