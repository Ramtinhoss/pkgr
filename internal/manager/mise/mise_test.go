package mise

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

func TestMiseList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"mise ls --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "mise"}
	got, _ := a.List(context.Background())
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestMiseSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"mise plugins ls-remote": {Stdout: fx(t, "plugins.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "mise"}
	got, _ := a.Search(context.Background(), "py")
	if len(got) != 1 {
		t.Fatalf("%+v", got)
	}
}
