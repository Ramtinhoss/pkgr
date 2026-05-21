package manager

import (
	"errors"
	"testing"
)

func TestErrorFormatting(t *testing.T) {
	base := errors.New("connection refused")
	e := &Error{
		Manager: "brew",
		Op:      OpInstall,
		Code:    CodeNetworkFailure,
		Err:     base,
		Cmd:     "brew install ripgrep",
		Stderr:  "curl: (7) Failed to connect",
	}
	s := e.Error()
	wantParts := []string{"brew", "install", "network_failure", "connection refused"}
	for _, w := range wantParts {
		if !contains(s, w) {
			t.Errorf("Error() = %q, missing %q", s, w)
		}
	}
	if !errors.Is(e, base) {
		t.Error("errors.Is should unwrap to base")
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (indexOf(s, sub) >= 0))
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
