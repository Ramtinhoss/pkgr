// Package cache provides a TTL'd JSON file cache with per-file flock.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

type entry struct {
	FetchedAt   time.Time       `json:"fetched_at"`
	TTLSeconds  int64           `json:"ttl_seconds"`
	Data        json.RawMessage `json:"data"`
}

type Cache struct {
	Root string
}

func New(root string) *Cache { return &Cache{Root: root} }

var ErrUnsafeKey = errors.New("cache: unsafe key")

// PathFor maps a logical key to a file path under Root.
func (c *Cache) PathFor(key string) string {
	return filepath.Join(c.Root, key+".json")
}

func (c *Cache) checkKey(key string) error {
	cleaned := filepath.Clean(key)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return ErrUnsafeKey
	}
	return nil
}

func (c *Cache) Set(key string, v any, ttl time.Duration) error {
	if err := c.checkKey(key); err != nil {
		return err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	e := entry{FetchedAt: time.Now(), TTLSeconds: int64(ttl.Seconds()), Data: raw}
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	path := c.PathFor(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	lock := flock.New(path + ".lock")
	if err := lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Get unmarshals into dst and returns (hit, error). A miss returns (false, nil).
func (c *Cache) Get(key string, dst any) (bool, error) {
	if err := c.checkKey(key); err != nil {
		return false, err
	}
	path := c.PathFor(key)
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	var e entry
	if err := json.Unmarshal(body, &e); err != nil {
		return false, fmt.Errorf("cache: parse %s: %w", path, err)
	}
	if time.Since(e.FetchedAt) > time.Duration(e.TTLSeconds)*time.Second {
		return false, nil
	}
	if err := json.Unmarshal(e.Data, dst); err != nil {
		return false, fmt.Errorf("cache: parse data: %w", err)
	}
	return true, nil
}

func (c *Cache) Invalidate(key string) error {
	if err := c.checkKey(key); err != nil {
		return err
	}
	path := c.PathFor(key)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
