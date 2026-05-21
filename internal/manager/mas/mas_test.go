package mas

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

func TestMasSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"mas search Magnet": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "mas"}
	pkgs, _ := a.Search(context.Background(), "Magnet")
	if len(pkgs) != 2 {
		t.Fatalf("%+v", pkgs)
	}
}
func TestMasList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"mas list": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "mas"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 {
		t.Fatalf("len=%d", len(pkgs))
	}
}
func TestMasOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"mas outdated": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "mas"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 || pkgs[0].Latest != "15.5" {
		t.Fatalf("%+v", pkgs)
	}
}
