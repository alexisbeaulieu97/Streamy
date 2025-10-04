package lineinfileplugin

import (
	"fmt"
	"regexp"
)

// MatchResult describes the outcome of applying a regex over file lines.
type MatchResult struct {
	Matched      bool
	LineNumbers  []int
	MatchedLines []string
	MatchCount   int
}

func findMatches(lines []string, pattern *regexp.Regexp) *MatchResult {
	result := &MatchResult{}
	if pattern == nil {
		return result
	}
	for idx, line := range lines {
		if pattern.MatchString(line) {
			result.Matched = true
			result.LineNumbers = append(result.LineNumbers, idx)
			result.MatchedLines = append(result.MatchedLines, line)
			result.MatchCount++
		}
	}
	return result
}

func appendLineIfMissing(lines []string, line string) ([]string, bool) {
	for _, existing := range lines {
		if existing == line {
			return lines, false
		}
	}
	return append(lines, line), true
}

func replaceLines(lines []string, result *MatchResult, newLine string, strategy string) ([]string, bool, error) {
	if result == nil || !result.Matched {
		return lines, false, nil
	}

	switch strategy {
	case onMultipleError:
		if result.MatchCount > 1 {
			return lines, false, fmt.Errorf("multiple matches found")
		}
		fallthrough
	case onMultipleFirst, "":
		changed := false
		idx := result.LineNumbers[0]
		if lines[idx] != newLine {
			lines[idx] = newLine
			changed = true
		}
		return lines, changed, nil
	case onMultipleAll:
		changed := false
		for _, idx := range result.LineNumbers {
			if lines[idx] != newLine {
				lines[idx] = newLine
				changed = true
			}
		}
		return lines, changed, nil
	case onMultiplePrompt:
		if result.MatchCount > 1 {
			return lines, false, fmt.Errorf("multiple matches require interactive prompt")
		}
		changed := false
		idx := result.LineNumbers[0]
		if lines[idx] != newLine {
			lines[idx] = newLine
			changed = true
		}
		return lines, changed, nil
	default:
		return lines, false, fmt.Errorf("unsupported on_multiple_matches strategy: %s", strategy)
	}
}

func removeMatchedLines(lines []string, result *MatchResult) ([]string, bool) {
	if result == nil || !result.Matched {
		return lines, false
	}
	filtered := make([]string, 0, len(lines))
	matchIdx := 0
	for i, line := range lines {
		if matchIdx < len(result.LineNumbers) && i == result.LineNumbers[matchIdx] {
			matchIdx++
			continue
		}
		filtered = append(filtered, line)
	}
	changed := len(filtered) != len(lines)
	return filtered, changed
}
