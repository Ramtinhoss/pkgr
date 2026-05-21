package update

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type fakeRT struct {
	body   string
	status int
}

func (f *fakeRT) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: f.status,
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     http.Header{},
	}, nil
}

func TestCheckLatest(t *testing.T) {
	c := &Checker{Client: &http.Client{Transport: &fakeRT{status: 200, body: `{"tag_name":"v1.2.0"}`}}}
	got, err := c.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.2.0" {
		t.Fatalf("got %q", got)
	}
}

func TestCheckLatestNon200(t *testing.T) {
	c := &Checker{Client: &http.Client{Transport: &fakeRT{status: 404, body: ""}}}
	got, err := c.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected empty string for non-200, got %q", got)
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		have, latest string
		want         bool
	}{
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.1", "v1.0.0", false},
		{"v1.0.0", "v1.0.0", false},
		{"dev", "v1.0.0", false}, // don't nag in dev builds
		{"", "v1.0.0", false},    // empty have: skip
		{"v1.0.0", "", false},    // empty latest: skip
		{"v2.0.0", "v1.9.9", false},
		{"v1.10.0", "v1.9.0", false},
	}
	for _, c := range cases {
		if got := IsNewer(c.have, c.latest); got != c.want {
			t.Errorf("IsNewer(%s, %s) = %v, want %v", c.have, c.latest, got, c.want)
		}
	}
}
