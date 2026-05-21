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
