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
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestChocoSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"choco search ripgrep -r": {Stdout: []byte("ripgrep|14.1.0\nripgrep.install|14.1.0\n")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "choco"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 2 {
		t.Fatalf("%+v", pkgs)
	}
}

func TestChocoList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"choco list -r": {Stdout: []byte("ripgrep|14.1.0\njq|1.7.1\n")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "choco"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 {
		t.Fatalf("len=%d", len(pkgs))
	}
}

func TestChocoOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"choco outdated -r": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "choco"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 || pkgs[0].Latest != "14.1.0" {
		t.Fatalf("%+v", pkgs)
	}
}
