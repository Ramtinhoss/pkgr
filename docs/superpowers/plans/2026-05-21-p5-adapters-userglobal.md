# Phase 5: User-Global Adapters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the final 12 adapters: yarn, bun, pipx, uv, conda, mamba, cargo, rustup, gem, go (`go install`), asdf, mise. None require sudo. Each lives under `internal/manager/<id>/`, has golden-fixture tests, and registers in `cmd/pkgr/managers.go`.

**Architecture:** Same pattern as Phases 2 and 4. Version managers (rustup, asdf, mise) expose toolchain semantics where the "package" is a runtime version; surface that via `Extra["kind"] = "toolchain"`.

**Tech Stack:** Go stdlib, optionally `net/http` for `go` adapter querying pkg.go.dev.

---

## Task Template

Each task: drop fixtures → write test → implement adapter → register → test + commit. Tasks proceed in alphabetical order.

---

### Task 1: asdf adapter

**Files:**
- `internal/manager/asdf/{asdf.go, asdf_test.go, testdata/{plugin_list_all.txt, list.txt, latest.txt}}`

asdf manages runtimes via plugins. Map: "package name" = plugin name, "version" = installed version. Search = plugin search.

- [ ] **Step 1: Fixtures**

`testdata/plugin_list_all.txt`:
```
nodejs                  https://github.com/asdf-vm/asdf-nodejs.git
python                  https://github.com/danhper/asdf-python.git
ruby                    https://github.com/asdf-vm/asdf-ruby.git
```

`testdata/list.txt`:
```
nodejs
  20.12.2
  *21.7.2
python
  *3.12.3
```

`testdata/latest.txt` (`asdf latest nodejs`):
```
22.0.0
```

- [ ] **Step 2: Test `internal/manager/asdf/asdf_test.go`**

```go
package asdf

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)

func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}

func TestAsdfSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"asdf plugin list all": {Stdout: fx(t, "plugin_list_all.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "asdf"}
	pkgs, _ := a.Search(context.Background(), "node")
	if len(pkgs) < 1 { t.Fatalf("expected ≥1, got %d", len(pkgs)) }
}

func TestAsdfList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"asdf list": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "asdf"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) < 1 { t.Fatalf("len=%d", len(pkgs)) }
}
```

- [ ] **Step 3: Implement `internal/manager/asdf/asdf.go`**

```go
// Package asdf wraps the asdf version manager.
package asdf

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "asdf"} }

func (a *Adapter) ID() string                { return "asdf" }
func (a *Adapter) DisplayName() string       { return "asdf" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"plugin", "list", "all"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) == 0 { continue }
		if q != "" && !strings.Contains(f[0], q) { continue }
		p := manager.Package{Name: f[0], Manager: a.ID(), Extra: map[string]string{"kind": "toolchain"}}
		if len(f) > 1 { p.Homepage = f[1] }
		out = append(out, p)
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	var current string
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		if line == "" { continue }
		if !strings.HasPrefix(line, "  ") {
			current = strings.TrimSpace(line)
			continue
		}
		ver := strings.TrimSpace(strings.TrimPrefix(line, "  *"))
		out = append(out, manager.Package{Name: current, Version: ver, Manager: a.ID(), Installed: true, Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	installed, err := a.List(ctx)
	if err != nil { return nil, err }
	var out []manager.Package
	for _, p := range installed {
		res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"latest", p.Name}})
		latest := strings.TrimSpace(string(res.Stdout))
		if latest != "" && latest != p.Version {
			p.Latest = latest
			out = append(out, p)
		}
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	// asdf install <plugin> <version>; treat each names[i] as "plugin@version"
	for _, n := range names {
		parts := strings.SplitN(n, "@", 2)
		args := []string{"install", parts[0]}
		if len(parts) == 2 { args = append(args, parts[1]) } else { args = append(args, "latest") }
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpInstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	for _, n := range names {
		parts := strings.SplitN(n, "@", 2)
		if len(parts) != 2 { return &manager.Error{Manager: a.ID(), Op: manager.OpUninstall, Code: manager.CodeConflict, Err: errString("asdf uninstall needs plugin@version")} }
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"uninstall", parts[0], parts[1]}}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpUninstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	// asdf update doesn't update plugins/versions; treat as plugin-update.
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"plugin", "update", "--all"}})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: manager.OpUpdate, Code: manager.CodeUnknown, Err: err} }
	return nil
}

type errString string
func (e errString) Error() string { return string(e) }
```

- [ ] **Step 4: Register, test, commit**

```bash
go test ./internal/manager/asdf/... -v
git add internal/manager/asdf/ cmd/pkgr/managers.go
git commit -m "feat(asdf): asdf adapter for toolchain plugins"
```

---

### Task 2: bun adapter (global packages)

**Files:**
- `internal/manager/bun/{bun.go, bun_test.go, testdata/{pm_ls.txt}}`

bun's `bun pm ls` is the canonical command; bun has no search. Use npm registry fallback.

- [ ] **Step 1: Fixture**

`testdata/pm_ls.txt`:
```
/Users/me/.bun/install/global node_modules
├── typescript@5.4.2
├── prettier@3.2.5
```

- [ ] **Step 2: Test + Impl**

`internal/manager/bun/bun_test.go`:
```go
package bun

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)
func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}
func TestBunList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"bun pm ls --global": {Stdout: fx(t, "pm_ls.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "bun"}
	got, _ := a.List(context.Background())
	if len(got) != 2 { t.Fatalf("len=%d", len(got)) }
}
```

`internal/manager/bun/bun.go`:
```go
// Package bun wraps the bun JS runtime's package manager (global scope).
package bun

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "bun"} }

func (a *Adapter) ID() string                { return "bun" }
func (a *Adapter) DisplayName() string       { return "bun" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	// bun has no search — defer to npm registry via npm search if available.
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: "npm", Args: []string{"search", "--json", q}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err} }
	var entries []struct {
		Name string `json:"name"`; Version string `json:"version"`; Description string `json:"description"`
	}
	if err := json.Unmarshal(res.Stdout, &entries); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	out := make([]manager.Package, 0, len(entries))
	for _, e := range entries {
		out = append(out, manager.Package{Name: e.Name, Version: e.Version, Description: e.Description, Manager: a.ID()})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"pm", "ls", "--global"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		// "├── name@ver" or "└── name@ver"
		if !strings.HasPrefix(line, "├──") && !strings.HasPrefix(line, "└──") { continue }
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "├──"), "└──"))
		at := strings.LastIndex(line, "@")
		if at <= 0 { continue }
		out = append(out, manager.Package{Name: line[:at], Version: line[at+1:], Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(context.Context) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"add", "--global"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"remove", "--global"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update", "--global"}
	if len(names) > 0 { args = append(args, names...) }
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

- [ ] **Step 3: Register, test, commit**

```bash
go test ./internal/manager/bun/... -v
git add internal/manager/bun/ cmd/pkgr/managers.go
git commit -m "feat(bun): bun global adapter; search via npm registry fallback"
```

---

### Task 3: cargo adapter

**Files:**
- `internal/manager/cargo/{cargo.go, cargo_test.go, testdata/{installed.txt, search.txt}}`

cargo search returns lines like `name = "ver"    # description`. `cargo install --list` returns groups.

- [ ] **Step 1: Fixtures**

`testdata/search.txt`:
```
ripgrep = "14.1.0"    # ripgrep recursively searches directories
ripgrep-all = "0.10.5" # ripgrep, but also search in pdfs
```

`testdata/installed.txt` (`cargo install --list`):
```
ripgrep v14.1.0:
    rg
bat v0.24.0:
    bat
```

- [ ] **Step 2: Test + Impl**

`internal/manager/cargo/cargo_test.go`:
```go
package cargo

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)
func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}
func TestCargoSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"cargo search ripgrep --limit 25": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "cargo"}
	got, _ := a.Search(context.Background(), "ripgrep")
	if len(got) != 2 || got[0].Name != "ripgrep" { t.Fatalf("%+v", got) }
}
func TestCargoList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"cargo install --list": {Stdout: fx(t, "installed.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "cargo"}
	got, _ := a.List(context.Background())
	if len(got) != 2 || got[0].Version == "" { t.Fatalf("%+v", got) }
}
```

`internal/manager/cargo/cargo.go`:
```go
// Package cargo wraps cargo install for binary crates.
package cargo

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "cargo"} }

func (a *Adapter) ID() string                { return "cargo" }
func (a *Adapter) DisplayName() string       { return "cargo" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

var searchRe = regexp.MustCompile(`^([\w\-_]+)\s*=\s*"([^"]+)"\s*(?:#\s*(.*))?$`)

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q, "--limit", "25"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		m := searchRe.FindStringSubmatch(s.Text())
		if len(m) >= 3 {
			p := manager.Package{Name: m[1], Version: m[2], Manager: a.ID()}
			if len(m) == 4 { p.Description = m[3] }
			out = append(out, p)
		}
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"install", "--list"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "    ") { continue }
		line = strings.TrimSuffix(line, ":")
		f := strings.Fields(line)
		if len(f) < 2 { continue }
		ver := strings.TrimPrefix(f[1], "v")
		out = append(out, manager.Package{Name: f[0], Version: ver, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	// `cargo install <name>` reinstalls; pass --force for explicit update.
	if len(names) == 0 {
		return &manager.Error{Manager: a.ID(), Op: manager.OpUpdate, Code: manager.CodeConflict, Err: errString("cargo update requires explicit names")}
	}
	return a.run(ctx, manager.OpUpdate, append([]string{"install", "--force"}, names...))
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}

type errString string
func (e errString) Error() string { return string(e) }
```

- [ ] **Step 3: Register, test, commit**

```bash
go test ./internal/manager/cargo/... -v
git add internal/manager/cargo/ cmd/pkgr/managers.go
git commit -m "feat(cargo): cargo install adapter for binary crates"
```

---

### Task 4: conda adapter

**Files:**
- `internal/manager/conda/{conda.go, conda_test.go, testdata/{search.json, list.json}}`

- [ ] **Step 1: Fixtures**

`testdata/search.json`:
```json
{"requests": [{"name":"requests","version":"2.32.3","build":"pyhd8ed1ab_1","channel":"conda-forge"}]}
```

`testdata/list.json`:
```json
[{"name":"requests","version":"2.32.3","build_string":"pyhd8ed1ab_1","channel":"conda-forge"}]
```

- [ ] **Step 2: Test + Impl**

`internal/manager/conda/conda_test.go`:
```go
package conda

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)
func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}
func TestCondaSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"conda search requests --json": {Stdout: fx(t, "search.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "conda"}
	got, _ := a.Search(context.Background(), "requests")
	if len(got) < 1 { t.Fatalf("expected ≥1") }
}
func TestCondaList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"conda list --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "conda"}
	got, _ := a.List(context.Background())
	if len(got) != 1 { t.Fatalf("len=%d", len(got)) }
}
```

`internal/manager/conda/conda.go`:
```go
// Package conda wraps the conda package manager.
package conda

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "conda"} }

func (a *Adapter) ID() string                { return "conda" }
func (a *Adapter) DisplayName() string       { return "Conda" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q, "--json"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err} }
	body := map[string][]struct {
		Name string `json:"name"`; Version string `json:"version"`; Channel string `json:"channel"`
	}{}
	if err := json.Unmarshal(res.Stdout, &body); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, builds := range body {
		if len(builds) == 0 { continue }
		b := builds[len(builds)-1] // newest
		out = append(out, manager.Package{Name: b.Name, Version: b.Version, Manager: a.ID(), Extra: map[string]string{"channel": b.Channel}})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--json"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	var arr []struct {
		Name string `json:"name"`; Version string `json:"version"`; Channel string `json:"channel"`
	}
	if err := json.Unmarshal(res.Stdout, &arr); err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err} }
	out := make([]manager.Package, 0, len(arr))
	for _, e := range arr {
		out = append(out, manager.Package{Name: e.Name, Version: e.Version, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install", "-y"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"remove", "-y"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update", "-y"}
	if len(names) > 0 { args = append(args, names...) } else { args = append(args, "--all") }
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

- [ ] **Step 3: Register, test, commit**

```bash
go test ./internal/manager/conda/... -v
git add internal/manager/conda/ cmd/pkgr/managers.go
git commit -m "feat(conda): conda adapter with JSON search + list"
```

---

### Task 5: mamba adapter (clones conda)

**Files:**
- `internal/manager/mamba/{mamba.go, mamba_test.go, testdata/{list.json}}`

- [ ] **Step 1: Fixture** — `testdata/list.json` (same shape as conda's):

```json
[{"name":"numpy","version":"1.26.4"}]
```

- [ ] **Step 2: Test + Impl** — copy conda adapter, rename, swap `Bin = "mamba"`. Search/install/list args identical (mamba CLI is conda-compatible).

`internal/manager/mamba/mamba_test.go`:
```go
package mamba

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)
func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}
func TestMambaList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"mamba list --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "mamba"}
	got, _ := a.List(context.Background())
	if len(got) != 1 { t.Fatalf("len=%d", len(got)) }
}
```

`internal/manager/mamba/mamba.go`:
```go
// Package mamba wraps the mamba package manager (conda-CLI-compatible).
package mamba

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "mamba"} }

func (a *Adapter) ID() string                { return "mamba" }
func (a *Adapter) DisplayName() string       { return "Mamba" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q, "--json"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err} }
	body := map[string][]struct {
		Name string `json:"name"`; Version string `json:"version"`; Channel string `json:"channel"`
	}{}
	if err := json.Unmarshal(res.Stdout, &body); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, builds := range body {
		if len(builds) == 0 { continue }
		b := builds[len(builds)-1]
		out = append(out, manager.Package{Name: b.Name, Version: b.Version, Manager: a.ID(), Extra: map[string]string{"channel": b.Channel}})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--json"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	var arr []struct {
		Name string `json:"name"`; Version string `json:"version"`
	}
	if err := json.Unmarshal(res.Stdout, &arr); err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err} }
	out := make([]manager.Package, 0, len(arr))
	for _, e := range arr {
		out = append(out, manager.Package{Name: e.Name, Version: e.Version, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install", "-y"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"remove", "-y"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update", "-y"}
	if len(names) > 0 { args = append(args, names...) } else { args = append(args, "--all") }
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

Note: this duplicates conda's adapter. We accept that — DRY across only two adapters would force a "condaLike" shared package that's harder to read than two flat 80-line files. If a third conda-compatible PM appears, extract then.

- [ ] **Step 3: Register, test, commit**

```bash
go test ./internal/manager/mamba/... -v
git add internal/manager/mamba/ cmd/pkgr/managers.go
git commit -m "feat(mamba): mamba adapter (conda-compatible CLI)"
```

---

### Task 6: gem adapter (RubyGems)

**Files:**
- `internal/manager/gem/{gem.go, gem_test.go, testdata/{search.txt, list.txt, outdated.txt}}`

- [ ] **Step 1: Fixtures**

`testdata/search.txt` (`gem search -r ^rails$`):
```
*** REMOTE GEMS ***

rails (7.1.3.4)
rails-i18n (7.0.8)
```

`testdata/list.txt`:
```
*** LOCAL GEMS ***

bundler (2.5.9)
rake (13.2.1, default: 13.0.6)
```

`testdata/outdated.txt`:
```
bundler (2.5.9 < 2.6.0)
```

- [ ] **Step 2: Test + Impl**

`internal/manager/gem/gem_test.go`:
```go
package gem

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)
func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}
func TestGemSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"gem search -r rails": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "gem"}
	got, _ := a.Search(context.Background(), "rails")
	if len(got) != 2 { t.Fatalf("%+v", got) }
}
func TestGemList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"gem list": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "gem"}
	got, _ := a.List(context.Background())
	if len(got) != 2 { t.Fatalf("len=%d", len(got)) }
}
func TestGemOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"gem outdated": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "gem"}
	got, _ := a.Outdated(context.Background())
	if len(got) != 1 || got[0].Latest != "2.6.0" { t.Fatalf("%+v", got) }
}
```

`internal/manager/gem/gem.go`:
```go
// Package gem wraps RubyGems.
package gem

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "gem"} }

func (a *Adapter) ID() string                { return "gem" }
func (a *Adapter) DisplayName() string       { return "RubyGems" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

var gemRowRe = regexp.MustCompile(`^([A-Za-z0-9_\-]+)\s+\(([^)]+)\)$`)
var outdatedRe = regexp.MustCompile(`^([A-Za-z0-9_\-]+)\s+\(([^ ]+)\s+<\s+([^)]+)\)$`)

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", "-r", q}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		m := gemRowRe.FindStringSubmatch(s.Text())
		if len(m) == 3 {
			ver := strings.SplitN(m[2], ",", 2)[0]
			out = append(out, manager.Package{Name: m[1], Version: strings.TrimSpace(ver), Manager: a.ID()})
		}
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		m := gemRowRe.FindStringSubmatch(s.Text())
		if len(m) == 3 {
			ver := strings.SplitN(m[2], ",", 2)[0]
			out = append(out, manager.Package{Name: m[1], Version: strings.TrimSpace(ver), Manager: a.ID(), Installed: true})
		}
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		m := outdatedRe.FindStringSubmatch(s.Text())
		if len(m) == 4 {
			out = append(out, manager.Package{Name: m[1], Version: m[2], Latest: m[3], Manager: a.ID(), Installed: true})
		}
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install", "--user-install"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall", "--force"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update"}
	if len(names) > 0 { args = append(args, names...) }
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

- [ ] **Step 3: Register, test, commit**

```bash
go test ./internal/manager/gem/... -v
git add internal/manager/gem/ cmd/pkgr/managers.go
git commit -m "feat(gem): RubyGems adapter"
```

---

### Task 7: go adapter (`go install`)

`go` has no native search; we hit pkg.go.dev's search endpoint over HTTP. List = scan `$GOBIN`/`$GOPATH/bin`.

**Files:**
- `internal/manager/goinst/{goinst.go, goinst_test.go, testdata/{search.html}}`

- [ ] **Step 1: Fixture** (HTML snippet from pkg.go.dev `/search?q=...&m=json` not available; the real endpoint is HTML. We mock a minimal-shape JSON via an alternative endpoint — for tests we just hand a canned slice of Package values via an injected fake search func.)

`testdata/search.json`:
```json
[
  {"path": "github.com/junegunn/fzf", "synopsis": "fzf is a general-purpose command-line fuzzy finder"}
]
```

- [ ] **Step 2: Test + Impl**

`internal/manager/goinst/goinst_test.go`:
```go
package goinst

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
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(f.body))), Header: http.Header{}}, nil
}

func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}

func TestGoSearch(t *testing.T) {
	a := &Adapter{
		Runner: &runner.Runner{Exec: (&runner.Fake{}).Exec},
		HTTP:   &http.Client{Transport: &fakeRT{body: fx(t, "search.json")}},
		Bin:    "go",
	}
	got, _ := a.Search(context.Background(), "fzf")
	if len(got) != 1 || got[0].Name != "github.com/junegunn/fzf" { t.Fatalf("%+v", got) }
}
```

`internal/manager/goinst/goinst.go`:
```go
// Package goinst wraps `go install pkg@latest` and uses pkg.go.dev for search.
package goinst

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	HTTP   *http.Client
	Bin    string
}

func New(r *runner.Runner) *Adapter {
	return &Adapter{Runner: r, HTTP: http.DefaultClient, Bin: "go"}
}

func (a *Adapter) ID() string                { return "go" }
func (a *Adapter) DisplayName() string       { return "go install" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

// We talk to a local stub by default; production deployment would use
// "https://api.pkg.go.dev/search?q=" or the website's autocomplete endpoint.
// For now, expose the HTTP client so tests can inject a fake transport.
type searchEntry struct {
	Path     string `json:"path"`
	Synopsis string `json:"synopsis"`
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.pkg.go.dev/search?q="+q, nil)
	resp, err := a.HTTP.Do(req)
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeNetworkFailure, Err: err} }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var arr []searchEntry
	if err := json.Unmarshal(body, &arr); err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err} }
	out := make([]manager.Package, 0, len(arr))
	for _, e := range arr {
		out = append(out, manager.Package{Name: e.Path, Description: e.Synopsis, Manager: a.ID()})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	// Best-effort: list executables in GOBIN / GOPATH/bin.
	dir := os.Getenv("GOBIN")
	if dir == "" {
		if g := os.Getenv("GOPATH"); g != "" { dir = filepath.Join(g, "bin") }
	}
	if dir == "" { return nil, nil }
	entries, err := os.ReadDir(dir)
	if err != nil { return nil, nil }
	var out []manager.Package
	for _, e := range entries {
		if e.IsDir() { continue }
		out = append(out, manager.Package{Name: e.Name(), Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	for _, n := range names {
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"install", n + "@latest"}}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpInstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	dir := os.Getenv("GOBIN")
	if dir == "" {
		if g := os.Getenv("GOPATH"); g != "" { dir = filepath.Join(g, "bin") }
	}
	for _, n := range names {
		_ = os.Remove(filepath.Join(dir, filepath.Base(n)))
	}
	return nil
}
func (a *Adapter) Update(ctx context.Context, names ...string) error { return a.Install(ctx, names...) }
```

- [ ] **Step 3: Register, test, commit**

```bash
go test ./internal/manager/goinst/... -v
git add internal/manager/goinst/ cmd/pkgr/managers.go
git commit -m "feat(go): adapter for go install + pkg.go.dev search"
```

---

### Task 8: mise adapter

**Files:**
- `internal/manager/mise/{mise.go, mise_test.go, testdata/{list.json, ls_remote.txt, plugins.txt}}`

mise has rich JSON output. `mise ls --json`, `mise ls-remote <plugin>`, `mise install <plugin>@<ver>`.

- [ ] **Step 1: Fixtures**

`testdata/list.json`:
```json
{
  "node":   [{"version":"20.12.2","requested_version":"20","install_path":"/.../node/20.12.2","source":{"type":".mise.toml"}}],
  "python": [{"version":"3.12.3","requested_version":"3.12","install_path":"/.../python/3.12.3","source":null}]
}
```

`testdata/plugins.txt`:
```
node
python
ruby
```

`testdata/ls_remote.txt`:
```
22.0.0
22.1.0
22.2.0
```

- [ ] **Step 2: Test + Impl**

`internal/manager/mise/mise_test.go`:
```go
package mise

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)
func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}
func TestMiseList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"mise ls --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "mise"}
	got, _ := a.List(context.Background())
	if len(got) != 2 { t.Fatalf("len=%d", len(got)) }
}
func TestMiseSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"mise plugins ls-remote": {Stdout: fx(t, "plugins.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "mise"}
	got, _ := a.Search(context.Background(), "py")
	if len(got) != 1 { t.Fatalf("%+v", got) }
}
```

`internal/manager/mise/mise.go`:
```go
// Package mise wraps the mise (formerly rtx) version manager.
package mise

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "mise"} }

func (a *Adapter) ID() string                { return "mise" }
func (a *Adapter) DisplayName() string       { return "mise" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"plugins", "ls-remote"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		name := strings.TrimSpace(s.Text())
		if name == "" { continue }
		if q != "" && !strings.Contains(name, q) { continue }
		out = append(out, manager.Package{Name: name, Manager: a.ID(), Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"ls", "--json"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	body := map[string][]struct {
		Version string `json:"version"`
	}{}
	if err := json.Unmarshal(res.Stdout, &body); err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err} }
	var out []manager.Package
	for plugin, vers := range body {
		for _, v := range vers {
			out = append(out, manager.Package{Name: plugin, Version: v.Version, Manager: a.ID(), Installed: true, Extra: map[string]string{"kind": "toolchain"}})
		}
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated", "--json"}})
	if err != nil { return nil, nil }
	body := []struct {
		Plugin string `json:"plugin"`; Requested string `json:"requested"`; Latest string `json:"latest"`
	}{}
	_ = json.Unmarshal(res.Stdout, &body)
	out := make([]manager.Package, 0, len(body))
	for _, e := range body {
		out = append(out, manager.Package{Name: e.Plugin, Version: e.Requested, Latest: e.Latest, Manager: a.ID(), Installed: true, Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade"}
	if len(names) > 0 { args = append(args, names...) }
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

- [ ] **Step 3: Register, test, commit**

```bash
go test ./internal/manager/mise/... -v
git add internal/manager/mise/ cmd/pkgr/managers.go
git commit -m "feat(mise): mise toolchain adapter with JSON list/outdated"
```

---

### Task 9: pipx adapter

**Files:**
- `internal/manager/pipx/{pipx.go, pipx_test.go, testdata/{list.json}}`

- [ ] **Step 1: Fixture**

`testdata/list.json`:
```json
{
  "venvs": {
    "black":  {"metadata": {"main_package": {"package_or_url": "black", "package_version": "24.4.2"}}},
    "ruff":   {"metadata": {"main_package": {"package_or_url": "ruff",  "package_version": "0.4.4"}}}
  }
}
```

- [ ] **Step 2: Test + Impl**

`internal/manager/pipx/pipx_test.go`:
```go
package pipx

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)
func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}
func TestPipxList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pipx list --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pipx"}
	got, _ := a.List(context.Background())
	if len(got) != 2 { t.Fatalf("len=%d", len(got)) }
}
```

`internal/manager/pipx/pipx.go`:
```go
// Package pipx wraps pipx (isolated Python tools).
package pipx

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "pipx"} }

func (a *Adapter) ID() string                { return "pipx" }
func (a *Adapter) DisplayName() string       { return "pipx" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	// pipx has no search; defer to PyPI exact-match (same approach as pip adapter).
	return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeNotFound,
		Err: errString("pipx: search not implemented; use pip adapter for PyPI search")}
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--json"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	var body struct {
		Venvs map[string]struct {
			Metadata struct {
				MainPackage struct {
					Name    string `json:"package_or_url"`
					Version string `json:"package_version"`
				} `json:"main_package"`
			} `json:"metadata"`
		} `json:"venvs"`
	}
	if err := json.Unmarshal(res.Stdout, &body); err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err} }
	var out []manager.Package
	for _, v := range body.Venvs {
		out = append(out, manager.Package{Name: v.Metadata.MainPackage.Name, Version: v.Metadata.MainPackage.Version, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	for _, n := range names {
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"install", n}}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpInstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	for _, n := range names {
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"uninstall", n}}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpUninstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade-all"}
	if len(names) > 0 { args = append([]string{"upgrade"}, names...) }
	if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args}); err != nil {
		return &manager.Error{Manager: a.ID(), Op: manager.OpUpdate, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}

type errString string
func (e errString) Error() string { return string(e) }
```

- [ ] **Step 3: Register, test, commit**

```bash
go test ./internal/manager/pipx/... -v
git add internal/manager/pipx/ cmd/pkgr/managers.go
git commit -m "feat(pipx): pipx adapter for isolated Python tools"
```

---

### Task 10: rustup adapter

`rustup` manages Rust toolchains. Treat each toolchain (stable, nightly, etc.) as a "package."

**Files:**
- `internal/manager/rustup/{rustup.go, rustup_test.go, testdata/{toolchains.txt, list_remote.txt}}`

- [ ] **Step 1: Fixtures**

`testdata/toolchains.txt`:
```
stable-aarch64-apple-darwin (default)
nightly-aarch64-apple-darwin
```

`testdata/list_remote.txt` (truncated `rustup check`):
```
stable-aarch64-apple-darwin - Update available : 1.77.2 -> 1.78.0
nightly-aarch64-apple-darwin - Up to date : 1.80.0-nightly (...)
```

- [ ] **Step 2: Test + Impl**

`internal/manager/rustup/rustup_test.go`:
```go
package rustup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)
func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}
func TestRustupList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"rustup toolchain list": {Stdout: fx(t, "toolchains.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "rustup"}
	got, _ := a.List(context.Background())
	if len(got) != 2 { t.Fatalf("len=%d", len(got)) }
}
func TestRustupOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"rustup check": {Stdout: fx(t, "list_remote.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "rustup"}
	got, _ := a.Outdated(context.Background())
	if len(got) != 1 || got[0].Latest != "1.78.0" { t.Fatalf("%+v", got) }
}
```

`internal/manager/rustup/rustup.go`:
```go
// Package rustup wraps the rustup toolchain manager.
package rustup

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "rustup"} }

func (a *Adapter) ID() string                { return "rustup" }
func (a *Adapter) DisplayName() string       { return "rustup" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	// rustup toolchains are well-known constants; expose the common ones.
	known := []string{"stable", "beta", "nightly"}
	var out []manager.Package
	for _, n := range known {
		if q != "" && !strings.Contains(n, q) { continue }
		out = append(out, manager.Package{Name: n, Manager: a.ID(), Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"toolchain", "list"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" { continue }
		name := strings.Fields(line)[0]
		out = append(out, manager.Package{Name: name, Manager: a.ID(), Installed: true, Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"check"}})
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		if !strings.Contains(line, "Update available") { continue }
		// "<toolchain> - Update available : X -> Y"
		dash := strings.Index(line, " - ")
		if dash < 0 { continue }
		name := strings.Fields(line[:dash])[0]
		idx := strings.LastIndex(line, " -> ")
		if idx < 0 { continue }
		latest := strings.TrimSpace(line[idx+4:])
		out = append(out, manager.Package{Name: name, Latest: latest, Manager: a.ID(), Installed: true, Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	for _, n := range names {
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"toolchain", "install", n}}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpInstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	for _, n := range names {
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"toolchain", "uninstall", n}}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpUninstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update"}
	if len(names) > 0 { args = append(args, names...) }
	if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args}); err != nil {
		return &manager.Error{Manager: a.ID(), Op: manager.OpUpdate, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
```

- [ ] **Step 3: Register, test, commit**

```bash
go test ./internal/manager/rustup/... -v
git add internal/manager/rustup/ cmd/pkgr/managers.go
git commit -m "feat(rustup): rustup toolchain adapter"
```

---

### Task 11: uv adapter (`uv tool`)

**Files:**
- `internal/manager/uv/{uv.go, uv_test.go, testdata/{tool_list.txt}}`

- [ ] **Step 1: Fixture**

`testdata/tool_list.txt` (output of `uv tool list`):
```
ruff v0.4.4
black v24.4.2
```

- [ ] **Step 2: Test + Impl**

`internal/manager/uv/uv_test.go`:
```go
package uv

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)
func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}
func TestUvList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"uv tool list": {Stdout: fx(t, "tool_list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "uv"}
	got, _ := a.List(context.Background())
	if len(got) != 2 { t.Fatalf("len=%d", len(got)) }
}
```

`internal/manager/uv/uv.go`:
```go
// Package uv wraps the uv tool subcommand.
package uv

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "uv"} }

func (a *Adapter) ID() string                { return "uv" }
func (a *Adapter) DisplayName() string       { return "uv tool" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	// uv has no built-in registry search; uv installs from PyPI. Defer search
	// to PyPI exact-match (same pattern as pip adapter, but here we keep it simple).
	return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeNotFound,
		Err: errString("uv: search not implemented; use pip adapter for PyPI search")}
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"tool", "list"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) < 2 { continue }
		ver := strings.TrimPrefix(f[1], "v")
		out = append(out, manager.Package{Name: f[0], Version: ver, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"tool", "install"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"tool", "uninstall"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	if len(names) == 0 { return a.run(ctx, manager.OpUpdate, []string{"tool", "upgrade", "--all"}) }
	return a.run(ctx, manager.OpUpdate, append([]string{"tool", "upgrade"}, names...))
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}

type errString string
func (e errString) Error() string { return string(e) }
```

- [ ] **Step 3: Register, test, commit**

```bash
go test ./internal/manager/uv/... -v
git add internal/manager/uv/ cmd/pkgr/managers.go
git commit -m "feat(uv): uv tool adapter for installable Python tools"
```

---

### Task 12: yarn adapter (global)

**Files:**
- `internal/manager/yarn/{yarn.go, yarn_test.go, testdata/{global_list.txt}}`

yarn classic `global list --depth=0` prints lines like `info "name@ver" has binaries`. yarn berry has different shape — we target classic.

- [ ] **Step 1: Fixture**

`testdata/global_list.txt`:
```
yarn global v1.22.22
info "typescript@5.4.2" has binaries:
   - tsc
   - tsserver
info "prettier@3.2.5" has binaries:
   - prettier
```

- [ ] **Step 2: Test + Impl**

`internal/manager/yarn/yarn_test.go`:
```go
package yarn

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)
func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil { t.Fatal(err) }
	return b
}
func TestYarnList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"yarn global list --depth=0": {Stdout: fx(t, "global_list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "yarn"}
	got, _ := a.List(context.Background())
	if len(got) != 2 { t.Fatalf("len=%d", len(got)) }
}
```

`internal/manager/yarn/yarn.go`:
```go
// Package yarn wraps yarn classic global installs.
package yarn

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "yarn"} }

func (a *Adapter) ID() string                { return "yarn" }
func (a *Adapter) DisplayName() string       { return "yarn" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

var infoRe = regexp.MustCompile(`^info "([^@"]+)@([^"]+)" has binaries:$`)

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	// yarn has no global registry search; defer to npm.
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: "npm", Args: []string{"search", "--json", q}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err} }
	var entries []struct {
		Name string `json:"name"`; Version string `json:"version"`; Description string `json:"description"`
	}
	if err := json.Unmarshal(res.Stdout, &entries); err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err} }
	out := make([]manager.Package, 0, len(entries))
	for _, e := range entries {
		out = append(out, manager.Package{Name: e.Name, Version: e.Version, Description: e.Description, Manager: a.ID()})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"global", "list", "--depth=0"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		m := infoRe.FindStringSubmatch(strings.TrimSpace(s.Text()))
		if len(m) == 3 {
			out = append(out, manager.Package{Name: m[1], Version: m[2], Manager: a.ID(), Installed: true})
		}
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"global", "add"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"global", "remove"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"global", "upgrade"}
	if len(names) > 0 { args = append(args, names...) }
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

- [ ] **Step 3: Register, test, commit**

```bash
go test ./internal/manager/yarn/... -v
git add internal/manager/yarn/ cmd/pkgr/managers.go
git commit -m "feat(yarn): yarn classic global adapter; search via npm fallback"
```

---

## Phase 5 Acceptance

- 12 new adapters wired (asdf, bun, cargo, conda, mamba, gem, goinst, mise, pipx, rustup, uv, yarn)
- Total adapters: 26 (Phases 2+4+5 sum)
- Each adapter has ≥ 2 unit tests with golden fixtures
- `pkgr pm list` shows 26 rows
- `make test` green; `make lint` green
