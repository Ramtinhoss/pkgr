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
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"apt-cache search ripgrep": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "apt-cache"}
	pkgs, err := a.Search(context.Background(), "ripgrep")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 2 || pkgs[0].Name != "ripgrep" {
		t.Fatalf("%+v", pkgs)
	}
}

func TestList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"apt list --installed": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "apt"}
	pkgs, err := a.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("len=%d", len(pkgs))
	}
}

func TestOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"apt list --upgradable": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "apt"}
	pkgs, err := a.Outdated(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].Latest == "" {
		t.Fatalf("%+v", pkgs)
	}
}
