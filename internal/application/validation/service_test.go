package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
)

func TestServiceRunValidations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "check.txt")
	if err := os.WriteFile(filePath, []byte("welcome to streamy"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	svc := NewService(logging.NewNoOpLogger())

	validations := []domain.Validation{
		{
			Type: domain.ValidationFileExists,
			Config: map[string]interface{}{
				"path": filePath,
			},
		},
		{
			Type: domain.ValidationPathContains,
			Config: map[string]interface{}{
				"file": filePath,
				"text": "streamy",
			},
		},
	}

	summary, err := svc.RunValidations(context.Background(), validations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalChecks != 2 || summary.PassedChecks != 2 || summary.FailedChecks != 0 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
}

func TestServiceRunValidationsFailure(t *testing.T) {
	svc := NewService(logging.NewNoOpLogger())

	validations := []domain.Validation{
		{
			Type:   domain.ValidationFileExists,
			Config: map[string]interface{}{"path": "/path/does/not/exist"},
		},
	}

	summary, err := svc.RunValidations(context.Background(), validations)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if summary.FailedChecks != 1 {
		t.Fatalf("expected one failed check, got %+v", summary)
	}
}
