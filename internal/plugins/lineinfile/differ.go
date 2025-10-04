package lineinfileplugin

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// ChangeSet summarises the differences between original and modified content.
type ChangeSet struct {
	Action  string
	Diff    string
	Changed bool
}

func generateChangeSet(original, modified []string) *ChangeSet {
	cs := &ChangeSet{}
	if len(original) == 0 && len(modified) > 0 {
		cs.Action = "append"
	} else if len(original) > 0 && len(modified) == 0 {
		cs.Action = "remove"
	} else if equalLines(original, modified) {
		cs.Action = "none"
	} else {
		cs.Action = "update"
	}
	cs.Changed = cs.Action != "none"

	ud := difflib.UnifiedDiff{
		A:        original,
		B:        modified,
		FromFile: "original",
		ToFile:   "modified",
		Context:  3,
	}
	diff, _ := difflib.GetUnifiedDiffString(ud)
	cs.Diff = strings.TrimSpace(diff)
	return cs
}

func equalLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
