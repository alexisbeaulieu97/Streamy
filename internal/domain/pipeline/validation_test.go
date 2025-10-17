package pipeline

import (
	"errors"
	"testing"
)

func TestValidationValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      Validation
		wantErr  bool
		wantCode ErrorCode
	}{
		{
			name: "command exists valid",
			val: Validation{
				Type:   ValidationCommandExists,
				Config: map[string]interface{}{"command": "git"},
			},
		},
		{
			name: "file exists valid",
			val: Validation{
				Type:   ValidationFileExists,
				Config: map[string]interface{}{"path": "/tmp/file"},
			},
		},
		{
			name: "path contains valid",
			val: Validation{
				Type:   ValidationPathContains,
				Config: map[string]interface{}{"file": "/tmp/file", "text": "needle"},
			},
		},
		{
			name:     "missing type",
			val:      Validation{Config: map[string]interface{}{}},
			wantErr:  true,
			wantCode: ErrCodeMissing,
		},
		{
			name:     "unsupported type",
			val:      Validation{Type: ValidationType("bogus"), Config: map[string]interface{}{}},
			wantErr:  true,
			wantCode: ErrCodeType,
		},
		{
			name:     "command exists missing config",
			val:      Validation{Type: ValidationCommandExists},
			wantErr:  true,
			wantCode: ErrCodeValidation,
		},
		{
			name: "command exists empty string",
			val: Validation{
				Type:   ValidationCommandExists,
				Config: map[string]interface{}{"command": "   "},
			},
			wantErr:  true,
			wantCode: ErrCodeValidation,
		},
		{
			name: "file exists missing path",
			val: Validation{
				Type:   ValidationFileExists,
				Config: map[string]interface{}{},
			},
			wantErr:  true,
			wantCode: ErrCodeMissing,
		},
		{
			name: "path contains missing file",
			val: Validation{
				Type:   ValidationPathContains,
				Config: map[string]interface{}{"text": "needle"},
			},
			wantErr:  true,
			wantCode: ErrCodeMissing,
		},
		{
			name: "path contains non-string",
			val: Validation{
				Type:   ValidationPathContains,
				Config: map[string]interface{}{"file": "/tmp/file", "text": 42},
			},
			wantErr:  true,
			wantCode: ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.val.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr {
				return
			}
			var domainErr *DomainError
			if !errors.As(err, &domainErr) {
				t.Fatalf("expected DomainError, got %T", err)
			}
			if domainErr.Code != tt.wantCode {
				t.Fatalf("unexpected code: got %s want %s", domainErr.Code, tt.wantCode)
			}
		})
	}
}
