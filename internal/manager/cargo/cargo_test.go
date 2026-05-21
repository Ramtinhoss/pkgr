package cargo

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

func TestCargoSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"cargo search ripgrep --limit 25": {Stdout: fx(t, "search.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "cargo"}
	got, _ := a.Search(context.Background(), "ripgrep")
	if len(got) != 2 || got[0].Name != "ripgrep" {
		t.Fatalf("%+v", got)
	}
}

func TestCargoList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"cargo install --list": {Stdout: fx(t, "installed.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "cargo"}
	got, _ := a.List(context.Background())
	if len(got) != 2 || got[0].Version == "" {
		t.Fatalf("%+v", got)
	}
}
