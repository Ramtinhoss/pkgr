// Package spec parses package specs of the form "name[==ver][@pm]".
// Names may include leading "@scope/" (npm scoped packages).
package spec

import (
	"errors"
	"fmt"
	"strings"
)

type Spec struct {
	Name    string
	Version string
	PM      string
}

var (
	ErrEmpty      = errors.New("spec: empty")
	ErrNoName     = errors.New("spec: missing name")
	ErrMultiplePM = errors.New("spec: multiple @pm separators")
	ErrEmptyVer   = errors.New("spec: '==' with empty version")
)

// Parse interprets a spec string and returns a Spec or error.
// Format: name[==version][@pm]
// Names may include "@scope/" prefix (npm scoped packages).
// Whitespace anywhere in the string is rejected.
func Parse(s string) (Spec, error) {
	if s == "" {
		return Spec{}, ErrEmpty
	}
	// Reject any whitespace — specs must be compact tokens.
	if strings.TrimSpace(s) != s || strings.ContainsAny(s, " \t\n\r") {
		return Spec{}, fmt.Errorf("spec: whitespace not allowed: %q", s)
	}

	// Extract @pm suffix (but allow @ at start for scoped names).
	var pm, body string
	if strings.HasPrefix(s, "@") {
		// Scoped name: @scope/pkg[==ver][@pm]
		// Find the last @, which separates the body from @pm.
		idx := strings.LastIndex(s, "@")
		if idx > 0 {
			// Found @ after first char: could be "@scope/pkg@pm"
			// Check if there's a / before this @ (indicating scope).
			slashIdx := strings.Index(s, "/")
			if slashIdx > 0 && slashIdx < idx {
				// Scoped package with @pm: @scope/pkg@pm
				pm = s[idx+1:]
				body = s[:idx]
			} else {
				// Just @pm after non-scoped start: treat the whole thing as body.
				body = s
			}
		} else {
			// Only one @ at position 0: pure scope, no pm separator.
			body = s
		}
	} else {
		// Non-scoped name: name[==ver][@pm]
		// Count @ occurrences outside of the body.
		if strings.Count(s, "@") > 1 {
			return Spec{}, ErrMultiplePM
		}
		idx := strings.LastIndex(s, "@")
		if idx >= 0 {
			pm = s[idx+1:]
			body = s[:idx]
		} else {
			body = s
		}
	}

	if body == "" {
		return Spec{}, ErrNoName
	}

	// Parse body: name[==version]
	var name, ver string
	idx := strings.Index(body, "==")
	if idx >= 0 {
		name = body[:idx]
		ver = body[idx+2:]
		if ver == "" {
			return Spec{}, ErrEmptyVer
		}
	} else {
		name = body
	}

	if name == "" {
		return Spec{}, ErrNoName
	}

	// Reject scoped names with empty package part: "@scope/" is not valid.
	if strings.HasPrefix(name, "@") {
		slashIdx := strings.Index(name, "/")
		if slashIdx >= 0 && slashIdx == len(name)-1 {
			return Spec{}, fmt.Errorf("spec: empty package name after scope in %q", name)
		}
	}

	return Spec{Name: name, Version: ver, PM: pm}, nil
}
