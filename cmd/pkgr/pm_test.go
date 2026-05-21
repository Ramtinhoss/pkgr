package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPMListJSON(t *testing.T) {
	var out bytes.Buffer
	root := newRootCmd(buildInfo{Version: "test"})
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--json", "pm", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var v any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &v); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out.String())
	}
	// Verify it's an array of objects with the expected keys.
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("expected JSON array, got %T", v)
	}
	if len(arr) == 0 {
		t.Fatal("expected at least one adapter in the list")
	}
	first, ok := arr[0].(map[string]any)
	if !ok {
		t.Fatalf("expected JSON object element, got %T", arr[0])
	}
	for _, key := range []string{"id", "scope", "detected", "enabled"} {
		if _, exists := first[key]; !exists {
			t.Errorf("missing key %q in first element", key)
		}
	}
}
