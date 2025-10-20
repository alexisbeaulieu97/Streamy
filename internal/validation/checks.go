// Package validation provides reusable validation helpers for pipelines.
package validation

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// CheckCommandExists verifies a command is available on PATH.
func CheckCommandExists(command string) error {
	if command == "" {
		return fmt.Errorf("command name is required")
	}

	if _, err := exec.LookPath(command); err != nil {
		return fmt.Errorf("locate command %q: %w", command, err)
	}

	return nil
}

// CheckFileExists verifies a file or directory exists at the given path.
func CheckFileExists(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}

	path = filepath.Clean(path)

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("path %s does not exist", path)
		}

		return fmt.Errorf("stat path %s: %w", path, err)
	}

	return nil
}

// CheckPathContains verifies that file contains the provided text or pattern.
func CheckPathContains(path, text string) error {
	if path == "" {
		return fmt.Errorf("file path is required")
	}

	if text == "" {
		return fmt.Errorf("text is required")
	}

	path = filepath.Clean(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file %s: %w", path, err)
	}

	pattern, err := regexp.Compile(text)
	if err != nil {
		return fmt.Errorf("compile pattern %q: %w", text, err)
	}

	if !pattern.Match(data) {
		return fmt.Errorf("pattern %q not found in %s", text, path)
	}

	return nil
}
