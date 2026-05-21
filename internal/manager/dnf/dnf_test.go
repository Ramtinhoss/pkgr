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
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSearchDnf(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"dnf search ripgrep": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "dnf"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) < 1 {
		t.Fatalf("expected ≥1, got %d", len(pkgs))
	}
}

func TestListDnf(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"dnf list --installed": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "dnf"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 {
		t.Fatalf("len=%d", len(pkgs))
	}
}

func TestOutdatedDnf(t *testing.T) {
	// dnf check-update exits 100 when updates exist.
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"dnf check-update": {Stdout: fx(t, "outdated.txt"), Code: 100},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "dnf"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 2 {
		t.Fatalf("len=%d", len(pkgs))
	}
}
