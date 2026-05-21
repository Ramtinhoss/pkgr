package spec

import "testing"

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
