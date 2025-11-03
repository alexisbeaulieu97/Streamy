package registry

import "testing"

func TestBuildCanonicalPipelineID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		id      string
		version string
		want    string
		wantErr bool
	}{
		{
			name:    "valid inputs",
			id:      "app-deploy",
			version: "1.0",
			want:    "app-deploy@1.0",
		},
		{
			name:    "trims whitespace",
			id:      " backend-api ",
			version: " 2.3.4 ",
			want:    "backend-api@2.3.4",
		},
		{
			name:    "invalid id format",
			id:      "Backend",
			version: "1.0",
			wantErr: true,
		},
		{
			name:    "empty version",
			id:      "service",
			version: "",
			wantErr: true,
		},
		{
			name:    "version with whitespace",
			id:      "service",
			version: "1.0 beta",
			wantErr: true,
		},
		{
			name:    "version with separator",
			id:      "service",
			version: "1.0@alpha",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := BuildCanonicalPipelineID(tc.id, tc.version)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("BuildCanonicalPipelineID(%q, %q) expected error, got nil", tc.id, tc.version)
				}

				return
			}

			if err != nil {
				t.Fatalf("BuildCanonicalPipelineID(%q, %q) unexpected error: %v", tc.id, tc.version, err)
			}

			if got != tc.want {
				t.Fatalf("BuildCanonicalPipelineID(%q, %q) = %q, want %q", tc.id, tc.version, got, tc.want)
			}
		})
	}
}

func TestParseCanonicalPipelineID(t *testing.T) {
	t.Parallel()

	id, version, err := ParseCanonicalPipelineID("core-db@1.6.0")
	if err != nil {
		t.Fatalf("ParseCanonicalPipelineID returned error: %v", err)
	}

	if id != "core-db" {
		t.Fatalf("ParseCanonicalPipelineID returned id %q, want %q", id, "core-db")
	}

	if version != "1.6.0" {
		t.Fatalf("ParseCanonicalPipelineID returned version %q, want %q", version, "1.6.0")
	}
}

func TestParseCanonicalPipelineID_Invalid(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"",
		"invalid-id",
		"bad!@1.0",
		"core@app@1.0",
		"core@1.0 beta",
	}

	for _, value := range invalid {
		value := value
		t.Run(value, func(t *testing.T) {
			if _, _, err := ParseCanonicalPipelineID(value); err == nil {
				t.Fatalf("ParseCanonicalPipelineID(%q) expected error, got nil", value)
			}
		})
	}
}

func TestValidateCanonicalPipelineID(t *testing.T) {
	t.Parallel()

	if err := ValidateCanonicalPipelineID("scheduler@2024.10"); err != nil {
		t.Fatalf("ValidateCanonicalPipelineID returned error: %v", err)
	}
}

func TestIsCanonicalPipelineID(t *testing.T) {
	t.Parallel()

	if !IsCanonicalPipelineID("logging@1.0") {
		t.Fatalf("IsCanonicalPipelineID returned false for valid identifier")
	}

	if IsCanonicalPipelineID("logging") {
		t.Fatalf("IsCanonicalPipelineID returned true for invalid identifier")
	}
}
