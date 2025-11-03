package registry

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".streamy-*")
	if err != nil {
		return fmt.Errorf("create temporary file in %q: %w", dir, err)
	}

	tmpPath := tmpFile.Name()
	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := tmpFile.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temporary file %q: %w", tmpPath, err)
	}

	if err := tmpFile.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync temporary file %q: %w", tmpPath, err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temporary file %q: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename %q to %q: %w", tmpPath, path, err)
	}

	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("set permissions on %q: %w", path, err)
	}

	if err := syncDir(dir); err != nil {
		return fmt.Errorf("sync directory %q: %w", dir, err)
	}

	return nil
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()

	return f.Sync()
}
