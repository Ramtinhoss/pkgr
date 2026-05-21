package rustup

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

func TestRustupList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"rustup toolchain list": {Stdout: fx(t, "toolchains.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "rustup"}
	got, _ := a.List(context.Background())
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestRustupOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"rustup check": {Stdout: fx(t, "list_remote.txt")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "rustup"}
	got, _ := a.Outdated(context.Background())
	if len(got) != 1 || got[0].Latest != "1.78.0" {
		t.Fatalf("%+v", got)
	}
}
