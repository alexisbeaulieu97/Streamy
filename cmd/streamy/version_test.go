package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCommandOutputsBuildInfo(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalDate := date
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
	})

	version = "1.2.3"
	commit = "abcdef1"
	date = "2025-10-03"

	root := newRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"version"})

	require.NoError(t, root.Execute())

	output := buf.String()
	require.Contains(t, output, "1.2.3")
	require.Contains(t, output, "abcdef1")
	require.Contains(t, output, "2025-10-03")
}
