package runner

import (
	"bytes"
	"context"
	"strings"
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

func TestFakeReturnsErrorOnMissingKey(t *testing.T) {
	f := &Fake{Returns: map[string]FakeResult{}}
	r := &Runner{Exec: f.Exec}
	_, err := r.Run(context.Background(), Cmd{Bin: "brew", Args: []string{"unknown"}})
	if err == nil {
		t.Fatal("expected error for missing canned reply")
	}
}

func TestDryRunSudoPrefix(t *testing.T) {
	var out bytes.Buffer
	r := &Runner{Out: &out, DryRun: true}
	if _, err := r.Run(context.Background(), Cmd{Bin: "apt-get", Args: []string{"install", "git"}, Sudo: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "sudo apt-get install git") {
		t.Fatalf("expected sudo prefix in dry-run, got %q", out.String())
	}
}
