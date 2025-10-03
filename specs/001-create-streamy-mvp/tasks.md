# Tasks: Create Streamy MVP

**Branch**: `001-create-streamy-mvp` | **Date**: 2025-10-03  
**Input**: Design documents from `/home/alexis/Projects/Streamy/specs/001-create-streamy-mvp/`  
**Prerequisites**: plan.md, research.md, data-model.md, quickstart.md

## Overview

This document breaks down the Streamy MVP implementation into 83 ordered, dependency-aware tasks across 12 phases. Tasks marked `[P]` can be executed in parallel (different files, no dependencies). All tasks follow TDD principles with tests written before implementation.

**Update**: Added T083 for test coverage enforcement (80%+ threshold) per constitution principle V.

---

## Phase 3.1: Project Setup & Initialization

- [X] **T001** Initialize Go module with `go.mod` at repository root
  - Run: `go mod init github.com/alexisbeaulieu97/streamy`
  - Set Go version to 1.25.1
  - File: `go.mod`

- [X] **T002** Create project directory structure per plan.md
  - Create: `cmd/streamy/`, `internal/{config,engine,plugin,plugins/{package,repo,symlink,copy,command},validation,tui,logger}`, `pkg/errors/`, `testdata/configs/`, `tests/`, `scripts/`, `docs/`, `.github/workflows/`
  - No code, just directories

- [X] **T003** Install core dependencies
  - Add: `github.com/spf13/cobra`, `github.com/spf13/viper`, `gopkg.in/yaml.v3`, `github.com/go-playground/validator/v10`
  - Run: `go get` for each dependency
  - File: `go.mod`

- [X] **T004** Install TUI and logging dependencies
  - Add: `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`, `github.com/charmbracelet/bubbles`, `github.com/rs/zerolog`
  - Run: `go get` for each dependency
  - File: `go.mod`

- [X] **T005** Install git and testing dependencies
  - Add: `github.com/go-git/go-git/v5`, `github.com/stretchr/testify`
  - Run: `go get` for each dependency
  - File: `go.mod`

- [X] **T006** Configure linting and formatting tools
  - Create: `.golangci.yml` with linter configuration (govet, staticcheck, errcheck, gofmt, goimports)
  - Add GitHub Actions workflow reference (configured in later task)

- [X] **T007** Create basic README.md
  - Include: Project overview, quick install, basic usage example (`streamy apply config.yaml`)
  - Note: Comprehensive docs come later
  - File: `README.md`

---

## Phase 3.2: Core Foundation - Tests First (TDD)

**CRITICAL**: All test tasks (T008-T017) MUST be completed and MUST show failing tests before ANY implementation tasks.

- [X] **T008** [P] Create config parser test in `internal/config/parser_test.go`
  - Test cases: valid YAML, invalid YAML, missing required fields, schema version validation
  - Must fail: No parser implementation yet
  - File: `internal/config/parser_test.go`

- [X] **T009** [P] Create config validator test in `internal/config/validator_test.go`
  - Test cases: unique step IDs, valid dependency references, no circular deps, type validation (package/repo/symlink/copy/command)
  - Must fail: No validator implementation yet
  - File: `internal/config/validator_test.go`

- [X] **T010** [P] Create Step struct validation test in `internal/config/step_test.go`
  - Test cases: each step type with valid/invalid fields, embedded type-specific structs
  - Must fail: No structs defined yet
  - File: `internal/config/step_test.go`

- [X] **T011** [P] Create DAG builder test in `internal/engine/dag_test.go`
  - Test cases: simple graph, multi-level graph, detect cycles, topological sort, level grouping
  - Must fail: No DAG implementation yet
  - File: `internal/engine/dag_test.go`

- [X] **T012** [P] Create dry-run planner test in `internal/engine/planner_test.go`
  - Test cases: generate execution plan, estimate timing, group parallel steps, format output
  - Must fail: No planner implementation yet
  - File: `internal/engine/planner_test.go`

- [X] **T013** [P] Create executor test in `internal/engine/executor_test.go`
  - Test cases: sequential execution, parallel execution, fail-fast on error, timeout handling, context cancellation
  - Must fail: No executor implementation yet
  - File: `internal/engine/executor_test.go`

- [X] **T014** [P] Create plugin interface test in `internal/plugin/plugin_test.go`
  - Test cases: interface compliance, metadata retrieval, schema validation
  - Must fail: No interface defined yet
  - File: `internal/plugin/plugin_test.go`

- [X] **T015** [P] Create plugin registry test in `internal/plugin/registry_test.go`
  - Test cases: register plugins, lookup by type, duplicate registration prevention
  - Must fail: No registry implementation yet
  - File: `internal/plugin/registry_test.go`

- [X] **T016** [P] Create logger wrapper test in `internal/logger/logger_test.go`
  - Test cases: structured logging, level filtering, context fields
  - Must fail: No logger wrapper yet
  - File: `internal/logger/logger_test.go`

- [X] **T017** [P] Create error types test in `pkg/errors/errors_test.go`
  - Test cases: wrapped errors with context, error unwrapping, error chains
  - Must fail: No error types defined yet
  - File: `pkg/errors/errors_test.go`

---

## Phase 3.3: Core Foundation - Implementation (ONLY after tests fail)

- [X] **T018** Implement Config struct and type-specific step structs in `internal/config/types.go`
  - Structs: Config, Settings, Step, PackageStep, RepoStep, SymlinkStep, CopyStep, CommandStep, Validation
  - Use struct tags: `yaml:` for parsing, `validate:` for validation rules
  - Reference: data-model.md sections 1-9
  - File: `internal/config/types.go`

- [X] **T019** Implement YAML parser in `internal/config/parser.go`
  - Function: `ParseConfig(path string) (*Config, error)`
  - Use `gopkg.in/yaml.v3` for parsing
  - Return parse errors with line numbers
  - Tests: T008 should now pass
  - File: `internal/config/parser.go`

- [X] **T020** Implement config validator in `internal/config/validator.go`
  - Function: `ValidateConfig(cfg *Config) error`
  - Use `go-playground/validator/v10` for struct validation
  - Custom validators: unique IDs, valid dependencies, no cycles
  - Tests: T009 should now pass
  - File: `internal/config/validator.go`

- [X] **T021** Implement DAG data structures in `internal/engine/dag.go`
  - Structs: Node, Graph, Level
  - Methods: AddNode, AddEdge, DetectCycles, TopologicalSort, GroupByLevel
  - Tests: T011 should now pass
  - File: `internal/engine/dag.go`

- [X] **T022** Implement DAG builder in `internal/engine/dag_builder.go`
  - Function: `BuildDAG(steps []config.Step) (*Graph, error)`
  - Create nodes from steps, edges from depends_on
  - Detect cycles and return error
  - Tests: T011 should now pass
  - File: `internal/engine/dag_builder.go`

- [X] **T023** Implement dry-run planner in `internal/engine/planner.go`
  - Function: `GeneratePlan(graph *Graph) (*ExecutionPlan, error)`
  - Output: Level-by-level execution order with parallel step grouping
  - Tests: T012 should now pass
  - File: `internal/engine/planner.go`

- [X] **T024** Implement execution context in `internal/engine/context.go`
  - Struct: ExecutionContext with worker pool, timeout, cancel, logger
  - Reference: data-model.md section 13
  - File: `internal/engine/context.go`

- [X] **T025** Implement executor in `internal/engine/executor.go`
  - Function: `Execute(ctx *ExecutionContext, plan *ExecutionPlan) ([]StepResult, error)`
  - Use goroutines for parallel level execution with worker pool
  - Fail-fast on errors
  - Tests: T013 should now pass
  - File: `internal/engine/executor.go`

- [X] **T026** Implement plugin interface in `internal/plugin/interface.go`
  - Interface: Plugin with methods: Metadata(), Schema(), Check(step), Apply(step, ctx), DryRun(step)
  - Types: PluginMetadata, StepResult (success, skipped, failed)
  - Reference: data-model.md section 16
  - Tests: T014 should now pass
  - File: `internal/plugin/interface.go`

- [X] **T027** Implement plugin registry in `internal/plugin/registry.go`
  - Functions: RegisterPlugin(name string, p Plugin), GetPlugin(name string) (Plugin, error)
  - Global registry with init() registration pattern
  - Tests: T015 should now pass
  - File: `internal/plugin/registry.go`

- [X] **T028** Implement logger wrapper in `internal/logger/logger.go`
  - Wrapper around `zerolog` with Info/Debug/Warn/Error methods
  - Support context fields
  - Human-readable console writer for TUI
  - Tests: T016 should now pass
  - File: `internal/logger/logger.go`

- [X] **T029** Implement error types in `pkg/errors/errors.go`
  - Types: ParseError, ValidationError, ExecutionError, PluginError
  - Wrap errors with context using `fmt.Errorf` and `%w`
  - Tests: T017 should now pass
  - File: `pkg/errors/errors.go`

---

## Phase 3.4: Plugin System - Tests First

- [X] **T030** [P] Create package plugin test in `internal/plugins/package/package_test.go`
  - Test cases: idempotency check (dpkg-query), apt install command generation, check mode, apply mode, dry-run mode
  - Must fail: No plugin implementation yet
  - File: `internal/plugins/package/package_test.go`

- [X] **T031** [P] Create repo plugin test in `internal/plugins/repo/repo_test.go`
  - Test cases: clone with go-git, idempotency check (dir exists), branch checkout, shallow clone with depth
  - Must fail: No plugin implementation yet
  - File: `internal/plugins/repo/repo_test.go`

- [X] **T032** [P] Create symlink plugin test in `internal/plugins/symlink/symlink_test.go`
  - Test cases: idempotency check (readlink), create symlink, force flag, target validation
  - Must fail: No plugin implementation yet
  - File: `internal/plugins/symlink/symlink_test.go`

- [X] **T033** [P] Create copy plugin test in `internal/plugins/copy/copy_test.go`
  - Test cases: idempotency check (file hash), copy file, recursive copy, preserve permissions, overwrite flag
  - Must fail: No plugin implementation yet
  - File: `internal/plugins/copy/copy_test.go`

- [X] **T034** [P] Create command plugin test in `internal/plugins/command/command_test.go`
  - Test cases: shell detection, command execution, check command, env var passing, working directory
  - Must fail: No plugin implementation yet
  - File: `internal/plugins/command/command_test.go`

---

## Phase 3.5: Plugin System - Implementation

- [X] **T035** Implement package plugin in `internal/plugins/package/package.go`
  - Implement Plugin interface: Metadata(), Schema(), Check(), Apply(), DryRun()
  - Check: Use `dpkg-query -W` for idempotency
  - Apply: Run `apt-get install -y` with packages list
  - Register plugin in init()
  - Tests: T030 should now pass
  - File: `internal/plugins/package/package.go`

- [X] **T036** Implement repo plugin in `internal/plugins/repo/repo.go`
  - Implement Plugin interface: Metadata(), Schema(), Check(), Apply(), DryRun()
  - Check: Verify destination directory exists
  - Apply: Use go-git PlainClone with branch and depth options
  - Register plugin in init()
  - Tests: T031 should now pass
  - File: `internal/plugins/repo/repo.go`

- [X] **T037** Implement symlink plugin in `internal/plugins/symlink/symlink.go`
  - Implement Plugin interface: Metadata(), Schema(), Check(), Apply(), DryRun()
  - Check: Use os.Readlink to verify symlink exists
  - Apply: Use os.Symlink with force flag handling
  - Register plugin in init()
  - Tests: T032 should now pass
  - File: `internal/plugins/symlink/symlink.go`

- [X] **T038** Implement copy plugin in `internal/plugins/copy/copy.go`
  - Implement Plugin interface: Metadata(), Schema(), Check(), Apply(), DryRun()
  - Check: Compare file hashes for idempotency
  - Apply: Copy with filepath.Walk for recursive, preserve permissions
  - Register plugin in init()
  - Tests: T033 should now pass
  - File: `internal/plugins/copy/copy.go`

- [X] **T039** Implement command plugin in `internal/plugins/command/command.go`
  - Implement Plugin interface: Metadata(), Schema(), Check(), Apply(), DryRun()
  - Check: Run check_command if provided, else return needs execution
  - Apply: Use exec.Command with shell detection (/bin/bash, /bin/sh)
  - Register plugin in init()
  - Tests: T034 should now pass
  - File: `internal/plugins/command/command.go`

---

## Phase 3.6: Validation System - Tests First

- [X] **T040** [P] Create validation runner test in `internal/validation/validator_test.go`
  - Test cases: run all validations, collect results, pass/fail reporting
  - Must fail: No validator implementation yet
  - File: `internal/validation/validator_test.go`

- [X] **T041** [P] Create validation checks test in `internal/validation/checks_test.go`
  - Test cases: command_exists (PATH search), file_exists (stat), path_contains (regex match)
  - Must fail: No checks implementation yet
  - File: `internal/validation/checks_test.go`

---

## Phase 3.7: Validation System - Implementation

- [X] **T042** Implement validation types in `internal/validation/types.go`
  - Structs: Validation, ValidationResult (from data-model.md sections 10-12)
  - Types: command_exists, file_exists, path_contains
  - File: `internal/validation/types.go`

- [X] **T043** Implement validation runner in `internal/validation/validator.go`
  - Function: `RunValidations(validations []config.Validation) ([]ValidationResult, error)`
  - Execute each validation, collect results
  - Tests: T040 should now pass
  - File: `internal/validation/validator.go`

- [X] **T044** Implement validation checks in `internal/validation/checks.go`
  - Functions: CheckCommandExists(), CheckFileExists(), CheckPathContains()
  - Use exec.LookPath for command_exists, os.Stat for file_exists, regexp for path_contains
  - Tests: T041 should now pass
  - File: `internal/validation/checks.go`

---

## Phase 3.8: TUI (Bubbletea) - Tests First

- [X] **T045** [P] Create TUI model test in `internal/tui/model_test.go`
  - Test cases: initial state, state transitions, step status updates
  - Must fail: No model implementation yet
  - File: `internal/tui/model_test.go`

- [X] **T046** [P] Create TUI update test in `internal/tui/update_test.go`
  - Test cases: message handling, step progress updates, completion detection
  - Must fail: No update function yet
  - File: `internal/tui/update_test.go`

- [X] **T047** [P] Create TUI view test in `internal/tui/view_test.go`
  - Test cases: rendering, lipgloss styling, component integration
  - Must fail: No view function yet
  - File: `internal/tui/view_test.go`

---

## Phase 3.9: TUI (Bubbletea) - Implementation

- [X] **T048** Implement TUI model in `internal/tui/model.go`
  - Struct: Model with config, plan, execution state, results
  - Init() method for Bubbletea
  - Reference: data-model.md section 15
  - Tests: T045 should now pass
  - File: `internal/tui/model.go`

- [X] **T049** Implement TUI update function in `internal/tui/update.go`
  - Function: Update(msg tea.Msg) (tea.Model, tea.Cmd)
  - Handle messages: StepStartMsg, StepCompleteMsg, ValidationMsg
  - State transitions
  - Tests: T046 should now pass
  - File: `internal/tui/update.go`

- [X] **T050** Implement TUI view function in `internal/tui/view.go`
  - Function: View() string
  - Render with lipgloss: header, step list, progress bars, summary
  - Tests: T047 should now pass
  - File: `internal/tui/view.go`

- [X] **T051** [P] Implement progress bar component in `internal/tui/components/progress.go`
  - Use `bubbles` progress bar
  - Show current/total steps with percentage
  - File: `internal/tui/components/progress.go`

- [X] **T052** [P] Implement step list component in `internal/tui/components/steplist.go`
  - Render step list with status icons: ⏳ running, ✓ success, ✗ failed, ⊘ skipped
  - Show step names and timing
  - File: `internal/tui/components/steplist.go`

- [X] **T053** [P] Implement summary component in `internal/tui/components/summary.go`
  - Render execution summary: total steps, succeeded, failed, skipped
  - Show validation results
  - Display total execution time
  - File: `internal/tui/components/summary.go`

- [X] **T054** Implement lipgloss styles in `internal/tui/styles.go`
  - Define styles: header, success (green), error (red), warning (yellow), info (blue)
  - Border styles, padding, margins
  - File: `internal/tui/styles.go`

---

## Phase 3.10: CLI Commands (Cobra) - Tests First

- [X] **T055** [P] Create CLI apply command test in `cmd/streamy/apply_test.go`
  - Test cases: flag parsing (--dry-run, --verbose), config file validation
  - Must fail: No apply command yet
  - File: `cmd/streamy/apply_test.go`

- [X] **T056** [P] Create CLI version command test in `cmd/streamy/version_test.go`
  - Test cases: version output format, build info
  - Must fail: No version command yet
  - File: `cmd/streamy/version_test.go`

---

## Phase 3.11: CLI Commands (Cobra) - Implementation

- [X] **T057** Implement main entry point in `cmd/streamy/main.go`
  - Cobra root command setup
  - Global flags: --verbose, --dry-run
  - Command registration
  - File: `cmd/streamy/main.go`

- [X] **T058** Implement apply command in `cmd/streamy/apply.go`
  - Function: runApply(cmd *cobra.Command, args []string)
  - Flow: Parse config → Validate → Build DAG → Generate plan → Execute/DryRun → Run validations
  - Launch Bubbletea TUI for progress
  - Tests: T055 should now pass
  - File: `cmd/streamy/apply.go`

- [X] **T059** Implement version command in `cmd/streamy/version.go`
  - Function: runVersion(cmd *cobra.Command, args []string)
  - Output: Version, Go version, build date (ldflags)
  - Tests: T056 should now pass
  - File: `cmd/streamy/version.go`

- [X] **T060** Implement flag validation in `cmd/streamy/flags.go`
  - Validate config file exists
  - Validate flag combinations
  - File: `cmd/streamy/flags.go`

---

## Phase 3.12: Integration Testing & Test Fixtures

- [ ] **T061** Create test fixture: simple config in `testdata/configs/simple.yaml`
  - 2-3 steps, no dependencies
  - Based on quickstart.md Example 1
  - File: `testdata/configs/simple.yaml`

- [ ] **T062** Create test fixture: complex config in `testdata/configs/complex.yaml`
  - All 5 step types, multi-level dependencies
  - Based on quickstart.md Example 6
  - File: `testdata/configs/complex.yaml`

- [ ] **T063** Create test fixture: invalid YAML in `testdata/configs/invalid.yaml`
  - Malformed YAML for parser error testing
  - File: `testdata/configs/invalid.yaml`

- [ ] **T064** Create test fixture: circular dependency in `testdata/configs/cycle.yaml`
  - Steps with circular depends_on for cycle detection testing
  - File: `testdata/configs/cycle.yaml`

- [ ] **T065** Create test fixture: missing reference in `testdata/configs/missing_ref.yaml`
  - depends_on references non-existent step ID
  - File: `testdata/configs/missing_ref.yaml`

- [X] **T066** [P] Integration test: simple config execution in `tests/integration_test.go`
  - Test: Load simple.yaml, execute, verify all steps succeed
  - Based on quickstart.md Example 1
  - File: `tests/integration_test.go` (add test function)

- [X] **T067** [P] Integration test: complex config with DAG in `tests/integration_test.go`
  - Test: Load complex.yaml, verify correct execution order, parallel execution
  - Based on quickstart.md Example 6
  - File: `tests/integration_test.go` (add test function)

- [X] **T068** [P] Integration test: dry-run mode in `tests/integration_test.go`
  - Test: Load simple.yaml with --dry-run, verify no actual execution
  - Based on quickstart.md Example 3
  - File: `tests/integration_test.go` (add test function)

- [X] **T069** [P] Integration test: idempotency in `tests/integration_test.go`
  - Test: Execute same config twice, verify second run skips already-completed steps
  - Based on quickstart.md Example 8
  - File: `tests/integration_test.go` (add test function)

- [X] **T070** [P] Integration test: error handling in `tests/integration_test.go`
  - Test: Config with failing step, verify fail-fast behavior
  - Based on quickstart.md Example 7
  - File: `tests/integration_test.go` (add test function)

- [X] **T071** [P] Integration test: validation failures in `tests/integration_test.go`
  - Test: Config with post-execution validations, verify validation failures reported
  - File: `tests/integration_test.go` (add test function)

- [X] **T072** [P] Error scenario test: malformed YAML in `tests/integration_test.go`
  - Test: Load invalid.yaml, verify parse error with line numbers
  - File: `tests/integration_test.go` (add test function)

- [X] **T073** [P] Error scenario test: circular dependencies in `tests/integration_test.go`
  - Test: Load cycle.yaml, verify cycle detection error
  - File: `tests/integration_test.go` (add test function)

- [X] **T074** [P] Error scenario test: missing references in `tests/integration_test.go`
  - Test: Load missing_ref.yaml, verify validation error
  - File: `tests/integration_test.go` (add test function)

---

## Phase 3.13: Documentation

- [X] **T075** [P] Create architecture documentation in `docs/architecture.md`
  - Content: System design, component diagram, data flow, DAG execution
  - Reference: plan.md project structure
  - File: `docs/architecture.md`

- [X] **T076** [P] Create plugin development guide in `docs/plugins.md`
  - Content: Plugin interface, implementation example, registration, testing
  - Reference: data-model.md section 16
  - File: `docs/plugins.md`

- [X] **T077** [P] Create YAML schema reference in `docs/schema.md`
  - Content: Complete schema documentation, validation rules, examples
  - Reference: data-model.md sections 1-12
  - File: `docs/schema.md`

- [X] **T078** Update README.md with comprehensive content
  - Add: Features, installation instructions, quick start, configuration examples, links to docs
  - File: `README.md`

---

## Phase 3.14: Build & Release

- [X] **T079** Create multi-platform build script in `scripts/build.sh`
  - Build for: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
  - Use ldflags for version injection: `-ldflags "-X main.version=$(git describe --tags)"`
  - Output: `dist/streamy-{os}-{arch}`
  - File: `scripts/build.sh`

- [X] **T080** Create install script in `scripts/install.sh`
  - Detect platform, download appropriate binary, install to /usr/local/bin or ~/bin
  - Curl | sh pattern
  - File: `scripts/install.sh`

- [X] **T081** Create CI workflow in `.github/workflows/ci.yml`
  - On: pull request, push to main
  - Jobs: test (go test ./...), lint (golangci-lint), build (all platforms)
  - File: `.github/workflows/ci.yml`

- [X] **T082** Create release workflow in `.github/workflows/release.yml`
  - On: push tag (v*)
  - Use Goreleaser for multi-platform binaries, GitHub release, checksums
  - Create `.goreleaser.yml` config
  - Files: `.github/workflows/release.yml`, `.goreleaser.yml`

- [X] **T083** Configure test coverage reporting and enforcement
  - Add coverage profile generation to CI workflow (T081): `go test ./... -coverprofile=coverage.out`
  - Add coverage threshold enforcement: fail CI if coverage < 80%
  - Use tool: `go tool cover -func=coverage.out` or `gocov`
  - Display coverage badge/report in GitHub Actions
  - File: `.github/workflows/ci.yml` (update existing)

---

## Dependencies

### Critical Path (Blocking Sequences)
1. **Setup → Tests → Implementation**: T001-T007 → T008-T017 → T018-T029
2. **Config → DAG → Executor**: T018-T020 → T021-T023 → T024-T025
3. **Plugin Interface → Implementations**: T026-T027 → T035-T039
4. **Core Engine → TUI → CLI**: T018-T029 → T048-T054 → T057-T060
5. **All Core → Integration Tests**: T001-T060 → T061-T074
6. **All Features → Documentation → Build → Coverage**: T001-T074 → T075-T078 → T079-T082 → T083

### Phase Dependencies
- Phase 3.1 (Setup) blocks all other phases
- Phase 3.2 (Tests) blocks Phase 3.3 (Implementation)
- Phase 3.3 (Core) blocks Phase 3.4-3.11 (all feature implementations)
- Phase 3.4 (Plugin Tests) blocks Phase 3.5 (Plugin Implementations)
- Phase 3.6 (Validation Tests) blocks Phase 3.7 (Validation Implementation)
- Phase 3.8 (TUI Tests) blocks Phase 3.9 (TUI Implementation)
- Phase 3.10 (CLI Tests) blocks Phase 3.11 (CLI Implementation)
- Phase 3.12 (Integration) requires all implementation phases complete
- Phase 3.13 (Docs) can run after core features (T018-T060)
- Phase 3.14 (Build) requires all features and tests complete
- T083 (Coverage) requires T081 (CI workflow) and all tests complete

### Specific Task Dependencies
- T019 requires T018 (parser needs config types)
- T020 requires T018 (validator needs config types)
- T022 requires T018, T021 (DAG builder needs config types and DAG structs)
- T025 requires T024, T026 (executor needs context and plugin interface)
- T035-T039 require T026 (plugins need interface)
- T043-T044 require T042 (validator needs validation types)
- T049-T050 require T048 (update/view need model)
- T049-T050 require T048 (update/view need model)
- T058 requires T019, T020, T022, T025, T048 (apply orchestrates all core components)
- T083 requires T081 (coverage reporting depends on CI workflow)

---

## Parallel Execution Examples

### Example 1: Setup Phase (T003-T005)
All dependency installations can run in parallel:
```bash
# Launch T003, T004, T005 together
Task T003: Install core dependencies (cobra, viper, yaml, validator)
Task T004: Install TUI dependencies (bubbletea, lipgloss, bubbles, zerolog)
Task T005: Install git and testing dependencies (go-git, testify)
```

### Example 2: Test Creation (T008-T017)
All test files in different directories can be created in parallel:
```bash
# Launch T008-T017 together (10 tasks)
Task T008: Create config parser test
Task T009: Create config validator test
Task T010: Create Step struct test
Task T011: Create DAG builder test
Task T012: Create planner test
Task T013: Create executor test
Task T014: Create plugin interface test
Task T015: Create plugin registry test
Task T016: Create logger wrapper test
Task T017: Create error types test
```

### Example 3: Plugin Tests (T030-T034)
All plugin test files can be created in parallel:
```bash
# Launch T030-T034 together (5 tasks)
Task T030: Create package plugin test
Task T031: Create repo plugin test
Task T032: Create symlink plugin test
Task T033: Create copy plugin test
Task T034: Create command plugin test
```

### Example 4: Plugin Implementations (T035-T039)
Cannot fully parallelize due to shared registry, but can group:
```bash
# Group 1: Independent plugins (T035, T036, T037 in parallel)
Task T035: Implement package plugin
Task T036: Implement repo plugin
Task T037: Implement symlink plugin

# Group 2: After Group 1 (T038, T039 in parallel)
Task T038: Implement copy plugin
Task T039: Implement command plugin
```

### Example 5: TUI Components (T051-T053)
All component files can be created in parallel:
```bash
# Launch T051-T053 together (3 tasks)
Task T051: Implement progress bar component
Task T052: Implement step list component
Task T053: Implement summary component
```

### Example 6: Test Fixtures (T061-T065)
All fixture YAML files can be created in parallel:
```bash
# Launch T061-T065 together (5 tasks)
Task T061: Create simple.yaml fixture
Task T062: Create complex.yaml fixture
Task T063: Create invalid.yaml fixture
Task T064: Create cycle.yaml fixture
Task T065: Create missing_ref.yaml fixture
```

### Example 7: Integration Tests (T066-T074)
All integration test functions can be added in parallel (if using different test files or careful merge):
```bash
# Launch T066-T074 together (9 tasks)
Task T066: Integration test simple config
Task T067: Integration test complex config
Task T068: Integration test dry-run
Task T069: Integration test idempotency
Task T070: Integration test error handling
Task T071: Integration test validations
Task T072: Error test malformed YAML
Task T073: Error test circular dependencies
Task T074: Error test missing references
```

### Example 8: Documentation (T075-T077)
All doc files can be created in parallel:
```bash
# Launch T075-T077 together (3 tasks)
Task T075: Create architecture.md
Task T076: Create plugins.md
Task T077: Create schema.md
```

---

## Validation Checklist

Before marking this tasks.md as complete, verify:

- [x] All entities from data-model.md have corresponding implementation tasks
  - Config, Settings, Step (all 5 types), Validation (all 3 types), DAG, ExecutionContext, StepResult, Plugin interface
- [x] All quickstart examples have corresponding integration tests
  - Example 1 → T066, Example 2 → T067 (implicit), Example 3 → T068, Example 6 → T067, Example 7 → T070, Example 8 → T069
- [x] All tests come before implementation (TDD)
  - Phase 3.2 (T008-T017) before Phase 3.3 (T018-T029)
  - Phase 3.4 (T030-T034) before Phase 3.5 (T035-T039)
  - Phase 3.6 (T040-T041) before Phase 3.7 (T042-T044)
  - Phase 3.8 (T045-T047) before Phase 3.9 (T048-T054)
  - Phase 3.10 (T055-T056) before Phase 3.11 (T057-T060)
- [x] Parallel tasks [P] are truly independent (different files, no shared state)
- [x] Each task specifies exact file path
- [x] No task modifies same file as another [P] task (except test files with separate functions)
- [x] All constitutional principles addressed in tasks
  - Onboarding First: T079-T082 (single binary, install script)
  - Schema Clarity: T018-T020, T077 (validation, docs)
  - Plugin-Centric: T026-T039 (interface, implementations)
  - Safety by Default: T023, T058 (dry-run, fail-fast)
  - Performance: T025 (parallel execution), T083 (80%+ test coverage enforcement)
  - Extensibility: T026-T027 (plugin registry)
  - Consistency: T006 (linting), T083 (coverage), following Go conventions throughout

---

## Notes

- **TDD Discipline**: Never implement before tests are written and failing
- **Parallel Markers**: [P] indicates different files with no dependencies
- **File Paths**: All file paths are relative to repository root
- **Test Coverage**: T083 enforces 80%+ coverage threshold per constitution principle V
- **YAML Schema**: Config fields are at root level (no "streamy:" wrapper key required)
- **Commit Strategy**: Commit after each task or logical group
- **Testing**: Run `go test ./...` after each implementation phase
- **Constitutional Compliance**: Each task maintains alignment with Streamy constitution principles
- **MVP Scope**: 83 tasks covering all requirements for single-binary, cross-platform CLI with 5 step types, 3 validation types, TUI, DAG execution, complete testing, and 80%+ coverage enforcement

---

**Total Tasks**: 83  
**Estimated Parallel Groups**: 12 groups of 3-10 parallel tasks  
**Critical Path Length**: ~40 sequential tasks (with parallelization)  
**Status**: Ready for implementation

---

*Generated from plan.md, research.md, data-model.md, quickstart.md on 2025-10-03*
