package validation

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"regexp"
)

// CheckCommandExists verifies a command is available on PATH.
func CheckCommandExists(command string) error {
	if command == "" {
		return fmt.Errorf("command name is required")
	}

	if _, err := exec.LookPath(command); err != nil {
		return err
	}
	return nil
}

// CheckFileExists verifies a file or directory exists at the given path.
func CheckFileExists(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("path %s does not exist", path)
		}
		return err
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

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	pattern, err := regexp.Compile(text)
	if err != nil {
		return err
	}

	if !pattern.Match(data) {
		return fmt.Errorf("pattern %q not found in %s", text, path)
	}

	return nil
}
