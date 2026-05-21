# Phase 2: First Adapters + Orchestrator + CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `pkgr` actually useful from the command line. Three adapters (`brew`, `npm`, `pip`), the aggregate orchestrator, formatters (human + JSON), and the CLI subcommands `search`, `list`, `info`, `install`, `remove`, `update`, `outdated`, `pm`, `cache`, `doctor`, `config`, `completion`.

**Architecture:** Each adapter is its own package under `internal/manager/<id>/` and depends only on `internal/manager` (interface), `internal/runner` (exec), and `internal/cache` (caching). The orchestrator fans out aggregate calls in parallel via `errgroup`, merges + ranks, and emits typed `[]Error` for partial failures. CLI subcommands compose orchestrator + formatters; no direct adapter imports from cobra commands.

**Tech Stack:** Go stdlib (`sync/errgroup`), [cobra](https://github.com/spf13/cobra), [BurntSushi/toml](https://github.com/BurntSushi/toml), golden fixtures via `testdata/`.

---

## File Structure

```
internal/manager/brew/{brew.go, brew_test.go, testdata/}
internal/manager/npm/{npm.go, npm_test.go, testdata/}
internal/manager/pip/{pip.go, pip_test.go, testdata/}
internal/manager/_template/template.go         # copy-paste scaffold
internal/orchestrator/{orchestrator.go, orchestrator_test.go}
internal/format/{human.go, json.go, format_test.go}
cmd/pkgr/{managers.go, search.go, list.go, info.go, install.go, remove.go,
          update.go, outdated.go, pm.go, cache.go, doctor.go, config.go,
          completion.go, root_flags.go, app.go}
```

Each adapter file ≤ 250 LOC. Orchestrator ≤ 200 LOC. Each CLI subcommand file ≤ 120 LOC.

---

### Task 1: Adapter template skeleton

**Files:**
- Create: `internal/manager/_template/template.go`

- [ ] **Step 1: Write template scaffold**

```go
// Package _template is a copy-paste starting point for new adapters.
// Replace _template with the adapter ID throughout.
package _template

import (
	"context"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string // override for tests
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "_template"} }

func (a *Adapter) ID() string                  { return "_template" }
func (a *Adapter) DisplayName() string         { return "_template" }
func (a *Adapter) OSes() []manager.OS          { return []manager.OS{manager.Darwin, manager.Linux} }
func (a *Adapter) Scope() manager.Scope        { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool   { return false }
func (a *Adapter) Detect() bool                { return false }

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error)            { return nil, nil }
func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error)        { return nil, nil }
func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error)  { return manager.Package{}, nil }
func (a *Adapter) Install(ctx context.Context, names ...string) error              { return nil }
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error            { return nil }
func (a *Adapter) Update(ctx context.Context, names ...string) error               { return nil }
```

- [ ] **Step 2: Commit**

```bash
git add internal/manager/_template/
git commit -m "feat(manager): adapter template for future copy-paste"
```

---

### Task 2: brew adapter

**Files:**
- Create: `internal/manager/brew/brew.go`
- Create: `internal/manager/brew/brew_test.go`
- Create: `internal/manager/brew/testdata/list_installed.txt`
- Create: `internal/manager/brew/testdata/search.json`
- Create: `internal/manager/brew/testdata/info.json`
- Create: `internal/manager/brew/testdata/outdated.json`

- [ ] **Step 1: Drop golden fixtures**

`internal/manager/brew/testdata/list_installed.txt`:
```
ripgrep 14.1.0
jq 1.7.1
fzf 0.46.0
```

`internal/manager/brew/testdata/search.json`:
```json
{
  "formulae": [
    {"name": "ripgrep", "desc": "Search tool like grep written in Rust", "homepage": "https://github.com/BurntSushi/ripgrep", "versions": {"stable": "14.1.0"}},
    {"name": "ripgrep-all", "desc": "Wrapper that adds support for pdfs/docs", "homepage": "https://github.com/phiresky/ripgrep-all", "versions": {"stable": "0.10.5"}}
  ],
  "casks": []
}
```

`internal/manager/brew/testdata/info.json`:
```json
[
  {"name": "ripgrep", "desc": "Search tool like grep written in Rust", "homepage": "https://github.com/BurntSushi/ripgrep", "versions": {"stable": "14.1.0"}, "installed": [{"version": "14.1.0"}]}
]
```

`internal/manager/brew/testdata/outdated.json`:
```json
{"formulae": [{"name": "jq", "installed_versions": ["1.6"], "current_version": "1.7.1"}], "casks": []}
```

- [ ] **Step 2: Write failing test `internal/manager/brew/brew_test.go`**

```go
package brew

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return b
}

func TestSearchParses(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"brew search --formula --json=v2 ripgrep": {Stdout: loadFixture(t, "search.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "brew"}

	pkgs, err := a.Search(context.Background(), "ripgrep")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("len = %d", len(pkgs))
	}
	if pkgs[0].Name != "ripgrep" || pkgs[0].Version != "14.1.0" || pkgs[0].Manager != "brew" {
		t.Fatalf("pkgs[0] = %+v", pkgs[0])
	}
	if pkgs[0].Homepage == "" {
		t.Fatal("homepage empty")
	}
}

func TestListInstalledParses(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"brew list --versions": {Stdout: loadFixture(t, "list_installed.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "brew"}

	pkgs, err := a.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(pkgs) != 3 {
		t.Fatalf("len = %d", len(pkgs))
	}
	if !pkgs[0].Installed {
		t.Error("Installed=false on listed pkg")
	}
}

func TestOutdatedParses(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"brew outdated --json=v2": {Stdout: loadFixture(t, "outdated.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "brew"}

	pkgs, err := a.Outdated(context.Background())
	if err != nil {
		t.Fatalf("Outdated: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Name != "jq" || pkgs[0].Version != "1.6" || pkgs[0].Latest != "1.7.1" {
		t.Fatalf("pkgs = %+v", pkgs)
	}
}

func TestInstallShellsOut(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"brew install ripgrep": {Stdout: []byte("ok")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "brew"}
	if err := a.Install(context.Background(), "ripgrep"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(fake.Calls) != 1 {
		t.Fatalf("calls = %v", fake.Calls)
	}
}
```

- [ ] **Step 3: Run tests, expect FAIL**

```bash
go test ./internal/manager/brew/...
```
Expected: compile errors.

- [ ] **Step 4: Implement `internal/manager/brew/brew.go`**

```go
// Package brew is the Homebrew adapter.
package brew

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "brew"} }

func (a *Adapter) ID() string                { return "brew" }
func (a *Adapter) DisplayName() string       { return "Homebrew" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool {
	_, err := exec.LookPath(a.Bin)
	return err == nil
}

// ---- search ----------------------------------------------------------------

type searchPayload struct {
	Formulae []struct {
		Name     string            `json:"name"`
		Desc     string            `json:"desc"`
		Homepage string            `json:"homepage"`
		Versions map[string]string `json:"versions"`
	} `json:"formulae"`
	Casks []struct {
		Token    string   `json:"token"`
		Desc     string   `json:"desc"`
		Homepage string   `json:"homepage"`
		Version  string   `json:"version"`
		Names    []string `json:"name"`
	} `json:"casks"`
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{
		Bin:  a.Bin,
		Args: []string{"search", "--formula", "--json=v2", q},
	})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err, Stderr: string(res.Stderr)}
	}
	var p searchPayload
	if err := json.Unmarshal(res.Stdout, &p); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	out := make([]manager.Package, 0, len(p.Formulae)+len(p.Casks))
	for _, f := range p.Formulae {
		out = append(out, manager.Package{
			Name:        f.Name,
			Version:     f.Versions["stable"],
			Manager:     a.ID(),
			Description: f.Desc,
			Homepage:    f.Homepage,
		})
	}
	for _, c := range p.Casks {
		name := c.Token
		out = append(out, manager.Package{
			Name:        name,
			Version:     c.Version,
			Manager:     a.ID(),
			Description: c.Desc,
			Homepage:    c.Homepage,
			Extra:       map[string]string{"type": "cask"},
		})
	}
	return out, nil
}

// ---- list ------------------------------------------------------------------

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--versions"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		ver := ""
		if len(parts) > 1 {
			ver = parts[len(parts)-1]
		}
		out = append(out, manager.Package{
			Name: parts[0], Version: ver, Manager: a.ID(), Installed: true,
		})
	}
	return out, nil
}

// ---- outdated --------------------------------------------------------------

type outdatedPayload struct {
	Formulae []struct {
		Name              string   `json:"name"`
		InstalledVersions []string `json:"installed_versions"`
		CurrentVersion    string   `json:"current_version"`
	} `json:"formulae"`
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated", "--json=v2"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	var p outdatedPayload
	if err := json.Unmarshal(res.Stdout, &p); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeParseError, Err: err}
	}
	out := make([]manager.Package, 0, len(p.Formulae))
	for _, f := range p.Formulae {
		ver := ""
		if len(f.InstalledVersions) > 0 {
			ver = f.InstalledVersions[0]
		}
		out = append(out, manager.Package{
			Name: f.Name, Version: ver, Latest: f.CurrentVersion, Manager: a.ID(), Installed: true,
		})
	}
	return out, nil
}

// ---- info -------------------------------------------------------------------

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"info", "--json=v2", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeUnknown, Err: err}
	}
	var arr []struct {
		Name      string            `json:"name"`
		Desc      string            `json:"desc"`
		Homepage  string            `json:"homepage"`
		Versions  map[string]string `json:"versions"`
		Installed []struct {
			Version string `json:"version"`
		} `json:"installed"`
	}
	if err := json.Unmarshal(res.Stdout, &arr); err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeParseError, Err: err}
	}
	if len(arr) == 0 {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: fmt.Errorf("brew: not found: %s", name)}
	}
	f := arr[0]
	p := manager.Package{
		Name:        f.Name,
		Version:     f.Versions["stable"],
		Manager:     a.ID(),
		Description: f.Desc,
		Homepage:    f.Homepage,
	}
	if len(f.Installed) > 0 {
		p.Installed = true
		p.Version = f.Installed[0].Version
		p.Latest = f.Versions["stable"]
	}
	return p, nil
}

// ---- mutations -------------------------------------------------------------

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpInstall, append([]string{"install"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpUninstall, append([]string{"uninstall"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade"}
	if len(names) > 0 {
		args = append(args, names...)
	}
	return a.exec(ctx, manager.OpUpdate, args...)
}
func (a *Adapter) exec(ctx context.Context, op manager.Op, args ...string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
```

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/manager/brew/... -v
```
Expected: 4 tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/manager/brew/
git commit -m "feat(brew): adapter with search/list/info/outdated/install/remove/update + fixtures"
```

---

### Task 3: npm adapter

**Files:**
- Create: `internal/manager/npm/npm.go`
- Create: `internal/manager/npm/npm_test.go`
- Create: `internal/manager/npm/testdata/search.json`
- Create: `internal/manager/npm/testdata/list_global.json`
- Create: `internal/manager/npm/testdata/outdated.json`

- [ ] **Step 1: Drop fixtures**

`testdata/search.json` (npm search emits an array):
```json
[
  {"name":"react","version":"18.3.1","description":"React is a JavaScript library","links":{"homepage":"https://react.dev"}},
  {"name":"react-dom","version":"18.3.1","description":"React package for DOM","links":{"homepage":"https://react.dev"}}
]
```

`testdata/list_global.json`:
```json
{
  "dependencies": {
    "typescript": {"version": "5.4.2"},
    "prettier":   {"version": "3.2.5"}
  }
}
```

`testdata/outdated.json`:
```json
{
  "typescript": {"current": "5.4.2", "wanted": "5.5.4", "latest": "5.5.4", "location": "global"}
}
```

- [ ] **Step 2: Write failing test `internal/manager/npm/npm_test.go`**

```go
package npm

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)

func fix(t *testing.T, name string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil { t.Fatal(err) }
	return b
}

func TestSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"npm search --json react": {Stdout: fix(t, "search.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "npm"}
	got, err := a.Search(context.Background(), "react")
	if err != nil { t.Fatal(err) }
	if len(got) != 2 || got[0].Name != "react" || got[0].Version != "18.3.1" {
		t.Fatalf("got %+v", got)
	}
}

func TestListGlobal(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"npm list -g --depth=0 --json": {Stdout: fix(t, "list_global.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "npm"}
	got, err := a.List(context.Background())
	if err != nil { t.Fatal(err) }
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
}

func TestOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"npm outdated -g --json": {Stdout: fix(t, "outdated.json"), Code: 1}, // npm exits non-zero when outdated exist
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "npm"}
	got, err := a.Outdated(context.Background())
	if err != nil { t.Fatal(err) }
	if len(got) != 1 || got[0].Name != "typescript" || got[0].Latest != "5.5.4" {
		t.Fatalf("got %+v", got)
	}
}
```

Note: npm exits 1 when outdated packages exist — adapter must tolerate that exit code (treat as success when stdout parses).

- [ ] **Step 3: Run, expect FAIL**

```bash
go test ./internal/manager/npm/...
```

- [ ] **Step 4: Implement `internal/manager/npm/npm.go`**

```go
// Package npm wraps the npm CLI for global packages.
package npm

import (
	"context"
	"encoding/json"
	"os/exec"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "npm"} }

func (a *Adapter) ID() string                 { return "npm" }
func (a *Adapter) DisplayName() string        { return "npm" }
func (a *Adapter) OSes() []manager.OS         { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope       { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool  { return false }
func (a *Adapter) Detect() bool {
	_, err := exec.LookPath(a.Bin)
	return err == nil
}

type searchEntry struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Links       struct {
		Homepage string `json:"homepage"`
	} `json:"links"`
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", "--json", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var entries []searchEntry
	if err := json.Unmarshal(res.Stdout, &entries); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	out := make([]manager.Package, 0, len(entries))
	for _, e := range entries {
		out = append(out, manager.Package{
			Name: e.Name, Version: e.Version, Description: e.Description,
			Homepage: e.Links.Homepage, Manager: a.ID(),
		})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "-g", "--depth=0", "--json"}})
	if err != nil {
		// npm list -g can exit 1 with stdout still valid (peer-dep warnings).
		if len(res.Stdout) == 0 {
			return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
		}
	}
	var body struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(res.Stdout, &body); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err}
	}
	out := make([]manager.Package, 0, len(body.Dependencies))
	for name, d := range body.Dependencies {
		out = append(out, manager.Package{
			Name: name, Version: d.Version, Manager: a.ID(), Installed: true,
		})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated", "-g", "--json"}})
	// npm exits 1 when outdated entries exist; ignore the error if stdout parses.
	if len(res.Stdout) == 0 {
		return nil, nil
	}
	var body map[string]struct {
		Current string `json:"current"`
		Latest  string `json:"latest"`
	}
	if err := json.Unmarshal(res.Stdout, &body); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for name, d := range body {
		out = append(out, manager.Package{
			Name: name, Version: d.Current, Latest: d.Latest, Manager: a.ID(), Installed: true,
		})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"view", name, "--json"}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	var v struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Homepage    string `json:"homepage"`
	}
	if err := json.Unmarshal(res.Stdout, &v); err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeParseError, Err: err}
	}
	return manager.Package{
		Name: v.Name, Version: v.Version, Description: v.Description,
		Homepage: v.Homepage, Manager: a.ID(),
	}, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install", "-g"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall", "-g"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update", "-g"}
	if len(names) > 0 { args = append(args, names...) }
	return a.run(ctx, manager.OpUpdate, args...)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args ...string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
```

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/manager/npm/... -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/manager/npm/
git commit -m "feat(npm): adapter for global packages with golden fixtures"
```

---

### Task 4: pip adapter

**Files:**
- Create: `internal/manager/pip/pip.go`
- Create: `internal/manager/pip/pip_test.go`
- Create: `internal/manager/pip/testdata/search.json` (PyPI JSON API)
- Create: `internal/manager/pip/testdata/list.json`
- Create: `internal/manager/pip/testdata/outdated.json`

Note: `pip search` was disabled by PyPI; adapter uses the PyPI HTTP JSON API for search instead. Tests inject the HTTP client via a transport.

- [ ] **Step 1: Drop fixtures**

`testdata/search.json` (truncated PyPI shape for one package):
```json
{
  "info": {"name": "requests", "version": "2.32.3", "summary": "Python HTTP for Humans", "home_page": "https://requests.readthedocs.io/"},
  "releases": {}
}
```

`testdata/list.json`:
```json
[
  {"name": "requests", "version": "2.32.3"},
  {"name": "rich",     "version": "13.7.1"}
]
```

`testdata/outdated.json`:
```json
[
  {"name": "rich", "version": "13.6.0", "latest_version": "13.7.1", "latest_filetype": "wheel"}
]
```

- [ ] **Step 2: Write failing test `internal/manager/pip/pip_test.go`**

```go
package pip

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)

type fakeRT struct{ body []byte }
func (f *fakeRT) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(string(f.body))),
		Header:     make(http.Header),
	}, nil
}

func loadFix(t *testing.T, name string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil { t.Fatal(err) }
	return b
}

func TestSearchViaPyPI(t *testing.T) {
	a := &Adapter{
		Runner: &runner.Runner{Exec: (&runner.Fake{}).Exec},
		HTTP:   &http.Client{Transport: &fakeRT{body: loadFix(t, "search.json")}},
		Bin:    "pip",
	}
	got, err := a.Search(context.Background(), "requests")
	if err != nil { t.Fatal(err) }
	if len(got) != 1 || got[0].Name != "requests" || got[0].Version != "2.32.3" {
		t.Fatalf("got %+v", got)
	}
}

func TestList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pip list --format json": {Stdout: loadFix(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pip"}
	got, err := a.List(context.Background())
	if err != nil { t.Fatal(err) }
	if len(got) != 2 { t.Fatalf("len=%d", len(got)) }
}

func TestOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pip list --outdated --format json": {Stdout: loadFix(t, "outdated.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pip"}
	got, err := a.Outdated(context.Background())
	if err != nil { t.Fatal(err) }
	if len(got) != 1 || got[0].Latest != "13.7.1" {
		t.Fatalf("got %+v", got)
	}
}
```

- [ ] **Step 3: Run, expect FAIL**

```bash
go test ./internal/manager/pip/...
```

- [ ] **Step 4: Implement `internal/manager/pip/pip.go`**

```go
// Package pip wraps pip + PyPI for search.
package pip

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	HTTP   *http.Client
	Bin    string
}

func New(r *runner.Runner) *Adapter {
	return &Adapter{Runner: r, HTTP: http.DefaultClient, Bin: "pip"}
}

func (a *Adapter) ID() string                { return "pip" }
func (a *Adapter) DisplayName() string       { return "pip" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool {
	_, err := exec.LookPath(a.Bin)
	return err == nil
}

// pip search was deprecated by PyPI. We hit the JSON API directly.
// Exact-match endpoint: https://pypi.org/pypi/<name>/json
func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", q)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeNetworkFailure, Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, nil
	}
	if resp.StatusCode >= 400 {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeNetworkFailure, Err: fmt.Errorf("pypi %d", resp.StatusCode)}
	}
	body, _ := io.ReadAll(resp.Body)
	var p struct {
		Info struct {
			Name     string `json:"name"`
			Version  string `json:"version"`
			Summary  string `json:"summary"`
			HomePage string `json:"home_page"`
		} `json:"info"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	return []manager.Package{{
		Name: p.Info.Name, Version: p.Info.Version, Description: p.Info.Summary,
		Homepage: p.Info.HomePage, Manager: a.ID(),
	}}, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--format", "json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var arr []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(res.Stdout, &arr); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err}
	}
	out := make([]manager.Package, 0, len(arr))
	for _, p := range arr {
		out = append(out, manager.Package{Name: p.Name, Version: p.Version, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--outdated", "--format", "json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	var arr []struct {
		Name          string `json:"name"`
		Version       string `json:"version"`
		LatestVersion string `json:"latest_version"`
	}
	if err := json.Unmarshal(res.Stdout, &arr); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeParseError, Err: err}
	}
	out := make([]manager.Package, 0, len(arr))
	for _, p := range arr {
		out = append(out, manager.Package{Name: p.Name, Version: p.Version, Latest: p.LatestVersion, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	pkgs, err := a.Search(ctx, name)
	if err != nil || len(pkgs) == 0 {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	return pkgs[0], nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install", "--user"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall", "-y"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"install", "--user", "--upgrade"}
	if len(names) == 0 {
		// pip has no "update all"; require explicit names. Surface friendly error.
		return &manager.Error{Manager: a.ID(), Op: manager.OpUpdate, Code: manager.CodeConflict,
			Err: fmt.Errorf("pip update requires explicit package names")}
	}
	args = append(args, names...)
	return a.run(ctx, manager.OpUpdate, args...)
}

func (a *Adapter) run(ctx context.Context, op manager.Op, args ...string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
```

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/manager/pip/... -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/manager/pip/
git commit -m "feat(pip): adapter using pip CLI + PyPI JSON API for search"
```

---

### Task 5: Orchestrator (`internal/orchestrator`)

**Files:**
- Create: `internal/orchestrator/orchestrator.go`
- Create: `internal/orchestrator/orchestrator_test.go`

- [ ] **Step 1: Write failing test `internal/orchestrator/orchestrator_test.go`**

```go
package orchestrator

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

type fakeMgr struct {
	id   string
	pkgs []manager.Package
	err  error
}

func (f *fakeMgr) ID() string                                                  { return f.id }
func (f *fakeMgr) DisplayName() string                                         { return f.id }
func (f *fakeMgr) OSes() []manager.OS                                          { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (f *fakeMgr) Detect() bool                                                { return true }
func (f *fakeMgr) NeedsSudo(manager.Op) bool                                   { return false }
func (f *fakeMgr) Scope() manager.Scope                                        { return manager.ScopeUserGlobal }
func (f *fakeMgr) List(context.Context) ([]manager.Package, error)             { return f.pkgs, f.err }
func (f *fakeMgr) Outdated(context.Context) ([]manager.Package, error)         { return f.pkgs, f.err }
func (f *fakeMgr) Search(context.Context, string) ([]manager.Package, error)   { return f.pkgs, f.err }
func (f *fakeMgr) Info(context.Context, string) (manager.Package, error)       { return manager.Package{}, f.err }
func (f *fakeMgr) Install(context.Context, ...string) error                    { return f.err }
func (f *fakeMgr) Uninstall(context.Context, ...string) error                  { return f.err }
func (f *fakeMgr) Update(context.Context, ...string) error                     { return f.err }

func TestSearchFansOutAndMergesResults(t *testing.T) {
	mgrs := []manager.Manager{
		&fakeMgr{id: "brew", pkgs: []manager.Package{{Name: "ripgrep", Manager: "brew", Version: "14.1.0"}}},
		&fakeMgr{id: "cargo", pkgs: []manager.Package{{Name: "ripgrep", Manager: "cargo", Version: "14.0.0"}}},
	}
	o := New(Ranking{Preferred: []string{"brew", "cargo"}})
	res, errs := o.Search(context.Background(), mgrs, "ripgrep")
	if len(errs) != 0 {
		t.Fatalf("errors: %v", errs)
	}
	if len(res) != 2 {
		t.Fatalf("len = %d", len(res))
	}
	// brew should rank first (preferred order).
	sort.Slice(res, func(i, j int) bool { return res[i].Rank < res[j].Rank })
	if res[0].Pkg.Manager != "brew" {
		t.Fatalf("first = %s", res[0].Pkg.Manager)
	}
}

func TestSearchCollectsPartialErrors(t *testing.T) {
	mgrs := []manager.Manager{
		&fakeMgr{id: "ok", pkgs: []manager.Package{{Name: "x", Manager: "ok"}}},
		&fakeMgr{id: "boom", err: errors.New("kaboom")},
	}
	o := New(Ranking{})
	res, errs := o.Search(context.Background(), mgrs, "x")
	if len(res) != 1 {
		t.Fatalf("len res = %d", len(res))
	}
	if len(errs) != 1 {
		t.Fatalf("len errs = %d", len(errs))
	}
}
```

- [ ] **Step 2: Run, expect FAIL**

```bash
go test ./internal/orchestrator/...
```

- [ ] **Step 3: Implement `internal/orchestrator/orchestrator.go`**

```go
// Package orchestrator fans out aggregate ops across multiple managers
// and merges/ranks the results.
package orchestrator

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

type Ranking struct {
	Preferred []string
}

type Result struct {
	Pkg  manager.Package
	Rank int // lower is better
}

type Orchestrator struct {
	rank Ranking
}

func New(r Ranking) *Orchestrator { return &Orchestrator{rank: r} }

func (o *Orchestrator) rankFor(pmID, query string, name string) int {
	r := 1000
	for i, p := range o.rank.Preferred {
		if p == pmID {
			r = i
			break
		}
	}
	if strings.EqualFold(name, query) {
		r -= 500 // exact match boost
	}
	return r
}

func (o *Orchestrator) Search(ctx context.Context, mgrs []manager.Manager, q string) ([]Result, []error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var out []Result
	var errs []error

	for _, m := range mgrs {
		wg.Add(1)
		go func(m manager.Manager) {
			defer wg.Done()
			pkgs, err := m.Search(ctx, q)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			for _, p := range pkgs {
				out = append(out, Result{Pkg: p, Rank: o.rankFor(m.ID(), q, p.Name)})
			}
		}(m)
	}
	wg.Wait()

	sort.Slice(out, func(i, j int) bool {
		if out[i].Rank != out[j].Rank {
			return out[i].Rank < out[j].Rank
		}
		return out[i].Pkg.Name < out[j].Pkg.Name
	})
	return out, errs
}

// List fans out List across all managers; results carry their Manager field set.
func (o *Orchestrator) List(ctx context.Context, mgrs []manager.Manager) ([]manager.Package, []error) {
	return fanOutPkgs(ctx, mgrs, func(m manager.Manager) ([]manager.Package, error) { return m.List(ctx) })
}

func (o *Orchestrator) Outdated(ctx context.Context, mgrs []manager.Manager) ([]manager.Package, []error) {
	return fanOutPkgs(ctx, mgrs, func(m manager.Manager) ([]manager.Package, error) { return m.Outdated(ctx) })
}

func fanOutPkgs(ctx context.Context, mgrs []manager.Manager, fn func(manager.Manager) ([]manager.Package, error)) ([]manager.Package, []error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var pkgs []manager.Package
	var errs []error
	for _, m := range mgrs {
		wg.Add(1)
		go func(m manager.Manager) {
			defer wg.Done()
			p, err := fn(m)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			pkgs = append(pkgs, p...)
		}(m)
	}
	wg.Wait()
	return pkgs, errs
}
```

- [ ] **Step 4: Run tests, expect PASS**

```bash
go test ./internal/orchestrator/... -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/
git commit -m "feat(orchestrator): parallel fan-out + rank-aware merge"
```

---

### Task 6: Formatters (`internal/format`)

**Files:**
- Create: `internal/format/human.go`
- Create: `internal/format/json.go`
- Create: `internal/format/format_test.go`

- [ ] **Step 1: Write failing test `internal/format/format_test.go`**

```go
package format

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

func TestHumanSearchRendersTable(t *testing.T) {
	pkgs := []manager.Package{
		{Name: "ripgrep", Version: "14.1.0", Manager: "brew", Description: "Search like grep"},
		{Name: "ripgrep", Version: "14.0.0", Manager: "cargo"},
	}
	var buf bytes.Buffer
	if err := HumanSearch(&buf, pkgs); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	for _, want := range []string{"ripgrep", "14.1.0", "brew", "cargo"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestJSONSearchOutputsStableSchema(t *testing.T) {
	pkgs := []manager.Package{{Name: "x", Manager: "brew", Version: "1.0"}}
	var buf bytes.Buffer
	if err := JSONResult(&buf, pkgs, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"packages"`) || !strings.Contains(buf.String(), `"errors"`) {
		t.Fatalf("missing keys: %s", buf.String())
	}
}
```

- [ ] **Step 2: Run, expect FAIL**

```bash
go test ./internal/format/...
```

- [ ] **Step 3: Implement `internal/format/human.go`**

```go
// Package format renders package results as human tables or JSON.
package format

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

func HumanSearch(w io.Writer, pkgs []manager.Package) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tVERSION\tPM\tINSTALLED\tDESCRIPTION")
	for _, p := range pkgs {
		inst := "no"
		if p.Installed {
			inst = "yes"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", p.Name, p.Version, p.Manager, inst, p.Description)
	}
	return tw.Flush()
}

func HumanList(w io.Writer, pkgs []manager.Package) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tVERSION\tPM\tLATEST")
	for _, p := range pkgs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", p.Name, p.Version, p.Manager, p.Latest)
	}
	return tw.Flush()
}

func HumanInfo(w io.Writer, p manager.Package) error {
	fmt.Fprintf(w, "Name:        %s\n", p.Name)
	fmt.Fprintf(w, "Manager:     %s\n", p.Manager)
	fmt.Fprintf(w, "Version:     %s\n", p.Version)
	if p.Latest != "" && p.Latest != p.Version {
		fmt.Fprintf(w, "Latest:      %s\n", p.Latest)
	}
	fmt.Fprintf(w, "Installed:   %v\n", p.Installed)
	if p.Homepage != "" {
		fmt.Fprintf(w, "Homepage:    %s\n", p.Homepage)
	}
	if p.Description != "" {
		fmt.Fprintf(w, "Description: %s\n", p.Description)
	}
	return nil
}
```

- [ ] **Step 4: Implement `internal/format/json.go`**

```go
package format

import (
	"encoding/json"
	"io"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

type Envelope struct {
	Packages []manager.Package `json:"packages"`
	Errors   []errorRendered   `json:"errors"`
}

type errorRendered struct {
	Manager string `json:"manager"`
	Op      string `json:"op"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func renderErr(e error) errorRendered {
	if me, ok := e.(*manager.Error); ok {
		return errorRendered{Manager: me.Manager, Op: string(me.Op), Code: string(me.Code), Message: me.Err.Error()}
	}
	return errorRendered{Code: "unknown", Message: e.Error()}
}

func JSONResult(w io.Writer, pkgs []manager.Package, errs []error) error {
	env := Envelope{Packages: pkgs, Errors: make([]errorRendered, 0, len(errs))}
	for _, e := range errs {
		env.Errors = append(env.Errors, renderErr(e))
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}
```

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/format/... -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/format/
git commit -m "feat(format): human tabular + JSON envelope renderers"
```

---

### Task 7: App wiring (`cmd/pkgr/app.go`, `managers.go`, `root_flags.go`)

**Files:**
- Create: `cmd/pkgr/app.go`
- Create: `cmd/pkgr/managers.go`
- Create: `cmd/pkgr/root_flags.go`
- Modify: `cmd/pkgr/main.go`

- [ ] **Step 1: Implement `cmd/pkgr/app.go`** (singleton context shared across cobra commands)

```go
package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ramtinhoss/pkgr/internal/cache"
	"github.com/ramtinhoss/pkgr/internal/config"
	pkgrlog "github.com/ramtinhoss/pkgr/internal/log"
	"github.com/ramtinhoss/pkgr/internal/orchestrator"
	"github.com/ramtinhoss/pkgr/internal/registry"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type App struct {
	Cfg    config.Config
	Reg    *registry.Registry
	Orc    *orchestrator.Orchestrator
	Cache  *cache.Cache
	Run    *runner.Runner
	Log    *slog.Logger
	Closer func() error
}

func newApp(flags rootFlags) (*App, error) {
	cfgPath := flags.ConfigPath
	if cfgPath == "" {
		base, _ := os.UserConfigDir()
		cfgPath = filepath.Join(base, "pkgr", "config.toml")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	stateDir, _ := os.UserHomeDir()
	logPath := filepath.Join(stateDir, ".local", "state", "pkgr", "pkgr.log")
	l, closer, err := pkgrlog.Setup(pkgrlog.Options{Path: logPath, Verbose: flags.Verbose || cfg.General.Verbose})
	if err != nil {
		return nil, err
	}

	cacheDir, _ := os.UserCacheDir()
	c := cache.New(filepath.Join(cacheDir, "pkgr"))

	r := &runner.Runner{DryRun: flags.DryRun, Out: os.Stdout}
	reg := registry.New()
	registerAdapters(reg, r)
	enabled := make(map[string]bool)
	for id, m := range cfg.Managers {
		enabled[id] = m.Enabled
	}
	reg.SetEnabled(enabled)

	orc := orchestrator.New(orchestrator.Ranking{Preferred: cfg.Ranking.Preferred})

	return &App{Cfg: cfg, Reg: reg, Orc: orc, Cache: c, Run: r, Log: l, Closer: closer}, nil
}
```

- [ ] **Step 2: Implement `cmd/pkgr/managers.go`**

```go
package main

import (
	"github.com/ramtinhoss/pkgr/internal/manager/brew"
	"github.com/ramtinhoss/pkgr/internal/manager/npm"
	"github.com/ramtinhoss/pkgr/internal/manager/pip"
	"github.com/ramtinhoss/pkgr/internal/registry"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

// registerAdapters wires every known adapter. Each phase appends here.
func registerAdapters(reg *registry.Registry, r *runner.Runner) {
	reg.Register(brew.New(r))
	reg.Register(npm.New(r))
	reg.Register(pip.New(r))
	// phases 4 & 5 will append the rest.
}
```

- [ ] **Step 3: Implement `cmd/pkgr/root_flags.go`**

```go
package main

import "github.com/spf13/cobra"

type rootFlags struct {
	PMs        []string
	JSON       bool
	NoColor    bool
	Yes        bool
	DryRun     bool
	NoCache    bool
	Verbose    bool
	ConfigPath string
}

func bindRootFlags(cmd *cobra.Command, f *rootFlags) {
	cmd.PersistentFlags().StringSliceVar(&f.PMs, "pm", nil, "restrict to one or more PMs")
	cmd.PersistentFlags().BoolVar(&f.JSON, "json", false, "machine output")
	cmd.PersistentFlags().BoolVar(&f.NoColor, "no-color", false, "disable ANSI colors")
	cmd.PersistentFlags().BoolVarP(&f.Yes, "yes", "y", false, "auto-confirm prompts")
	cmd.PersistentFlags().BoolVar(&f.DryRun, "dry-run", false, "print commands without executing")
	cmd.PersistentFlags().BoolVar(&f.NoCache, "no-cache", false, "bypass cache")
	cmd.PersistentFlags().BoolVarP(&f.Verbose, "verbose", "v", false, "verbose logging to stderr")
	cmd.PersistentFlags().StringVar(&f.ConfigPath, "config", "", "alternative config file path")
}
```

- [ ] **Step 4: Modify `cmd/pkgr/main.go`** to bind flags + build app

Replace `newRootCmd` with:

```go
func newRootCmd(b buildInfo) *cobra.Command {
	flags := &rootFlags{}
	root := &cobra.Command{
		Use:           "pkgr",
		Short:         "Cross-platform package manager TUI/CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	bindRootFlags(root, flags)
	root.AddCommand(newVersionCmd(b))
	// subcommands added below
	addSearchCmd(root, flags)
	addListCmd(root, flags)
	addInfoCmd(root, flags)
	addInstallCmd(root, flags)
	addRemoveCmd(root, flags)
	addUpdateCmd(root, flags)
	addOutdatedCmd(root, flags)
	addPMCmd(root, flags)
	addCacheCmd(root, flags)
	addDoctorCmd(root, flags)
	addConfigCmd(root, flags)
	addCompletionCmd(root)
	return root
}
```

- [ ] **Step 5: Build (will fail until subcommands exist; stub them next task)**

```bash
go build ./...
```
Expected: errors about `addSearchCmd` etc., to be resolved by Tasks 8–14.

- [ ] **Step 6: Commit**

```bash
git add cmd/pkgr/app.go cmd/pkgr/managers.go cmd/pkgr/root_flags.go cmd/pkgr/main.go
git commit -m "feat(cli): app wiring, root flags, adapter registration"
```

---

### Task 8: `pkgr search` subcommand

**Files:**
- Create: `cmd/pkgr/search.go`
- Create: `cmd/pkgr/search_test.go`

- [ ] **Step 1: Write failing test `cmd/pkgr/search_test.go`**

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSearchCommandDryRun(t *testing.T) {
	var out bytes.Buffer
	root := newRootCmd(buildInfo{Version: "test"})
	root.SetOut(&out); root.SetErr(&out)
	root.SetArgs([]string{"--dry-run", "search", "ripgrep"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "would exec:") && !strings.Contains(out.String(), "NAME") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
```

- [ ] **Step 2: Implement `cmd/pkgr/search.go`**

```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
)

func addSearchCmd(root *cobra.Command, flags *rootFlags) {
	var limit int
	var installedOnly bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search packages across all detected PMs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()

			mgrs := app.Reg.Active()
			if len(flags.PMs) > 0 {
				mgrs = filterPMs(mgrs, flags.PMs)
			}
			results, errs := app.Orc.Search(cmd.Context(), mgrs, args[0])

			pkgs := make([]packagePtr(0), 0, len(results))
			for _, r := range results {
				if installedOnly && !r.Pkg.Installed { continue }
				pkgs = append(pkgs, r.Pkg)
				if limit > 0 && len(pkgs) >= limit { break }
			}

			if flags.JSON {
				return format.JSONResult(cmd.OutOrStdout(), pkgs, errs)
			}
			return format.HumanSearch(cmd.OutOrStdout(), pkgs)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	cmd.Flags().BoolVar(&installedOnly, "installed-only", false, "only show installed pkgs")
	root.AddCommand(cmd)
}

// packagePtr is a tiny alias to keep the loop above terse without an extra import alias.
type packagePtr = struct{}
```

Note: replace the `packagePtr` alias block with proper `manager.Package` slice. Corrected version:

```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
	"github.com/ramtinhoss/pkgr/internal/manager"
)

func addSearchCmd(root *cobra.Command, flags *rootFlags) {
	var limit int
	var installedOnly bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search packages across all detected PMs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()

			mgrs := app.Reg.Active()
			if len(flags.PMs) > 0 {
				mgrs = filterPMs(mgrs, flags.PMs)
			}
			results, errs := app.Orc.Search(cmd.Context(), mgrs, args[0])

			pkgs := make([]manager.Package, 0, len(results))
			for _, r := range results {
				if installedOnly && !r.Pkg.Installed { continue }
				pkgs = append(pkgs, r.Pkg)
				if limit > 0 && len(pkgs) >= limit { break }
			}

			if flags.JSON {
				return format.JSONResult(cmd.OutOrStdout(), pkgs, errs)
			}
			return format.HumanSearch(cmd.OutOrStdout(), pkgs)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	cmd.Flags().BoolVar(&installedOnly, "installed-only", false, "only show installed pkgs")
	root.AddCommand(cmd)
}

func filterPMs(all []manager.Manager, ids []string) []manager.Manager {
	allow := make(map[string]bool, len(ids))
	for _, id := range ids { allow[id] = true }
	out := make([]manager.Manager, 0, len(all))
	for _, m := range all {
		if allow[m.ID()] { out = append(out, m) }
	}
	return out
}
```

Replace the file with the corrected version above.

- [ ] **Step 3: Run tests, expect PASS**

```bash
go test ./cmd/pkgr/... -v
```

- [ ] **Step 4: Commit**

```bash
git add cmd/pkgr/search.go cmd/pkgr/search_test.go
git commit -m "feat(cli): search subcommand using orchestrator + format"
```

---

### Task 9: `pkgr list` and `pkgr outdated` subcommands

**Files:**
- Create: `cmd/pkgr/list.go`
- Create: `cmd/pkgr/outdated.go`

- [ ] **Step 1: Implement `cmd/pkgr/list.go`**

```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
)

func addListCmd(root *cobra.Command, flags *rootFlags) {
	var outdated bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed packages across PMs",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()

			mgrs := app.Reg.Active()
			if len(flags.PMs) > 0 { mgrs = filterPMs(mgrs, flags.PMs) }

			var pkgs, errs2 = func() (any, any) { return nil, nil }()
			_ = pkgs; _ = errs2

			if outdated {
				p, errs := app.Orc.Outdated(cmd.Context(), mgrs)
				if flags.JSON { return format.JSONResult(cmd.OutOrStdout(), p, errs) }
				return format.HumanList(cmd.OutOrStdout(), p)
			}
			p, errs := app.Orc.List(cmd.Context(), mgrs)
			if flags.JSON { return format.JSONResult(cmd.OutOrStdout(), p, errs) }
			return format.HumanList(cmd.OutOrStdout(), p)
		},
	}
	cmd.Flags().BoolVar(&outdated, "outdated", false, "show only outdated packages")
	root.AddCommand(cmd)
}
```

Remove the dead `var pkgs, errs2 = ...` block; cleaned version:

```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
)

func addListCmd(root *cobra.Command, flags *rootFlags) {
	var outdated bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed packages across PMs",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()

			mgrs := app.Reg.Active()
			if len(flags.PMs) > 0 { mgrs = filterPMs(mgrs, flags.PMs) }

			if outdated {
				p, errs := app.Orc.Outdated(cmd.Context(), mgrs)
				if flags.JSON { return format.JSONResult(cmd.OutOrStdout(), p, errs) }
				return format.HumanList(cmd.OutOrStdout(), p)
			}
			p, errs := app.Orc.List(cmd.Context(), mgrs)
			if flags.JSON { return format.JSONResult(cmd.OutOrStdout(), p, errs) }
			return format.HumanList(cmd.OutOrStdout(), p)
		},
	}
	cmd.Flags().BoolVar(&outdated, "outdated", false, "show only outdated packages")
	root.AddCommand(cmd)
}
```

- [ ] **Step 2: Implement `cmd/pkgr/outdated.go`** (alias for `list --outdated`)

```go
package main

import "github.com/spf13/cobra"

func addOutdatedCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "Show outdated packages across PMs (alias for 'list --outdated')",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Re-dispatch through list cmd with the flag set.
			root.SetArgs(append([]string{"list", "--outdated"}, args...))
			return root.Execute()
		},
	}
	root.AddCommand(cmd)
}
```

- [ ] **Step 3: Build + smoke test**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add cmd/pkgr/list.go cmd/pkgr/outdated.go
git commit -m "feat(cli): list (+ --outdated) and outdated subcommands"
```

---

### Task 10: `pkgr info` subcommand

**Files:**
- Create: `cmd/pkgr/info.go`

- [ ] **Step 1: Implement `cmd/pkgr/info.go`**

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
	"github.com/ramtinhoss/pkgr/internal/spec"
)

func addInfoCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "info <spec>",
		Short: "Show details for a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()

			s, err := spec.Parse(args[0])
			if err != nil { return err }

			mgrs := app.Reg.Active()
			if s.PM != "" {
				m, ok := app.Reg.Get(s.PM)
				if !ok { return fmt.Errorf("unknown pm: %s", s.PM) }
				mgrs = []manager.Manager{m}
			} else if len(flags.PMs) > 0 {
				mgrs = filterPMs(mgrs, flags.PMs)
			}

			for _, m := range mgrs {
				p, err := m.Info(cmd.Context(), s.Name)
				if err == nil {
					if flags.JSON {
						return format.JSONResult(cmd.OutOrStdout(), []manager.Package{p}, nil)
					}
					return format.HumanInfo(cmd.OutOrStdout(), p)
				}
			}
			return fmt.Errorf("not found: %s", s.Name)
		},
	}
	root.AddCommand(cmd)
}
```

Add missing import `"github.com/ramtinhoss/pkgr/internal/manager"` at top of file.

Final imports block:

```go
import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/spec"
)
```

- [ ] **Step 2: Build**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add cmd/pkgr/info.go
git commit -m "feat(cli): info subcommand with PM-pinning via spec"
```

---

### Task 11: `pkgr install`, `remove`, `update` subcommands

**Files:**
- Create: `cmd/pkgr/install.go`
- Create: `cmd/pkgr/remove.go`
- Create: `cmd/pkgr/update.go`

- [ ] **Step 1: Implement `cmd/pkgr/install.go`**

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/spec"
)

func addInstallCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "install <spec>...",
		Short: "Install one or more packages",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()

			byPM := make(map[string][]string)
			for _, s := range args {
				parsed, err := spec.Parse(s)
				if err != nil { return err }
				pm := parsed.PM
				if pm == "" {
					pm, err = resolvePM(app, parsed.Name, flags.Yes)
					if err != nil { return err }
				}
				byPM[pm] = append(byPM[pm], parsed.Name)
			}

			for pm, names := range byPM {
				m, ok := app.Reg.Get(pm)
				if !ok { return fmt.Errorf("unknown pm: %s", pm) }
				if err := m.Install(cmd.Context(), names...); err != nil {
					return err
				}
				_ = app.Cache.Invalidate(pm + "/installed")
				_ = app.Cache.Invalidate(pm + "/outdated")
			}
			return nil
		},
	}
	root.AddCommand(cmd)
}

// resolvePM picks the PM for a bare name. If multiple PMs return a hit,
// prompt the user unless yes=true, in which case pick by ranking.preferred.
func resolvePM(app *App, name string, yes bool) (string, error) {
	type cand struct{ m manager.Manager }
	var cands []cand
	for _, m := range app.Reg.Active() {
		if pkgs, err := m.Search(nil, name); err == nil && len(pkgs) > 0 {
			cands = append(cands, cand{m: m})
		}
	}
	if len(cands) == 0 {
		return "", fmt.Errorf("no PM has %q", name)
	}
	if len(cands) == 1 || yes {
		// pick first by ranking.preferred order
		for _, p := range app.Cfg.Ranking.Preferred {
			for _, c := range cands {
				if c.m.ID() == p { return p, nil }
			}
		}
		return cands[0].m.ID(), nil
	}
	// interactive prompt
	fmt.Printf("Package %q exists in multiple PMs:\n", name)
	for i, c := range cands {
		fmt.Printf("  %d) %s\n", i+1, c.m.ID())
	}
	fmt.Print("Pick [1]: ")
	var pick int
	if _, err := fmt.Scanln(&pick); err != nil || pick < 1 || pick > len(cands) {
		pick = 1
	}
	return cands[pick-1].m.ID(), nil
}
```

- [ ] **Step 2: Implement `cmd/pkgr/remove.go`**

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/spec"
)

func addRemoveCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:     "remove <spec>...",
		Aliases: []string{"uninstall", "rm"},
		Short:   "Uninstall one or more packages",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()

			byPM := make(map[string][]string)
			for _, s := range args {
				parsed, err := spec.Parse(s)
				if err != nil { return err }
				if parsed.PM == "" {
					return fmt.Errorf("remove requires explicit @pm: %q", s)
				}
				byPM[parsed.PM] = append(byPM[parsed.PM], parsed.Name)
			}
			for pm, names := range byPM {
				m, ok := app.Reg.Get(pm)
				if !ok { return fmt.Errorf("unknown pm: %s", pm) }
				if err := m.Uninstall(cmd.Context(), names...); err != nil {
					return err
				}
				_ = app.Cache.Invalidate(pm + "/installed")
				_ = app.Cache.Invalidate(pm + "/outdated")
			}
			return nil
		},
	}
	root.AddCommand(cmd)
}
```

- [ ] **Step 3: Implement `cmd/pkgr/update.go`**

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/spec"
)

func addUpdateCmd(root *cobra.Command, flags *rootFlags) {
	var all bool
	cmd := &cobra.Command{
		Use:   "update [spec]...",
		Short: "Update packages. No args + --all updates everything.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()

			if len(args) == 0 {
				if !all { return fmt.Errorf("no specs given; use --all to update everything") }
				for _, m := range app.Reg.Active() {
					if err := m.Update(cmd.Context()); err != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), err)
					}
					_ = app.Cache.Invalidate(m.ID() + "/installed")
					_ = app.Cache.Invalidate(m.ID() + "/outdated")
				}
				return nil
			}

			byPM := make(map[string][]string)
			for _, s := range args {
				parsed, err := spec.Parse(s)
				if err != nil { return err }
				if parsed.PM == "" { return fmt.Errorf("update requires explicit @pm: %q", s) }
				byPM[parsed.PM] = append(byPM[parsed.PM], parsed.Name)
			}
			for pm, names := range byPM {
				m, ok := app.Reg.Get(pm)
				if !ok { return fmt.Errorf("unknown pm: %s", pm) }
				if err := m.Update(cmd.Context(), names...); err != nil { return err }
				_ = app.Cache.Invalidate(pm + "/installed")
				_ = app.Cache.Invalidate(pm + "/outdated")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "update everything across all PMs")
	root.AddCommand(cmd)
}
```

- [ ] **Step 4: Build + smoke**

```bash
go build ./...
./pkgr --dry-run install ripgrep@brew
./pkgr --dry-run remove ripgrep@brew
./pkgr --dry-run update --all
```
Expected: dry-run prints `→ would exec: brew install ripgrep` etc.

- [ ] **Step 5: Commit**

```bash
git add cmd/pkgr/install.go cmd/pkgr/remove.go cmd/pkgr/update.go
git commit -m "feat(cli): install/remove/update with PM resolution + cache invalidation"
```

---

### Task 12: `pkgr pm`, `pkgr cache`, `pkgr doctor`, `pkgr config`, `pkgr completion`

**Files:**
- Create: `cmd/pkgr/pm.go`
- Create: `cmd/pkgr/cache.go`
- Create: `cmd/pkgr/doctor.go`
- Create: `cmd/pkgr/config.go`
- Create: `cmd/pkgr/completion.go`

- [ ] **Step 1: Implement `cmd/pkgr/pm.go`**

```go
package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func addPMCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "pm",
		Short: "Manage package-manager adapters",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List adapters",
		RunE: func(c *cobra.Command, _ []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()
			tw := tabwriter.NewWriter(c.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tDETECTED\tENABLED\tSCOPE")
			for _, m := range app.Reg.All() {
				en := "yes"
				if v, ok := app.Cfg.Managers[m.ID()]; ok && !v.Enabled { en = "no" }
				fmt.Fprintf(tw, "%s\t%v\t%s\t%s\n", m.ID(), m.Detect(), en, m.Scope())
			}
			return tw.Flush()
		},
	})
	root.AddCommand(cmd)
}
```

- [ ] **Step 2: Implement `cmd/pkgr/cache.go`**

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func addCacheCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{Use: "cache", Short: "Manage local cache"}
	cmd.AddCommand(&cobra.Command{
		Use:   "clear [pm]",
		Short: "Clear cache (all PMs or one)",
		RunE: func(c *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()
			target := app.Cache.Root
			if len(args) == 1 {
				target = filepath.Join(app.Cache.Root, args[0])
			}
			fmt.Fprintf(c.OutOrStdout(), "removing %s\n", target)
			return os.RemoveAll(target)
		},
	})
	root.AddCommand(cmd)
}
```

- [ ] **Step 3: Implement `cmd/pkgr/doctor.go`**

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func addDoctorCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose adapter health",
		RunE: func(c *cobra.Command, _ []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()
			for _, m := range app.Reg.All() {
				status := "ok"
				if !m.Detect() { status = "binary not found" }
				fmt.Fprintf(c.OutOrStdout(), "%-10s %s\n", m.ID(), status)
			}
			return nil
		},
	}
	root.AddCommand(cmd)
}
```

- [ ] **Step 4: Implement `cmd/pkgr/config.go`**

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func addConfigCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{Use: "config", Short: "Inspect or edit config"}

	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print config path",
		RunE: func(c *cobra.Command, _ []string) error {
			base, _ := os.UserConfigDir()
			fmt.Fprintln(c.OutOrStdout(), filepath.Join(base, "pkgr", "config.toml"))
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Print effective config",
		RunE: func(c *cobra.Command, _ []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()
			fmt.Fprintf(c.OutOrStdout(), "%+v\n", app.Cfg)
			return nil
		},
	})
	root.AddCommand(cmd)
}
```

- [ ] **Step 5: Implement `cmd/pkgr/completion.go`**

```go
package main

import (
	"os"

	"github.com/spf13/cobra"
)

func addCompletionCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:                   "completion <bash|zsh|fish|powershell>",
		Short:                 "Generate shell completion script",
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(_ *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(os.Stdout)
			case "zsh":
				return root.GenZshCompletion(os.Stdout)
			case "fish":
				return root.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return root.GenPowerShellCompletion(os.Stdout)
			}
			return nil
		},
	}
	root.AddCommand(cmd)
}
```

- [ ] **Step 6: Build + smoke**

```bash
go build ./...
./pkgr pm list
./pkgr doctor
./pkgr config path
./pkgr completion bash | head
```
Expected: each prints without error.

- [ ] **Step 7: Commit**

```bash
git add cmd/pkgr/{pm,cache,doctor,config,completion}.go
git commit -m "feat(cli): pm/cache/doctor/config/completion subcommands"
```

---

## Phase 2 Acceptance

- `pkgr search ripgrep` returns merged results from brew + npm + pip (where applicable)
- `pkgr list` shows installed across all detected PMs
- `pkgr install ripgrep@brew` shells out to brew (dry-runnable)
- `pkgr pm list`, `pkgr doctor`, `pkgr cache clear` all work
- All adapters covered by golden-fixture unit tests; `make test` green
- `make lint` green
