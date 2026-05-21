package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommandPrints(t *testing.T) {
	var out bytes.Buffer
	root := newRootCmd(buildInfo{Version: "1.2.3", Commit: "abc", Date: "2026-01-01"})
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got := out.String()
	for _, want := range []string{"1.2.3", "abc", "2026-01-01"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %s", want, got)
		}
	}
}
