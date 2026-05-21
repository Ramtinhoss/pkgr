// Package pip wraps pip + PyPI for search.
package pip

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/manager/httpquery"
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
	body, status, err := httpquery.New(a.HTTP).Get(ctx, url)
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeNetworkFailure, Err: err}
	}
	if status == 404 {
		return nil, nil
	}
	if status >= 400 {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeNetworkFailure, Err: fmt.Errorf("pypi %d", status)}
	}
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
