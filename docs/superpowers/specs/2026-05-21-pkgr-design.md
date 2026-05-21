# pkgr — Cross-Platform Package Manager TUI/CLI

**Status:** Approved design — ready for implementation plan
**Date:** 2026-05-21
**Owner:** ramtinhoss@gmail.com
**Working name:** `pkgr` (rename pre-1.0 if desired)

---

## 1. Problem & Goals

Developers juggle 5–15 package managers (brew, apt, dnf, pacman, snap, flatpak, mas, nix, scoop, choco, winget, npm, pnpm, yarn, bun, pip, pipx, uv, conda, mamba, cargo, rustup, gem, go install, asdf, mise). Each has its own syntax, search UX, and update story. There is no single way to:

- Discover what's installed across them all
- Search for a tool without knowing which manager carries it
- Update everything in one shot
- Audit outdated packages globally

**Goal:** one tool — `pkgr` — that wraps the native managers as adapters, exposes a uniform CLI for scripting, and ships a Bubbletea TUI for interactive use. macOS + Linux + Windows from day one.

**Non-goals (v1):**

- Reimplementing PM resolution / dependency solving
- Acting as a sandbox or replacing the underlying PMs
- Cross-PM dependency graph
- Mobile package managers (Cocoapods, etc.)

## 2. Success Criteria

- One static binary, < 20 MB, no runtime deps beyond the native PM binaries
- Cold `pkgr search ripgrep` returns merged results from all detected PMs in < 2 s on a warm machine
- `pkgr` (TUI) usable in < 1 s after launch (initial detection cached)
- Adapters: 26 listed in §5
- Test coverage: parsers ≥ 90 %, integration e2e for each Linux distro per nightly CI
- Distribution: brew, scoop, winget, .deb, .rpm + install one-liner

## 3. Architecture

Layered Go project. Each layer narrow, single-purpose, replaceable.

```
cmd/pkgr/                       # main, version stamping
internal/
├── cli/                        # cobra subcommands (search, install, ...)
├── tui/                        # bubbletea screens (home, search, detail, pm, log)
├── manager/                    # Manager interface + Package/Op/OS types
│   └── <pm-id>/                # one dir per adapter
├── registry/                   # PM detection + enable/disable
├── orchestrator/               # aggregate fan-out + merge + ranking
├── runner/                     # exec wrapper, sudo flow, dry-run
├── cache/                      # TTL JSON cache, flock concurrency
├── config/                     # TOML loader
├── spec/                       # "name@pm==ver" parser
├── format/                     # human + JSON renderers
└── log/                        # slog + rotation
```

**Dependency direction:** `cli`/`tui` → `orchestrator` → `registry` → `manager` impls → `runner`. `cache`, `config`, `format`, `log`, `spec` are cross-cutting utilities; no upward references.

**Tech stack:**

- Go 1.22+
- [cobra](https://github.com/spf13/cobra) — CLI
- [bubbletea](https://github.com/charmbracelet/bubbletea) + [bubbles](https://github.com/charmbracelet/bubbles) + [lipgloss](https://github.com/charmbracelet/lipgloss) — TUI
- [BurntSushi/toml](https://github.com/BurntSushi/toml) — config
- `log/slog` (stdlib) — logging
- `goreleaser` + `nfpm` — release
- `cosign` (keyless OIDC) — signing

No PM-specific Go libraries. All adapters shell out via `internal/runner`.

## 4. Adapter Interface

```go
package manager

type OS string
const (
    Darwin  OS = "darwin"
    Linux   OS = "linux"
    Windows OS = "windows"
)

type Op string
const (
    OpInstall   Op = "install"
    OpUninstall Op = "uninstall"
    OpUpdate    Op = "update"
    OpSearch    Op = "search"
    OpList      Op = "list"
    OpInfo      Op = "info"
    OpOutdated  Op = "outdated"
)

type Scope string
const (
    ScopeSystem       Scope = "system"
    ScopeUserGlobal   Scope = "user-global"
    ScopeProjectLocal Scope = "project-local"
)

type Package struct {
    Name        string
    Version     string
    Latest      string            // populated for outdated
    Manager     string
    Installed   bool
    Description string
    Homepage    string
    Extra       map[string]string
}

type Manager interface {
    ID() string
    DisplayName() string
    OSes() []OS
    Detect() bool
    NeedsSudo(op Op) bool
    Scope() Scope

    List(ctx context.Context) ([]Package, error)
    Outdated(ctx context.Context) ([]Package, error)
    Search(ctx context.Context, q string) ([]Package, error)
    Info(ctx context.Context, name string) (Package, error)
    Install(ctx context.Context, names ...string) error
    Uninstall(ctx context.Context, names ...string) error
    Update(ctx context.Context, names ...string) error // empty = all
}
```

Each adapter: ~150 LOC, one file. Parses native output (JSON flag preferred where available, else line/regex). Wraps errors using the typed `Error` (§9).

## 5. PM Matrix (v1)

| # | id | OS | sudo? | scope | native cmd hint |
|---|---|---|---|---|---|
| 1 | brew | darwin, linux | no | user-global | `brew search --formula --json=v2` |
| 2 | mas | darwin | no | system | `mas list / search` |
| 3 | apt | linux | yes | system | `apt-cache search`, `apt-get install` |
| 4 | dnf | linux | yes | system | `dnf search`, `dnf install` |
| 5 | pacman | linux | yes | system | `pacman -Ss`, `-S` |
| 6 | snap | linux | yes | system | `snap find`, `snap install` |
| 7 | flatpak | linux | no | user-global | `flatpak search`, `install --user` |
| 8 | nix | darwin, linux | no | user-global | `nix search nixpkgs`, `nix profile install` |
| 9 | scoop | windows | no | user-global | `scoop search`, `scoop install` |
| 10 | choco | windows | yes | system | `choco search`, `choco install -y` |
| 11 | winget | windows | no | system | `winget search`, `winget install` |
| 12 | npm | all | no | user-global | `npm search --json`, `npm i -g` |
| 13 | pnpm | all | no | user-global | `pnpm search`, `pnpm add -g` |
| 14 | yarn | all | no | user-global | `yarn global add` |
| 15 | bun | all | no | user-global | `bun add -g` |
| 16 | pip | all | no | user-global | `pip install --user` |
| 17 | pipx | all | no | user-global | `pipx install` |
| 18 | uv | all | no | user-global | `uv tool install` |
| 19 | conda | all | no | user-global | `conda search/install` |
| 20 | mamba | all | no | user-global | `mamba search/install` |
| 21 | cargo | all | no | user-global | `cargo search`, `cargo install` |
| 22 | rustup | all | no | user-global | toolchain ops only |
| 23 | gem | all | no | user-global | `gem search -r`, `gem install` |
| 24 | go | all | no | user-global | `go install pkg@latest`; search via pkg.go.dev API |
| 25 | asdf | all | no | user-global | plugin + version ops |
| 26 | mise | all | no | user-global | `mise install`, `mise ls-remote` |

Volta dropped (redundant with mise). Version managers (rustup, asdf, mise) expose toolchain semantics — adapter normalizes to the common interface but UX surfaces the distinction in detail view.

## 6. Discovery

- `internal/registry` calls `Detect()` on every adapter at startup
- Result cached in `~/.cache/pkgr/registry.json` (1 h TTL)
- Only **enabled + detected** PMs participate in aggregate ops
- `pkgr pm` and TUI PM Manager screen show live status; `pkgr pm enable|disable <id>` writes config

## 7. CLI Surface

**Invocation:** `pkgr [global flags] <command> [args] [flags]`

**Global flags:**

- `--pm <id>[,<id>...]` — restrict
- `--os auto|darwin|linux|windows` — override (testing)
- `--json`
- `--no-color`
- `--yes / -y` — auto-confirm
- `--dry-run`
- `--no-cache`
- `--verbose / -v`
- `--config <path>`

**Commands:**

| cmd | purpose |
|---|---|
| `pkgr` / `pkgr tui` | launch TUI |
| `pkgr search <query>` | aggregate search (`--limit`, `--installed-only`) |
| `pkgr install <spec>...` | install (`name`, `name@pm`, `name==1.2.3@pm`) |
| `pkgr remove <spec>...` | uninstall (aliases: `uninstall`, `rm`) |
| `pkgr update [spec]...` | update; no args = update all |
| `pkgr list` | installed (`--outdated`, `--pm`) |
| `pkgr info <spec>` | package details |
| `pkgr outdated` | shortcut for `list --outdated` |
| `pkgr pm` | list detected PMs |
| `pkgr pm enable\|disable <id>` | toggle PM |
| `pkgr doctor` | health check |
| `pkgr cache clear [<id>]` | wipe cache |
| `pkgr completion <shell>` | bash/zsh/fish/pwsh |
| `pkgr config [edit\|path\|show]` | config helpers |
| `pkgr version` | version info |

**Spec syntax:** `name`, `name@pm`, `name==ver@pm`. Parsed once in `internal/spec`.

**Ambiguity rule:** if `name` matches > 1 detected PM, prompt user; with `--yes`, pick by `[ranking] preferred` order.

**Exit codes:** 0 ok · 1 partial/full failure · 2 usage error · 3 not found · 4 user aborted · 5 sudo required & refused.

## 8. TUI Screens

Root `App` owns a screen stack; each screen is its own Bubbletea sub-model.

**Layout (every screen):**

```
┌─ pkgr ──────────── brew(213) npm(42) pip(18) … ─┐
│  <screen content>                               │
├─ status: searching brew, npm…   3/8 done ───────┤
│ [/] search [i] install [u] update [d] remove [?]│
└─────────────────────────────────────────────────┘
```

**Screens:**

1. **Home / Dashboard** — totals, outdated count, last-sync per PM, quick actions
2. **Search** (`/`) — debounced 250 ms input, incremental results table `pkg | ver | pm | installed? | summary`, sort, mini-DSL filter (`pm:brew installed:no`), multi-select
3. **Detail** — full metadata, install / update / remove / open homepage / copy cmd actions
4. **Installed** — same table, pre-filtered; group-by toggle (by PM / flat); bulk select
5. **Outdated** — current → latest column; `U` all, `u` selected
6. **PM Manager** — table `id | detected | enabled | version | scope | sudo | last_sync`; enable/disable, resync
7. **Confirm Modal** — shows resolved native command(s); labels sudo ops clearly
8. **Operation Log** (`L`) — tailing pane, collapsible per-op blocks

**Global keys:** `?` help · `q`/`ctrl-c` quit · `esc` pop · `:` command palette

**Concurrency:** long ops dispatched as `tea.Cmd` returning typed result msgs; aggregate search spawns one goroutine per PM, table accumulates `searchPartialMsg`s so fast PMs render immediately. Cancellation via `ctrl-x` cancels ctx.

**Style:** lipgloss theme in `internal/tui/theme.go`; light/dark auto-detect, override via `[ui] theme`.

## 9. Cache · Config · Sudo · Errors · Dry-run

### Cache

- `${XDG_CACHE_HOME:-~/.cache}/pkgr/`
- Layout: `<pm>/installed.json`, `<pm>/search/<sha1(q)>.json`, `<pm>/info/<name>.json`, `registry.json`
- TTLs: installed 5 min, outdated 30 min, search 1 h, info 24 h, registry 1 h
- Format `{fetched_at, ttl_seconds, data}`
- Per-file `flock` for cross-process safety
- Mutating op busts `installed` + `outdated` for that PM immediately

### Config

- `${XDG_CONFIG_HOME:-~/.config}/pkgr/config.toml`, auto-created with defaults
- Schema includes `[general]`, `[cache]`, `[ranking]`, `[managers.<id>]` blocks
- `[ranking] preferred` resolves cross-PM ambiguity
- `[managers.<id>] enabled`, `extra_args`, `sudo` override auto-detection
- No hot-reload (process is short-lived); TUI re-reads on `r` in PM Manager

### Sudo

- `NeedsSudo(op)` per adapter + config override
- Execution: spawn `sudo -p "pkgr needs sudo for %p (op: <op>): "` — interactive terminal prompt
- Never store password; never `sudo -S` with stdin
- CI: require `--yes` AND (`SUDO_ASKPASS` set OR passwordless sudo); else exit 5
- Every sudo invocation logged to operation log with timestamp + redacted cmd

### Error Model

```go
type Code string
const (
    CodeNotFound         Code = "not_found"
    CodeNotDetected      Code = "not_detected"
    CodeNeedsSudo        Code = "needs_sudo"
    CodeNetworkFailure   Code = "network_failure"
    CodeParseError       Code = "parse_error"
    CodeConflict         Code = "conflict"
    CodePermissionDenied Code = "permission_denied"
    CodeCancelled        Code = "cancelled"
    CodeUnknown          Code = "unknown"
)

type Error struct {
    Manager string
    Op      Op
    Code    Code
    Err     error
    Cmd     string
    Stderr  string
}
```

Aggregate ops collect `[]Error` — one PM failing does not kill others. `--json` emits errors array.

### Dry-run

- Runner prints `→ would exec: brew install ripgrep` and skips
- TUI shows ghost confirm modal
- Used in tests for non-mutating verification

### Logging

- `slog` JSON → `${XDG_STATE_HOME:-~/.local/state}/pkgr/pkgr.log`
- Custom rotator, 5 files × 5 MB; no extra dep
- `--verbose` mirrors to stderr

## 10. Testing

1. **Adapter unit tests** — golden fixtures under `internal/manager/<pm>/testdata/`; runner injects fake exec returning fixture bytes
2. **Spec parser** — table-driven
3. **Cache** — TTL, flock, mutation invalidation
4. **Orchestrator** — fan-out merge, ranking, partial failures, cancellation
5. **TUI** — `teatest` golden frames for key flows
6. **CLI integration** — cobra tree exercised with `--dry-run`
7. **E2E (opt-in)** — ephemeral containers per distro (ubuntu, fedora, arch); macOS + Windows on GHA runners

**CI:** golangci-lint, `go test ./...` on linux/macos/windows, e2e nightly, codecov.

## 11. Distribution

- `goreleaser` builds: linux/amd64+arm64, darwin/amd64+arm64, windows/amd64
- Archives signed with cosign (keyless OIDC), sha256 checksums, syft SBOM attached
- Channels: homebrew tap, scoop bucket, winget manifest, `.deb`/`.rpm` via `nfpm`
- Install one-liners: `brew install <tap>/pkgr`, `scoop install pkgr`, `winget install pkgr`, plus a hosted `install.sh` (final domain decided pre-1.0)
- Self-update check: 24 h TTL hint, opt-out via `[general] update_check = false`

## 12. Repo Layout (final)

```
pkgr/
├── cmd/pkgr/                main.go
├── internal/
│   ├── cli/
│   ├── tui/
│   ├── manager/
│   │   ├── brew/ apt/ dnf/ … (26 dirs)
│   ├── registry/
│   ├── orchestrator/
│   ├── runner/
│   ├── cache/
│   ├── config/
│   ├── spec/
│   ├── format/
│   └── log/
├── docs/
│   ├── superpowers/specs/
│   ├── adapters.md
│   └── architecture.md
├── tests/
│   ├── e2e/docker/
│   └── golden/
├── .goreleaser.yaml
├── .github/workflows/       ci.yml, release.yml, e2e.yml
├── Makefile
├── go.mod
├── README.md
├── CLAUDE.md
├── AGENTS.md
├── SKILLS.md
└── LICENSE
```

## 13. Risks & Open Questions

- **Native output drift:** PMs change their output format between versions. Mitigation: golden fixtures per PM major version; CI nightly e2e catches drift; parsers fail loud with `ParseError` not silent miss.
- **Sudo UX on TUI:** terminal hand-off to `sudo` interrupts Bubbletea redraw. Mitigation: suspend tea program → run sudo cmd → resume on completion. Reuses `tea.ExecProcess`.
- **Search rate limits:** some PMs (pkg.go.dev, conda) throttle. Mitigation: cache aggressively, surface `429` via `NetworkFailure`, never retry storm.
- **Windows path:** v1 ships but is least-tested. Container e2e n/a; rely on GHA windows runner.
- **License of fixtures:** real PM outputs may contain copyrighted summaries. Mitigation: fixtures store our re-paraphrased text where licensing unclear.

## 14. Out of Scope (deferred)

- Cross-PM dependency resolution
- pkgr's own package format
- Cloud sync of installed lists
- Plugin system for third-party adapters
- GUI (non-terminal)
- snap/flatpak as **distribution channels for pkgr itself** (v1.1) — note: flatpak/snap are still supported as **adapters** in v1
