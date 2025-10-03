# Repository Guidelines

## Project Structure & Module Organization
Streamy is a Go module (`go.mod` at the root). Place runnable entry points under `cmd/<service>/main.go`. Share reusable packages via `internal/` for private logic and `pkg/` for public utilities. Keep HTTP handlers, gRPC services, and background jobs grouped by feature folders. Store configuration presets in `configs/`, fixtures in `testdata/`, and static assets in `assets/`. Add architectural notes under `docs/` (use `docs/adr/` for longer-lived decisions).

## Build, Test, and Development Commands
`go mod tidy` cleans dependency metadata; run it whenever Go files are added or imports change. `go build ./...` validates that all packages compile. `go test ./...` executes the unit tests; pair it with `-cover` to check coverage. During local development, execute `go run ./cmd/server` (or whichever binary you are touching) to start the service with the default configuration.

## Coding Style & Naming Conventions
Always format files with `gofmt` (or `go fmt ./...`); the project expects Go’s tab-based indentation. Organize imports using `goimports` to group standard, third-party, and local packages. Favor descriptive exported names (`PlayerService`, `NewStreamStore`) and keep package names short and lower-case. Tests should follow the same naming scheme (`TestHandlePlayback`). Prefer `snake_case` for JSON or YAML config keys and hyphenated file names for CLI scripts.

## Testing Guidelines
Co-locate tests beside the code by creating `*_test.go` files. Use table-driven tests and helper functions marked with `t.Helper()`. Mark slower suites with build tags such as `//go:build integration` and run them via `go test -tags=integration ./...`. Capture coverage reports with `go test ./... -coverprofile=coverage.out` before submitting larger changes.

## Commit & Pull Request Guidelines
The history is new, so use Conventional Commits (e.g., `feat: add stream session store`). Keep each commit scoped to a single concern and include any schema or config migrations. Pull requests must summarize the change set, link related issues, and note manual verification (`go test ./...`, local run, etc.). Add screenshots or API examples when touching user-facing flows.
