# Changelog

## Unreleased

### Added
- Comprehensive architecture documentation (`docs/architecture.md`) with layered diagrams and workflow sequences.
- Plugin development guide updates plus dedicated onboarding guide (`docs/adding-plugins.md`) and testing guide (`docs/testing-guide.md`).
- Performance baseline documentation (`docs/performance-baseline.md`) and executor benchmark for 500-step pipelines.
- Port placement ADR (`docs/ADR/002-port-placement-at-boundary.md`).

### Changed
- Refactored Streamy into domain/application/infrastructure layers with ports at the application boundary.
- Migrated built-in plugins to ports-native implementations and removed legacy `internal/config`, `internal/engine`, `internal/plugin`, and `internal/model` packages.
- Updated CLI wiring to use port-based registries with structured logging, metrics, tracing, and event publishers.

### Removed
- Legacy plugin adapters, config schema types, and engine services replaced by new domain and infrastructure layers.

