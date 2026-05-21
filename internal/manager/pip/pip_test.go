package pip

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/runner"
)

type fakeRT struct{ body []byte }

func (f *fakeRT) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(string(f.body))),
		Header:     make(http.Header),
	}, nil
}

func loadFix(t *testing.T, name string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSearchViaPyPI(t *testing.T) {
	a := &Adapter{
		Runner: &runner.Runner{Exec: (&runner.Fake{}).Exec},
		HTTP:   &http.Client{Transport: &fakeRT{body: loadFix(t, "search.json")}},
		Bin:    "pip",
	}
	got, err := a.Search(context.Background(), "requests")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "requests" || got[0].Version != "2.32.3" {
		t.Fatalf("got %+v", got)
	}
}

func TestList(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pip list --format json": {Stdout: loadFix(t, "list.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pip"}
	got, err := a.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestOutdated(t *testing.T) {
	fake := &runner.Fake{Returns: map[string]runner.FakeResult{
		"pip list --outdated --format json": {Stdout: loadFix(t, "outdated.json")},
	}}
	a := &Adapter{Runner: &runner.Runner{Exec: fake.Exec}, Bin: "pip"}
	got, err := a.Outdated(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Latest != "13.7.1" {
		t.Fatalf("got %+v", got)
	}
}
