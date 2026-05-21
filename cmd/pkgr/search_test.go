package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSearchCommandDryRun(t *testing.T) {
	var out bytes.Buffer
	root := newRootCmd(buildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--dry-run", "search", "ripgrep"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "would exec:") &&
		!strings.Contains(got, "NAME") &&
		!strings.Contains(got, "no canned reply") &&
		!strings.Contains(got, "exec brew") {
		t.Fatalf("unexpected output: %s", got)
	}
}
