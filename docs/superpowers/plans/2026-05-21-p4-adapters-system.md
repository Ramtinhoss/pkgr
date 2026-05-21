# Phase 4: System-Scope Adapters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement 11 more adapters: apt, dnf, pacman, snap, flatpak, nix, scoop, choco, winget, mas, pnpm. Wire each into `cmd/pkgr/managers.go`. Each adapter has golden fixtures and full unit tests using `runner.Fake`.

**Architecture:** Identical adapter pattern from Phase 2. New adapters share no code with each other except the interface they implement. Sudo-aware adapters (apt, dnf, pacman, snap, choco) return `true` from `NeedsSudo` for mutating ops; runner wraps with `sudo` automatically when invoked.

**Tech Stack:** Go stdlib JSON / line parsing; golden fixtures in `testdata/`.

---

## Task Template (per adapter)

Each task follows the same shape:

1. Drop golden fixtures into `internal/manager/<id>/testdata/`
2. Write failing test in `internal/manager/<id>/<id>_test.go`
3. Run `go test ./internal/manager/<id>/...` to confirm FAIL
4. Implement `internal/manager/<id>/<id>.go`
5. Run tests, expect PASS
6. Register adapter in `cmd/pkgr/managers.go`
7. Commit with `feat(<id>): adapter ...`

The interface methods to implement: `ID/DisplayName/OSes/Detect/NeedsSudo/Scope/List/Outdated/Search/Info/Install/Uninstall/Update`.

---

### Task 1: apt adapter (linux, sudo)

**Files:**
- Create: `internal/manager/apt/{apt.go, apt_test.go, testdata/{search.txt, list.txt, outdated.txt}}`

- [ ] **Step 1: Fixtures**

`testdata/search.txt` (output of `apt-cache search`):
```
ripgrep - line-oriented search tool
ripgrep-all - ripgrep wrapper that searches in pdfs, ebooks, …
```

`testdata/list.txt` (output of `apt list --installed`):
```
Listing...
ripgrep/jammy,now 13.0.0-2ubuntu0.1 amd64 [installed]
jq/jammy,now 1.6-2.1ubuntu3 amd64 [installed,automatic]
```

`testdata/outdated.txt` (output of `apt list --upgradable`):
```
Listing...
curl/jammy-updates 7.81.0-1ubuntu1.13 amd64 [upgradable from: 7.81.0-1ubuntu1.10]
```

- [ ] **Step 2: Test `internal/manager/apt/apt_test.go`**

```go
package apt

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

func TestSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"apt-cache search ripgrep": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "apt-cache"}
	pkgs, err := a.Search(context.Background(), "ripgrep")
	if err != nil { t.Fatal(err) }
	if len(pkgs) != 2 || pkgs[0].Name != "ripgrep" { t.Fatalf("%+v", pkgs) }
}

func TestList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"apt list --installed": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "apt"}
	pkgs, err := a.List(context.Background())
	if err != nil { t.Fatal(err) }
	if len(pkgs) != 2 { t.Fatalf("len=%d", len(pkgs)) }
}

func TestOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"apt list --upgradable": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "apt"}
	pkgs, err := a.Outdated(context.Background())
	if err != nil { t.Fatal(err) }
	if len(pkgs) != 1 || pkgs[0].Latest == "" { t.Fatalf("%+v", pkgs) }
}
```

- [ ] **Step 3: Implement `internal/manager/apt/apt.go`**

```go
// Package apt is the Debian/Ubuntu APT adapter.
package apt

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string // "apt" by default; some ops use "apt-cache"
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "apt"} }

func (a *Adapter) ID() string                { return "apt" }
func (a *Adapter) DisplayName() string       { return "APT" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(op manager.Op) bool {
	switch op {
	case manager.OpInstall, manager.OpUninstall, manager.OpUpdate:
		return true
	}
	return false
}
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: "apt-cache", Args: []string{"search", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		idx := strings.Index(line, " - ")
		if idx < 0 { continue }
		out = append(out, manager.Package{
			Name: line[:idx], Description: line[idx+3:], Manager: a.ID(),
		})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--installed"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	return parseAptList(res.Stdout, a.ID(), false), nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--upgradable"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	return parseAptList(res.Stdout, a.ID(), true), nil
}

func parseAptList(b []byte, pmID string, upgradable bool) []manager.Package {
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "Listing") { continue }
		// format: <name>/<dist>,now <ver> <arch> [installed,...]
		slash := strings.Index(line, "/")
		if slash < 0 { continue }
		name := line[:slash]
		rest := strings.Fields(line[slash:])
		if len(rest) < 2 { continue }
		ver := rest[1]
		p := manager.Package{Name: name, Version: ver, Manager: pmID, Installed: !upgradable}
		if upgradable {
			// rest[len-1] like "[upgradable from: X]"; salvage X if present
			if idx := strings.Index(line, "from: "); idx > 0 {
				p.Version = strings.TrimSuffix(line[idx+6:], "]")
				p.Latest = ver
			}
		}
		out = append(out, p)
	}
	return out
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: "apt-cache", Args: []string{"show", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	p := manager.Package{Name: name, Manager: a.ID()}
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "Version: "):
			p.Version = strings.TrimPrefix(line, "Version: ")
		case strings.HasPrefix(line, "Homepage: "):
			p.Homepage = strings.TrimPrefix(line, "Homepage: ")
		case strings.HasPrefix(line, "Description: "):
			p.Description = strings.TrimPrefix(line, "Description: ")
		}
	}
	return p, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpInstall, append([]string{"install", "-y"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpUninstall, append([]string{"remove", "-y"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade", "-y"}
	if len(names) > 0 { args = append([]string{"install", "-y", "--only-upgrade"}, names...) }
	return a.exec(ctx, manager.OpUpdate, args...)
}
func (a *Adapter) exec(ctx context.Context, op manager.Op, args ...string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: "apt-get", Args: args, Sudo: a.NeedsSudo(op)})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err, Cmd: fmt.Sprintf("apt-get %s", strings.Join(args, " "))}
	}
	return nil
}
```

- [ ] **Step 4: Register in `cmd/pkgr/managers.go`**

```go
import "github.com/ramtinhoss/pkgr/internal/manager/apt"
// in registerAdapters:
reg.Register(apt.New(r))
```

- [ ] **Step 5: Run tests, expect PASS; commit**

```bash
go test ./internal/manager/apt/... -v
go build ./...
git add internal/manager/apt/ cmd/pkgr/managers.go
git commit -m "feat(apt): Debian/Ubuntu adapter with sudo-aware mutations"
```

---

### Task 2: dnf adapter (Fedora/RHEL, sudo)

**Files:**
- Create: `internal/manager/dnf/{dnf.go, dnf_test.go, testdata/{search.txt, list.txt, outdated.txt}}`

- [ ] **Step 1: Fixtures**

`testdata/search.txt`:
```
Last metadata expiration check: 0:30:00 ago.
============================ Name & Summary Matched: ripgrep ============================
ripgrep.x86_64 : Line oriented search tool using Rust's regex library
ripgrep-doc.noarch : Documentation files for ripgrep
```

`testdata/list.txt` (output of `dnf list --installed`):
```
Installed Packages
ripgrep.x86_64                14.1.0-1.fc40    @fedora
jq.x86_64                     1.7.1-3.fc40     @fedora
```

`testdata/outdated.txt` (output of `dnf check-update`):
```
Last metadata expiration check: 0:01:23 ago.

curl.x86_64                   8.5.0-3.fc40     updates
glibc.x86_64                  2.39-12.fc40     updates
```

- [ ] **Step 2: Test `internal/manager/dnf/dnf_test.go`**

```go
package dnf

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

func TestSearchDnf(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"dnf search ripgrep": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "dnf"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) < 1 { t.Fatalf("expected ≥1, got %d", len(pkgs)) }
}

func TestListDnf(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"dnf list --installed": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "dnf"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 { t.Fatalf("len=%d", len(pkgs)) }
}

func TestOutdatedDnf(t *testing.T) {
	// dnf check-update exits 100 when updates exist.
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"dnf check-update": {Stdout: fx(t, "outdated.txt"), Code: 100},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "dnf"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 2 { t.Fatalf("len=%d", len(pkgs)) }
}
```

- [ ] **Step 3: Implement `internal/manager/dnf/dnf.go`**

```go
// Package dnf is the Fedora/RHEL DNF adapter.
package dnf

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "dnf"} }

func (a *Adapter) ID() string                { return "dnf" }
func (a *Adapter) DisplayName() string       { return "DNF" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(op manager.Op) bool {
	switch op {
	case manager.OpInstall, manager.OpUninstall, manager.OpUpdate:
		return true
	}
	return false
}
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "===") || strings.HasPrefix(line, "Last metadata") || line == "" {
			continue
		}
		// format: name.arch : description
		idx := strings.Index(line, " : ")
		if idx < 0 { continue }
		nameArch := line[:idx]
		name := nameArch
		if dot := strings.LastIndex(nameArch, "."); dot > 0 {
			name = nameArch[:dot]
		}
		out = append(out, manager.Package{Name: name, Description: line[idx+3:], Manager: a.ID()})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--installed"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	return parseDnfList(res.Stdout, a.ID(), false), nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"check-update"}})
	// exit code 100 = updates available; treat that as success
	return parseDnfList(res.Stdout, a.ID(), true), nil
}

func parseDnfList(b []byte, pmID string, outdated bool) []manager.Package {
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "Installed") || strings.HasPrefix(line, "Last") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 2 { continue }
		nameArch := f[0]
		name := nameArch
		if dot := strings.LastIndex(nameArch, "."); dot > 0 {
			name = nameArch[:dot]
		}
		p := manager.Package{Name: name, Version: f[1], Manager: pmID, Installed: !outdated}
		if outdated { p.Latest = f[1]; p.Version = "" }
		out = append(out, p)
	}
	return out
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"info", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	p := manager.Package{Name: name, Manager: a.ID()}
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "Version "):
			p.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version :"))
		case strings.HasPrefix(line, "URL "):
			p.Homepage = strings.TrimSpace(strings.TrimPrefix(line, "URL :"))
		case strings.HasPrefix(line, "Summary "):
			p.Description = strings.TrimSpace(strings.TrimPrefix(line, "Summary :"))
		}
	}
	return p, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpInstall, append([]string{"install", "-y"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpUninstall, append([]string{"remove", "-y"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade", "-y"}
	if len(names) > 0 { args = append(args, names...) }
	return a.exec(ctx, manager.OpUpdate, args...)
}
func (a *Adapter) exec(ctx context.Context, op manager.Op, args ...string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args, Sudo: a.NeedsSudo(op)})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
```

- [ ] **Step 4: Register, test, commit**

```bash
# add `reg.Register(dnf.New(r))` and import in cmd/pkgr/managers.go
go test ./internal/manager/dnf/... -v
go build ./...
git add internal/manager/dnf/ cmd/pkgr/managers.go
git commit -m "feat(dnf): Fedora/RHEL adapter"
```

---

### Task 3: pacman adapter (Arch, sudo)

**Files:**
- Create: `internal/manager/pacman/{pacman.go, pacman_test.go, testdata/{search.txt, list.txt, outdated.txt}}`

- [ ] **Step 1: Fixtures**

`testdata/search.txt` (`pacman -Ss ripgrep`):
```
extra/ripgrep 14.1.0-1
    A search tool that combines the usability of ag with the raw speed of grep
extra/ripgrep-all 0.10.5-1
    rga: ripgrep, but also search in PDFs, ebooks, …
```

`testdata/list.txt` (`pacman -Q`):
```
ripgrep 14.1.0-1
jq 1.7.1-1
```

`testdata/outdated.txt` (`pacman -Qu`):
```
curl 8.5.0-1 -> 8.6.0-1
```

- [ ] **Step 2: Test `internal/manager/pacman/pacman_test.go`**

```go
package pacman

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

func TestSearchPacman(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pacman -Ss ripgrep": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pacman"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 2 { t.Fatalf("got %+v", pkgs) }
}

func TestListPacman(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pacman -Q": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pacman"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 { t.Fatalf("len=%d", len(pkgs)) }
}

func TestOutdatedPacman(t *testing.T) {
	// pacman -Qu exits 1 when there are no upgrades; we test the upgrade case.
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pacman -Qu": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pacman"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 || pkgs[0].Latest != "8.6.0-1" { t.Fatalf("%+v", pkgs) }
}
```

- [ ] **Step 3: Implement `internal/manager/pacman/pacman.go`**

```go
// Package pacman is the Arch Linux pacman adapter.
package pacman

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "pacman"} }

func (a *Adapter) ID() string           { return "pacman" }
func (a *Adapter) DisplayName() string  { return "pacman" }
func (a *Adapter) OSes() []manager.OS   { return []manager.OS{manager.Linux} }
func (a *Adapter) Scope() manager.Scope { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(op manager.Op) bool {
	return op == manager.OpInstall || op == manager.OpUninstall || op == manager.OpUpdate
}
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"-Ss", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	var cur manager.Package
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "    ") {
			cur.Description = strings.TrimSpace(line)
			out = append(out, cur)
			cur = manager.Package{}
			continue
		}
		// repo/name version
		parts := strings.Fields(line)
		if len(parts) < 2 { continue }
		nameParts := strings.SplitN(parts[0], "/", 2)
		name := nameParts[len(nameParts)-1]
		cur = manager.Package{Name: name, Version: parts[1], Manager: a.ID()}
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"-Q"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) < 2 { continue }
		out = append(out, manager.Package{Name: f[0], Version: f[1], Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"-Qu"}})
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		f := strings.Fields(line)
		if len(f) < 4 || f[2] != "->" { continue }
		out = append(out, manager.Package{Name: f[0], Version: f[1], Latest: f[3], Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"-Si", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	p := manager.Package{Name: name, Manager: a.ID()}
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "Version "):
			p.Version = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		case strings.HasPrefix(line, "URL "):
			p.Homepage = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		case strings.HasPrefix(line, "Description "):
			p.Description = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
	}
	return p, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpInstall, append([]string{"-S", "--noconfirm"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpUninstall, append([]string{"-R", "--noconfirm"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"-Syu", "--noconfirm"}
	if len(names) > 0 { args = append([]string{"-S", "--noconfirm"}, names...) }
	return a.exec(ctx, manager.OpUpdate, args...)
}
func (a *Adapter) exec(ctx context.Context, op manager.Op, args ...string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args, Sudo: a.NeedsSudo(op)})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
```

- [ ] **Step 4: Register, test, commit**

```bash
go test ./internal/manager/pacman/... -v
git add internal/manager/pacman/ cmd/pkgr/managers.go
git commit -m "feat(pacman): Arch Linux pacman adapter"
```

---

### Task 4: snap adapter (Ubuntu, sudo)

**Files:**
- Create: `internal/manager/snap/{snap.go, snap_test.go, testdata/{find.txt, list.txt, refresh.txt}}`

- [ ] **Step 1: Fixtures**

`testdata/find.txt`:
```
Name        Version       Publisher    Notes  Summary
ripgrep     14.0.3        bossm                grep alternative
spotify     1.2.13.661.g  spotify✓     -      Music for everyone
```

`testdata/list.txt`:
```
Name       Version          Rev   Tracking         Publisher    Notes
bare       1.0              5     latest/stable    canonical✓   base
core22     20240408         1380  latest/stable    canonical✓   base
ripgrep    14.0.3           21    latest/stable    bossm        -
```

`testdata/refresh.txt` (`snap refresh --list`):
```
Name       Version      Rev    Size    Publisher    Notes
core22     20240601     1500   77MB    canonical✓   base
```

- [ ] **Step 2: Test `internal/manager/snap/snap_test.go`**

```go
package snap

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

func TestSnapSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"snap find ripgrep": {Stdout: fx(t, "find.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "snap"}
	got, _ := a.Search(context.Background(), "ripgrep")
	if len(got) != 2 { t.Fatalf("%+v", got) }
}

func TestSnapList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"snap list": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "snap"}
	got, _ := a.List(context.Background())
	if len(got) != 3 { t.Fatalf("len=%d", len(got)) }
}

func TestSnapOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"snap refresh --list": {Stdout: fx(t, "refresh.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "snap"}
	got, _ := a.Outdated(context.Background())
	if len(got) != 1 { t.Fatalf("%+v", got) }
}
```

- [ ] **Step 3: Implement `internal/manager/snap/snap.go`**

```go
// Package snap is the Ubuntu/Linux Snap adapter.
package snap

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "snap"} }

func (a *Adapter) ID() string           { return "snap" }
func (a *Adapter) DisplayName() string  { return "Snap" }
func (a *Adapter) OSes() []manager.OS   { return []manager.OS{manager.Linux} }
func (a *Adapter) Scope() manager.Scope { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(op manager.Op) bool {
	return op == manager.OpInstall || op == manager.OpUninstall || op == manager.OpUpdate
}
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"find", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	return parseSnapTable(res.Stdout, a.ID(), false, true), nil
}
func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	return parseSnapTable(res.Stdout, a.ID(), true, false), nil
}
func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"refresh", "--list"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	return parseSnapTable(res.Stdout, a.ID(), true, false), nil
}

func parseSnapTable(b []byte, pmID string, installed bool, withSummary bool) []manager.Package {
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(b))
	first := true
	for s.Scan() {
		line := s.Text()
		if first && strings.HasPrefix(line, "Name") { first = false; continue }
		first = false
		if strings.TrimSpace(line) == "" { continue }
		f := strings.Fields(line)
		if len(f) < 2 { continue }
		p := manager.Package{Name: f[0], Version: f[1], Manager: pmID, Installed: installed}
		if withSummary && len(f) > 4 {
			p.Description = strings.Join(f[4:], " ")
		}
		out = append(out, p)
	}
	return out
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"info", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	p := manager.Package{Name: name, Manager: a.ID()}
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "summary:"):
			p.Description = strings.TrimSpace(strings.TrimPrefix(line, "summary:"))
		case strings.HasPrefix(line, "publisher:"):
			p.Extra = map[string]string{"publisher": strings.TrimSpace(strings.TrimPrefix(line, "publisher:"))}
		}
	}
	return p, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpInstall, append([]string{"install"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpUninstall, append([]string{"remove"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"refresh"}
	if len(names) > 0 { args = append(args, names...) }
	return a.exec(ctx, manager.OpUpdate, args...)
}
func (a *Adapter) exec(ctx context.Context, op manager.Op, args ...string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args, Sudo: a.NeedsSudo(op)})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
```

- [ ] **Step 4: Register, test, commit**

```bash
go test ./internal/manager/snap/... -v
git add internal/manager/snap/ cmd/pkgr/managers.go
git commit -m "feat(snap): Snap adapter with table parsing"
```

---

### Task 5: flatpak adapter (linux, no sudo, user)

**Files:**
- Create: `internal/manager/flatpak/{flatpak.go, flatpak_test.go, testdata/{search.txt, list.txt, outdated.txt}}`

- [ ] **Step 1: Fixtures**

`testdata/search.txt` (output of `flatpak search --columns=name,application,version,description ripgrep`):
```
ripgrep	com.github.BurntSushi.ripgrep	14.1.0	Fast grep alternative
```

`testdata/list.txt` (`flatpak list --app --columns=name,application,version`):
```
GIMP	org.gimp.GIMP	2.10.36
LibreOffice	org.libreoffice.LibreOffice	24.2.4
```

`testdata/outdated.txt` (`flatpak remote-ls --updates --columns=name,application`):
```
GIMP	org.gimp.GIMP
```

- [ ] **Step 2: Test `internal/manager/flatpak/flatpak_test.go`**

```go
package flatpak

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

func TestFlatpakSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"flatpak search --columns=name,application,version,description ripgrep": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "flatpak"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 1 || pkgs[0].Name != "ripgrep" { t.Fatalf("%+v", pkgs) }
}

func TestFlatpakList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"flatpak list --app --columns=name,application,version": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "flatpak"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 { t.Fatalf("len=%d", len(pkgs)) }
}

func TestFlatpakOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"flatpak remote-ls --updates --columns=name,application": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "flatpak"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 { t.Fatalf("%+v", pkgs) }
}
```

- [ ] **Step 3: Implement `internal/manager/flatpak/flatpak.go`**

```go
// Package flatpak is the Flatpak adapter (per-user install scope).
package flatpak

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "flatpak"} }

func (a *Adapter) ID() string                { return "flatpak" }
func (a *Adapter) DisplayName() string       { return "Flatpak" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func tsv(b []byte) [][]string {
	var rows [][]string
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := strings.TrimRight(s.Text(), "\r")
		if line == "" { continue }
		rows = append(rows, strings.Split(line, "\t"))
	}
	return rows
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", "--columns=name,application,version,description", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range tsv(res.Stdout) {
		if len(row) < 2 { continue }
		p := manager.Package{Name: row[0], Manager: a.ID(), Extra: map[string]string{"app_id": row[1]}}
		if len(row) > 2 { p.Version = row[2] }
		if len(row) > 3 { p.Description = row[3] }
		out = append(out, p)
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--app", "--columns=name,application,version"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range tsv(res.Stdout) {
		if len(row) < 2 { continue }
		p := manager.Package{Name: row[0], Manager: a.ID(), Installed: true, Extra: map[string]string{"app_id": row[1]}}
		if len(row) > 2 { p.Version = row[2] }
		out = append(out, p)
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"remote-ls", "--updates", "--columns=name,application"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range tsv(res.Stdout) {
		if len(row) < 1 { continue }
		out = append(out, manager.Package{Name: row[0], Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"info", "--show-summary", "--show-permissions", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	return manager.Package{Name: name, Manager: a.ID(), Description: string(res.Stdout)}, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpInstall, append([]string{"install", "-y", "--user"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpUninstall, append([]string{"uninstall", "-y"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update", "-y"}
	if len(names) > 0 { args = append(args, names...) }
	return a.exec(ctx, manager.OpUpdate, args...)
}
func (a *Adapter) exec(ctx context.Context, op manager.Op, args ...string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

- [ ] **Step 4: Register, test, commit**

```bash
go test ./internal/manager/flatpak/... -v
git add internal/manager/flatpak/ cmd/pkgr/managers.go
git commit -m "feat(flatpak): Flatpak adapter using TSV columns and --user install"
```

---

### Task 6: nix adapter (`nix profile`)

**Files:**
- Create: `internal/manager/nix/{nix.go, nix_test.go, testdata/{search.json, list.json}}`

- [ ] **Step 1: Fixtures**

`testdata/search.json` (`nix search nixpkgs ripgrep --json`):
```json
{
  "legacyPackages.x86_64-linux.ripgrep": {
    "pname": "ripgrep",
    "version": "14.1.0",
    "description": "Search like grep written in Rust"
  }
}
```

`testdata/list.json` (`nix profile list --json`):
```json
{
  "elements": [
    {"attrPath": "legacyPackages.x86_64-linux.ripgrep", "active": true, "storePaths": ["/nix/store/...ripgrep-14.1.0"]}
  ]
}
```

- [ ] **Step 2: Test `internal/manager/nix/nix_test.go`**

```go
package nix

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

func TestSearchNix(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"nix search nixpkgs ripgrep --json": {Stdout: fx(t, "search.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "nix"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 1 || pkgs[0].Version != "14.1.0" { t.Fatalf("%+v", pkgs) }
}

func TestListNix(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"nix profile list --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "nix"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 1 || pkgs[0].Name == "" { t.Fatalf("%+v", pkgs) }
}
```

- [ ] **Step 3: Implement `internal/manager/nix/nix.go`**

```go
// Package nix wraps Nix's modern CLI (nix profile).
package nix

import (
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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "nix"} }

func (a *Adapter) ID() string                { return "nix" }
func (a *Adapter) DisplayName() string       { return "Nix" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", "nixpkgs", q, "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	body := map[string]struct {
		Pname       string `json:"pname"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}{}
	if err := json.Unmarshal(res.Stdout, &body); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for attr, v := range body {
		out = append(out, manager.Package{
			Name: v.Pname, Version: v.Version, Description: v.Description,
			Manager: a.ID(), Extra: map[string]string{"attr": attr},
		})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"profile", "list", "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var body struct {
		Elements []struct {
			AttrPath   string   `json:"attrPath"`
			StorePaths []string `json:"storePaths"`
		} `json:"elements"`
	}
	if err := json.Unmarshal(res.Stdout, &body); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, e := range body.Elements {
		name := e.AttrPath
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			name = name[idx+1:]
		}
		out = append(out, manager.Package{Name: name, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	// nix profile upgrade --dry-run prints upgrade candidates; parse line-by-line.
	res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"profile", "upgrade", "--dry-run", ".*"}})
	var out []manager.Package
	for _, line := range strings.Split(string(res.Stdout), "\n") {
		if strings.Contains(line, "would be replaced by") {
			out = append(out, manager.Package{Name: line, Manager: a.ID(), Installed: true})
		}
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	pkgs, err := a.Search(ctx, name)
	if err != nil || len(pkgs) == 0 {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound}
	}
	return pkgs[0], nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	args := []string{"profile", "install"}
	for _, n := range names { args = append(args, "nixpkgs#"+n) }
	return a.run(ctx, manager.OpInstall, args)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"profile", "remove"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	if len(names) == 0 { return a.run(ctx, manager.OpUpdate, []string{"profile", "upgrade", ".*"}) }
	return a.run(ctx, manager.OpUpdate, append([]string{"profile", "upgrade"}, names...))
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

- [ ] **Step 4: Register, test, commit**

```bash
go test ./internal/manager/nix/... -v
git add internal/manager/nix/ cmd/pkgr/managers.go
git commit -m "feat(nix): Nix profile adapter using flake search + JSON list"
```

---

### Task 7: scoop adapter (Windows, no sudo)

**Files:**
- Create: `internal/manager/scoop/{scoop.go, scoop_test.go, testdata/{search.json, list.json, status.json}}`

- [ ] **Step 1: Fixtures**

`testdata/search.json` (`scoop search <q> --json` from scoop-search plugin; falls back to `scoop search` otherwise):
```json
[
  {"name":"ripgrep","version":"14.1.0","description":"A search tool that combines the usability of ag with the raw speed of grep","bucket":"main"}
]
```

`testdata/list.json` (`scoop list --json` from `scoop-export` plugin or simulated):
```json
[
  {"Name":"ripgrep","Version":"14.1.0","Source":"main"},
  {"Name":"jq","Version":"1.7.1","Source":"main"}
]
```

`testdata/status.json` (`scoop status --json`):
```json
[
  {"Name":"jq","Installed Version":"1.7.0","Latest Version":"1.7.1"}
]
```

- [ ] **Step 2: Test `internal/manager/scoop/scoop_test.go`**

```go
package scoop

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

func TestScoopSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"scoop search ripgrep --json": {Stdout: fx(t, "search.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "scoop"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 1 || pkgs[0].Name != "ripgrep" { t.Fatalf("%+v", pkgs) }
}

func TestScoopList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"scoop list --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "scoop"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 { t.Fatalf("len=%d", len(pkgs)) }
}

func TestScoopStatus(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"scoop status --json": {Stdout: fx(t, "status.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "scoop"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 || pkgs[0].Latest != "1.7.1" { t.Fatalf("%+v", pkgs) }
}
```

- [ ] **Step 3: Implement `internal/manager/scoop/scoop.go`**

```go
// Package scoop is the Windows Scoop adapter.
package scoop

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "scoop"} }

func (a *Adapter) ID() string                { return "scoop" }
func (a *Adapter) DisplayName() string       { return "Scoop" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q, "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var entries []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Bucket      string `json:"bucket"`
	}
	if err := json.Unmarshal(res.Stdout, &entries); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, e := range entries {
		out = append(out, manager.Package{
			Name: e.Name, Version: e.Version, Description: e.Description, Manager: a.ID(),
			Extra: map[string]string{"bucket": e.Bucket},
		})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var entries []struct {
		Name    string `json:"Name"`
		Version string `json:"Version"`
		Source  string `json:"Source"`
	}
	if err := json.Unmarshal(res.Stdout, &entries); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, e := range entries {
		out = append(out, manager.Package{Name: e.Name, Version: e.Version, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"status", "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	var entries []struct {
		Name             string `json:"Name"`
		InstalledVersion string `json:"Installed Version"`
		LatestVersion    string `json:"Latest Version"`
	}
	if err := json.Unmarshal(res.Stdout, &entries); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, e := range entries {
		out = append(out, manager.Package{Name: e.Name, Version: e.InstalledVersion, Latest: e.LatestVersion, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"info", name, "--json"}})
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
	return manager.Package{Name: v.Name, Version: v.Version, Description: v.Description, Homepage: v.Homepage, Manager: a.ID()}, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update"}
	if len(names) > 0 { args = append(args, names...) } else { args = append(args, "*") }
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

- [ ] **Step 4: Register, test, commit**

```bash
go test ./internal/manager/scoop/... -v
git add internal/manager/scoop/ cmd/pkgr/managers.go
git commit -m "feat(scoop): Windows Scoop adapter with JSON outputs"
```

---

### Task 8: choco adapter (Windows, sudo via UAC)

**Files:**
- Create: `internal/manager/choco/{choco.go, choco_test.go, testdata/{search.txt, list.txt, outdated.txt}}`

- [ ] **Step 1: Fixtures**

`testdata/search.txt`:
```
Chocolatey v2.2.2
ripgrep 14.1.0 [Approved]
ripgrep.install 14.1.0 [Approved]
2 packages found.
```

`testdata/list.txt`:
```
Chocolatey v2.2.2
ripgrep 14.1.0
jq 1.7.1
2 packages installed.
```

`testdata/outdated.txt` (`choco outdated -r`):
```
ripgrep|14.0.0|14.1.0|false
```

- [ ] **Step 2: Test `internal/manager/choco/choco_test.go`**

```go
package choco

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

func TestChocoSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"choco search ripgrep -r": {Stdout: []byte("ripgrep|14.1.0\nripgrep.install|14.1.0\n")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "choco"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 2 { t.Fatalf("%+v", pkgs) }
}

func TestChocoList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"choco list -r": {Stdout: []byte("ripgrep|14.1.0\njq|1.7.1\n")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "choco"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 { t.Fatalf("len=%d", len(pkgs)) }
}

func TestChocoOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"choco outdated -r": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "choco"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 || pkgs[0].Latest != "14.1.0" { t.Fatalf("%+v", pkgs) }
}
```

- [ ] **Step 3: Implement `internal/manager/choco/choco.go`**

```go
// Package choco is the Windows Chocolatey adapter.
// We use `-r` (limited output) which prints pipe-delimited lines.
package choco

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "choco"} }

func (a *Adapter) ID() string           { return "choco" }
func (a *Adapter) DisplayName() string  { return "Chocolatey" }
func (a *Adapter) OSes() []manager.OS   { return []manager.OS{manager.Windows} }
func (a *Adapter) Scope() manager.Scope { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(op manager.Op) bool {
	return op == manager.OpInstall || op == manager.OpUninstall || op == manager.OpUpdate
}
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func parsePipe(b []byte) [][]string {
	var rows [][]string
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := s.Text()
		if line == "" { continue }
		rows = append(rows, strings.Split(line, "|"))
	}
	return rows
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q, "-r"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range parsePipe(res.Stdout) {
		if len(row) < 2 { continue }
		out = append(out, manager.Package{Name: row[0], Version: row[1], Manager: a.ID()})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "-r"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range parsePipe(res.Stdout) {
		if len(row) < 2 { continue }
		out = append(out, manager.Package{Name: row[0], Version: row[1], Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated", "-r"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range parsePipe(res.Stdout) {
		if len(row) < 3 { continue }
		out = append(out, manager.Package{Name: row[0], Version: row[1], Latest: row[2], Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"info", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	return manager.Package{Name: name, Manager: a.ID(), Description: string(res.Stdout)}, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install", "-y"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall", "-y"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade", "-y"}
	if len(names) > 0 { args = append(args, names...) } else { args = append(args, "all") }
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args, Sudo: a.NeedsSudo(op)})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

- [ ] **Step 4: Register, test, commit**

```bash
go test ./internal/manager/choco/... -v
git add internal/manager/choco/ cmd/pkgr/managers.go
git commit -m "feat(choco): Chocolatey adapter with pipe-delimited -r output"
```

---

### Task 9: winget adapter (Windows, no sudo)

**Files:**
- Create: `internal/manager/winget/{winget.go, winget_test.go, testdata/{search.txt, list.txt, upgrade.txt}}`

- [ ] **Step 1: Fixtures** — winget output is column-formatted; capture verbatim, parser must handle column widths.

`testdata/search.txt`:
```
Name                   Id                              Version   Source
-------------------------------------------------------------------------
Ripgrep                BurntSushi.ripgrep              14.1.0    winget
PowerShell             Microsoft.PowerShell            7.4.1.0   winget
```

`testdata/list.txt`:
```
Name                Id                  Version       Available    Source
-----------------------------------------------------------------------------
Ripgrep             BurntSushi.ripgrep  14.0.0        14.1.0       winget
Git                 Git.Git             2.45.0                     winget
```

`testdata/upgrade.txt`:
```
Name      Id                  Version    Available    Source
---------------------------------------------------------------
Ripgrep   BurntSushi.ripgrep  14.0.0     14.1.0       winget
```

- [ ] **Step 2: Test `internal/manager/winget/winget_test.go`**

```go
package winget

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

func TestWingetSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"winget search ripgrep": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "winget"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 2 { t.Fatalf("%+v", pkgs) }
}

func TestWingetList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"winget list": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "winget"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 { t.Fatalf("len=%d", len(pkgs)) }
}

func TestWingetUpgrade(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"winget upgrade": {Stdout: fx(t, "upgrade.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "winget"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 || pkgs[0].Latest != "14.1.0" { t.Fatalf("%+v", pkgs) }
}
```

- [ ] **Step 3: Implement `internal/manager/winget/winget.go`**

```go
// Package winget is the Windows winget adapter. winget prints a fixed-column
// table; we detect column widths from the header row.
package winget

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "winget"} }

func (a *Adapter) ID() string                { return "winget" }
func (a *Adapter) DisplayName() string       { return "winget" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

type col struct{ name string; start, end int }

func parseHeader(line string) []col {
	var cols []col
	// detect spans of non-space; each span is a column header.
	i := 0
	for i < len(line) {
		for i < len(line) && line[i] == ' ' { i++ }
		start := i
		for i < len(line) && line[i] != ' ' { i++ }
		if start == i { break }
		cols = append(cols, col{name: line[start:i], start: start, end: i})
	}
	return cols
}

// cell extracts the value for column c from line, honoring the column start
// and stopping at the next column's start.
func cell(line string, cols []col, idx int) string {
	if idx >= len(cols) { return "" }
	s := cols[idx].start
	if s >= len(line) { return "" }
	e := len(line)
	if idx+1 < len(cols) { e = cols[idx+1].start }
	if e > len(line) { e = len(line) }
	return strings.TrimSpace(line[s:e])
}

func parseTable(b []byte) ([]col, []string) {
	var lines []string
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := s.Text()
		if line == "" { continue }
		lines = append(lines, line)
	}
	// find header line (one with "Name" at start)
	headerIdx := -1
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "Name ") || strings.HasPrefix(strings.TrimSpace(l), "Name\t") {
			headerIdx = i; break
		}
	}
	if headerIdx < 0 { return nil, nil }
	cols := parseHeader(lines[headerIdx])
	// skip separator
	return cols, lines[headerIdx+2:]
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	cols, body := parseTable(res.Stdout)
	if cols == nil { return nil, nil }
	idx := indexOf(cols, "Name"); idIdx := indexOf(cols, "Id"); verIdx := indexOf(cols, "Version")
	var out []manager.Package
	for _, l := range body {
		name := cell(l, cols, idx)
		if name == "" { continue }
		out = append(out, manager.Package{
			Name: name, Version: cell(l, cols, verIdx), Manager: a.ID(),
			Extra: map[string]string{"id": cell(l, cols, idIdx)},
		})
	}
	return out, nil
}

func indexOf(cols []col, name string) int {
	for i, c := range cols { if c.name == name { return i } }
	return -1
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	cols, body := parseTable(res.Stdout)
	if cols == nil { return nil, nil }
	var out []manager.Package
	for _, l := range body {
		name := cell(l, cols, indexOf(cols, "Name"))
		if name == "" { continue }
		out = append(out, manager.Package{
			Name:    name,
			Version: cell(l, cols, indexOf(cols, "Version")),
			Latest:  cell(l, cols, indexOf(cols, "Available")),
			Manager: a.ID(), Installed: true,
		})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"upgrade"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	cols, body := parseTable(res.Stdout)
	if cols == nil { return nil, nil }
	var out []manager.Package
	for _, l := range body {
		name := cell(l, cols, indexOf(cols, "Name"))
		if name == "" { continue }
		out = append(out, manager.Package{
			Name: name, Version: cell(l, cols, indexOf(cols, "Version")),
			Latest: cell(l, cols, indexOf(cols, "Available")), Manager: a.ID(), Installed: true,
		})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"show", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	return manager.Package{Name: name, Manager: a.ID(), Description: string(res.Stdout)}, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade"}
	if len(names) == 0 { args = append(args, "--all") } else { args = append(args, names...) }
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

- [ ] **Step 4: Register, test, commit**

```bash
go test ./internal/manager/winget/... -v
git add internal/manager/winget/ cmd/pkgr/managers.go
git commit -m "feat(winget): Windows winget adapter with column-aware table parser"
```

---

### Task 10: mas adapter (Mac App Store)

**Files:**
- Create: `internal/manager/mas/{mas.go, mas_test.go, testdata/{search.txt, list.txt, outdated.txt}}`

- [ ] **Step 1: Fixtures**

`testdata/search.txt`:
```
   441258766  Magnet  (2.13.0)
   1295203466 Microsoft Remote Desktop  (10.9.4)
```

`testdata/list.txt`:
```
441258766  Magnet  (2.13.0)
497799835  Xcode   (15.4)
```

`testdata/outdated.txt`:
```
497799835  Xcode   (15.4 -> 15.5)
```

- [ ] **Step 2: Test + Implement together (similar pattern)**

`internal/manager/mas/mas_test.go`:
```go
package mas

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

func TestMasSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"mas search Magnet": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "mas"}
	pkgs, _ := a.Search(context.Background(), "Magnet")
	if len(pkgs) != 2 { t.Fatalf("%+v", pkgs) }
}
func TestMasList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"mas list": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "mas"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 { t.Fatalf("len=%d", len(pkgs)) }
}
func TestMasOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"mas outdated": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "mas"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 || pkgs[0].Latest != "15.5" { t.Fatalf("%+v", pkgs) }
}
```

`internal/manager/mas/mas.go`:
```go
// Package mas wraps the Mac App Store CLI (mas).
package mas

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "mas"} }

func (a *Adapter) ID() string                { return "mas" }
func (a *Adapter) DisplayName() string       { return "Mac App Store" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

var rowRe = regexp.MustCompile(`^\s*(\d+)\s+(.+?)\s+\((.+)\)\s*$`)

func parseMas(b []byte, pmID string, installed bool, outdated bool) []manager.Package {
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		m := rowRe.FindStringSubmatch(s.Text())
		if len(m) != 4 { continue }
		p := manager.Package{Name: strings.TrimSpace(m[2]), Manager: pmID, Installed: installed, Extra: map[string]string{"appid": m[1]}}
		if outdated {
			parts := strings.Split(m[3], " -> ")
			if len(parts) == 2 { p.Version = parts[0]; p.Latest = parts[1] }
		} else {
			p.Version = m[3]
		}
		out = append(out, p)
	}
	return out
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err} }
	return parseMas(res.Stdout, a.ID(), false, false), nil
}
func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	return parseMas(res.Stdout, a.ID(), true, false), nil
}
func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err} }
	return parseMas(res.Stdout, a.ID(), true, true), nil
}
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	// mas install requires app IDs, not names. Caller must pass numeric IDs.
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
go test ./internal/manager/mas/... -v
git add internal/manager/mas/ cmd/pkgr/managers.go
git commit -m "feat(mas): Mac App Store adapter via mas CLI"
```

---

### Task 11: pnpm adapter (global)

**Files:**
- Create: `internal/manager/pnpm/{pnpm.go, pnpm_test.go, testdata/{list.json, outdated.json}}`

pnpm's `search` is registry-bound; pnpm doesn't ship a `search` subcommand so we reuse npm's search endpoint via the registry HTTP API (or shell out to `npm search`). For symmetry we shell out to the pnpm CLI for everything mutating + list, and use `npm` for search (a transitive runtime dep that's typically present alongside).

- [ ] **Step 1: Fixtures**

`testdata/list.json`:
```json
[
  {"name": "global", "dependencies": {"typescript": {"version": "5.4.2"}, "prettier": {"version": "3.2.5"}}}
]
```

`testdata/outdated.json`:
```json
{
  "typescript": {"current": "5.4.2", "latest": "5.5.4"}
}
```

- [ ] **Step 2: Test `internal/manager/pnpm/pnpm_test.go`**

```go
package pnpm

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

func TestPnpmList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pnpm list -g --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pnpm"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 { t.Fatalf("len=%d", len(pkgs)) }
}

func TestPnpmOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pnpm outdated -g --format json": {Stdout: fx(t, "outdated.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pnpm"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 { t.Fatalf("%+v", pkgs) }
}
```

- [ ] **Step 3: Implement `internal/manager/pnpm/pnpm.go`**

```go
// Package pnpm wraps pnpm for global packages.
package pnpm

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "pnpm"} }

func (a *Adapter) ID() string                { return "pnpm" }
func (a *Adapter) DisplayName() string       { return "pnpm" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	// pnpm has no search command; defer to `npm search --json` if available.
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: "npm", Args: []string{"search", "--json", q}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err} }
	var entries []struct {
		Name string `json:"name"`; Version string `json:"version"`
		Description string `json:"description"`
		Links struct{ Homepage string `json:"homepage"` } `json:"links"`
	}
	if err := json.Unmarshal(res.Stdout, &entries); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	out := make([]manager.Package, 0, len(entries))
	for _, e := range entries {
		out = append(out, manager.Package{Name: e.Name, Version: e.Version, Description: e.Description, Homepage: e.Links.Homepage, Manager: a.ID()})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "-g", "--json"}})
	if err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err} }
	var arr []struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(res.Stdout, &arr); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, e := range arr {
		for name, d := range e.Dependencies {
			out = append(out, manager.Package{Name: name, Version: d.Version, Manager: a.ID(), Installed: true})
		}
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated", "-g", "--format", "json"}})
	if len(res.Stdout) == 0 { return nil, nil }
	var m map[string]struct {
		Current string `json:"current"`; Latest string `json:"latest"`
	}
	if err := json.Unmarshal(res.Stdout, &m); err != nil { return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeParseError, Err: err} }
	var out []manager.Package
	for name, v := range m {
		out = append(out, manager.Package{Name: name, Version: v.Current, Latest: v.Latest, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"add", "-g"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"remove", "-g"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update", "-g"}
	if len(names) > 0 { args = append(args, names...) }
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil { return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err} }
	return nil
}
```

- [ ] **Step 4: Register, test, commit**

```bash
go test ./internal/manager/pnpm/... -v
git add internal/manager/pnpm/ cmd/pkgr/managers.go
git commit -m "feat(pnpm): pnpm adapter (search via npm registry fallback)"
```

---

## Phase 4 Acceptance

- 11 new adapters present under `internal/manager/<id>/`
- Each registered in `cmd/pkgr/managers.go`
- `pkgr pm list` shows all 14 adapters (Phase 2's 3 + Phase 4's 11)
- `pkgr search ripgrep` aggregates whichever are detected on host
- `make test` green; each adapter ≥ 3 unit tests
- `make lint` green
- Sudo-aware adapters (apt, dnf, pacman, snap, choco) mark `Sudo: true` in runner.Cmd for mutating ops, verified via `--dry-run`
