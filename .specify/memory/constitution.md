<!--
Sync Impact Report:
Version: 1.0.0 (Initial ratification)
Modified Principles: None (initial creation)
Added Sections:
  - All 7 core principles established
  - Technical Design Constraints section added
  - Schema Evolution Rules section added
  - Governance section established
Removed Sections: None
Templates Requiring Updates:
  ✅ plan-template.md - Constitution Check section needs principle alignment
  ✅ spec-template.md - Requirements validation aligned with safety/clarity principles
  ✅ tasks-template.md - Task categorization reflects new principles
Follow-up TODOs: None
-->

# Streamy Constitution

## Core Principles

### I. Onboarding First (NON-NEGOTIABLE)

Every decision MUST reduce time-to-usable environment. The tool MUST always work on a 
fresh machine with zero dependencies besides the compiled binary.

**Rules:**
- The compiled Go binary is the ONLY prerequisite for execution
- No pre-installed languages, package managers, or system dependencies required
- First-run experience MUST be: download binary → run command → working environment
- Installation complexity is a design failure, not a deployment problem
- Any feature requiring external dependencies MUST be implemented as an optional plugin

**Rationale:** Developer onboarding friction kills adoption. A tool that "just works" 
builds trust and allows teams to focus on environment configuration, not tool setup.

### II. Schema Clarity & Fun

YAML configuration MUST be minimal, declarative, and enjoyable to write. Schema design 
prioritizes human readability and machine validation equally.

**Rules:**
- Flat flags for common options, nested structures only for rare/complex scenarios
- `id` fields for machine references, `name` fields for human-readable labels
- All plugins MUST expose JSON schemas for config validation
- Schemas drive interactive UX: autocomplete, inline docs, validation feedback
- Configuration errors MUST provide actionable fix suggestions with file/line context
- Schema evolution follows strict versioning: breaking changes require migration paths

**Rationale:** Configuration is the user interface. Clear, predictable schemas reduce 
cognitive load and enable tooling (LSPs, validators, generators) that improve the 
developer experience.

### III. Plugin-Centric Architecture

The Go core is lightweight (DAG execution, logging, validation). All environment actions
(repos, packages, symlinks, commands, etc.) are plugins with a stable API contract.

**Rules:**
- Core responsibilities: DAG resolution, plugin lifecycle, structured logging, error handling
- Plugins implement well-defined interfaces with semantic versioning
- Core MUST NOT contain domain-specific logic (package managers, git operations, etc.)
- Plugins MAY be bundled in core binary for MVP, but architecture supports external distribution
- Plugin API contracts MUST be backward compatible within major versions
- Breaking plugin API changes require core major version bump and migration tooling

**Rationale:** A plugin architecture enables independent evolution of connectors, 
community contributions, and domain-specific optimizations without destabilizing the 
core execution engine.

### IV. Safety by Default (NON-NEGOTIABLE)

Operations MUST be idempotent and reversible where possible. Failures fail fast unless 
explicitly configured otherwise. Parallelism MUST NOT break environments.

**Rules:**
- All state-changing operations MUST support dry-run mode
- Destructive actions (deletions, overwrites) REQUIRE explicit confirmation or flags
- Idempotency: running the same config twice produces identical results
- Backup strategy: reversible operations create restore points automatically
- Parallel execution defaults MUST be safe; dangerous concurrency requires opt-in
- Transactional semantics: partial failures leave environment in known, recoverable state
- Clear rollback procedures for every destructive operation

**Rationale:** Developers trust tools that respect their environments. Safety defaults 
prevent accidental destruction, enable confident automation, and reduce support burden.

### V. Performance & Reliability

DAG execution MUST be concurrent where safe, serialized where necessary. The core MUST 
be efficient, producing clear logs, predictable error handling, and fast dry-run planning.

**Rules:**
- Dependency analysis determines parallel vs. sequential execution automatically
- Dry-run mode completes in <1s for typical configs (50-100 tasks)
- Execution logs clearly show: task start/end, duration, dependencies, errors
- Error messages include: context, root cause, suggested fixes, and retry guidance
- Resource limits: plugins MUST declare memory/CPU bounds for scheduling
- Timeouts: long-running operations MUST have configurable limits with clear failure modes

**Rationale:** Fast feedback loops improve developer productivity. Predictable 
performance and clear diagnostics reduce debugging time and build confidence in automation.

### VI. Extensibility & Composability

The system MUST scale from minimal personal configs to complex team setups. Features 
like imports, groups, and cross-platform conditions MUST be built without breaking MVP.

**Rules:**
- Configuration imports: YAML files can reference other files for reuse
- Task groups: logical grouping for selective execution (e.g., "dev", "ci", "prod")
- Platform conditionals: tasks can specify OS/arch constraints declaratively
- Variable substitution: environment variables and config-defined values
- New features MUST NOT require breaking changes to existing configs
- Backward compatibility within major versions is mandatory
- Composability: small configs combine into larger ones without duplication

**Rationale:** Real-world usage evolves from simple to complex. A composable design 
allows gradual adoption of advanced features without forcing complexity on simple use cases.

### VII. Ecosystem Consistency

New connectors MUST follow consistent schema design principles. Plugin development 
MUST be documented, tested, and maintain API compatibility.

**Rules:**
- All plugins use consistent naming: `id`, `name`, `enabled`, `depends_on` fields
- Error handling: plugins return structured errors (code, message, context, remediation)
- Testing: plugins MUST include contract tests for interface compliance
- Documentation: plugins MUST provide JSON schema, examples, and troubleshooting guides
- Versioning: plugins declare compatibility with core API versions explicitly
- Breaking changes require: version bump, changelog entry, migration guide

**Rationale:** Consistency reduces learning curves. Users learn plugin patterns once 
and apply them everywhere, accelerating adoption and community contributions.

## Technical Design Constraints

### Core Technology Stack
- **Language:** Go 1.25+
- **Build:** Single statically-linked binary for all platforms (Linux, macOS, Windows)
- **Logging:** Structured logs (JSON option for machine parsing, human-readable default)
- **Configuration:** YAML with JSON schema validation
- **Plugin Interface:** Go interfaces compiled into binary (MVP), gRPC for external plugins (future)

### Architectural Boundaries
- **Core:** DAG engine, plugin registry, logger, config parser, CLI framework
- **Plugins:** All domain logic (git, apt, brew, npm, symlinks, shell commands, etc.)
- **State Management:** Declarative desired state, no persistent runtime state in core
- **Error Recovery:** Plugins declare rollback procedures; core orchestrates recovery

### Performance Targets
- Binary size: <20MB uncompressed (allows single-file distribution)
- Startup time: <100ms cold start (enables fast dry-run feedback)
- Memory usage: O(n) in number of tasks, <50MB for typical 100-task configs
- Concurrency: Auto-detect parallelism opportunities, respect plugin-declared limits

## Schema Evolution Rules

### Configuration Versioning
- Config files MAY declare schema version (e.g., `version: "1.0"`)
- Missing version assumes latest stable schema
- Core MUST support N-1 major version for graceful migration
- Breaking changes require migration tool and clear upgrade path

### Plugin Schema Changes
- **Additive changes** (new optional fields): MINOR version bump
- **Deprecations** (marked but still functional): MINOR version bump + warning logs
- **Removals or semantic changes**: MAJOR version bump + migration guide
- Deprecated fields MUST be supported for one full major version cycle

### Schema Documentation
- All fields MUST have inline descriptions in JSON schema
- Examples MUST cover common use cases and edge cases
- Breaking changes MUST be announced in release notes with before/after examples

## Governance

### Principle Precedence
These principles take precedence over convenience, feature requests, or implementation 
shortcuts. Any proposal violating a principle requires explicit justification and 
community consensus to amend this constitution.

### Feature Evaluation Criteria
New features and plugins MUST be evaluated against:
1. **Onboarding Impact:** Does it increase or reduce time-to-first-success?
2. **Schema Clarity:** Is the configuration intuitive and self-documenting?
3. **Safety Defaults:** Are destructive actions guarded? Is dry-run supported?
4. **Ecosystem Fit:** Does it follow established plugin patterns and naming conventions?

### Breaking Change Policy
- Breaking changes require MAJOR version bump (Streamy 1.x → 2.x)
- Migration tool MUST be provided for config upgrades
- Deprecation warnings MUST appear for at least one minor version before removal
- Changelog MUST document migration steps with examples

### Amendment Process
Constitutional amendments require:
1. Proposal documenting: rationale, impact analysis, affected principles
2. Community review period (minimum 14 days for public discussion)
3. Update to this document with version bump and sync impact report
4. Update to all dependent templates (plan, spec, tasks) for consistency
5. Migration guide for any workflow changes

### Compliance Review
- All feature specs MUST include Constitution Check section (ref: `plan-template.md`)
- Design reviews verify: safety defaults, schema clarity, plugin architecture fit
- Implementation reviews verify: error handling, logging, idempotency, dry-run support

### Continuous Improvement
- User feedback on onboarding friction MUST be prioritized
- Plugin API pain points drive architecture refinements
- Performance regressions are considered bugs, not acceptable trade-offs

**Version**: 1.0.0 | **Ratified**: 2025-10-03 | **Last Amended**: 2025-10-03