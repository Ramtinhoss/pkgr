package spec

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in    string
		name  string
		pm    string
		ver   string
		err   bool
	}{
		{"ripgrep", "ripgrep", "", "", false},
		{"ripgrep@brew", "ripgrep", "brew", "", false},
		{"ripgrep==13.0.0", "ripgrep", "", "13.0.0", false},
		{"ripgrep==13.0.0@brew", "ripgrep", "brew", "13.0.0", false},
		{"@scope/pkg@npm", "@scope/pkg", "npm", "", false},
		{"@scope/pkg==1.2.3@npm", "@scope/pkg", "npm", "1.2.3", false},
		{"", "", "", "", true},
		{"==1.2.3", "", "", "", true},
		{"foo@bar@baz", "", "", "", true},
		{"foo==", "", "", "", true},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if c.err {
			if err == nil {
				t.Errorf("Parse(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q) error: %v", c.in, err)
			continue
		}
		if got.Name != c.name || got.PM != c.pm || got.Version != c.ver {
			t.Errorf("Parse(%q) = %+v, want Name=%q PM=%q Ver=%q", c.in, got, c.name, c.pm, c.ver)
		}
	}
}

func TestParseExtraCases(t *testing.T) {
	cases := []struct {
		in   string
		name string
		pm   string
		ver  string
		err  bool
	}{
		{"name@", "name", "", "", false}, // trailing @ with empty pm is treated as body
		{"@only", "@only", "", "", false},
		{"a-b_c.d", "a-b_c.d", "", "", false},
		{"@scope/", "", "", "", true},          // empty pkg name after scope
		{"@scope/pkg==1.2.3@npm", "@scope/pkg", "npm", "1.2.3", false},
		{"  spaces  ", "", "", "", true},       // whitespace rejected
		{" leading", "", "", "", true},         // leading space rejected
		{"trailing ", "", "", "", true},        // trailing space rejected
		{"with\ttab", "", "", "", true},        // tab rejected
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if c.err {
			if err == nil {
				t.Errorf("Parse(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q): %v", c.in, err)
			continue
		}
		if got.Name != c.name || got.PM != c.pm || got.Version != c.ver {
			t.Errorf("Parse(%q) = %+v, want Name=%q PM=%q Ver=%q", c.in, got, c.name, c.pm, c.ver)
		}
	}
}

func TestParseWhitespaceAlwaysErrors(t *testing.T) {
	inputs := []string{"  ", "\t", "\n", " foo", "foo ", "foo bar"}
	for _, s := range inputs {
		_, err := Parse(s)
		if err == nil {
			t.Errorf("Parse(%q) should error on whitespace", s)
		}
		if err != nil && !strings.Contains(err.Error(), "whitespace") && err != ErrEmpty {
			// also acceptable: ErrEmpty for blank strings
		}
	}
}
