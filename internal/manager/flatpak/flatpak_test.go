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
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestFlatpakSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"flatpak search --columns=name,application,version,description ripgrep": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "flatpak"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 1 || pkgs[0].Name != "ripgrep" {
		t.Fatalf("%+v", pkgs)
	}
}

func TestFlatpakList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"flatpak list --app --columns=name,application,version": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "flatpak"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 {
		t.Fatalf("len=%d", len(pkgs))
	}
}

func TestFlatpakOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"flatpak remote-ls --updates --columns=name,application": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "flatpak"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 {
		t.Fatalf("%+v", pkgs)
	}
}
