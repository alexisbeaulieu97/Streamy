package plugin

import "testing"

func TestMetadataValidate(t *testing.T) {
	meta := Metadata{
		ID:           "pkg",
		Name:         "Package Plugin",
		Version:      "1.0.0",
		Type:         TypePackage,
		Dependencies: []string{"repo"},
		APIVersion:   "1.0",
	}

	if err := meta.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	invalid := Metadata{}
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected validation error for empty metadata")
	}

	cases := []struct {
		name string
		meta Metadata
	}{
		{
			name: "missing id",
			meta: Metadata{
				Name:    "Pkg",
				Version: "1.0.0",
				Type:    TypePackage,
			},
		},
		{
			name: "missing name",
			meta: Metadata{
				ID:      "pkg",
				Version: "1.0.0",
				Type:    TypePackage,
			},
		},
		{
			name: "missing version",
			meta: Metadata{
				ID:   "pkg",
				Name: "Pkg",
				Type: TypePackage,
			},
		},
		{
			name: "unsupported type",
			meta: Metadata{
				ID:      "pkg",
				Name:    "Pkg",
				Version: "1.0.0",
				Type:    Type("other"),
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.meta.Validate(); err == nil {
				t.Fatalf("expected validation failure for %s", tc.name)
			}
		})
	}
}
