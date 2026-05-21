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
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

type searchEntry struct {
	Path     string `json:"path"`
	Synopsis string `json:"synopsis"`
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.pkg.go.dev/search?q="+q, nil)
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeNetworkFailure, Err: err}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var arr []searchEntry
	if err := json.Unmarshal(body, &arr); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
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
		if g := os.Getenv("GOPATH"); g != "" {
			dir = filepath.Join(g, "bin")
		}
	}
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	var out []manager.Package
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
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
		if g := os.Getenv("GOPATH"); g != "" {
			dir = filepath.Join(g, "bin")
		}
	}
	for _, n := range names {
		_ = os.Remove(filepath.Join(dir, filepath.Base(n)))
	}
	return nil
}
func (a *Adapter) Update(ctx context.Context, names ...string) error { return a.Install(ctx, names...) }
