// Package log configures slog with JSON output to a rotating file,
// optionally mirroring to stderr.
package log

import (
	"io"
	"log/slog"
	"os"
)

type Options struct {
	Path    string
	Verbose bool
	Stderr  io.Writer // injected for tests; defaults to os.Stderr
}

// Setup returns a configured slog.Logger and a closer that flushes
// and closes underlying file handles. The log file is rotated at 5 MB
// and up to 5 backups are kept.
func Setup(opts Options) (*slog.Logger, func() error, error) {
	rot := &Rotator{Path: opts.Path, MaxBytes: 5 << 20, MaxFiles: 5}

	var w io.Writer = rot
	if opts.Verbose {
		stderr := opts.Stderr
		if stderr == nil {
			stderr = os.Stderr
		}
		w = io.MultiWriter(rot, stderr)
	}

	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := slog.New(h)

	return l, rot.Close, nil
}
