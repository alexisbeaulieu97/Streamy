package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// MigrateRegistryFile upgrades registry storage files to the current schema version.
func MigrateRegistryFile(path string) error {
	// #nosec G304 -- migration reads user-managed registry files from disk.
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("read registry file: %w", err)
	}

	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse registry: %w", err)
	}

	originalVersion := file.Version
	if originalVersion == "" {
		originalVersion = legacyFileVersion
	}

	if originalVersion == registryFileVersion {
		return nil
	}

	switch originalVersion {
	case legacyFileVersion:
		for i := range file.Pipelines {
			if file.Pipelines[i].Dependencies == nil {
				file.Pipelines[i].Dependencies = []string{}
			}
		}
	default:
		return fmt.Errorf("unsupported registry version %q", originalVersion)
	}

	file.Version = registryFileVersion

	if err := writeBackup(path, originalVersion, data); err != nil {
		return err
	}

	updated, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize registry: %w", err)
	}

	if err := writeAtomic(path, updated, 0o600); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	return nil
}

// MigrateStatusCacheFile upgrades the status cache file to the current schema version.
func MigrateStatusCacheFile(path string) error {
	// #nosec G304 -- migration reads user-managed status cache from disk.
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("read status cache: %w", err)
	}

	var file StatusCacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse status cache: %w", err)
	}

	originalVersion := file.Version
	if originalVersion == "" {
		originalVersion = legacyFileVersion
	}

	if originalVersion == statusCacheVersion {
		return nil
	}

	switch originalVersion {
	case legacyFileVersion:
		if file.Statuses == nil {
			file.Statuses = make(map[string]CachedStatus)
		}
	default:
		return fmt.Errorf("unsupported status cache version %q", originalVersion)
	}

	file.Version = statusCacheVersion

	if err := writeBackup(path, originalVersion, data); err != nil {
		return err
	}

	updated, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize status cache: %w", err)
	}

	if err := writeAtomic(path, updated, 0o600); err != nil {
		return fmt.Errorf("write status cache: %w", err)
	}

	return nil
}

func writeBackup(path, originalVersion string, data []byte) error {
	backupPath := fmt.Sprintf("%s.v%s.bak", path, originalVersion)
	if err := writeAtomic(backupPath, data, 0o600); err != nil {
		return fmt.Errorf("create backup %q: %w", backupPath, err)
	}

	return nil
}
