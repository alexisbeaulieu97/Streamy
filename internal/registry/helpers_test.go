package registry

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		prefix string
	}{
		{name: "simple", input: "Dev Setup", want: "dev-setup"},
		{name: "mixedCharacters", input: "My_Project v1.0", want: "my-project-v1-0"},
		{name: "leadingTrailing", input: "--Prod--", want: "prod"},
		{name: "consecutiveSeparators", input: "A    B!!C", want: "a-b-c"},
		{name: "trimToMaxLength", input: strings.Repeat("abc", 25), prefix: "abcabcabcabcabcabcabcabcabcabc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.input)

			if tt.want != "" && got != tt.want {
				t.Fatalf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}

			if tt.prefix != "" && !strings.HasPrefix(got, tt.prefix) {
				t.Fatalf("SanitizeFilename(%q) = %q, expected prefix %q", tt.input, got, tt.prefix)
			}

			if len(got) > pipelineIDMaxLength {
				t.Fatalf("SanitizeFilename(%q) length = %d, exceeds max %d", tt.input, len(got), pipelineIDMaxLength)
			}

			if got != "" {
				if strings.HasPrefix(got, "-") || strings.HasSuffix(got, "-") {
					t.Fatalf("SanitizeFilename(%q) produced ID with leading/trailing hyphen: %q", tt.input, got)
				}
			}
		})
	}
}

func TestValidatePipelineID(t *testing.T) {
	valid := []string{
		"dev",
		"dev-setup",
		"abc123",
		strings.Repeat("a", pipelineIDMaxLength),
	}

	for _, id := range valid {
		if err := ValidatePipelineID(id); err != nil {
			t.Fatalf("ValidatePipelineID(%q) returned error: %v", id, err)
		}
	}

	invalid := []struct {
		id string
	}{
		{""},
		{"Dev"},
		{"-leading"},
		{"trailing-"},
		{"has_underscore"},
		{strings.Repeat("a", pipelineIDMaxLength+1)},
	}

	for _, tt := range invalid {
		if err := ValidatePipelineID(tt.id); err == nil {
			t.Fatalf("ValidatePipelineID(%q) expected error, got nil", tt.id)
		}
	}
}

func TestGeneratePipelineID(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		want   string
		prefix string
	}{
		{name: "simple", path: "/tmp/dev-setup.yaml", want: "dev-setup"},
		{name: "uppercaseAndSpaces", path: "/configs/Prod Setup.yml", want: "prod-setup"},
		{name: "noExtension", path: "/configs/staging", want: "staging"},
		{name: "longName", path: "/configs/" + strings.Repeat("abc", 30) + ".yaml", prefix: "abcabcabcabc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePipelineID(tt.path)
			if tt.want != "" && got != tt.want {
				t.Fatalf("GeneratePipelineID(%q) = %q, want %q", tt.path, got, tt.want)
			}

			if tt.prefix != "" && !strings.HasPrefix(got, tt.prefix) {
				t.Fatalf("GeneratePipelineID(%q) = %q, expected prefix %q", tt.path, got, tt.prefix)
			}

			if err := ValidatePipelineID(got); err != nil {
				t.Fatalf("generated ID %q is invalid: %v", got, err)
			}

			if len(got) > pipelineIDMaxLength {
				t.Fatalf("GeneratePipelineID(%q) produced ID exceeding max length: %d", tt.path, len(got))
			}
		})
	}

	t.Run("nonAlphanumericOnly", func(t *testing.T) {
		path := filepath.Join("/tmp", "!!!.yaml")
		got := GeneratePipelineID(path)
		if !strings.HasPrefix(got, "pipeline-") {
			t.Fatalf("expected fallback prefix for %q, got %q", path, got)
		}
		if len(got) > pipelineIDMaxLength {
			t.Fatalf("fallback ID length = %d exceeds max %d", len(got), pipelineIDMaxLength)
		}
		if err := ValidatePipelineID(got); err != nil {
			t.Fatalf("fallback ID %q failed validation: %v", got, err)
		}
	})
}
