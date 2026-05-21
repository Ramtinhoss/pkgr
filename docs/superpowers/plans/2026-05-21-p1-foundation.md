# Phase 1: Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scaffold the `pkgr` Go module, ship a working `pkgr version` binary, and build all utility internals (logger, config, spec parser, error model, Manager interface, runner, cache, registry) with TDD coverage. No actual adapters yet.

**Architecture:** Layered Go project. `cmd/pkgr/` is the cobra entry; `internal/*` packages are narrow utilities consumed by later phases. Each package is independently testable with no upward dependencies. Mutual exclusion on cache files via `flock`. Logging via stdlib `slog`. Config via TOML.

**Tech Stack:** Go 1.22, [spf13/cobra](https://github.com/spf13/cobra), [BurntSushi/toml](https://github.com/BurntSushi/toml), [gofrs/flock](https://github.com/gofrs/flock), `log/slog` stdlib, `golangci-lint`, GitHub Actions.

---

## File Structure

```
pkgr/
├── cmd/pkgr/main.go
├── go.mod, go.sum
├── Makefile
├── .gitignore, .editorconfig
├── .golangci.yml
├── .github/workflows/ci.yml
├── LICENSE (MIT)
├── README.md
├── internal/
│   ├── log/log.go, log_test.go
│   ├── config/config.go, config_test.go, defaults.go
│   ├── spec/spec.go, spec_test.go
│   ├── manager/types.go, errors.go, types_test.go
│   ├── runner/runner.go, runner_test.go, fake.go
│   ├── cache/cache.go, cache_test.go
│   └── registry/registry.go, registry_test.go
```

Each file ≤ 200 LOC. Tests live next to source.

---

### Task 1: Scaffold module + Makefile + gitignore

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `.editorconfig`
- Create: `Makefile`
- Create: `LICENSE`
- Create: `README.md`

- [ ] **Step 1: Initialize Go module**

Run:
```bash
cd ~/pkgr && go mod init github.com/ramtinhoss/pkgr
```
Expected: creates `go.mod` with `module github.com/ramtinhoss/pkgr` and `go 1.22`.

- [ ] **Step 2: Write `.gitignore`**

```gitignore
# binaries
/pkgr
/dist/
*.exe

# go
*.test
*.out
coverage.txt
vendor/

# editor
.idea/
.vscode/
*.swp
.DS_Store

# pkgr local state (in case dev uses tool on own repo)
.pkgr-cache/
```

- [ ] **Step 3: Write `.editorconfig`**

```editorconfig
root = true

[*]
charset = utf-8
end_of_line = lf
indent_style = tab
indent_size = 4
insert_final_newline = true
trim_trailing_whitespace = true

[*.{md,yml,yaml,toml,json}]
indent_style = space
indent_size = 2
```

- [ ] **Step 4: Write `Makefile`**

```makefile
.PHONY: build test lint fmt vet tidy run clean

BINARY := pkgr
PKG    := github.com/ramtinhoss/pkgr
GO     ?= go

build:
	$(GO) build -trimpath -ldflags "-s -w -X main.version=dev" -o $(BINARY) ./cmd/pkgr

test:
	$(GO) test -race -coverprofile=coverage.txt ./...

lint:
	golangci-lint run

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

run:
	$(GO) run ./cmd/pkgr

clean:
	rm -f $(BINARY) coverage.txt
	rm -rf dist
```

- [ ] **Step 5: Write `LICENSE` (MIT)**

```
MIT License

Copyright (c) 2026 Ramtin Hosseini

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 6: Write `README.md` stub**

```markdown
# pkgr

Cross-platform package manager TUI/CLI. Wraps brew, apt, dnf, snap, flatpak, nix, scoop, choco, winget, npm, pnpm, yarn, bun, pip, pipx, uv, conda, mamba, cargo, gem, go, asdf, mise, mas, rustup behind one uniform interface.

**Status:** under construction (Phase 1: foundation).

See `docs/superpowers/specs/2026-05-21-pkgr-design.md` for the v1 design.

## Build

```
make build
./pkgr version
```
```

- [ ] **Step 7: Commit**

```bash
git add go.mod .gitignore .editorconfig Makefile LICENSE README.md
git commit -m "chore: scaffold Go module + Makefile + LICENSE"
```

---

### Task 2: Logger (`internal/log`)

**Files:**
- Create: `internal/log/log.go`
- Create: `internal/log/log_test.go`

- [ ] **Step 1: Write failing test `internal/log/log_test.go`**

```go
package log

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupWritesJSONToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgr.log")

	l, closer, err := Setup(Options{Path: path, Verbose: false})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer closer()

	l.Info("hello", slog.String("k", "v"))

	if err := closer(); err != nil {
		t.Fatalf("close: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(b), `"msg":"hello"`) {
		t.Fatalf("missing msg: %s", b)
	}
	if !strings.Contains(string(b), `"k":"v"`) {
		t.Fatalf("missing attr: %s", b)
	}
}

func TestSetupVerboseMirrorsToStderr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgr.log")
	var buf bytes.Buffer

	opts := Options{Path: path, Verbose: true, Stderr: &buf}
	l, closer, err := Setup(opts)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	l.Info("mirror")
	_ = closer()

	if !strings.Contains(buf.String(), "mirror") {
		t.Fatalf("expected stderr mirror, got %q", buf.String())
	}
}
```

- [ ] **Step 2: Run test, expect FAIL**

```bash
go test ./internal/log/...
```
Expected: `undefined: Setup` compilation failure.

- [ ] **Step 3: Implement `internal/log/log.go`**

```go
// Package log configures slog with JSON output to a rotating file,
// optionally mirroring to stderr.
package log

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type Options struct {
	Path    string
	Verbose bool
	Stderr  io.Writer // injected for tests; defaults to os.Stderr
}

// Setup returns a configured slog.Logger and a closer that flushes
// and closes underlying file handles.
func Setup(opts Options) (*slog.Logger, func() error, error) {
	if err := os.MkdirAll(filepath.Dir(opts.Path), 0o755); err != nil {
		return nil, nil, err
	}
	f, err := os.OpenFile(opts.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	var w io.Writer = f
	if opts.Verbose {
		stderr := opts.Stderr
		if stderr == nil {
			stderr = os.Stderr
		}
		w = io.MultiWriter(f, stderr)
	}

	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := slog.New(h)

	return l, f.Close, nil
}
```

- [ ] **Step 4: Run tests, expect PASS**

```bash
go test ./internal/log/... -v
```
Expected: both tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/log/
git commit -m "feat(log): slog JSON logger with optional stderr mirror"
```

---

### Task 3: Config (`internal/config`)

**Files:**
- Create: `internal/config/defaults.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test `internal/config/config_test.go`**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaultsWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.General.Theme != "auto" {
		t.Fatalf("Theme = %q, want auto", cfg.General.Theme)
	}
	if cfg.Cache.InstalledTTL != 5*time.Minute {
		t.Fatalf("InstalledTTL = %v, want 5m", cfg.Cache.InstalledTTL)
	}
	if len(cfg.Ranking.Preferred) == 0 {
		t.Fatalf("Ranking.Preferred is empty")
	}
}

func TestLoadParsesTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := `
[general]
theme = "dark"
verbose = true

[cache]
installed_ttl = "1m"

[managers.brew]
enabled = false
extra_args = ["--quiet"]
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.General.Theme != "dark" {
		t.Fatalf("Theme = %q", cfg.General.Theme)
	}
	if !cfg.General.Verbose {
		t.Fatal("Verbose not parsed")
	}
	if cfg.Cache.InstalledTTL != time.Minute {
		t.Fatalf("InstalledTTL = %v", cfg.Cache.InstalledTTL)
	}
	brew, ok := cfg.Managers["brew"]
	if !ok {
		t.Fatal("brew block missing")
	}
	if brew.Enabled {
		t.Fatal("brew should be disabled")
	}
	if len(brew.ExtraArgs) != 1 || brew.ExtraArgs[0] != "--quiet" {
		t.Fatalf("ExtraArgs = %v", brew.ExtraArgs)
	}
}
```

- [ ] **Step 2: Run test, expect FAIL**

```bash
go test ./internal/config/...
```
Expected: `undefined: Load` compile error.

- [ ] **Step 3: Implement `internal/config/defaults.go`**

```go
package config

import "time"

func Defaults() Config {
	return Config{
		General: General{
			DefaultAssumeYes: false,
			Verbose:          false,
			Theme:            "auto",
			JSONOutput:       false,
			UpdateCheck:      true,
		},
		Cache: Cache{
			Enabled:      true,
			InstalledTTL: 5 * time.Minute,
			OutdatedTTL:  30 * time.Minute,
			SearchTTL:    time.Hour,
			InfoTTL:      24 * time.Hour,
			RegistryTTL:  time.Hour,
		},
		Ranking: Ranking{
			Preferred: []string{
				"brew", "apt", "dnf", "pacman", "winget", "scoop",
				"uv", "pipx", "pip", "cargo", "npm", "pnpm", "bun",
			},
		},
		Managers: map[string]Manager{},
	}
}
```

- [ ] **Step 4: Implement `internal/config/config.go`**

```go
// Package config loads TOML configuration and applies defaults.
package config

import (
	"errors"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General  General            `toml:"general"`
	Cache    Cache              `toml:"cache"`
	Ranking  Ranking            `toml:"ranking"`
	Managers map[string]Manager `toml:"managers"`
}

type General struct {
	DefaultAssumeYes bool   `toml:"default_assume_yes"`
	Verbose          bool   `toml:"verbose"`
	Theme            string `toml:"theme"`
	JSONOutput       bool   `toml:"json_output"`
	UpdateCheck      bool   `toml:"update_check"`
}

type Cache struct {
	Enabled      bool          `toml:"enabled"`
	InstalledTTL time.Duration `toml:"installed_ttl"`
	OutdatedTTL  time.Duration `toml:"outdated_ttl"`
	SearchTTL    time.Duration `toml:"search_ttl"`
	InfoTTL      time.Duration `toml:"info_ttl"`
	RegistryTTL  time.Duration `toml:"registry_ttl"`
}

type Ranking struct {
	Preferred []string `toml:"preferred"`
}

type Manager struct {
	Enabled   bool     `toml:"enabled"`
	ExtraArgs []string `toml:"extra_args"`
	Sudo      *bool    `toml:"sudo"` // nil = use adapter default
}

// Load reads a TOML config, applying defaults for any missing fields.
// If the file does not exist, returns full defaults.
func Load(path string) (Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}

	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return Config{}, err
	}

	// re-apply defaults for zero-valued duration fields
	d := Defaults()
	if cfg.Cache.InstalledTTL == 0 {
		cfg.Cache.InstalledTTL = d.Cache.InstalledTTL
	}
	if cfg.Cache.OutdatedTTL == 0 {
		cfg.Cache.OutdatedTTL = d.Cache.OutdatedTTL
	}
	if cfg.Cache.SearchTTL == 0 {
		cfg.Cache.SearchTTL = d.Cache.SearchTTL
	}
	if cfg.Cache.InfoTTL == 0 {
		cfg.Cache.InfoTTL = d.Cache.InfoTTL
	}
	if cfg.Cache.RegistryTTL == 0 {
		cfg.Cache.RegistryTTL = d.Cache.RegistryTTL
	}
	if cfg.General.Theme == "" {
		cfg.General.Theme = d.General.Theme
	}
	if cfg.Managers == nil {
		cfg.Managers = map[string]Manager{}
	}

	return cfg, nil
}
```

- [ ] **Step 5: Add BurntSushi/toml dep**

```bash
go get github.com/BurntSushi/toml@latest
go mod tidy
```

- [ ] **Step 6: Run tests, expect PASS**

```bash
go test ./internal/config/... -v
```
Expected: both tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat(config): TOML loader with sane defaults"
```

---

### Task 4: Spec parser (`internal/spec`)

**Files:**
- Create: `internal/spec/spec.go`
- Create: `internal/spec/spec_test.go`

- [ ] **Step 1: Write failing test `internal/spec/spec_test.go`**

```go
package spec

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in    string
		name  string
		pm    string
		ver   string
		err   bool
	}{
		{"ripgrep", "ripgrep", "", "", false},
		{"ripgrep@brew", "ripgrep", "brew", "", false},
		{"ripgrep==13.0.0", "ripgrep", "", "13.0.0", false},
		{"ripgrep==13.0.0@brew", "ripgrep", "brew", "13.0.0", false},
		{"@scope/pkg@npm", "@scope/pkg", "npm", "", false},
		{"@scope/pkg==1.2.3@npm", "@scope/pkg", "npm", "1.2.3", false},
		{"", "", "", "", true},
		{"==1.2.3", "", "", "", true},
		{"foo@bar@baz", "", "", "", true},
		{"foo==", "", "", "", true},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if c.err {
			if err == nil {
				t.Errorf("Parse(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q) error: %v", c.in, err)
			continue
		}
		if got.Name != c.name || got.PM != c.pm || got.Version != c.ver {
			t.Errorf("Parse(%q) = %+v", c.in, got)
		}
	}
}
```

- [ ] **Step 2: Run test, expect FAIL**

```bash
go test ./internal/spec/...
```
Expected: `undefined: Parse` compile error.

- [ ] **Step 3: Implement `internal/spec/spec.go`**

```go
// Package spec parses package specs of the form "name[==ver][@pm]".
// Names may include leading "@scope/" (npm scoped packages).
package spec

import (
	"errors"
	"strings"
)

type Spec struct {
	Name    string
	Version string
	PM      string
}

var (
	ErrEmpty       = errors.New("spec: empty")
	ErrNoName      = errors.New("spec: missing name")
	ErrMultiplePM  = errors.New("spec: multiple @pm separators")
	ErrEmptyVer    = errors.New("spec: '==' with empty version")
)

// Parse interprets a spec string.
func Parse(s string) (Spec, error) {
	if s == "" {
		return Spec{}, ErrEmpty
	}

	// Detect @pm at the end. For scoped npm packages the leading "@" is
	// part of the name; the PM @ is always preceded by a non-empty body.
	var pm, body string
	if strings.HasPrefix(s, "@") {
		// scoped name. PM separator is the *last* '@' if any.
		if idx := strings.LastIndex(s, "@"); idx > 0 {
			pm = s[idx+1:]
			body = s[:idx]
		} else {
			body = s
		}
	} else if strings.Count(s, "@") > 1 {
		return Spec{}, ErrMultiplePM
	} else if idx := strings.LastIndex(s, "@"); idx >= 0 {
		pm = s[idx+1:]
		body = s[:idx]
	} else {
		body = s
	}

	if body == "" {
		return Spec{}, ErrNoName
	}

	var name, ver string
	if idx := strings.Index(body, "=="); idx >= 0 {
		name = body[:idx]
		ver = body[idx+2:]
		if ver == "" {
			return Spec{}, ErrEmptyVer
		}
	} else {
		name = body
	}

	if name == "" {
		return Spec{}, ErrNoName
	}

	return Spec{Name: name, Version: ver, PM: pm}, nil
}
```

- [ ] **Step 4: Run tests, expect PASS**

```bash
go test ./internal/spec/... -v
```
Expected: all subcases pass.

- [ ] **Step 5: Commit**

```bash
git add internal/spec/
git commit -m "feat(spec): parser for name[==ver][@pm] including scoped npm names"
```

---

### Task 5: Manager types + errors (`internal/manager`)

**Files:**
- Create: `internal/manager/types.go`
- Create: `internal/manager/errors.go`
- Create: `internal/manager/types_test.go`

- [ ] **Step 1: Write test `internal/manager/types_test.go`**

```go
package manager

import (
	"errors"
	"testing"
)

func TestErrorFormatting(t *testing.T) {
	base := errors.New("connection refused")
	e := &Error{
		Manager: "brew",
		Op:      OpInstall,
		Code:    CodeNetworkFailure,
		Err:     base,
		Cmd:     "brew install ripgrep",
		Stderr:  "curl: (7) Failed to connect",
	}
	s := e.Error()
	wantParts := []string{"brew", "install", "network_failure", "connection refused"}
	for _, w := range wantParts {
		if !contains(s, w) {
			t.Errorf("Error() = %q, missing %q", s, w)
		}
	}
	if !errors.Is(e, base) {
		t.Error("errors.Is should unwrap to base")
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (indexOf(s, sub) >= 0))
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Run test, expect FAIL**

```bash
go test ./internal/manager/...
```
Expected: `undefined: Error / OpInstall / CodeNetworkFailure` compile errors.

- [ ] **Step 3: Implement `internal/manager/types.go`**

```go
// Package manager defines the adapter interface and shared types.
package manager

import "context"

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
	Name        string            `json:"name"`
	Version     string            `json:"version,omitempty"`
	Latest      string            `json:"latest,omitempty"`
	Manager     string            `json:"manager"`
	Installed   bool              `json:"installed"`
	Description string            `json:"description,omitempty"`
	Homepage    string            `json:"homepage,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
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
	Update(ctx context.Context, names ...string) error
}
```

- [ ] **Step 4: Implement `internal/manager/errors.go`**

```go
package manager

import "fmt"

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

func (e *Error) Error() string {
	return fmt.Sprintf("manager=%s op=%s code=%s err=%v",
		e.Manager, e.Op, e.Code, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }
```

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/manager/... -v
```
Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/manager/
git commit -m "feat(manager): interface, Package type, typed Error"
```

---

### Task 6: Runner (`internal/runner`)

**Files:**
- Create: `internal/runner/runner.go`
- Create: `internal/runner/fake.go`
- Create: `internal/runner/runner_test.go`

- [ ] **Step 1: Write failing test `internal/runner/runner_test.go`**

```go
package runner

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunDryRunPrintsAndSkips(t *testing.T) {
	var out bytes.Buffer
	r := &Runner{Out: &out, DryRun: true}

	res, err := r.Run(context.Background(), Cmd{Bin: "brew", Args: []string{"install", "ripgrep"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Skipped != true {
		t.Fatal("expected Skipped=true under DryRun")
	}
	if !strings.Contains(out.String(), "brew install ripgrep") {
		t.Fatalf("dry-run output missing cmd: %q", out.String())
	}
}

func TestRunUsesFakeExecutor(t *testing.T) {
	fake := &Fake{
		Returns: map[string]FakeResult{
			"brew search ripgrep": {Stdout: []byte("ripgrep\n"), Code: 0},
		},
	}
	r := &Runner{Exec: fake.Exec}

	res, err := r.Run(context.Background(), Cmd{Bin: "brew", Args: []string{"search", "ripgrep"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(string(res.Stdout), "ripgrep") {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestRunFakeNonZeroExitReturnsError(t *testing.T) {
	fake := &Fake{
		Returns: map[string]FakeResult{
			"brew install missing": {Stderr: []byte("not found"), Code: 1},
		},
	}
	r := &Runner{Exec: fake.Exec}

	_, err := r.Run(context.Background(), Cmd{Bin: "brew", Args: []string{"install", "missing"}})
	if err == nil {
		t.Fatal("expected error on non-zero exit")
	}
}
```

- [ ] **Step 2: Run test, expect FAIL**

```bash
go test ./internal/runner/...
```
Expected: compile errors for `Runner / Cmd / Fake`.

- [ ] **Step 3: Implement `internal/runner/runner.go`**

```go
// Package runner provides an exec wrapper with dry-run support and an
// injectable Exec function for tests.
package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Cmd struct {
	Bin   string
	Args  []string
	Env   []string // additional environment vars; nil = inherit
	Sudo  bool     // wrap with sudo
	Stdin io.Reader
}

type Result struct {
	Stdout  []byte
	Stderr  []byte
	Code    int
	Skipped bool // true when DryRun was set
}

// ExecFunc runs a process and returns its result. Real impl execs;
// tests inject a fake.
type ExecFunc func(ctx context.Context, c Cmd) (Result, error)

type Runner struct {
	Out    io.Writer // for dry-run printout; defaults to os.Stdout
	DryRun bool
	Exec   ExecFunc // if nil, uses RealExec
}

// Run is the single entry point. Honors DryRun and delegates to Exec.
func (r *Runner) Run(ctx context.Context, c Cmd) (Result, error) {
	if r.DryRun {
		w := r.Out
		if w == nil {
			w = os.Stdout
		}
		prefix := ""
		if c.Sudo {
			prefix = "sudo "
		}
		fmt.Fprintf(w, "→ would exec: %s%s %s\n", prefix, c.Bin, strings.Join(c.Args, " "))
		return Result{Skipped: true}, nil
	}
	exe := r.Exec
	if exe == nil {
		exe = RealExec
	}
	return exe(ctx, c)
}

// RealExec runs the command via os/exec. Captures stdout+stderr.
func RealExec(ctx context.Context, c Cmd) (Result, error) {
	args := c.Args
	bin := c.Bin
	if c.Sudo {
		args = append([]string{"-p", fmt.Sprintf("pkgr needs sudo for %s: ", bin), bin}, args...)
		bin = "sudo"
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	if len(c.Env) > 0 {
		cmd.Env = append(os.Environ(), c.Env...)
	}
	cmd.Stdin = c.Stdin

	var stdout, stderr strings_Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := Result{
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
		Code:   cmd.ProcessState.ExitCode(),
	}
	if err != nil {
		return res, fmt.Errorf("exec %s: %w", bin, err)
	}
	return res, nil
}

// strings_Builder is a tiny bytes.Buffer-equivalent kept inline to avoid an
// extra import; reuses standard buffer behavior.
type strings_Builder struct{ b []byte }

func (s *strings_Builder) Write(p []byte) (int, error) { s.b = append(s.b, p...); return len(p), nil }
func (s *strings_Builder) Bytes() []byte               { return s.b }
```

- [ ] **Step 4: Replace inline buffer with stdlib `bytes.Buffer`**

Edit `internal/runner/runner.go` to use `bytes.Buffer`:

```go
import "bytes"
// ...
var stdout, stderr bytes.Buffer
// remove the strings_Builder type entirely
```

- [ ] **Step 5: Implement `internal/runner/fake.go`**

```go
package runner

import (
	"context"
	"fmt"
	"strings"
)

type FakeResult struct {
	Stdout []byte
	Stderr []byte
	Code   int
}

// Fake is a deterministic Exec impl. Returns[k] looks up by
// "<bin> <space-joined args>".
type Fake struct {
	Returns map[string]FakeResult
	Calls   []string // populated as commands run, for assertion
}

func (f *Fake) Exec(_ context.Context, c Cmd) (Result, error) {
	key := strings.TrimSpace(c.Bin + " " + strings.Join(c.Args, " "))
	f.Calls = append(f.Calls, key)
	r, ok := f.Returns[key]
	if !ok {
		return Result{}, fmt.Errorf("fake runner: no canned reply for %q", key)
	}
	res := Result{Stdout: r.Stdout, Stderr: r.Stderr, Code: r.Code}
	if r.Code != 0 {
		return res, fmt.Errorf("fake runner: exit %d for %q", r.Code, key)
	}
	return res, nil
}
```

- [ ] **Step 6: Run tests, expect PASS**

```bash
go test ./internal/runner/... -v
```
Expected: all three tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/runner/
git commit -m "feat(runner): exec wrapper with dry-run and injectable Fake"
```

---

### Task 7: Cache (`internal/cache`)

**Files:**
- Create: `internal/cache/cache.go`
- Create: `internal/cache/cache_test.go`

- [ ] **Step 1: Write failing test `internal/cache/cache_test.go`**

```go
package cache

import (
	"path/filepath"
	"testing"
	"time"
)

func TestGetSetRoundTrip(t *testing.T) {
	c := New(t.TempDir())
	type doc struct {
		Items []string
	}
	in := doc{Items: []string{"a", "b"}}
	if err := c.Set("brew/installed", in, 1*time.Hour); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var out doc
	hit, err := c.Get("brew/installed", &out)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hit {
		t.Fatal("expected cache hit")
	}
	if len(out.Items) != 2 || out.Items[0] != "a" {
		t.Fatalf("Items = %v", out.Items)
	}
}

func TestExpiredEntryIsMiss(t *testing.T) {
	c := New(t.TempDir())
	if err := c.Set("k", 1, 10*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	var v int
	hit, err := c.Get("k", &v)
	if err != nil {
		t.Fatal(err)
	}
	if hit {
		t.Fatal("expected miss after expiry")
	}
}

func TestInvalidateRemovesKey(t *testing.T) {
	c := New(t.TempDir())
	_ = c.Set("brew/installed", []int{1}, time.Hour)
	if err := c.Invalidate("brew/installed"); err != nil {
		t.Fatal(err)
	}
	var out []int
	hit, _ := c.Get("brew/installed", &out)
	if hit {
		t.Fatal("expected miss after invalidate")
	}
}

func TestPathIsSandboxed(t *testing.T) {
	c := New(t.TempDir())
	if err := c.Set("../escape", 1, time.Hour); err == nil {
		t.Fatal("expected error on path traversal key")
	}
	got := filepath.Clean(c.PathFor("brew/installed"))
	want := filepath.Clean(filepath.Join(c.Root, "brew", "installed.json"))
	if got != want {
		t.Fatalf("PathFor = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run test, expect FAIL**

```bash
go test ./internal/cache/...
```
Expected: `undefined: New` compile error.

- [ ] **Step 3: Implement `internal/cache/cache.go`**

```go
// Package cache provides a TTL'd JSON file cache with per-file flock.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

type entry struct {
	FetchedAt   time.Time       `json:"fetched_at"`
	TTLSeconds  int64           `json:"ttl_seconds"`
	Data        json.RawMessage `json:"data"`
}

type Cache struct {
	Root string
}

func New(root string) *Cache { return &Cache{Root: root} }

var ErrUnsafeKey = errors.New("cache: unsafe key")

// PathFor maps a logical key to a file path under Root.
func (c *Cache) PathFor(key string) string {
	return filepath.Join(c.Root, key+".json")
}

func (c *Cache) checkKey(key string) error {
	cleaned := filepath.Clean(key)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return ErrUnsafeKey
	}
	return nil
}

func (c *Cache) Set(key string, v any, ttl time.Duration) error {
	if err := c.checkKey(key); err != nil {
		return err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	e := entry{FetchedAt: time.Now(), TTLSeconds: int64(ttl.Seconds()), Data: raw}
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	path := c.PathFor(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	lock := flock.New(path + ".lock")
	if err := lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Get unmarshals into dst and returns (hit, error). A miss returns (false, nil).
func (c *Cache) Get(key string, dst any) (bool, error) {
	if err := c.checkKey(key); err != nil {
		return false, err
	}
	path := c.PathFor(key)
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	var e entry
	if err := json.Unmarshal(body, &e); err != nil {
		return false, fmt.Errorf("cache: parse %s: %w", path, err)
	}
	if time.Since(e.FetchedAt) > time.Duration(e.TTLSeconds)*time.Second {
		return false, nil
	}
	if err := json.Unmarshal(e.Data, dst); err != nil {
		return false, fmt.Errorf("cache: parse data: %w", err)
	}
	return true, nil
}

func (c *Cache) Invalidate(key string) error {
	if err := c.checkKey(key); err != nil {
		return err
	}
	path := c.PathFor(key)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
```

- [ ] **Step 4: Add flock dep**

```bash
go get github.com/gofrs/flock@latest
go mod tidy
```

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/cache/... -v
```
Expected: 4 tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/cache/ go.mod go.sum
git commit -m "feat(cache): TTL JSON file cache with flock and path sandbox"
```

---

### Task 8: Registry (`internal/registry`)

**Files:**
- Create: `internal/registry/registry.go`
- Create: `internal/registry/registry_test.go`

- [ ] **Step 1: Write failing test `internal/registry/registry_test.go`**

```go
package registry

import (
	"context"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

type stub struct {
	id       string
	detected bool
}

func (s *stub) ID() string                                                   { return s.id }
func (s *stub) DisplayName() string                                          { return s.id }
func (s *stub) OSes() []manager.OS                                           { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (s *stub) Detect() bool                                                 { return s.detected }
func (s *stub) NeedsSudo(manager.Op) bool                                    { return false }
func (s *stub) Scope() manager.Scope                                         { return manager.ScopeUserGlobal }
func (s *stub) List(context.Context) ([]manager.Package, error)              { return nil, nil }
func (s *stub) Outdated(context.Context) ([]manager.Package, error)          { return nil, nil }
func (s *stub) Search(context.Context, string) ([]manager.Package, error)    { return nil, nil }
func (s *stub) Info(context.Context, string) (manager.Package, error)        { return manager.Package{}, nil }
func (s *stub) Install(context.Context, ...string) error                     { return nil }
func (s *stub) Uninstall(context.Context, ...string) error                   { return nil }
func (s *stub) Update(context.Context, ...string) error                      { return nil }

func TestActiveReturnsDetectedAndEnabled(t *testing.T) {
	r := New()
	r.Register(&stub{id: "a", detected: true})
	r.Register(&stub{id: "b", detected: false})
	r.Register(&stub{id: "c", detected: true})
	r.SetEnabled(map[string]bool{"a": true, "b": true, "c": false})

	got := r.Active()
	if len(got) != 1 || got[0].ID() != "a" {
		t.Fatalf("Active = %+v", got)
	}
}

func TestAllReturnsEverythingRegardlessOfDetection(t *testing.T) {
	r := New()
	r.Register(&stub{id: "a", detected: true})
	r.Register(&stub{id: "b", detected: false})
	if len(r.All()) != 2 {
		t.Fatalf("All = %d", len(r.All()))
	}
}
```

- [ ] **Step 2: Run test, expect FAIL**

```bash
go test ./internal/registry/...
```
Expected: `undefined: New / Register / Active` compile errors.

- [ ] **Step 3: Implement `internal/registry/registry.go`**

```go
// Package registry holds all known package-manager adapters and resolves
// which are "active" (detected on PATH + enabled via config).
package registry

import (
	"sort"
	"sync"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

type Registry struct {
	mu       sync.RWMutex
	managers map[string]manager.Manager
	enabled  map[string]bool // user override; missing = use Detect()
}

func New() *Registry {
	return &Registry{
		managers: make(map[string]manager.Manager),
		enabled:  make(map[string]bool),
	}
}

func (r *Registry) Register(m manager.Manager) {
	r.mu.Lock()
	r.managers[m.ID()] = m
	r.mu.Unlock()
}

func (r *Registry) Get(id string) (manager.Manager, bool) {
	r.mu.RLock()
	m, ok := r.managers[id]
	r.mu.RUnlock()
	return m, ok
}

// SetEnabled overrides per-PM enable state. Pass map[id]bool from config.
func (r *Registry) SetEnabled(m map[string]bool) {
	r.mu.Lock()
	for k, v := range m {
		r.enabled[k] = v
	}
	r.mu.Unlock()
}

// All returns every registered manager, sorted by ID.
func (r *Registry) All() []manager.Manager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]manager.Manager, 0, len(r.managers))
	for _, m := range r.managers {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// Active returns managers that are both detected and enabled.
// A manager is enabled by default unless config explicitly disables it.
func (r *Registry) Active() []manager.Manager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []manager.Manager
	for id, m := range r.managers {
		enabled, override := r.enabled[id]
		if override && !enabled {
			continue
		}
		if !m.Detect() {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}
```

- [ ] **Step 4: Run tests, expect PASS**

```bash
go test ./internal/registry/... -v
```
Expected: 2 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/registry/
git commit -m "feat(registry): adapter registration + active resolution"
```

---

### Task 9: CLI entry (`cmd/pkgr`) + cobra root + version

**Files:**
- Create: `cmd/pkgr/main.go`
- Create: `cmd/pkgr/version.go`
- Create: `cmd/pkgr/main_test.go`

- [ ] **Step 1: Write failing test `cmd/pkgr/main_test.go`**

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommandPrints(t *testing.T) {
	var out bytes.Buffer
	root := newRootCmd(buildInfo{Version: "1.2.3", Commit: "abc", Date: "2026-01-01"})
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got := out.String()
	for _, want := range []string{"1.2.3", "abc", "2026-01-01"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run test, expect FAIL**

```bash
go test ./cmd/pkgr/...
```
Expected: `undefined: newRootCmd / buildInfo` compile errors.

- [ ] **Step 3: Implement `cmd/pkgr/main.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// build-time stamped values
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type buildInfo struct {
	Version, Commit, Date string
}

func main() {
	root := newRootCmd(buildInfo{Version: version, Commit: commit, Date: date})
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func newRootCmd(b buildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:           "pkgr",
		Short:         "Cross-platform package manager TUI/CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCmd(b))
	return root
}
```

- [ ] **Step 4: Implement `cmd/pkgr/version.go`**

```go
package main

import (
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd(b buildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit, and build info",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf("pkgr %s\ncommit:  %s\ndate:    %s\ngo:      %s\nplatform: %s/%s\n",
				b.Version, b.Commit, b.Date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}
```

- [ ] **Step 5: Add cobra dep**

```bash
go get github.com/spf13/cobra@latest
go mod tidy
```

- [ ] **Step 6: Run tests, expect PASS**

```bash
go test ./cmd/pkgr/... -v
make build
./pkgr version
```
Expected: test passes; binary prints `pkgr dev` + build info.

- [ ] **Step 7: Commit**

```bash
git add cmd/pkgr/ go.mod go.sum
git commit -m "feat(cli): cobra root + version subcommand"
```

---

### Task 10: golangci-lint config + CI workflow

**Files:**
- Create: `.golangci.yml`
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Write `.golangci.yml`**

```yaml
run:
  timeout: 5m
  go: "1.22"

linters:
  enable:
    - govet
    - staticcheck
    - errcheck
    - gosec
    - revive
    - ineffassign
    - unused
    - gofmt
    - goimports
    - misspell

linters-settings:
  gosec:
    excludes:
      - G204  # subprocess with variable: that's our entire purpose
  revive:
    rules:
      - name: var-naming
        disabled: false
```

- [ ] **Step 2: Write `.github/workflows/ci.yml`**

```yaml
name: ci
on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          cache: true
      - run: go mod tidy && git diff --exit-code go.mod go.sum
      - run: make test

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest
```

- [ ] **Step 3: Run lint locally to verify config**

```bash
golangci-lint run
```
Expected: no errors (some warnings OK for stub code).

- [ ] **Step 4: Commit**

```bash
git add .golangci.yml .github/workflows/ci.yml
git commit -m "ci: golangci-lint config + GitHub Actions test+lint matrix"
```

---

### Task 11: Auto-scribe scaffold docs

**Files:**
- Create: `CLAUDE.md`
- Create: `AGENTS.md`
- Create: `SKILLS.md`
- Modify: `README.md`

- [ ] **Step 1: Write `CLAUDE.md`** (invariants, layout, conventions)

```markdown
# pkgr — CLAUDE Project Notes

## Invariants
- Never shell out without going through `internal/runner`. It owns dry-run and sudo handling.
- Every adapter MUST live in its own package under `internal/manager/<id>/`.
- Adapters MUST NOT import each other. Cross-adapter logic lives in `internal/orchestrator`.
- `cmd/pkgr` is the ONLY entry point. No `init()` side effects elsewhere.

## Layout
See `docs/superpowers/specs/2026-05-21-pkgr-design.md` §12.

## Conventions
- File names: `snake_case.go` is wrong; use Go default `lowercase.go`.
- Each adapter file ≤ 200 LOC; if larger, split parser into `<id>_parse.go`.
- Tests live next to code; golden fixtures in `testdata/`.
- Commits: Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`, `test:`, `ci:`, `refactor:`).
- PR titles same prefix.

## Recipes
- Add an adapter: copy `internal/manager/_template/` to `internal/manager/<id>/`, rename, implement, register in `cmd/pkgr/managers.go`.
- Refresh fixtures: `make refresh-fixtures PM=brew` (added in phase 4).

## Watch-outs
- Native PM output formats drift between versions; parsers must fail loud (CodeParseError) not silently mis-parse.
- Windows tests run on GHA only; do not assume POSIX paths.
- Never store sudo password. Never `sudo -S`.
```

- [ ] **Step 2: Write `AGENTS.md`**

```markdown
# pkgr — Agents Guide

## Tool selection
- File reads/edits: use Read/Edit, not cat/sed.
- Search: `rg` over `grep`. `find . -name` over `find /`.
- Build/test: `make test`, `make lint`, `make build`.

## Editing checklist
- [ ] Read file before editing.
- [ ] Run tests for touched package.
- [ ] Run `make lint`.
- [ ] Update docs if behavior changed (README, CLAUDE.md, design.md).
- [ ] One logical commit per change.

## Subagent dispatch
- Big refactors → `superpowers:code-reviewer` after.
- Stuck on a parser → spawn `Explore` with golden fixture content.
- Never spawn an agent to write a commit.

## Risky actions (require confirmation)
- `git push`, `git reset --hard`, `git rebase -i`
- Editing CI secrets
- Deleting fixtures
```

- [ ] **Step 3: Write `SKILLS.md`**

```markdown
# pkgr — Skills

## Slash commands
- `/scribe`: refresh scaffold docs + propose commit (auto-scribe plugin).
- `/code-review`: review changeset against design + lint rules.

## Recipes
- **Add adapter**: see CLAUDE.md "Recipes".
- **Refresh fixtures**: capture real PM output, sanitize for licensing, drop into `testdata/`.
- **Update version**: bump tag, CI release workflow handles rest.

## Anti-patterns
- Don't add a PM-specific Go library; shell out.
- Don't parse PM output without a golden fixture covering the format.
- Don't cache mutating ops; bust cache instead.
```

- [ ] **Step 4: Append to `README.md`**

Edit `README.md`, append:

```markdown
## Development

- Go 1.22+
- `make build` — build local binary
- `make test` — run unit tests
- `make lint` — golangci-lint
- See `docs/superpowers/specs/2026-05-21-pkgr-design.md` for the v1 design.
- See `docs/superpowers/plans/` for implementation plans.
```

- [ ] **Step 5: Commit**

```bash
git add CLAUDE.md AGENTS.md SKILLS.md README.md
git commit -m "docs: add scaffold docs (CLAUDE, AGENTS, SKILLS, README dev section)"
```

---

## Phase 1 Acceptance

- `make build && ./pkgr version` prints version/commit/date/go/platform
- `make test` passes (≥ 5 packages)
- `make lint` passes
- CI workflow exists and green on push
- `internal/{log,config,spec,manager,runner,cache,registry}` all present, ≤ 200 LOC, ≥ 80 % coverage on parsers/utils
