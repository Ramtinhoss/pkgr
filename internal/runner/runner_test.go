package runner

import (
	"bytes"
	"context"
	"testing"
)

func TestRunDryRunPrintsAndSkips(t *testing.T) {
	out := &bytes.Buffer{}
	r := &Runner{
		Out:    out,
		DryRun: true,
		Exec:   RealExec,
	}

	cmd := Cmd{
		Bin:  "echo",
		Args: []string{"hello"},
	}

	result, err := r.Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	if !result.Skipped {
		t.Errorf("Skipped = %v, want true", result.Skipped)
	}

	output := out.String()
	if output != "→ would exec: echo hello\n" {
		t.Errorf("output = %q, want %q", output, "→ would exec: echo hello\n")
	}
}

func TestRunUsesFakeExecutor(t *testing.T) {
	fake := &Fake{
		Returns: map[string]FakeResult{
			"test": {
				Stdout: []byte("fake output"),
				Code:   0,
			},
		},
	}

	r := &Runner{
		Out:    &bytes.Buffer{},
		DryRun: false,
		Exec:   fake.Exec,
	}

	cmd := Cmd{
		Bin: "test",
	}

	result, err := r.Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	if string(result.Stdout) != "fake output" {
		t.Errorf("Stdout = %q, want %q", string(result.Stdout), "fake output")
	}

	if result.Code != 0 {
		t.Errorf("Code = %d, want 0", result.Code)
	}
}

func TestRunFakeNonZeroExitReturnsError(t *testing.T) {
	fake := &Fake{
		Returns: map[string]FakeResult{
			"fail": {
				Stderr: []byte("command failed"),
				Code:   1,
			},
		},
	}

	r := &Runner{
		Out:    &bytes.Buffer{},
		DryRun: false,
		Exec:   fake.Exec,
	}

	cmd := Cmd{
		Bin: "fail",
	}

	result, err := r.Run(context.Background(), cmd)
	if err == nil {
		t.Fatalf("Run() error = nil, want non-nil")
	}

	if result.Code != 1 {
		t.Errorf("Code = %d, want 1", result.Code)
	}

	if string(result.Stderr) != "command failed" {
		t.Errorf("Stderr = %q, want %q", string(result.Stderr), "command failed")
	}
}
