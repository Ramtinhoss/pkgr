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
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSearchPacman(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pacman -Ss ripgrep": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pacman"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 2 {
		t.Fatalf("got %+v", pkgs)
	}
}

func TestListPacman(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pacman -Q": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pacman"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 {
		t.Fatalf("len=%d", len(pkgs))
	}
}

func TestOutdatedPacman(t *testing.T) {
	// pacman -Qu exits 1 when there are no upgrades; we test the upgrade case.
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pacman -Qu": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pacman"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 || pkgs[0].Latest != "8.6.0-1" {
		t.Fatalf("%+v", pkgs)
	}
}
