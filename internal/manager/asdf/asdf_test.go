package asdf

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

func TestAsdfSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"asdf plugin list all": {Stdout: fx(t, "plugin_list_all.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "asdf"}
	pkgs, _ := a.Search(context.Background(), "node")
	if len(pkgs) < 1 {
		t.Fatalf("expected ≥1, got %d", len(pkgs))
	}
}

func TestAsdfList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"asdf list": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "asdf"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) < 1 {
		t.Fatalf("len=%d", len(pkgs))
	}
}
