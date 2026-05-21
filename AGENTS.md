# AGENTS.md — Agent Tool Selection & Editing Checklists

**For:** AI agents (Claude, Copilot, etc.) working on pkgr codebase.

## Tool Selection

### File Reading & Modification

| Task | Tool | Notes |
|------|------|-------|
| Read file (any size) | **Read** | Faster, structured. NOT `cat`. |
| Modify file | **Edit** (old_string/new_string) | Surgical precision. NOT `sed`. |
| Write new file | **Write** | Clean creation. NOT `echo >> file`. |
| List directory | **Bash**: `ls -la <dir>` | Quick inspection. |

### Search & Discovery

| Task | Tool | Notes |
|------|------|-------|
| Find text in codebase | **Bash**: `rg "pattern" .` | Ripgrep (fast, context). NOT `grep -r`. |
| Find file by name | **Bash**: `find . -name "*.go" -type f` | Located patterns. |
| Show function signature | **Read** + inspect | Never guess — read the file. |

### Build & Test

| Command | Use Case |
|---------|----------|
| `make build` | Compile pkgr binary; verify no errors. |
| `make test` | Run all unit tests; expect green. |
| `make lint` | Run golangci-lint; catch style issues. |
| `go test ./... -v` | Verbose test output with individual test names. |
| `go test ./... -run TestName` | Run single test by name. |

## Editing Checklist

Before committing changes:

- [ ] **Read first.** Use Read tool to inspect the file before editing.
- [ ] **Edit surgically.** Use Edit with precise old_string/new_string, not wholesale rewrites.
- [ ] **Test locally.** Run `make test` and `make lint` after changes.
- [ ] **Update docs.** If API/behavior changes, update CLAUDE.md or relevant design doc.
- [ ] **Single logical commit.** One feature per commit; use Conventional Commits format.
- [ ] **Verify layout.** Confirm new files follow pkgr directory structure (internal/manager/<adapter-id>/).

## Subagent Dispatch Guidance

When dispatching work to a subagent:

- **Clearly specify** the task (e.g., "Add npm adapter following CLAUDE.md layout").
- **Link to CLAUDE.md** so subagent knows project invariants.
- **Provide before/after examples** if non-obvious.
- **Set exit criteria** (e.g., "All tests pass, no lint errors, commit includes golden fixtures").

## Risky Actions Requiring Confirmation

Before executing:

1. **Deleting files or directories** → Confirm intent.
2. **Modifying internal/runner/** → Core execution layer; test extra carefully.
3. **Changing cmd/pkgr/main.go** → Affects all adapters; verify build still works.
4. **Bulk refactors** (e.g., rename `Manager` struct) → Confirm scope first.
5. **Updating version** → Confirm semver increment is correct.
6. **Force git operations** (`git reset --hard`, `git push --force`) → Never without explicit user approval.

For risky actions: stop, explain the change, wait for user confirmation.
