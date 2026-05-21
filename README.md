# pkgr

Cross-platform package manager TUI/CLI. Wraps brew, apt, dnf, snap, flatpak, nix, scoop, choco, winget, npm, pnpm, yarn, bun, pip, pipx, uv, conda, mamba, cargo, gem, go, asdf, mise, mas, rustup behind one uniform interface.

**Status:** under construction (Phase 1: foundation).

See `docs/superpowers/specs/2026-05-21-pkgr-design.md` for the v1 design.

## Build

```
make build
./pkgr version
```

## Development

### Requirements

- Go 1.22 or later

### Build, Test, Lint

```bash
make build      # Compile pkgr binary
make test       # Run all unit tests
make lint       # Run golangci-lint
```

### Documentation

- **Design & Architecture:** See `docs/superpowers/specs/2026-05-21-pkgr-design.md`
- **Implementation Plans:** See `docs/superpowers/plans/2026-05-21-p1-foundation.md` (and subsequent phase plans)
- **Project Invariants:** See `CLAUDE.md` (layout, conventions, recipes)
- **Agent Guidelines:** See `AGENTS.md` (tool selection, editing checklist, risky actions)
- **Common Workflows:** See `SKILLS.md` (slash commands, adapter recipes, anti-patterns)

## Release Secrets (maintainer only)

- `TAP_TOKEN` — fine-grained PAT for `ramtinhoss/homebrew-pkgr`
- `SCOOP_TOKEN` — fine-grained PAT for `ramtinhoss/scoop-pkgr`
- (winget PRs use a GitHub App via the winget-releaser action; configure separately)
