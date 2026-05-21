package gem

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

func TestGemSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"gem search -r rails": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "gem"}
	got, _ := a.Search(context.Background(), "rails")
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestGemList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"gem list": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "gem"}
	got, _ := a.List(context.Background())
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestGemOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"gem outdated": {Stdout: fx(t, "outdated.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "gem"}
	got, _ := a.Outdated(context.Background())
	if len(got) != 1 || got[0].Latest != "2.6.0" {
		t.Fatalf("%+v", got)
	}
}
