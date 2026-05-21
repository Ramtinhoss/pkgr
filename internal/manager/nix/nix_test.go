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
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSearchNix(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"nix search nixpkgs ripgrep --json": {Stdout: fx(t, "search.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "nix"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 1 || pkgs[0].Version != "14.1.0" {
		t.Fatalf("%+v", pkgs)
	}
}

func TestListNix(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"nix profile list --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "nix"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 1 || pkgs[0].Name == "" {
		t.Fatalf("%+v", pkgs)
	}
}
