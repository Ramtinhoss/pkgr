package goinst

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
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(f.body))), Header: http.Header{}}, nil
}

func fx(t *testing.T, n string) []byte {
	b, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGoSearch(t *testing.T) {
	a := &Adapter{
		Runner: &runner.Runner{Exec: (&runner.Fake{}).Exec},
		HTTP:   &http.Client{Transport: &fakeRT{body: fx(t, "search.json")}},
		Bin:    "go",
	}
	got, _ := a.Search(context.Background(), "fzf")
	if len(got) != 1 || got[0].Name != "github.com/junegunn/fzf" {
		t.Fatalf("%+v", got)
	}
}
