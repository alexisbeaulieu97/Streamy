package lineinfileplugin

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

const defaultFileMode os.FileMode = 0o644

// FileState captures the state of a target file prior to modification.
type FileState struct {
	Path            string
	OriginalPath    string
	Exists          bool
	Permissions     os.FileMode
	Content         string
	Lines           []string
	TrailingNewline bool
}

func readFileState(cfg *LineInFileConfig) (*FileState, error) {
	expandedPath, err := expandPath(cfg.File)
	if err != nil {
		return nil, err
	}

	state := &FileState{
		Path:         expandedPath,
		OriginalPath: expandedPath,
	}

	resolvedPath, err := filepath.EvalSymlinks(expandedPath)
	if err == nil {
		state.Path = resolvedPath
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	info, err := os.Stat(state.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			state.Exists = false
			state.Permissions = defaultFileMode
			state.Lines = []string{}
			return state, nil
		}
		return nil, err
	}

	state.Exists = true
	state.Permissions = info.Mode().Perm()

	data, err := os.ReadFile(state.Path)
	if err != nil {
		return nil, err
	}

	decoded, err := decodeContent(data, cfg.Encoding)
	if err != nil {
		return nil, err
	}

	state.Content = decoded
	lines, trailing := splitLines(decoded)
	state.Lines = lines
	state.TrailingNewline = trailing
	return state, nil
}

func splitLines(content string) ([]string, bool) {
	if content == "" {
		return []string{}, false
	}
	trailing := strings.HasSuffix(content, "\n")
	trimmed := content
	if trailing {
		trimmed = strings.TrimSuffix(content, "\n")
	}
	if trimmed == "" {
		if trailing {
			return []string{}, true
		}
		return []string{""}, false
	}
	lines := strings.Split(trimmed, "\n")
	return lines, trailing
}

func joinLines(lines []string, trailing bool) string {
	if len(lines) == 0 {
		if trailing {
			return "\n"
		}
		return ""
	}
	joined := strings.Join(lines, "\n")
	if trailing {
		return joined + "\n"
	}
	return joined
}

func expandPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else if strings.HasPrefix(path, "~/") {
			path = filepath.Join(home, path[2:])
		}
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Abs(path)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".lineinfile-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}

	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}

	return nil
}

func createBackup(path, backupDir string, content []byte, perm os.FileMode) (string, error) {
	targetDir := filepath.Dir(path)
	if strings.TrimSpace(backupDir) != "" {
		expanded, err := expandPath(backupDir)
		if err != nil {
			return "", err
		}
		targetDir = expanded
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}

	base := filepath.Base(path)
	timestamp := time.Now().UTC().Format("20060102T150405")
	backupPath := filepath.Join(targetDir, fmt.Sprintf("%s.%s.bak", base, timestamp))

	if err := os.WriteFile(backupPath, content, perm); err != nil {
		return "", err
	}

	return backupPath, nil
}

func decodeContent(data []byte, name string) (string, error) {
	enc := encodingByName(name)
	if enc == nil {
		return string(data), nil
	}

	reader := transform.NewReader(bytes.NewReader(data), enc.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func encodeContent(content string, name string) ([]byte, error) {
	enc := encodingByName(name)
	if enc == nil {
		return []byte(content), nil
	}
	var buf bytes.Buffer
	writer := transform.NewWriter(&buf, enc.NewEncoder())
	if _, err := writer.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodingByName(name string) encoding.Encoding {
	switch strings.ToLower(name) {
	case "", "utf-8", "utf8":
		return nil
	case "latin-1", "latin1", "iso-8859-1":
		return charmap.ISO8859_1
	case "windows-1252":
		return charmap.Windows1252
	case "ascii":
		return charmap.ISO8859_1
	case "utf-16", "utf-16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM)
	case "utf-16be":
		return unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM)
	default:
		return nil
	}
}
