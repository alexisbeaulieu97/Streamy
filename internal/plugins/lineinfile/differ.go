package lineinfileplugin

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

const (
	actionAppend = "append"
	actionRemove = "remove"
	actionNone   = "none"
	actionUpdate = "update"
)

// ChangeSet summarises the differences between original and modified content.
type ChangeSet struct {
	Action  string
	Diff    string
	Changed bool
}

func generateChangeSet(original, modified []string) *ChangeSet {
	cs := &ChangeSet{}

	switch {
	case len(original) == 0 && len(modified) > 0:
		cs.Action = actionAppend
	case len(original) > 0 && len(modified) == 0:
		cs.Action = actionRemove
	case equalLines(original, modified):
		cs.Action = actionNone
	default:
		cs.Action = actionUpdate
	}

	cs.Changed = cs.Action != actionNone

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
