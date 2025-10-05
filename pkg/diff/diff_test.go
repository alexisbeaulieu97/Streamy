package diff

import (
	"strings"
	"testing"
)

func TestGenerateUnifiedDiff_IdenticalContent(t *testing.T) {
	expected := []byte("line1\nline2\nline3\n")
	actual := []byte("line1\nline2\nline3\n")
	
	result := GenerateUnifiedDiff(expected, actual, "expected", "actual")
	
	if result != "" {
		t.Errorf("Expected empty diff for identical content, got: %s", result)
	}
}

func TestGenerateUnifiedDiff_SingleLineChange(t *testing.T) {
	expected := []byte("line1\nline2\nline3\n")
	actual := []byte("line1\nmodified\nline3\n")
	
	result := GenerateUnifiedDiff(expected, actual, "expected", "actual")
	
	if result == "" {
		t.Error("Expected non-empty diff for different content")
	}
	
	if !strings.Contains(result, "---") || !strings.Contains(result, "+++") {
		t.Error("Diff should contain unified diff headers")
	}
	
	if !strings.Contains(result, "-line2") {
		t.Error("Diff should show removed line with - prefix")
	}
	
	if !strings.Contains(result, "+modified") {
		t.Error("Diff should show added line with + prefix")
	}
}

func TestGenerateUnifiedDiff_MultiLineChanges(t *testing.T) {
	expected := []byte("line1\nline2\nline3\nline4\nline5\n")
	actual := []byte("line1\nmodified2\nmodified3\nline4\nline5\n")
	
	result := GenerateUnifiedDiff(expected, actual, "expected.txt", "actual.txt")
	
	if result == "" {
		t.Error("Expected non-empty diff for different content")
	}
	
	// Check for context lines (unchanged lines around changes)
	if !strings.Contains(result, " line1") || !strings.Contains(result, " line4") {
		t.Error("Diff should include context lines")
	}
	
	// Check that changes are present (algorithm may split differently)
	if !strings.Contains(result, "modified") {
		t.Error("Diff should show modified lines")
	}
	
	// Verify we have both add and remove markers
	if !strings.Contains(result, "-") || !strings.Contains(result, "+") {
		t.Error("Diff should contain both additions and removals")
	}
}

func TestGenerateUnifiedDiff_Truncation(t *testing.T) {
	// Create content with > 10,000 lines
	var expectedLines []string
	var actualLines []string
	
	for i := 0; i < 11000; i++ {
		expectedLines = append(expectedLines, "expected line")
		if i%2 == 0 {
			actualLines = append(actualLines, "actual line")
		} else {
			actualLines = append(actualLines, "expected line")
		}
	}
	
	expected := []byte(strings.Join(expectedLines, "\n"))
	actual := []byte(strings.Join(actualLines, "\n"))
	
	result := GenerateUnifiedDiff(expected, actual, "expected", "actual")
	
	if result == "" {
		t.Error("Expected non-empty diff for different content")
	}
	
	if !strings.Contains(result, "truncated") {
		t.Error("Large diff should be truncated with truncation message")
	}
	
	lineCount := strings.Count(result, "\n")
	if lineCount > 10100 { // Allow some margin for headers
		t.Errorf("Truncated diff should not exceed ~10,000 lines, got %d", lineCount)
	}
}

func TestGenerateUnifiedDiff_EmptyContent(t *testing.T) {
	expected := []byte("")
	actual := []byte("new content\n")
	
	result := GenerateUnifiedDiff(expected, actual, "expected", "actual")
	
	if result == "" {
		t.Error("Expected non-empty diff when adding content to empty file")
	}
	
	if !strings.Contains(result, "+new content") {
		t.Error("Diff should show added content")
	}
}

func TestGenerateUnifiedDiff_Labels(t *testing.T) {
	expected := []byte("old")
	actual := []byte("new")
	
	result := GenerateUnifiedDiff(expected, actual, "file1.txt", "file2.txt")
	
	if !strings.Contains(result, "--- file1.txt") {
		t.Error("Diff should contain expected file label")
	}
	
	if !strings.Contains(result, "+++ file2.txt") {
		t.Error("Diff should contain actual file label")
	}
}
