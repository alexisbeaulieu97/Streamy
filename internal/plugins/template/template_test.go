package templateplugin

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

func TestTemplatePlugin_MetadataAndSchema(t *testing.T) {
	t.Parallel()

	plugin := New()
	meta := plugin.Metadata()
	require.Equal(t, "template-renderer", meta.Name)
	require.Equal(t, "1.0.0", meta.Version)
	require.Equal(t, templatePluginType, meta.Type)

	schema := plugin.Schema()
	require.NotNil(t, schema)
	_, ok := schema.(config.TemplateStep)
	require.True(t, ok, "schema should expose TemplateStep structure")
}

func TestTemplatePlugin_Check(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		prepare     func(t *testing.T, dir string) *config.Step
		wantOK      bool
		wantErr     bool
		errContains string
		skipWindows bool
	}{
		{
			name: "source missing returns error",
			prepare: func(t *testing.T, dir string) *config.Step {
				return &config.Step{
					ID:   "missing-source",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      filepath.Join(dir, "missing.tmpl"),
						Destination: filepath.Join(dir, "output.txt"),
						Vars:        map[string]string{"NAME": "Alice"},
						Env:         false,
					},
				}
			},
			wantOK:      false,
			wantErr:     true,
			errContains: "read template",
		},
		{
			name: "destination missing needs creation",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "template.tmpl")
				writeFile(t, src, "Hello {{.NAME}}", 0o644)
				dst := filepath.Join(dir, "result.txt")
				return &config.Step{
					ID:   "dest-missing",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Vars:        map[string]string{"NAME": "Alice"},
						Env:         false,
					},
				}
			},
			wantOK:  false,
			wantErr: false,
		},
		{
			name: "destination matches rendered output",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "template.tmpl")
				writeFile(t, src, "Hello {{.NAME}}", 0o644)
				dst := filepath.Join(dir, "result.txt")
				writeFile(t, dst, "Hello Alice", 0o644)
				return &config.Step{
					ID:   "dest-matches",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Vars:        map[string]string{"NAME": "Alice"},
						Env:         false,
					},
				}
			},
			wantOK:  true,
			wantErr: false,
		},
		{
			name: "destination differs from rendered output",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "template.tmpl")
				writeFile(t, src, "Hello {{.NAME}}", 0o644)
				dst := filepath.Join(dir, "result.txt")
				writeFile(t, dst, "Hello Bob", 0o644)
				return &config.Step{
					ID:   "dest-differs",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Vars:        map[string]string{"NAME": "Alice"},
						Env:         false,
					},
				}
			},
			wantOK:  false,
			wantErr: false,
		},
		{
			name: "template syntax error produces failure",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "broken.tmpl")
				writeFile(t, src, "{{.NAME", 0o644)
				dst := filepath.Join(dir, "result.txt")
				writeFile(t, dst, "placeholder", 0o644)
				return &config.Step{
					ID:   "syntax-error",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			wantOK:      false,
			wantErr:     true,
			errContains: "parse template",
		},
		{
			name: "missing variables without allow_missing returns error",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "template.tmpl")
				writeFile(t, src, "Hello {{.NAME}}", 0o644)
				dst := filepath.Join(dir, "result.txt")
				writeFile(t, dst, "Hello", 0o644)
				return &config.Step{
					ID:   "missing-var",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			wantOK:      false,
			wantErr:     true,
			errContains: "missing",
		},
		{
			name: "allow missing variables treats blanks as matches",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "template.tmpl")
				writeFile(t, src, "Hello {{.NAME}}", 0o644)
				dst := filepath.Join(dir, "result.txt")
				writeFile(t, dst, "Hello ", 0o644)
				return &config.Step{
					ID:   "allow-missing-match",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:       src,
						Destination:  dst,
						AllowMissing: true,
						Env:          false,
					},
				}
			},
			wantOK:  true,
			wantErr: false,
		},
		{
			name: "returns false when permissions differ",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "mode.tmpl")
				writeFile(t, src, "Hello {{.NAME}}", 0o644)
				dst := filepath.Join(dir, "mode.txt")
				writeFile(t, dst, "Hello Alice", 0o644)
				mode := uint32(0o600)
				require.NoError(t, os.Chmod(dst, 0o644))
				return &config.Step{
					ID:   "mode-check",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
						Vars:        map[string]string{"NAME": "Alice"},
						Mode:        &mode,
					},
				}
			},
			wantOK:      false,
			wantErr:     false,
			skipWindows: true,
		},
		{
			name: "allow missing does not skip when destination differs",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "template.tmpl")
				writeFile(t, src, "Hello {{.NAME}}", 0o644)
				dst := filepath.Join(dir, "result.txt")
				writeFile(t, dst, "Hello bob", 0o644)
				return &config.Step{
					ID:   "allow-missing-different",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:       src,
						Destination:  dst,
						AllowMissing: true,
						Env:          false,
					},
				}
			},
			wantOK:  false,
			wantErr: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipWindows && runtime.GOOS == "windows" {
				t.Skip("permission semantics differ on Windows")
			}
			t.Parallel()
			dir := t.TempDir()
			step := tc.prepare(t, dir)

			result, err := New().Check(context.Background(), step)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.ErrorContains(t, err, tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantOK, result)
		})
	}
}

func TestTemplatePlugin_Apply(t *testing.T) {

	type expectation struct {
		check func(t *testing.T, res *model.StepResult, err error, step *config.Step)
	}

	cases := []struct {
		name        string
		prepare     func(t *testing.T, dir string) *config.Step
		prepareEnv  func(t *testing.T)
		first       expectation
		second      *expectation
		skipWindows bool
	}{
		{
			name: "basic inline variable substitution",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "inline.tmpl")
				writeFile(t, src, "Hello {{.NAME}}", 0o644)
				dst := filepath.Join(dir, "out.txt")
				return &config.Step{
					ID:   "inline",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Vars:        map[string]string{"NAME": "Streamy"},
						Env:         false,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.Equal(t, model.StatusSuccess, res.Status)
				require.Equal(t, "Hello Streamy", readFile(t, step.Template.Destination))
			}},
		},
		{
			name: "environment variable substitution",
			prepareEnv: func(t *testing.T) {
				t.Setenv("API_KEY", "abc123")
			},
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "env.tmpl")
				writeFile(t, src, "API_KEY={{.API_KEY}}", 0o644)
				dst := filepath.Join(dir, "env.txt")
				return &config.Step{
					ID:   "env",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         true,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.Equal(t, model.StatusSuccess, res.Status)
				require.Equal(t, "API_KEY=abc123", readFile(t, step.Template.Destination))
			}},
		},
		{
			name: "inline variables override environment variables",
			prepareEnv: func(t *testing.T) {
				t.Setenv("GREETING", "hello-env")
			},
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "override.tmpl")
				writeFile(t, src, "{{.GREETING}} world", 0o644)
				dst := filepath.Join(dir, "override.txt")
				return &config.Step{
					ID:   "override",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         true,
						Vars:        map[string]string{"GREETING": "hello-inline"},
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.Equal(t, "hello-inline world", readFile(t, step.Template.Destination))
			}},
		},
		{
			name: "idempotent apply skips on second run",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "idempotent.tmpl")
				writeFile(t, src, "value={{.VALUE}}", 0o644)
				dst := filepath.Join(dir, "idempotent.txt")
				return &config.Step{
					ID:   "idempotent",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
						Vars:        map[string]string{"VALUE": "123"},
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.Equal(t, model.StatusSuccess, res.Status)
				require.Equal(t, "value=123", readFile(t, step.Template.Destination))
			}},
			second: &expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.Equal(t, model.StatusSkipped, res.Status)
			}},
		},
		{
			name: "repairs permissions when content unchanged",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "mode-repair.tmpl")
				writeFile(t, src, "value={{.VALUE}}", 0o600)
				dst := filepath.Join(dir, "mode-repair.txt")
				mode := uint32(0o600)
				return &config.Step{
					ID:   "mode-repair",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
						Vars:        map[string]string{"VALUE": "1"},
						Mode:        &mode,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.Equal(t, model.StatusSuccess, res.Status)
				require.Equal(t, "value=1", readFile(t, step.Template.Destination))
				info, statErr := os.Stat(step.Template.Destination)
				require.NoError(t, statErr)
				require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
				require.NoError(t, os.Chmod(step.Template.Destination, 0o644))
			}},
			second: &expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.Equal(t, model.StatusSuccess, res.Status)
				info, statErr := os.Stat(step.Template.Destination)
				require.NoError(t, statErr)
				require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
			}},
			skipWindows: true,
		},
		{
			name: "fails when source file is missing",
			prepare: func(t *testing.T, dir string) *config.Step {
				return &config.Step{
					ID:   "missing-source",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      filepath.Join(dir, "does-not-exist.tmpl"),
						Destination: filepath.Join(dir, "out.txt"),
						Env:         false,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.Error(t, err)
				require.NotNil(t, res)
				require.Equal(t, model.StatusFailed, res.Status)
				require.Contains(t, res.Message, "stat source")
			}},
		},
		{
			name: "missing variable fails when allow_missing is false",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "missing.tmpl")
				writeFile(t, src, "value={{.VALUE}}", 0o644)
				dst := filepath.Join(dir, "missing.txt")
				return &config.Step{
					ID:   "missing",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.Error(t, err)
				require.NotNil(t, res)
				require.Equal(t, model.StatusFailed, res.Status)
				require.Contains(t, res.Message, "missing")
			}},
		},
		{
			name: "allow_missing renders with empty values",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "allow.tmpl")
				writeFile(t, src, "value={{.VALUE}}", 0o644)
				dst := filepath.Join(dir, "allow.txt")
				return &config.Step{
					ID:   "allow",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:       src,
						Destination:  dst,
						Env:          false,
						AllowMissing: true,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.Equal(t, model.StatusSuccess, res.Status)
				require.Equal(t, "value=", readFile(t, step.Template.Destination))
			}},
		},
		{
			name: "template syntax error reports failure",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "broken.tmpl")
				writeFile(t, src, "{{.VALUE", 0o644)
				dst := filepath.Join(dir, "broken.txt")
				return &config.Step{
					ID:   "broken",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.Error(t, err)
				require.NotNil(t, res)
				require.Equal(t, model.StatusFailed, res.Status)
				require.Contains(t, res.Message, "parse template")
			}},
		},
		{
			name: "fails when destination is an existing directory",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "dir-dest.tmpl")
				writeFile(t, src, "value", 0o644)
				dst := filepath.Join(dir, "existing-dir")
				require.NoError(t, os.MkdirAll(dst, 0o755))
				return &config.Step{
					ID:   "dest-dir",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.Error(t, err)
				require.NotNil(t, res)
				require.Equal(t, model.StatusFailed, res.Status)
				require.Contains(t, res.Message, "directory")
			}},
		},
		{
			name: "write fails when parent directory is not writable",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "readonly-parent.tmpl")
				writeFile(t, src, "value", 0o644)
				parent := filepath.Join(dir, "readonly")
				require.NoError(t, os.MkdirAll(parent, 0o755))
				require.NoError(t, os.Chmod(parent, 0o555))
				t.Cleanup(func() {
					_ = os.Chmod(parent, 0o755)
				})
				dst := filepath.Join(parent, "out.txt")
				return &config.Step{
					ID:   "readonly-parent",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.Error(t, err)
				require.NotNil(t, res)
				require.Equal(t, model.StatusFailed, res.Status)
				require.Contains(t, res.Message, "write destination")
			}},
			skipWindows: true,
		},
		{
			name: "explicit mode sets destination permissions",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "explicit.tmpl")
				writeFile(t, src, "data", 0o644)
				dst := filepath.Join(dir, "explicit.txt")
				mode := uint32(0o600)
				return &config.Step{
					ID:   "explicit-mode",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
						Mode:        &mode,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.Equal(t, model.StatusSuccess, res.Status)
				info, statErr := os.Stat(step.Template.Destination)
				require.NoError(t, statErr)
				require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
			}},
			skipWindows: true,
		},
		{
			name: "mode copied from source when unspecified",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "copy-mode.tmpl")
				writeFile(t, src, "content", 0o600)
				require.NoError(t, os.Chmod(src, 0o700))
				dst := filepath.Join(dir, "copy-mode.txt")
				return &config.Step{
					ID:   "copy-mode",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.Equal(t, model.StatusSuccess, res.Status)
				info, statErr := os.Stat(step.Template.Destination)
				require.NoError(t, statErr)
				require.Equal(t, os.FileMode(0o700), info.Mode().Perm())
			}},
			skipWindows: true,
		},
		{
			name: "parent directories are created automatically",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "nested.tmpl")
				writeFile(t, src, "ok", 0o644)
				dst := filepath.Join(dir, "a", "b", "c", "nested.txt")
				return &config.Step{
					ID:   "nested",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.DirExists(t, filepath.Dir(step.Template.Destination))
				require.Equal(t, "ok", readFile(t, step.Template.Destination))
			}},
		},
		{
			name: "empty template renders empty file",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "empty.tmpl")
				writeFile(t, src, "", 0o644)
				dst := filepath.Join(dir, "empty.txt")
				return &config.Step{
					ID:   "empty",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				require.Equal(t, "", readFile(t, step.Template.Destination))
			}},
		},
		{
			name: "conditionals and loops render correctly",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "complex.tmpl")
				content := strings.Join([]string{
					"{{if eq .ENVIRONMENT \"production\"}}mode=prod{{else}}mode=dev{{end}}",
					"{{range $key, $value := .}}key={{$key}} value={{$value}}\\n{{end}}",
				}, "\n")
				writeFile(t, src, content, 0o644)
				dst := filepath.Join(dir, "complex.txt")
				return &config.Step{
					ID:   "complex",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
						Vars: map[string]string{
							"ENVIRONMENT": "production",
							"SERVICE":     "streamy",
						},
					},
				}
			},
			first: expectation{check: func(t *testing.T, res *model.StepResult, err error, step *config.Step) {
				require.NoError(t, err)
				output := readFile(t, step.Template.Destination)
				require.Contains(t, output, "mode=prod")
				require.Contains(t, output, "key=ENVIRONMENT value=production")
				require.Contains(t, output, "key=SERVICE value=streamy")
			}},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipWindows && runtime.GOOS == "windows" {
				t.Skip("file mode semantics differ on Windows")
			}

			dir := t.TempDir()
			if tc.prepareEnv != nil {
				tc.prepareEnv(t)
			}
			step := tc.prepare(t, dir)
			plugin := New()

			firstRes, firstErr := plugin.Apply(context.Background(), step)
			tc.first.check(t, firstRes, firstErr, step)

			if tc.second != nil {
				secondRes, secondErr := plugin.Apply(context.Background(), step)
				tc.second.check(t, secondRes, secondErr, step)
			}
		})
	}
}

func TestTemplatePlugin_DryRun(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		prepare      func(t *testing.T, dir string) *config.Step
		expectStatus string
		wantErr      bool
		errContains  string
		skipWindows  bool
	}{
		{
			name: "reports would create when destination missing",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "dryrun-create.tmpl")
				writeFile(t, src, "value={{.VALUE}}", 0o644)
				dst := filepath.Join(dir, "missing.txt")
				return &config.Step{
					ID:   "dryrun-create",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Vars:        map[string]string{"VALUE": "1"},
						Env:         false,
					},
				}
			},
			expectStatus: model.StatusWouldCreate,
		},
		{
			name: "skips when destination matches",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "dryrun-skip.tmpl")
				writeFile(t, src, "value={{.VALUE}}", 0o644)
				dst := filepath.Join(dir, "skip.txt")
				writeFile(t, dst, "value=1", 0o644)
				return &config.Step{
					ID:   "dryrun-skip",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Vars:        map[string]string{"VALUE": "1"},
						Env:         false,
					},
				}
			},
			expectStatus: model.StatusSkipped,
		},
		{
			name: "reports would update when content differs",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "dryrun-update.tmpl")
				writeFile(t, src, "value={{.VALUE}}", 0o644)
				dst := filepath.Join(dir, "update.txt")
				writeFile(t, dst, "value=0", 0o644)
				return &config.Step{
					ID:   "dryrun-update",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Vars:        map[string]string{"VALUE": "1"},
						Env:         false,
					},
				}
			},
			expectStatus: model.StatusWouldUpdate,
		},
		{
			name: "syntax error fails dry run",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "dryrun-broken.tmpl")
				writeFile(t, src, "{{.VALUE", 0o644)
				dst := filepath.Join(dir, "broken.txt")
				writeFile(t, dst, "value=0", 0o644)
				return &config.Step{
					ID:   "dryrun-broken",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			expectStatus: model.StatusFailed,
			wantErr:      true,
			errContains:  "parse template",
		},
		{
			name: "missing variable fails dry run",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "dryrun-missing.tmpl")
				writeFile(t, src, "value={{.VALUE}}", 0o644)
				dst := filepath.Join(dir, "missing.txt")
				writeFile(t, dst, "value=0", 0o644)
				return &config.Step{
					ID:   "dryrun-missing",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			expectStatus: model.StatusFailed,
			wantErr:      true,
			errContains:  "missing",
		},
		{
			name: "fails when destination is a directory",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "dryrun-dir.tmpl")
				writeFile(t, src, "value", 0o644)
				dst := filepath.Join(dir, "existing-dir")
				require.NoError(t, os.MkdirAll(dst, 0o755))
				return &config.Step{
					ID:   "dryrun-directory",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			expectStatus: model.StatusFailed,
			wantErr:      true,
			errContains:  "directory",
		},
		{
			name: "fails when destination is unreadable",
			prepare: func(t *testing.T, dir string) *config.Step {
				src := filepath.Join(dir, "dryrun-unreadable.tmpl")
				writeFile(t, src, "value", 0o644)
				dst := filepath.Join(dir, "unreadable.txt")
				writeFile(t, dst, "value", 0o644)
				require.NoError(t, os.Chmod(dst, 0o000))
				t.Cleanup(func() {
					_ = os.Chmod(dst, 0o644)
				})
				return &config.Step{
					ID:   "dryrun-unreadable",
					Type: templatePluginType,
					Template: &config.TemplateStep{
						Source:      src,
						Destination: dst,
						Env:         false,
					},
				}
			},
			expectStatus: model.StatusFailed,
			wantErr:      true,
			errContains:  "hash destination",
			skipWindows:  true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipWindows && runtime.GOOS == "windows" {
				t.Skip("permission semantics differ on Windows")
			}
			t.Parallel()
			dir := t.TempDir()
			step := tc.prepare(t, dir)

			res, err := New().DryRun(context.Background(), step)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.ErrorContains(t, err, tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
			require.NotNil(t, res)
			require.Equal(t, tc.expectStatus, res.Status)
		})
	}
}

func TestTemplatePlugin_QuickstartIntegration(t *testing.T) {
	dir := t.TempDir()
	templatesDir := filepath.Join(dir, "templates")
	require.NoError(t, os.MkdirAll(templatesDir, 0o755))

	appTemplate := `# Application Configuration
app_name = {{.APP_NAME}}
environment = {{.ENVIRONMENT}}
debug = {{.DEBUG_MODE}}

{{if eq .ENVIRONMENT "production"}}
max_connections = 100
timeout = 30
{{else}}
max_connections = 10
timeout = 300
{{end}}

database_url = {{.DATABASE_URL}}
`
	secretTemplate := "API_KEY={{.API_KEY}}\nSECRET={{.SECRET_TOKEN}}\n"
	optionalTemplate := `OptionalValue={{.OPTIONAL}}
{{if .EXTRA}}
ExtraValue={{.EXTRA}}
{{end}}
`
	brokenTemplate := "{{.VAR1}}\n{{if .VAR2}}"

	writeFile(t, filepath.Join(templatesDir, "app.conf.tmpl"), appTemplate, 0o644)
	writeFile(t, filepath.Join(templatesDir, "secret.txt.tmpl"), secretTemplate, 0o600)
	writeFile(t, filepath.Join(templatesDir, "optional.conf.tmpl"), optionalTemplate, 0o644)
	writeFile(t, filepath.Join(templatesDir, "broken.tmpl"), brokenTemplate, 0o644)

	t.Setenv("API_KEY", "test-key-12345")
	t.Setenv("SECRET_TOKEN", "super-secret")

	plug := New()

	appStep := &config.Step{
		ID:   "render-app-config",
		Type: templatePluginType,
		Template: &config.TemplateStep{
			Source:      filepath.Join(templatesDir, "app.conf.tmpl"),
			Destination: filepath.Join(dir, "config", "app.conf"),
			Env:         false,
			Vars: map[string]string{
				"APP_NAME":     "MyApp",
				"ENVIRONMENT":  "development",
				"DEBUG_MODE":   "true",
				"DATABASE_URL": "postgres://localhost/mydb",
			},
		},
	}

	secretsStep := &config.Step{
		ID:   "render-secrets",
		Type: templatePluginType,
		Template: &config.TemplateStep{
			Source:      filepath.Join(templatesDir, "secret.txt.tmpl"),
			Destination: filepath.Join(dir, "config", "secrets.txt"),
			Env:         true,
		},
	}

	optionalStep := &config.Step{
		ID:   "render-optional",
		Type: templatePluginType,
		Template: &config.TemplateStep{
			Source:       filepath.Join(templatesDir, "optional.conf.tmpl"),
			Destination:  filepath.Join(dir, "config", "optional.conf"),
			Env:          false,
			AllowMissing: true,
			Vars: map[string]string{
				"OPTIONAL": "present",
			},
		},
	}

	missingVarStep := &config.Step{
		ID:   "render-missing-var",
		Type: templatePluginType,
		Template: &config.TemplateStep{
			Source:      filepath.Join(templatesDir, "app.conf.tmpl"),
			Destination: filepath.Join(dir, "config", "missing.conf"),
			Env:         false,
			Vars: map[string]string{
				"APP_NAME": "TestApp",
			},
		},
	}

	brokenStep := &config.Step{
		ID:   "render-broken",
		Type: templatePluginType,
		Template: &config.TemplateStep{
			Source:      filepath.Join(templatesDir, "broken.tmpl"),
			Destination: filepath.Join(dir, "config", "broken.txt"),
			Env:         false,
		},
	}

	res, err := plug.DryRun(context.Background(), appStep)
	require.NoError(t, err)
	require.Equal(t, model.StatusWouldCreate, res.Status)

	res, err = plug.DryRun(context.Background(), secretsStep)
	require.NoError(t, err)
	require.Equal(t, model.StatusWouldCreate, res.Status)

	res, err = plug.Apply(context.Background(), appStep)
	require.NoError(t, err)
	require.Equal(t, model.StatusSuccess, res.Status)
	require.Contains(t, readFile(t, appStep.Template.Destination), "app_name = MyApp")

	res, err = plug.Apply(context.Background(), secretsStep)
	require.NoError(t, err)
	require.Equal(t, model.StatusSuccess, res.Status)
	info, statErr := os.Stat(secretsStep.Template.Destination)
	require.NoError(t, statErr)
	if runtime.GOOS != "windows" {
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
	secretsContent := readFile(t, secretsStep.Template.Destination)
	require.Contains(t, secretsContent, "API_KEY=test-key-12345")
	require.Contains(t, secretsContent, "SECRET=super-secret")
	if runtime.GOOS != "windows" {
		require.NoError(t, os.Chmod(secretsStep.Template.Destination, 0o644))
		res, err = plug.Apply(context.Background(), secretsStep)
		require.NoError(t, err)
		require.Equal(t, model.StatusSuccess, res.Status)
		info, statErr := os.Stat(secretsStep.Template.Destination)
		require.NoError(t, statErr)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}

	res, err = plug.Apply(context.Background(), optionalStep)
	require.NoError(t, err)
	require.Equal(t, model.StatusSuccess, res.Status)
	optionalContent := readFile(t, optionalStep.Template.Destination)
	require.Contains(t, optionalContent, "OptionalValue=present")

	res, err = plug.Apply(context.Background(), appStep)
	require.NoError(t, err)
	require.Equal(t, model.StatusSkipped, res.Status)

	res, err = plug.Apply(context.Background(), missingVarStep)
	require.Error(t, err)
	require.NotNil(t, res)
	require.Equal(t, model.StatusFailed, res.Status)
	require.Contains(t, res.Message, "ENVIRONMENT")

	res, err = plug.Apply(context.Background(), brokenStep)
	require.Error(t, err)
	require.NotNil(t, res)
	require.Equal(t, model.StatusFailed, res.Status)
	require.Contains(t, res.Message, "parse template")
}

func writeFile(t *testing.T, path string, content string, perm os.FileMode) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), perm))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	bytes, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(bytes)
}

func TestTemplatePlugin_Verify(t *testing.T) {
	t.Run("returns satisfied when rendered template matches destination", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateSrc := filepath.Join(tmpDir, "template.tmpl")
		destination := filepath.Join(tmpDir, "output.txt")

		writeFile(t, templateSrc, "Hello {{ .name }}!", 0o644)
		writeFile(t, destination, "Hello World!", 0o644)

		p := New()

		step := &config.Step{
			ID:   "render_template",
			Type: "template",
			Template: &config.TemplateStep{
				Source:      templateSrc,
				Destination: destination,
				Vars: map[string]string{
					"name": "World",
				},
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "satisfied", string(result.Status))
		require.Contains(t, result.Message, "matches")
	})

	t.Run("returns missing when destination does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateSrc := filepath.Join(tmpDir, "template.tmpl")
		destination := filepath.Join(tmpDir, "nonexistent.txt")

		writeFile(t, templateSrc, "Hello {{ .name }}!", 0o644)

		p := New()

		step := &config.Step{
			ID:   "render_template",
			Type: "template",
			Template: &config.TemplateStep{
				Source:      templateSrc,
				Destination: destination,
				Vars: map[string]string{
					"name": "World",
				},
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "missing", string(result.Status))
		require.Contains(t, result.Message, "does not exist")
	})

	t.Run("returns drifted when rendered content differs from destination", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateSrc := filepath.Join(tmpDir, "template.tmpl")
		destination := filepath.Join(tmpDir, "output.txt")

		writeFile(t, templateSrc, "Hello {{ .name }}!", 0o644)
		writeFile(t, destination, "Hello Universe!", 0o644)

		p := New()

		step := &config.Step{
			ID:   "render_template",
			Type: "template",
			Template: &config.TemplateStep{
				Source:      templateSrc,
				Destination: destination,
				Vars: map[string]string{
					"name": "World",
				},
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "drifted", string(result.Status))
		require.Contains(t, result.Message, "differs")
	})

	t.Run("returns blocked when context is cancelled", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateSrc := filepath.Join(tmpDir, "template.tmpl")
		destination := filepath.Join(tmpDir, "output.txt")

		writeFile(t, templateSrc, "Hello!", 0o644)

		p := New()

		step := &config.Step{
			ID:   "render_template",
			Type: "template",
			Template: &config.TemplateStep{
				Source:      templateSrc,
				Destination: destination,
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := p.Verify(ctx, step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "blocked", string(result.Status))
		require.Contains(t, result.Message, "cancelled")
		require.NotNil(t, result.Error)
	})

	t.Run("returns error when template config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:       "render_template",
			Type:     "template",
			Template: nil,
		}

		_, err := p.Verify(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "template configuration missing")
	})

	t.Run("returns blocked when template rendering fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateSrc := filepath.Join(tmpDir, "template.tmpl")
		destination := filepath.Join(tmpDir, "output.txt")

		// Template with error - undefined variable
		writeFile(t, templateSrc, "Hello {{ .undefined }}!", 0o644)

		p := New()

		step := &config.Step{
			ID:   "render_template",
			Type: "template",
			Template: &config.TemplateStep{
				Source:      templateSrc,
				Destination: destination,
				Vars:        map[string]string{},
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "blocked", string(result.Status))
		require.Contains(t, result.Message, "cannot render template")
	})
}
