package npm

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)

func fix(t *testing.T, name string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil { t.Fatal(err) }
	return b
}

func TestSearch(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"npm search --json react": {Stdout: fix(t, "search.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "npm"}
	got, err := a.Search(context.Background(), "react")
	if err != nil { t.Fatal(err) }
	if len(got) != 2 || got[0].Name != "react" || got[0].Version != "18.3.1" {
		t.Fatalf("got %+v", got)
	}
}

func TestListGlobal(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"npm list -g --depth=0 --json": {Stdout: fix(t, "list_global.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "npm"}
	got, err := a.List(context.Background())
	if err != nil { t.Fatal(err) }
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
}

func TestOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"npm outdated -g --json": {Stdout: fix(t, "outdated.json"), Code: 1}, // npm exits non-zero when outdated exist
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "npm"}
	got, err := a.Outdated(context.Background())
	if err != nil { t.Fatal(err) }
	if len(got) != 1 || got[0].Name != "typescript" || got[0].Latest != "5.5.4" {
		t.Fatalf("got %+v", got)
	}
}
