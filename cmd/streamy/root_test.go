package main

import (
	"context"
	"strings"
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

func TestNewRootCmdInvokesDashboardByDefault(t *testing.T) {
	original := rootDashboardLauncher

	var called bool

	rootDashboardLauncher = func(_ context.Context, _ *AppContext, _ ports.Logger) error {
		called = true
		return nil
	}

	t.Cleanup(func() { rootDashboardLauncher = original })

	app := &AppContext{Logger: noopLogger{}}
	cmd := newRootCmd(app)

	cmd.SetArgs(nil)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute root: %v", err)
	}

	if !called {
		t.Fatal("expected dashboard to be invoked")
	}
}

func TestNewRootCmdAppliesTimeoutFlag(t *testing.T) {
	original := rootDashboardLauncher

	var captured context.Context

	rootDashboardLauncher = func(ctx context.Context, _ *AppContext, _ ports.Logger) error {
		captured = ctx
		return nil
	}

	t.Cleanup(func() { rootDashboardLauncher = original })

	app := &AppContext{Logger: noopLogger{}}
	cmd := newRootCmd(app)
	cmd.SetArgs([]string{"--timeout", "10ms"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute root with timeout: %v", err)
	}

	if captured == nil {
		t.Fatal("expected context to be passed to dashboard")
	}

	if _, ok := captured.Deadline(); !ok {
		t.Fatal("expected deadline to be set by timeout flag")
	}
}

func TestRootCmdDisplaysHelp(t *testing.T) {
	app := &AppContext{Logger: noopLogger{}}
	cmd := newRootCmd(app)

	var output strings.Builder
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected help execution to succeed, got %v", err)
	}

	if !strings.Contains(output.String(), "Streamy automates environment setup") {
		t.Fatalf("expected help output, got %q", output.String())
	}
}

type noopLogger struct{}

func (noopLogger) Debug(_ context.Context, _ string, _ ...interface{}) {}
func (noopLogger) Info(_ context.Context, _ string, _ ...interface{})  {}
func (noopLogger) Warn(_ context.Context, _ string, _ ...interface{})  {}
func (noopLogger) Error(_ context.Context, _ string, _ ...interface{}) {}
func (noopLogger) With(_ ...interface{}) ports.Logger                  { return noopLogger{} }
