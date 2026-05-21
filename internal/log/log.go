// Package log configures slog with JSON output to a rotating file,
// optionally mirroring to stderr.
package log

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type Options struct {
	Path    string
	Verbose bool
	Stderr  io.Writer // injected for tests; defaults to os.Stderr
}

// Setup returns a configured slog.Logger and a closer that flushes
// and closes underlying file handles.
func Setup(opts Options) (*slog.Logger, func() error, error) {
	if err := os.MkdirAll(filepath.Dir(opts.Path), 0o755); err != nil {
		return nil, nil, err
	}
	f, err := os.OpenFile(opts.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	var w io.Writer = f
	if opts.Verbose {
		stderr := opts.Stderr
		if stderr == nil {
			stderr = os.Stderr
		}
		w = io.MultiWriter(f, stderr)
	}

	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := slog.New(h)

	return l, f.Close, nil
}
