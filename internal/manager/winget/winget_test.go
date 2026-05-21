package winget

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

func TestWingetSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"winget search ripgrep": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "winget"}
	pkgs, _ := a.Search(context.Background(), "ripgrep")
	if len(pkgs) != 2 {
		t.Fatalf("%+v", pkgs)
	}
}

func TestWingetList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"winget list": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "winget"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 {
		t.Fatalf("len=%d", len(pkgs))
	}
}

func TestWingetUpgrade(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"winget upgrade": {Stdout: fx(t, "upgrade.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "winget"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 || pkgs[0].Latest != "14.1.0" {
		t.Fatalf("%+v", pkgs)
	}
}
