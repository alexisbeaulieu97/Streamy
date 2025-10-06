package internalexec

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Result captures stdout/stderr emitted by a streaming command run.
type Result struct {
	Stdout string
	Stderr string
}

// RunStreaming wires the command's stdout/stderr through to the parent process
// while collecting the output for later inspection.
func RunStreaming(cmd *exec.Cmd) (Result, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	if cmd.Stdout != nil {
		cmd.Stdout = io.MultiWriter(cmd.Stdout, &stdoutBuf)
	} else {
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	}
	if cmd.Stderr != nil {
		cmd.Stderr = io.MultiWriter(cmd.Stderr, &stderrBuf)
	} else {
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	}

	err := cmd.Run()

	return Result{
		Stdout: strings.TrimSpace(stdoutBuf.String()),
		Stderr: strings.TrimSpace(stderrBuf.String()),
	}, err
}

// PrimaryOutput returns stderr if present, otherwise stdout.
func PrimaryOutput(res Result) string {
	if res.Stderr != "" {
		return res.Stderr
	}
	return res.Stdout
}
