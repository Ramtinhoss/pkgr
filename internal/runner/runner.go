package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Cmd describes a command to execute.
type Cmd struct {
	Bin   string
	Args  []string
	Env   []string
	Sudo  bool
	Stdin io.Reader
}

// Result is the outcome of command execution.
type Result struct {
	Stdout  []byte
	Stderr  []byte
	Code    int
	Skipped bool
}

// ExecFunc is a function that executes a command.
type ExecFunc func(ctx context.Context, c Cmd) (Result, error)

// Runner manages command execution with dry-run support and injectable Exec.
type Runner struct {
	Out    io.Writer // Output sink for dry-run messages
	DryRun bool      // If true, print but don't execute
	Exec   ExecFunc  // Command executor; nil means RealExec
}

// Run executes the command with dry-run and error handling.
func (r *Runner) Run(ctx context.Context, c Cmd) (Result, error) {
	if r.DryRun {
		// Format: "→ would exec: [sudo ]<bin> <args>"
		prefix := ""
		if c.Sudo {
			prefix = "sudo "
		}
		fmt.Fprintf(r.Out, "→ would exec: %s%s %s\n", prefix, c.Bin, strings.Join(c.Args, " "))
		return Result{Skipped: true}, nil
	}

	execFn := r.Exec
	if execFn == nil {
		execFn = RealExec
	}

	result, err := execFn(ctx, c)
	if err != nil {
		return result, err
	}

	if result.Code != 0 {
		return result, fmt.Errorf("command exited with code %d", result.Code)
	}

	return result, nil
}

// RealExec actually executes the command using os/exec.
func RealExec(ctx context.Context, c Cmd) (Result, error) {
	cmd := exec.CommandContext(ctx, c.Bin, c.Args...)

	if len(c.Env) > 0 {
		cmd.Env = append(os.Environ(), c.Env...)
	}

	if c.Stdin != nil {
		cmd.Stdin = c.Stdin
	}

	// Capture stdout and stderr in buffers
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
		Code:   0,
	}

	// Extract exit code
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.Code = exitErr.ExitCode()
		} else {
			return result, err
		}
	}

	return result, nil
}
