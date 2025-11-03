# review-findings

## Description
Run CodeRabbit analysis and process the output to update `.agents/CODERABBIT.md` with potential issues requiring verification.

## When to Use
When you want to analyze the codebase for potential issues and create actionable tasks from the results.

## Workflow Steps

### 1. Run CodeRabbit Analysis
```bash
coderabbit --prompt-only
```
- This starts the CodeRabbit review in plain text mode
- The command will run in the background and take several minutes to complete

### 2. Monitor Progress
- Use the `BashOutput` tool to check command progress
- Typical progression: "Connecting to review service" → "Setting up" → "Analyzing" → "Reviewing"
- Wait for completion status before proceeding

### 3. Extract Issues from Output
- Copy the potential issues from the CodeRabbit output
- Each issue will have: File path, Line numbers, Type (usually "potential_issue"), and Prompt for AI Agent

### 4. Update Findings Document
- Open `.agents/CODERABBIT.md` for cross-project compatibility
- Add new issues using the established format:
  ```markdown
  ## [ ] Brief descriptive title
  **File:** `path/to/file.md`
  **Lines:** XXX-XXX
  **Reported Issue:** Clear description of what was reported
  **Suggested Fix:** What CodeRabbit suggests doing
  **Verification Needed:** Specific guidance on what to confirm

  ---
  ```

### 5. Format Guidelines
- Use "Reported Issue" instead of "Issue" or "Suggested Issue"
- Use "Suggested Fix" for the proposed solution
- Always include "Verification Needed" with specific confirmation steps
- Use code backticks for file paths, type names, and function references
- Separate issues with horizontal rules (`---`)
- Mark completed items with `[x]` instead of `[ ]`

### 6. Cross-Project Compatibility
- Use relative path `.agents/CODERABBIT.md` instead of absolute paths
- This allows the same command to work across different projects
- Ensure the `.agents/` directory exists before running

### 7. Key Principles
- These are **reported issues requiring verification**, not confirmed problems
- Each issue needs investigation before taking action
- Focus on making the document LLM-friendly and scannable
- Maintain consistent terminology throughout the document

## Example Entry
```markdown
## [ ] Type mismatch in domain struct
**File:** `internal/domain/pipeline/types.go`
**Lines:** 45-48
**Reported Issue:** Field name mismatch between struct definition and usage

**Suggested Fix:** Update field name to match usage across the codebase

**Verification Needed:** Confirm the field name mismatch exists and the suggested name is appropriate
```

## Notes
- The `.agents/CODERABBIT.md` file should already exist with established format
- Follow the existing Process section guidelines for handling issues
- Remove incomplete or cut-off issue entries rather than adding partial information
- The document serves as both a task tracker and verification guide
- Using `.agents/CODERABBIT.md` makes this command reusable across different projects