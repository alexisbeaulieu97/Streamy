package config

import (
	"testing"
)

func TestGetValidator(t *testing.T) {
	v1 := GetValidator()
	v2 := GetValidator()

	// Should return the same instance (singleton)
	if v1 != v2 {
		t.Error("GetValidator should return the same instance (singleton pattern)")
	}
}

func TestGitURLValidation(t *testing.T) {
	v := GetValidator()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Empty strings
		{"empty string", "", true},
		{"space", " ", false},

		// Valid HTTP/HTTPS URLs
		{"valid https", "https://github.com/user/repo.git", true},
		{"valid http", "http://git.example.com/project", true},
		{"https with port", "https://github.com:443/user/repo.git", true},
		{"http with subdomain", "http://gitlab.company.com/group/project", true},

		// Invalid HTTP/HTTPS URLs
		{"no host", "https:///path", false},
		{"empty host", "http://", false},
		{"invalid scheme", "ftp://example.com/repo.git", false},
		{"file scheme", "file:///path/to/repo", false},

		// Valid SSH URLs
		{"ssh git@github", "git@github.com:user/repo.git", true},
		{"ssh with domain", "deploy@server.com:project.git", true},
		{"ssh complex", "user.name@gitlab.com:group/subgroup/project.git", true},
		{"ssh with dots", "john.doe@dev.company.com:awesome-project.git", true},

		// Invalid SSH URLs
		{"ssh no colon", "git@github.com/user/repo.git", false},
		{"ssh no @", "github.com:user/repo.git", false},
		{"ssh empty host", "git@:repo.git", false},
		{"ssh empty path", "git@github.com:", false},

		// Valid local paths
		{"absolute path", "/tmp/example.git", true},
		{"absolute complex", "/home/user/projects/my-repo.git", true},
		{"relative current", "./local/repo", true},
		{"relative parent", "../upstream/project", true},
		{"relative nested", "./../sibling/repo.git", true},

		// Invalid local paths
		{"relative without prefix", "local/repo", false},
		{"just name", "my-repo", false},
		{"nul character", "/tmp/repo\x00.git", false},
		{"path traversal absolute", "/etc/../usr/share/repo", false},
		{"path traversal suffix", "/tmp/..", false},
		{"path traversal middle", "/home/../../../etc/passwd", false},

		// Edge cases
		{"just dots", "../", true},
		{"current dir", "./", true},
		{"root", "/", true},
		{"multiple slashes", "//tmp//repo.git", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Var(tt.url, "git_url")
			got := err == nil

			if got != tt.expected {
				t.Errorf("git_url validation for %q: got %v, expected %v (error: %v)", tt.url, got, tt.expected, err)
			}
		})
	}
}

func TestSemverValidation(t *testing.T) {
	v := GetValidator()

	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		// Valid semver
		{"simple", "1.0.0", true},
		{"with prerelease", "1.0.0-alpha", true},
		{"with build metadata", "1.0.0+build.1", true},
		{"complex", "2.1.3-beta.2+build.123", true},

		// Invalid semver
		{"empty", "", false},
		{"single digit", "1.0", false},
		{"no dots", "v1", false},
		{"too many dots", "1.2.3.4", false},
		{"invalid chars", "1.x.0", false},
		{"negative", "-1.0.0", false},
		{"leading v", "v1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Var(tt.version, "semver")
			got := err == nil

			if got != tt.expected {
				t.Errorf("semver validation for %q: got %v, expected %v (error: %v)", tt.version, got, tt.expected, err)
			}
		})
	}
}

func TestStepIDValidation(t *testing.T) {
	v := GetValidator()

	tests := []struct {
		name     string
		stepID   string
		expected bool
	}{
		// Valid step IDs
		{"simple lowercase", "step1", true},
		{"with underscores", "my_step", true},
		{"with hyphens", "my-step", true},
		{"mixed", "step_1-2", true},
		{"numbers only", "123", true},
		{"single char", "a", true},

		// Invalid step IDs
		{"empty", "", false},
		{"uppercase", "Step1", false},
		{"camel case", "myStep", false},
		{"spaces", "step 1", false},
		{"dots", "step.1", false},
		{"at symbol", "step@1", false},
		{"special chars", "step#1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Var(tt.stepID, "step_id")
			got := err == nil

			if got != tt.expected {
				t.Errorf("step_id validation for %q: got %v, expected %v (error: %v)", tt.stepID, got, tt.expected, err)
			}
		})
	}
}

func TestIsValidFilePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Valid absolute paths
		{"root absolute", "/", true},
		{"simple absolute", "/tmp", true},
		{"complex absolute", "/home/user/projects/repo", true},
		{"absolute with git", "/path/to/repo.git", true},

		// Valid relative paths
		{"current dir", "./", true},
		{"relative file", "./file.txt", true},
		{"relative nested", "./dir/file", true},
		{"parent dir", "../", true},
		{"parent file", "../file.txt", true},
		{"parent nested", "../dir/file", true},
		{"both prefixes", "./../file", true},

		// Invalid paths
		{"empty", "", false},
		{"no prefix", "file.txt", false},
		{"just dots", "..", false},
		{"just file", "filename", false},
		{"windows style", "C:\\path\\file", false},

		// Security concerns
		{"nul char", "/tmp\x00file", false},
		{"nul relative", "./file\x00", false},
		{"path traversal absolute", "/etc/../usr/share", false},
		{"path traversal suffix", "/tmp/..", false},
		{"complex traversal", "/home/user/../../../etc/passwd", false},

		// Edge cases
		{"multiple slashes", "//tmp//file", true},
		{"trailing slash absolute", "/tmp/", true},
		{"trailing slash relative", "./dir/", true},
		{"dots in name", "./my.file.txt", true},
		{"multiple dots", "./.././../file", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidFilePath(tt.path)

			if got != tt.expected {
				t.Errorf("isValidFilePath(%q): got %v, expected %v", tt.path, got, tt.expected)
			}
		})
	}
}

// Test that our custom validators are properly registered
func TestValidatorRegistration(t *testing.T) {
	v := GetValidator()

	// Test that our custom validators exist and are callable
	type TestStruct struct {
		SemverField string `validate:"semver"`
		StepIDField string `validate:"step_id"`
		GitURLField string `validate:"git_url"`
	}

	// Valid case
	valid := TestStruct{
		SemverField: "1.0.0",
		StepIDField: "test_step",
		GitURLField: "https://github.com/user/repo.git",
	}

	if err := v.Struct(valid); err != nil {
		t.Errorf("Valid struct should pass validation: %v", err)
	}

	// Invalid case
	invalid := TestStruct{
		SemverField: "invalid",
		StepIDField: "Invalid Step",
		GitURLField: "not-a-valid-url",
	}

	if err := v.Struct(invalid); err == nil {
		t.Error("Invalid struct should fail validation")
	}
}
