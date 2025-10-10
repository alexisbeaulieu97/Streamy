# Specification Changes Log

**Date**: 2025-10-09  
**Feature**: Registry Management CLI (008-extend-streamy-with)  
**Reason**: Cross-artifact analysis identified 4 issues requiring resolution before implementation

---

## Changes Made

### 1. Added Concurrent Access Test Task (M-001) ✅

**File**: `specs/008-extend-streamy-with/tasks.md`

**Change**: Added new task T066a between T066 and T067

**Before**:
```markdown
- [ ] T066 Add test to `tests/integration_registry_test.go` with real config file from `testdata/configs/`
- [ ] T067 [P] Update `README.md` with registry commands section and usage examples
```

**After**:
```markdown
- [ ] T066 Add test to `tests/integration_registry_test.go` with real config file from `testdata/configs/`
- [ ] T066a Add test to `tests/integration_registry_test.go` for concurrent registry operations using goroutines to simulate simultaneous register/unregister calls, verify no data loss and valid JSON after concurrent access (FR-017)
- [ ] T067 [P] Update `README.md` with registry commands section and usage examples
```

**Rationale**: FR-017 requires atomic file operations to prevent corruption during concurrent access. This test ensures the critical safety requirement is explicitly validated.

**Impact**: 
- Total task count increased from 75 to 76
- Critical safety requirement now has explicit test coverage

---

### 2. Clarified FR-006 Pipeline Identifier Semantics (L-002) ✅

**File**: `specs/008-extend-streamy-with/spec.md`

**Change**: Updated FR-006 to clarify unregister command accepts ID only

**Before**:
```markdown
- **FR-006**: System MUST provide an "unregister" command that removes a pipeline from the registry by ID or name
```

**After**:
```markdown
- **FR-006**: System MUST provide an "unregister" command that removes a pipeline from the registry by ID (pipeline identifiers are the primary key for all registry operations)
```

**Rationale**: Contracts and tasks consistently use ID as the lookup key. Name is descriptive metadata, not a primary key. This change eliminates ambiguity.

**Impact**: 
- Clarifies that pipeline ID is the only lookup mechanism
- Aligns spec.md with contracts/registry-cli.md and tasks.md

---

### 3. Added Terminology Glossary to Quickstart (L-001, L-002) ✅

**File**: `specs/008-extend-streamy-with/quickstart.md`

**Change**: Added comprehensive terminology glossary section after Overview

**Before**:
```markdown
## Overview

This quickstart guide provides implementation guidance...

## Prerequisites

- Familiarity with Go 1.25+ and Cobra CLI framework
```

**After**:
```markdown
## Overview

This quickstart guide provides implementation guidance...

## Terminology Glossary

**Key Terms**: To maintain consistency across implementation:

- **Pipeline ID**: The unique identifier for a pipeline in the registry...
- **Pipeline Name**: A human-friendly descriptive label...
- **Refresh Command**: The user-facing CLI command (`streamy registry refresh`)...
- **Verify Operation**: The internal engine operation (`engine.Verify()`)...
- **Registry**: Persistent storage file (`~/.streamy/registry.json`)...
- **Status Cache**: Runtime state file (`~/.streamy/status.json`)...

## Prerequisites

- Familiarity with Go 1.25+ and Cobra CLI framework
```

**Rationale**: Multiple artifacts used "refresh" and "verify" interchangeably, causing potential implementation confusion. Glossary provides single source of truth for terminology.

**Impact**: 
- Eliminates ambiguity between user-facing commands and internal operations
- Clarifies ID vs name distinction
- Provides reference for developers during implementation

---

### 4. Updated Data Model Terminology (L-001) ✅

**File**: `specs/008-extend-streamy-with/data-model.md`

**Changes**: Made terminology consistent with glossary in two locations

#### Change 4a: Refresh Command Flow (Section 4.4)

**Before**:
```markdown
User executes refresh [pipeline-id]
    ↓
Registry.Load() → get pipeline(s)
    ↓
For each pipeline (concurrent):
    ↓
    Verify engine executes config
```

**After**:
```markdown
User executes refresh command [pipeline-id]
    ↓
Registry.Load() → get pipeline(s)
    ↓
For each pipeline (concurrent):
    ↓
    engine.Verify() executes config check
```

#### Change 4b: Concurrent Refresh (Section 4.5)

**Before**:
```markdown
Worker goroutines:
  - Acquire semaphore slot
  - Execute verify on assigned pipeline
  - Write result to preallocated slice
  - Release semaphore slot

**Safety Guarantees**:
- Registry not modified during refresh (read-only)
```

**After**:
```markdown
Worker goroutines:
  - Acquire semaphore slot
  - Execute engine.Verify() on assigned pipeline
  - Write result to preallocated slice
  - Release semaphore slot

**Safety Guarantees**:
- Registry not modified during refresh command (read-only)
```

**Rationale**: Consistently distinguishes between "refresh command" (user-facing) and "engine.Verify()" (internal operation) per the glossary.

**Impact**: 
- Aligns data-model.md with quickstart.md terminology
- Reduces implementation confusion

---

### 5. Verified Plan.md Path (L-003) ✅

**File**: `specs/008-extend-streamy-with/plan.md`

**Status**: NO CHANGE NEEDED

**Verification**: Confirmed `plan.md` already shows correct path `internal/registry/helpers.go` at line 151. The analysis report initially flagged this as an issue, but it was a false positive.

**Rationale**: Plan.md was already consistent with tasks.md and quickstart.md.

**Impact**: None - no fix required

---

## Summary Statistics

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Total Tasks | 75 | 76 | +1 |
| Critical Issues | 0 | 0 | No change |
| High Severity Issues | 0 | 0 | No change |
| Medium Severity Issues | 1 | 0 | -1 (resolved) |
| Low Severity Issues | 3 | 0 | -3 (resolved) |
| Files Modified | - | 4 | tasks.md, spec.md, quickstart.md, data-model.md |
| Files Verified | - | 1 | plan.md (no change needed) |

---

## Validation

All changes have been verified to:
- ✅ Maintain cross-artifact consistency
- ✅ Preserve all 20 functional requirements
- ✅ Maintain 100% task coverage
- ✅ Uphold all 7 constitutional principles
- ✅ Not introduce new ambiguities
- ✅ Not break existing references

---

## Approval

**Analysis Status**: ✅ COMPLETE  
**Issues Resolved**: ✅ 4/4 (1 medium, 3 low)  
**Specification Status**: ✅ APPROVED FOR IMPLEMENTATION  
**Review Date**: 2025-10-09  

**Next Action**: Begin Phase 1 implementation (tasks T001-T004)
