package format

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

func TestHumanSearchRendersTable(t *testing.T) {
	pkgs := []manager.Package{
		{Name: "ripgrep", Version: "14.1.0", Manager: "brew", Description: "Search like grep"},
		{Name: "ripgrep", Version: "14.0.0", Manager: "cargo"},
	}
	var buf bytes.Buffer
	if err := HumanSearch(&buf, pkgs); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	for _, want := range []string{"ripgrep", "14.1.0", "brew", "cargo"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestJSONSearchOutputsStableSchema(t *testing.T) {
	pkgs := []manager.Package{{Name: "x", Manager: "brew", Version: "1.0"}}
	var buf bytes.Buffer
	if err := JSONResult(&buf, pkgs, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"packages"`) || !strings.Contains(buf.String(), `"errors"`) {
		t.Fatalf("missing keys: %s", buf.String())
	}
}
