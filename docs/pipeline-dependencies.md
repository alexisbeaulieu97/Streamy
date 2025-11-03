# Pipeline Dependencies

Streamy pipelines can now describe their upstream requirements directly in YAML. The registry understands these declarations, warns about missing entries, and renders a dependency tree so teams can see when downstream pipelines are blocked.

---

## 1. Declare Dependencies in YAML

Add `id`, `version`, and `dependencies` to your pipeline definition. Each dependency references a concrete registry entry in canonical `<id>@<version>` form.

```yaml
# deploy-frontend.yaml
id: frontend
version: "1.0"
name: "Frontend Deployment"
dependencies:
  - backend@1.2
  - network-config@2.0
steps:
  - id: deploy
    type: command
    command: ./deploy.sh
```

- `id` — lowercase letters, numbers, hyphen (`[a-z0-9-]`)
- `version` — any non-empty string without whitespace or `@`
- `dependencies` — optional list of canonical registry IDs

Streamy rejects configs that omit `id`/`version`, contain uppercase IDs, or include malformed dependency entries.

---

## 2. Register Pipelines

Registering a pipeline pulls in the dependency list, checks for cycles, and warns if referenced pipelines are not yet present in the registry.

```bash
streamy registry add deploy-frontend.yaml
```

Example output:

```
⚠ Warning: unregistered dependencies: [backend@1.2]
  Pipeline will remain blocked until each dependency is added to the registry.
✓ Added pipeline 'frontend@1.0' (Frontend Deployment)
  Path: /repos/env/deploy-frontend.yaml
  Status: 🟠 BLOCKED
  Missing dependencies:
    - backend@1.2
```

When you later register the missing dependency, Streamy reconciles the registry immediately and reports any pipelines that flipped back to ready status:

```bash
streamy registry add backend.yaml
```

```
✓ Added pipeline 'backend@1.2' (Backend Service)
  Path: /repos/env/backend.yaml
  Status: 🟢 READY
✓ Unblocked pipelines: frontend@1.0
```

The registry and status cache stay in sync, so follow-up commands (`registry list --tree`, `streamy run --registry ...`, dashboards) reflect the new readiness without additional manual steps.

- Pipelines remain **Ready** (`🟢`) when every dependency already exists.
- They are marked **Blocked** (`🟠`) when one or more dependencies are unregistered. Streamy accepts the pipeline but stores the missing IDs so you can add them later.
- Self-dependencies and cycles are rejected with clear error messages.

---

## 3. Inspect the Dependency Tree

Use the new tree view to understand how pipelines relate to each other. The command works even when some dependencies have not been registered.

```bash
streamy registry list --tree
```

Sample output:

```
└─ 🟢 platform@1.0 (Ready)
   ├─ 🟢 network-config@2.0 (Ready)
   ├─ 🟢 backend@1.2 (Ready)
   └─ 🟠 frontend@1.0 (Blocked: missing dependencies: auth@1.0)
      └─ 🟠 auth@1.0 (Blocked: not registered)
```

- The tree is sorted alphabetically at each level for stable output.
- Missing pipelines appear as blocked leaves with the reason `not registered`.
- ASCII-only environments fall back to `[RD]` / `[BL]` markers.

Use `streamy registry list` (table view) or `streamy registry list --json` when you need the traditional output. `--tree` cannot be combined with `--json`.

---

## 4. Run Orchestrated Execution

Once every dependency is registered you can execute the entire graph with a single command:

```bash
# Preview the execution plan without making changes
streamy run --registry app-deploy@1.0 --dry-run

# Apply the full graph in dependency order
streamy run --registry app-deploy@1.0

# Resolve the latest registered revision of a pipeline
streamy run --registry app-deploy@latest
```

During a run Streamy executes each dependency level in parallel, skips blocked pipelines, and records a summary in the status cache. Non-interactive environments can pass `--non-interactive` to suppress the TUI and print a textual summary instead.

Sample non-interactive output:

```
Executed 3 pipelines (3 ready, 0 blocked, 0 failed) in 3.1s
🟢 database@1.0 - Pipeline completed successfully
🟢 backend@1.0 - Pipeline completed successfully
🟢 frontend@1.0 - Pipeline completed successfully
```

If an upstream pipeline fails, downstream nodes are marked blocked and the summary lists the blocking pipeline ID so you can investigate quickly.

### Force Execution (use sparingly)

`--force` allows the orchestrator to keep running downstream pipelines even when an upstream dependency fails. Streamy treats the flag as a last resort: it shows an interactive confirmation that lists the blocked pipelines that will execute anyway, and it refuses to continue in non-interactive mode unless you also supply `--yes`.

```bash
streamy run --registry app-deploy@1.0 --force

WARNING: Executing the following blocked pipelines due to --force:
 - frontend@1.0 (blocked by backend@1.0)
Proceed with forced execution? [y/N]: y
```

Forced pipelines are annotated in the summary so that you can distinguish them from normal successes:

```
Executed 3 pipelines (2 ready, 0 blocked, 1 failed) in 4.6s
🔴 backend@1.0 - Pipeline failed: exit status 1
🟢 frontend@1.0 - Pipeline completed successfully (forced despite backend@1.0 failure) [FORCED: backend@1.0]
🟢 smoke-tests@1.0 - Pipeline completed successfully (forced despite backend@1.0 failure) [FORCED: backend@1.0]
```

CI jobs should pair `--force` with `--non-interactive --yes` to acknowledge the override explicitly.

---

## 5. Keep the Registry Ready

1. Register missing upstream pipelines using `streamy registry add`.
2. Re-run `streamy registry list --tree` to verify every node reports `🟢 Ready`.
3. Use dedicated registries per environment or branch to avoid cross-team conflicts.
4. Allow the automatic reconciliation step to update statuses—no manual refresh commands are required after registering dependencies.

---

## 6. Troubleshooting Checklist

| Symptom | Resolution |
|---------|------------|
| `missing pipeline id` | Ensure `id:` is present and lowercase. |
| `invalid dependency identifier` | Use canonical `<id>@<version>` entries. |
| `circular dependency detected` | Remove the cycle before adding the pipeline. |
| Tree shows `Blocked: not registered` | Add the referenced pipeline or remove it from `dependencies`. |
| Tree is empty | No pipelines registered yet; add one with `streamy registry add`. |

Need deeper guidance? Visit `docs/schema.md` for field descriptions or run `streamy registry list --tree --help` to see usage notes.
