package runner

import (
	"context"
	"fmt"
	"strings"
)

// FakeResult is a predefined result for a command.
type FakeResult struct {
	Stdout []byte
	Stderr []byte
	Code   int
}

// Fake is a test fixture that returns deterministic responses.
type Fake struct {
	Returns map[string]FakeResult // Keyed by "<bin> <space-joined args>"
	Calls   []string              // Record of all calls made
}

// Exec returns a deterministic response from the Returns map.
func (f *Fake) Exec(ctx context.Context, c Cmd) (Result, error) {
	// Build key from bin and args
	parts := []string{c.Bin}
	parts = append(parts, c.Args...)
	key := strings.Join(parts, " ")

	// Record call
	f.Calls = append(f.Calls, key)

	// Look up in Returns
	if fr, ok := f.Returns[key]; ok {
		result := Result{
			Stdout: fr.Stdout,
			Stderr: fr.Stderr,
			Code:   fr.Code,
		}
		if fr.Code != 0 {
			return result, fmt.Errorf("command exited with code %d", fr.Code)
		}
		return result, nil
	}

	// Not found; return error
	return Result{}, fmt.Errorf("fake runner: no canned reply for %q", key)
}
