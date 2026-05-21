package conda

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

func TestCondaSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"conda search requests --json": {Stdout: fx(t, "search.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "conda"}
	got, _ := a.Search(context.Background(), "requests")
	if len(got) < 1 {
		t.Fatalf("expected ≥1")
	}
}

func TestCondaList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"conda list --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "conda"}
	got, _ := a.List(context.Background())
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
}
