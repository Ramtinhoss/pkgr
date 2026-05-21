package cache

import (
	"path/filepath"
	"testing"
	"time"
)

func TestGetSetRoundTrip(t *testing.T) {
	c := New(t.TempDir())
	type doc struct {
		Items []string
	}
	in := doc{Items: []string{"a", "b"}}
	if err := c.Set("brew/installed", in, 1*time.Hour); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var out doc
	hit, err := c.Get("brew/installed", &out)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hit {
		t.Fatal("expected cache hit")
	}
	if len(out.Items) != 2 || out.Items[0] != "a" {
		t.Fatalf("Items = %v", out.Items)
	}
}

func TestExpiredEntryIsMiss(t *testing.T) {
	c := New(t.TempDir())
	if err := c.Set("k", 1, 10*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	var v int
	hit, err := c.Get("k", &v)
	if err != nil {
		t.Fatal(err)
	}
	if hit {
		t.Fatal("expected miss after expiry")
	}
}

func TestInvalidateRemovesKey(t *testing.T) {
	c := New(t.TempDir())
	_ = c.Set("brew/installed", []int{1}, time.Hour)
	if err := c.Invalidate("brew/installed"); err != nil {
		t.Fatal(err)
	}
	var out []int
	hit, _ := c.Get("brew/installed", &out)
	if hit {
		t.Fatal("expected miss after invalidate")
	}
}

func TestPathIsSandboxed(t *testing.T) {
	c := New(t.TempDir())
	if err := c.Set("../escape", 1, time.Hour); err == nil {
		t.Fatal("expected error on path traversal key")
	}
	got := filepath.Clean(c.PathFor("brew/installed"))
	want := filepath.Clean(filepath.Join(c.Root, "brew", "installed.json"))
	if got != want {
		t.Fatalf("PathFor = %q, want %q", got, want)
	}
}
