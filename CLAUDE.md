# CLAUDE.md — pkgr Project Invariants

**Project type:** Go CLI (cross-platform package manager TUI/CLI wrapper).  
**Framework:** Cobra CLI, structured for modular adapter design.

## Layout

```
pkgr/
├── cmd/pkgr/          # Entry point (main.go) — ONLY place to import cmd logic
├── internal/
│   ├── log/           # Logging (level, format)
│   ├── config/        # Configuration (TOML parsing, defaults)
│   ├── spec/          # Manager/adapter specs (golden fixtures)
│   ├── manager/       # Adapter implementations (one subdir per adapter ID)
│   │   ├── brew/      # Homebrew adapter
│   │   ├── apt/       # APT adapter
│   │   ├── ... (dnf, snap, flatpak, nix, scoop, choco, winget, npm, etc.)
│   ├── runner/        # Execution layer (Shell, sanitized output, streaming)
│   └── cache/         # Caching logic (optional, reserved for future)
└── docs/              # Design specs, plans, implementation guides
```

## Conventions

### Never Shell Out Without internal/runner

**RULE:** Do NOT invoke shell commands directly via `os/exec`, `syscall`, or similar. Always route through `internal/runner`:

```go
// ✅ CORRECT
runner.Run(ctx, spec.Cmd{Name: "brew", Args: []string{"list"}})

// ❌ WRONG
cmd := exec.Command("brew", "list")
cmd.Run()
```

Rationale: runner sanitizes output, handles cross-platform paths (Windows vs Unix), streams properly, and enforces exit-code semantics.

### Adapter Code Lives in internal/manager/<adapter-id>/

- `internal/manager/brew/parse.go` — parse logic
- `internal/manager/brew/cmd.go` — command builders
- `internal/manager/brew/integration_test.go` — integration tests

**No cross-adapter imports.** If two adapters share logic, extract to `internal/` package (e.g., `internal/semver/`), not `internal/manager/brew/` → `internal/manager/apt/`.

### cmd/pkgr as Only Entry Point

- cmd/pkgr/main.go is the ONLY place that constructs the root Cobra command tree.
- All commands (install, search, etc.) wired into cmd/ level.
- Do NOT import cmd/ logic into internal/ packages. Dependency direction: cmd/ → internal/ (one-way).

### File Naming

- `*_test.go` — unit tests (mocked, fixtures in `internal/spec/fixtures/`)
- `*_integration_test.go` — integration tests (real adapter calls, `// +build integration`)
- `parse.go` — parsing/formatting logic
- `cmd.go` — command builders

### Test Location Conventions

- Unit tests for `internal/manager/brew/parse.go` → `internal/manager/brew/parse_test.go`
- Integration tests → `internal/manager/brew/integration_test.go`
- Golden fixtures (expected outputs) → `internal/spec/fixtures/brew/`

## Recipes

### Adding a New Adapter

1. Create `internal/manager/<adapter-id>/` directory.
2. Implement `parse.go` with parsers (list, search, info, install, upgrade, remove, cleanup).
3. Implement `cmd.go` with builders (CommandSpec for each operation).
4. Add `*_test.go` with unit tests using fixtures from `internal/spec/fixtures/<adapter-id>/`.
5. Wire into cmd/pkgr/main.go Cobra tree.
6. Run `make test` and `make lint` to verify.
7. Commit with `feat(adapter): add <adapter-id> support`.

### Refreshing Parser Fixtures

After modifying parser behavior:

1. Run real adapter: `brew list --json` (or equiv.)
2. Capture output to `internal/spec/fixtures/brew/list_output.json`.
3. Update corresponding `*_test.go` assertion.
4. Run `make test` to verify golden fixture still matches parser output.

### Updating Version

1. Edit `internal/config/version.go`: update `Version = "0.1.0"` to new semver.
2. Run `make build` to verify.
3. Commit with `chore(release): bump version to 0.2.0`.

## Watch-outs

### Parser Output Format

- Parsers must normalize output (JSON, CSV, plaintext) into standardized `Manager` struct.
- Never assume shell output format is stable. Use fixtures and golden tests.
- Windows paths use `\`; normalize to `/` in cross-platform code (use `filepath` package).

### Commit Message Style

Follow Conventional Commits:

- `feat(adapter): add npm support`
- `fix(brew): handle missing cask gracefully`
- `docs: update README with new adapter list`
- `chore(deps): upgrade golangci-lint to 1.55.0`

### Golden Fixtures

- Keep fixtures in `internal/spec/fixtures/<adapter-id>/` as source of truth.
- Never hand-edit fixtures unless behavior intentionally changed.
- Always commit fixture updates alongside test updates.
