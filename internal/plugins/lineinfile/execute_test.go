package lineinfileplugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

func TestLineInFile_Apply_PresentNoMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		initial            string
		precreate          bool
		line               string
		expected           string
		expectFirstStatus  string
		expectSecondStatus string
	}{
		{
			name:               "append to empty existing file",
			precreate:          true,
			initial:            "",
			line:               "export PATH=\"$PATH:/opt/bin\"",
			expected:           "export PATH=\"$PATH:/opt/bin\"\n",
			expectFirstStatus:  model.StatusSuccess,
			expectSecondStatus: model.StatusSkipped,
		},
		{
			name:               "append to file without line",
			precreate:          true,
			initial:            "PATH=/usr/bin\n",
			line:               "PATH=$PATH:/opt/bin",
			expected:           "PATH=/usr/bin\nPATH=$PATH:/opt/bin\n",
			expectFirstStatus:  model.StatusSuccess,
			expectSecondStatus: model.StatusSkipped,
		},
		{
			name:               "line already present idempotent",
			precreate:          true,
			initial:            "export EDITOR=vim\n",
			line:               "export EDITOR=vim",
			expected:           "export EDITOR=vim\n",
			expectFirstStatus:  model.StatusSkipped,
			expectSecondStatus: model.StatusSkipped,
		},
		{
			name:               "create file when missing",
			precreate:          false,
			initial:            "",
			line:               "export JAVA_HOME=/opt/java",
			expected:           "export JAVA_HOME=/opt/java\n",
			expectFirstStatus:  model.StatusSuccess,
			expectSecondStatus: model.StatusSkipped,
		},
	}

	plugin := New()
	ctx := context.Background()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			target := filepath.Join(dir, "profile")
			if tt.precreate {
				writeFile(t, target, tt.initial, 0o644)
			}

			step := newLineInFileStep("present-no-match", &config.LineInFileStep{
				File:  target,
				Line:  tt.line,
				State: "present",
			})

			res, err := plugin.Apply(ctx, step)
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, tt.expectFirstStatus, res.Status)

			content := readFile(t, target)
			assert.Equal(t, tt.expected, content)

			res2, err := plugin.Apply(ctx, step)
			require.NoError(t, err)
			require.NotNil(t, res2)
			assert.Equal(t, tt.expectSecondStatus, res2.Status)
		})
	}
}

func TestLineInFile_Apply_PresentWithMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		initial           string
		match             string
		line              string
		strategy          string
		expected          string
		expectStatus      string
		expectError       bool
		expectErrContains string
	}{
		{
			name:         "replace single match",
			initial:      "debug=true\nmode=dev\n",
			match:        "^debug=",
			line:         "debug=false",
			strategy:     "first",
			expected:     "debug=false\nmode=dev\n",
			expectStatus: model.StatusSuccess,
		},
		{
			name:         "replace first of many",
			initial:      "export PATH=/usr/bin\nexport PATH=/usr/local/bin\n",
			match:        "^export PATH=",
			line:         "export PATH=/custom/bin",
			strategy:     "first",
			expected:     "export PATH=/custom/bin\nexport PATH=/usr/local/bin\n",
			expectStatus: model.StatusSuccess,
		},
		{
			name:         "replace all matches",
			initial:      "server=one\nserver=two\n",
			match:        "^server=",
			line:         "server=three",
			strategy:     "all",
			expected:     "server=three\nserver=three\n",
			expectStatus: model.StatusSuccess,
		},
		{
			name:              "error when multiple matches and strategy error",
			initial:           "server=one\nserver=two\n",
			match:             "^server=",
			line:              "server=three",
			strategy:          "error",
			expectError:       true,
			expectErrContains: "multiple matches",
		},
		{
			name:         "append when no match found",
			initial:      "mode=prod\n",
			match:        "^debug=",
			line:         "debug=false",
			strategy:     "first",
			expected:     "mode=prod\ndebug=false\n",
			expectStatus: model.StatusSuccess,
		},
		{
			name:         "already replaced idempotent",
			initial:      "debug=false\n",
			match:        "^debug=",
			line:         "debug=false",
			strategy:     "first",
			expected:     "debug=false\n",
			expectStatus: model.StatusSkipped,
		},
	}

	plugin := New()
	ctx := context.Background()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			target := filepath.Join(dir, "config.ini")
			writeFile(t, target, tt.initial, 0o644)

			step := newLineInFileStep("present-with-match", &config.LineInFileStep{
				File:              target,
				Line:              tt.line,
				State:             "present",
				Match:             tt.match,
				OnMultipleMatches: tt.strategy,
			})

			res, err := plugin.Apply(ctx, step)
			if tt.expectError {
				require.Error(t, err)
				if tt.expectErrContains != "" {
					assert.Contains(t, err.Error(), tt.expectErrContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, tt.expectStatus, res.Status)

			content := readFile(t, target)
			assert.Equal(t, tt.expected, content)
		})
	}
}

func TestLineInFile_Apply_Absent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		initial      string
		match        string
		expected     string
		expectStatus string
	}{
		{
			name:         "remove single matching line",
			initial:      "export OLD=value\nexport NEW=value\n",
			match:        "^export OLD=",
			expected:     "export NEW=value\n",
			expectStatus: model.StatusSuccess,
		},
		{
			name:         "remove multiple matches",
			initial:      "alias ll='ls'\nalias ll='ls -al'\n",
			match:        "^alias ll",
			expected:     "",
			expectStatus: model.StatusSuccess,
		},
		{
			name:         "no matches idempotent",
			initial:      "export NEW=value\n",
			match:        "^export OLD=",
			expected:     "export NEW=value\n",
			expectStatus: model.StatusSkipped,
		},
	}

	plugin := New()
	ctx := context.Background()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			target := filepath.Join(dir, "profile")
			writeFile(t, target, tt.initial, 0o644)

			step := newLineInFileStep("absent", &config.LineInFileStep{
				File:  target,
				Line:  "placeholder",
				State: "absent",
				Match: tt.match,
			})

			res, err := plugin.Apply(ctx, step)
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, tt.expectStatus, res.Status)

			content := readFile(t, target)
			assert.Equal(t, tt.expected, content)
		})
	}
}

func TestLineInFile_Apply_FileOperations(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on Windows")
	}

	plugin := New()
	ctx := context.Background()

	t.Run("preserve file permissions", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		target := filepath.Join(dir, "file.txt")
		writeFile(t, target, "initial\n", 0o600)

		step := newLineInFileStep("present", &config.LineInFileStep{
			File:  target,
			Line:  "new line",
			State: "present",
		})

		res, err := plugin.Apply(ctx, step)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, model.StatusSuccess, res.Status)

		info, err := os.Stat(target)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("follow symlinks", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		target := filepath.Join(dir, "target.txt")
		link := filepath.Join(dir, "link.txt")
		writeFile(t, target, "start\n", 0o644)
		require.NoError(t, os.Symlink(target, link))

		step := newLineInFileStep("present", &config.LineInFileStep{
			File:  link,
			Line:  "added",
			State: "present",
		})

		res, err := plugin.Apply(ctx, step)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, model.StatusSuccess, res.Status)

		content := readFile(t, target)
		assert.Equal(t, "start\nadded\n", content)
	})

	t.Run("error on read permission denied", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		target := filepath.Join(dir, "protected.txt")
		writeFile(t, target, "secret\n", 0o644)
		require.NoError(t, os.Chmod(target, 0o000))

		step := newLineInFileStep("present", &config.LineInFileStep{
			File:  target,
			Line:  "noop",
			State: "present",
		})

		_, err := plugin.Apply(ctx, step)
		require.Error(t, err)
		var execErr *streamyerrors.ExecutionError
		assert.True(t, errors.As(err, &execErr))
	})

	t.Run("error on write permission denied", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		sub := filepath.Join(dir, "sub")
		require.NoError(t, os.Mkdir(sub, 0o755))
		target := filepath.Join(sub, "file.txt")
		writeFile(t, target, "content\n", 0o644)
		require.NoError(t, os.Chmod(sub, 0o555))
		require.NoError(t, os.Chmod(target, 0o444))

		// Restore permissions for cleanup
		t.Cleanup(func() {
			os.Chmod(sub, 0o755)
			os.Chmod(target, 0o644)
		})

		step := newLineInFileStep("present", &config.LineInFileStep{
			File:  target,
			Line:  "extra",
			State: "present",
		})

		_, err := plugin.Apply(ctx, step)
		require.Error(t, err)
		var execErr *streamyerrors.ExecutionError
		assert.True(t, errors.As(err, &execErr))
	})
}

func TestLineInFile_Apply_Backup(t *testing.T) {
	t.Parallel()

	plugin := New()
	ctx := context.Background()

	t.Run("create backup when changes occur", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		target := filepath.Join(dir, "file.txt")
		writeFile(t, target, "original\n", 0o644)

		step := newLineInFileStep("backup", &config.LineInFileStep{
			File:   target,
			Line:   "modified",
			State:  "present",
			Backup: true,
		})

		res, err := plugin.Apply(ctx, step)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, model.StatusSuccess, res.Status)

		backups := listBackups(t, dir, "file.txt")
		require.Len(t, backups, 1)
		assert.True(t, strings.Contains(filepath.Base(backups[0]), ".bak"))
	})

	t.Run("no backup when file unchanged", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		target := filepath.Join(dir, "file.txt")
		writeFile(t, target, "line\n", 0o644)

		step := newLineInFileStep("backup", &config.LineInFileStep{
			File:   target,
			Line:   "line",
			State:  "present",
			Backup: true,
		})

		res, err := plugin.Apply(ctx, step)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, model.StatusSkipped, res.Status)

		backups := listBackups(t, dir, "file.txt")
		assert.Empty(t, backups)
	})

	t.Run("backup to custom directory", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		target := filepath.Join(dir, "file.txt")
		backupDir := filepath.Join(dir, "backups")
		writeFile(t, target, "data\n", 0o644)

		step := newLineInFileStep("backup", &config.LineInFileStep{
			File:      target,
			Line:      "data updated",
			State:     "present",
			Backup:    true,
			BackupDir: backupDir,
		})

		res, err := plugin.Apply(ctx, step)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, model.StatusSuccess, res.Status)

		backups := listBackups(t, backupDir, "file.txt")
		require.Len(t, backups, 1)
		info, err := os.Stat(backups[0])
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	})
}

func TestLineInFile_Apply_Encoding(t *testing.T) {
	t.Parallel()

	plugin := New()
	ctx := context.Background()

	tests := []struct {
		name         string
		fixtureName  string
		encoding     string
		line         string
		expected     string
		expectStatus string
	}{
		{
			name:         "utf-8 append",
			fixtureName:  "utf8.txt",
			encoding:     "utf-8",
			line:         "status=準備完了",
			expected:     "# Fixture: UTF-8 encoded file containing non-ASCII characters\nwelcome_message=こんにちは世界\nemoji=🤖\nstatus=準備完了\n",
			expectStatus: model.StatusSuccess,
		},
		{
			name:         "latin-1 append",
			fixtureName:  "latin1.txt",
			encoding:     "latin-1",
			line:         "update=Señora",
			expected:     "# Fixture: Latin-1 encoded file with accented characters\nstatus=Señor García está listo\nupdate=Señora\n",
			expectStatus: model.StatusSuccess,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := fixturePath(t, tt.fixtureName)
			dir := t.TempDir()
			target := filepath.Join(dir, tt.fixtureName)
			copyFile(t, src, target)

			step := newLineInFileStep("encoding", &config.LineInFileStep{
				File:     target,
				Line:     tt.line,
				State:    "present",
				Encoding: tt.encoding,
			})

			res, err := plugin.Apply(ctx, step)
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, tt.expectStatus, res.Status)

			content := readFileWithEncoding(t, target, tt.encoding)
			assert.Equal(t, tt.expected, content)
		})
	}
}

func TestLineInFile_DryRun(t *testing.T) {
	t.Parallel()

	plugin := New()
	ctx := context.Background()

	tests := []struct {
		name               string
		initial            string
		state              string
		line               string
		match              string
		expectStatus       string
		expectDiffContains []string
	}{
		{
			name:         "preview append",
			state:        "present",
			line:         "extra",
			expectStatus: model.StatusWouldCreate,
			expectDiffContains: []string{
				"+extra",
			},
		},
		{
			name:         "preview replace",
			initial:      "debug=true\n",
			state:        "present",
			line:         "debug=false",
			match:        "^debug=",
			expectStatus: model.StatusWouldUpdate,
			expectDiffContains: []string{
				"-debug=true",
				"+debug=false",
			},
		},
		{
			name:         "preview remove",
			initial:      "remove\nkeep\n",
			state:        "absent",
			match:        "^remove$",
			expectStatus: model.StatusWouldUpdate,
			expectDiffContains: []string{
				"-remove",
			},
		},
		{
			name:         "preview no change",
			initial:      "keep\n",
			state:        "present",
			line:         "keep",
			expectStatus: model.StatusSkipped,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			target := filepath.Join(dir, "file.txt")
			if tt.initial != "" {
				writeFile(t, target, tt.initial, 0o644)
			}

			step := newLineInFileStep("dry-run", &config.LineInFileStep{
				File:  target,
				Line:  tt.line,
				State: tt.state,
				Match: tt.match,
			})

			res, err := plugin.DryRun(ctx, step)
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, tt.expectStatus, res.Status)

			for _, fragment := range tt.expectDiffContains {
				assert.Contains(t, res.Message, fragment)
			}

			// Ensure file not modified
			if tt.initial != "" {
				assert.Equal(t, tt.initial, readFile(t, target))
			} else {
				_, err := os.Stat(target)
				if errors.Is(err, os.ErrNotExist) {
					// expected: dry run should not create file
				} else {
					require.NoError(t, err)
					assert.Equal(t, "", readFile(t, target))
				}
			}
		})
	}
}

func writeFile(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), perm))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dst, data, 0o644))
}

func listBackups(t *testing.T, dir, base string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	require.NoError(t, err)
	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), base+".") && strings.HasSuffix(entry.Name(), ".bak") {
			backups = append(backups, filepath.Join(dir, entry.Name()))
		}
	}
	return backups
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "testdata", "lineinfile", name)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.False(t, info.IsDir())
	return path
}

func readFileWithEncoding(t *testing.T, path, encoding string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	decoded, err := decodeContent(data, encoding)
	require.NoError(t, err)
	return decoded
}
