package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Rotator implements io.Writer with size-based rotation.
// When the current file exceeds MaxBytes, it is renamed to .1 and a fresh file
// is opened. Old backups beyond MaxFiles are deleted.
type Rotator struct {
	Path     string
	MaxBytes int64
	MaxFiles int

	mu sync.Mutex
	f  *os.File
	n  int64
}

func (r *Rotator) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		if err := r.open(); err != nil {
			return 0, err
		}
	}
	if r.n+int64(len(p)) > r.MaxBytes {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := r.f.Write(p)
	r.n += int64(n)
	return n, err
}

func (r *Rotator) open() error {
	if err := os.MkdirAll(filepath.Dir(r.Path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(r.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	r.f = f
	info, _ := f.Stat()
	if info != nil {
		r.n = info.Size()
	}
	return nil
}

func (r *Rotator) rotate() error {
	if r.f != nil {
		_ = r.f.Close()
		r.f = nil
	}
	// Shift: pkgr.log.(N-1) → pkgr.log.N, pkgr.log → pkgr.log.1
	for i := r.MaxFiles - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", r.Path, i)
		dst := fmt.Sprintf("%s.%d", r.Path, i+1)
		_ = os.Rename(src, dst)
	}
	_ = os.Rename(r.Path, r.Path+".1")
	r.n = 0

	// Trim backup files numbered > MaxFiles-1 (base counts as 1 slot).
	matches, _ := filepath.Glob(r.Path + ".*")
	maxBackups := r.MaxFiles - 1
	for _, m := range matches {
		ext := m[len(r.Path):]          // e.g. ".3"
		if !strings.HasPrefix(ext, ".") {
			continue
		}
		num, err := strconv.Atoi(ext[1:])
		if err != nil {
			continue
		}
		if num > maxBackups {
			_ = os.Remove(m)
		}
	}
	return r.open()
}

// Close flushes and closes the underlying file.
func (r *Rotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return nil
	}
	err := r.f.Close()
	r.f = nil
	return err
}
