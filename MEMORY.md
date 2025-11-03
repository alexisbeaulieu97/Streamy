# Session Memory (Coding Agent)

## Current Focus
- Feature 010 – Pipeline Dependencies: CodeRabbit clean-up completed; codecs aligned with latest spec/docs and CLI tree renderer fixes merged.

## Key Updates
- Registry/status cache mutations are now atomic and persist immediately (`writeAtomic`, updated `StatusCache` API); Ready ASCII fallback changed to `[RD]` with docs/tests updated accordingly.
- Orchestration use case persists cache status, supports forced execution, and carries new unit/integration coverage (see `internal/application/orchestration/*`, `cmd/streamy/orchestrate.go`, `tests/integration_orchestrate_test.go`).
- `streamy registry add` now loads the status cache, reconciles downstream readiness, and prints newly unblocked pipelines; unit tests in `internal/registry/registry_test.go` cover the workflow.
- Docs updated: `docs/pipeline-dependencies.md` documents reconciliation and `--force` safety, `docs/architecture.md` notes CLI safeguards and registry sync.
- Unified CLI entry point: `streamy run` replaces the old `apply`/`orchestrate` commands, choosing between file and registry execution based on mutually exclusive flags. Validation tests live in `cmd/streamy/run_test.go`.
- Documentation now covers `@latest` resolution rules and the quickstart/README examples use the unified `streamy run --registry` flow.
- Status cache setters now defer unlocks around `saveLocked`, closing the last mutex-safety gaps flagged by CodeRabbit.
- `golangci-lint run` and `go test ./...` currently fail because of long-standing project-wide findings (errcheck/gocyclo/wsl) and flaky CLI/TUI tests; rerun locally before committing if the baseline gets fixed.

- Coordinate with reviewers on release packaging (e.g., update release notes, decide on follow-up automation tasks if any).
- Follow-up: rerun `golangci-lint run` and `go test ./...` once baseline issues are resolved upstream (still failing on legacy findings/fixtures).
- Follow-up: address remaining CODERABBIT quickstart/design items (timestamp fields, canonical ID usage, mock executor docs, checklist refinements).

## Verification Status
- (working tree dirty; full suite pending — re-run `go test ./...` and `golangci-lint run` once baseline issues are cleared upstream)
