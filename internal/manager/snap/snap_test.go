package snap

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

func TestSnapSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"snap find ripgrep": {Stdout: fx(t, "find.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "snap"}
	got, _ := a.Search(context.Background(), "ripgrep")
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestSnapList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"snap list": {Stdout: fx(t, "list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "snap"}
	got, _ := a.List(context.Background())
	if len(got) != 3 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestSnapOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"snap refresh --list": {Stdout: fx(t, "refresh.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "snap"}
	got, _ := a.Outdated(context.Background())
	if len(got) != 1 {
		t.Fatalf("%+v", got)
	}
}
