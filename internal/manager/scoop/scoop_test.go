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
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestScoopSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"scoop search ripgrep --json": {Stdout: fx(t, "search.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "scoop"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 1 || pkgs[0].Name != "ripgrep" {
		t.Fatalf("%+v", pkgs)
	}
}

func TestScoopList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"scoop list --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "scoop"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 {
		t.Fatalf("len=%d", len(pkgs))
	}
}

func TestScoopStatus(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"scoop status --json": {Stdout: fx(t, "status.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "scoop"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 || pkgs[0].Latest != "1.7.1" {
		t.Fatalf("%+v", pkgs)
	}
}
