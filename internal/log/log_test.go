package log

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupWritesJSONToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgr.log")

	l, closer, err := Setup(Options{Path: path, Verbose: false})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer closer()

	l.Info("hello", slog.String("k", "v"))

	if err := closer(); err != nil {
		t.Fatalf("close: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(b), `"msg":"hello"`) {
		t.Fatalf("missing msg: %s", b)
	}
	if !strings.Contains(string(b), `"k":"v"`) {
		t.Fatalf("missing attr: %s", b)
	}
}

func TestSetupVerboseMirrorsToStderr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgr.log")
	var buf bytes.Buffer

	opts := Options{Path: path, Verbose: true, Stderr: &buf}
	l, closer, err := Setup(opts)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	l.Info("mirror")
	_ = closer()

	if !strings.Contains(buf.String(), "mirror") {
		t.Fatalf("expected stderr mirror, got %q", buf.String())
	}
}
