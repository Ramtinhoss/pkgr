package yarn

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

func TestYarnList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"yarn global list --depth=0": {Stdout: fx(t, "global_list.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "yarn"}
	got, _ := a.List(context.Background())
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
}
