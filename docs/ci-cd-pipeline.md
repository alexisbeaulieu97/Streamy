# CI/CD Pipeline

## Overview

The Streamy project uses GitHub Actions for automated testing, linting, building, and releasing. The pipeline ensures code quality, test coverage, and reliable releases across multiple platforms.

## Workflows

### 1. **CI Workflow** (`.github/workflows/ci.yml`)

Runs on every push to `main` and all pull requests, plus nightly at 2 AM UTC.

#### Jobs:

**Test Job** (Ubuntu Latest, Matrix: Go 1.24, 1.25)
- ✅ Verifies `go.mod` is tidy
- ✅ Checks code formatting with `gofmt`
- ✅ Runs all tests with race detector (`-race`)
- ✅ Measures unit test coverage (excluding `cmd/`, `internal/tui`, `internal/ui`)
- ✅ Enforces 80% coverage threshold
- ✅ Runs TUI integration tests

**Lint Job** (Ubuntu Latest)
- ✅ Runs golangci-lint with full caching
- ✅ Checks code style and potential bugs
- ✅ Configurable timeout (5m default)

**Build Job** (Ubuntu Latest, depends on test & lint)
- ✅ Builds binaries for all platforms using `scripts/build.sh`
- ✅ Uploads artifacts for inspection (5-day retention)

#### Key Features:
- **Matrix Testing**: Tests across multiple Go versions for compatibility
- **Fail-Fast Disabled**: All versions run even if one fails
- **Concurrency**: Cancels in-flight runs when new commits arrive
- **Caching**: Aggressive Go module and build caching
- **Permissions**: Minimal read-only permissions for security

### 2. **Release Workflow** (`.github/workflows/release.yml`)

Triggers on version tags (e.g., `v1.2.3`).

#### Steps:
1. ✅ Validates tag is semantic version (`v*.*.*` format)
2. ✅ Runs full test suite before release
3. ✅ Builds and publishes with GoReleaser
4. ✅ Generates checksums (SHA256)
5. ✅ Creates SBOM (Software Bill of Materials)

#### Permissions:
- `contents: write` - Create releases
- `packages: write` - Publish packages
- `id-token: write` - For future signing with Sigstore

### 3. **GoReleaser Configuration** (`.goreleaser.yml`)

#### Builds:
- Targets: Linux, Darwin (macOS), Windows
- Architectures: amd64, arm64
- Flags: `-trimpath` for reproducible builds
- Zero CGO for portability

#### Archives:
- **Linux/macOS**: tar.gz with docs and license
- **Windows**: ZIP format
- Consistent naming: `streamy_Linux_x86_64.tar.gz`

#### Artifacts:
- Checksums (SHA256)
- SBOM in SPDX JSON format
- Auto-generated changelog from git history

## Coverage Strategy

### What's Excluded:
- **`cmd/`**: Thin entry points (tested indirectly)
- **`internal/tui`**: Terminal UI (integration tested separately)
- **`internal/ui`**: UI components (difficult to unit test)

### What's Measured:
- **`internal/`**: Core business logic
- **`pkg/`**: Public utilities

### Threshold:
- **80% minimum** on covered packages
- PR fails if below threshold

## Running Locally

Replicate CI behavior locally:

```bash
# Verify formatting
go fmt ./...

# Verify go.mod is tidy
go mod tidy

# Run all tests with race detector
go test -race ./...

# Generate coverage report (matching CI)
go test $(go list ./internal/... ./pkg/... | grep -v '/internal/tui' | grep -v '/internal/ui') -coverprofile=coverage.out -covermode=atomic

# View coverage
go tool cover -html=coverage.out

# Run TUI integration tests
go test -v ./tests/...

# Build binaries
./scripts/build.sh
```

## Release Process

1. **Create a semantic version tag**:
   ```bash
   git tag v1.2.3
   git push origin v1.2.3
   ```

2. **Monitor the release workflow** in GitHub Actions

3. **Verify artifacts** in the GitHub Release page

## Best Practices

✅ **Always test locally before pushing**
✅ **Keep `go.mod` tidy** with `go mod tidy`
✅ **Format code** with `go fmt ./...`
✅ **Run with race detector** during development
✅ **Maintain coverage above 80%** for core logic
✅ **Use semantic versioning** for tags
✅ **Write descriptive commit messages** for changelogs
