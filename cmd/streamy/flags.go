package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validateApplyOptions(opts applyOptions) error {
	if strings.TrimSpace(opts.ConfigPath) == "" {
		return fmt.Errorf("config file is required")
	}

	abs, err := filepath.Abs(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("config file does not exist: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("config path %s is a directory", abs)
	}

	return nil
}
