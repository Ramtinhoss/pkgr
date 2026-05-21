package pnpm

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

func TestPnpmList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pnpm list -g --json": {Stdout: fx(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pnpm"}
	pkgs, _ := a.List(context.Background())
	if len(pkgs) != 2 {
		t.Fatalf("len=%d", len(pkgs))
	}
}

func TestPnpmOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pnpm outdated -g --format json": {Stdout: fx(t, "outdated.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pnpm"}
	pkgs, _ := a.Outdated(context.Background())
	if len(pkgs) != 1 {
		t.Fatalf("%+v", pkgs)
	}
}
