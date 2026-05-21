package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatorTrimsFiles(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "pkgr.log")

	r := &Rotator{Path: base, MaxBytes: 100, MaxFiles: 3}
	for i := 0; i < 5; i++ {
		_, err := r.Write([]byte(strings.Repeat("x", 60)))
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "pkgr.log*"))
	if len(files) > 3 {
		t.Fatalf("expected ≤3 files, got %d (%v)", len(files), files)
	}
	if _, err := os.Stat(base); err != nil {
		t.Fatalf("base file missing: %v", err)
	}
}

func TestRotatorConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "pkgr.log")

	r := &Rotator{Path: base, MaxBytes: 200, MaxFiles: 3}
	defer r.Close()

	done := make(chan struct{})
	for i := 0; i < 4; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 5; j++ {
				if _, err := r.Write([]byte(strings.Repeat("y", 40))); err != nil {
					t.Errorf("concurrent Write: %v", err)
					return
				}
			}
		}()
	}
	for i := 0; i < 4; i++ {
		<-done
	}
}
