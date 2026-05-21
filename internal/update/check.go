// Package update fetches the latest pkgr release tag and compares versions.
package update

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const releasesURL = "https://api.github.com/repos/ramtinhoss/pkgr/releases/latest"

// Checker fetches the latest GitHub release tag for pkgr.
type Checker struct {
	Client *http.Client
}

// Latest returns the latest release tag (e.g. "v1.2.3"), or "" on network
// errors or when no release exists yet.
func (c *Checker) Latest() (string, error) {
	cli := c.Client
	if cli == nil {
		cli = http.DefaultClient
	}
	req, _ := http.NewRequest(http.MethodGet, releasesURL, nil)
	req.Header.Set("User-Agent", "pkgr-update-check")
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", nil
	}
	body, _ := io.ReadAll(resp.Body)
	var v struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		return "", err
	}
	return v.TagName, nil
}

// IsNewer returns true when latest is a higher semver than have.
// "dev" or "" as have is never considered older (skip nag in dev builds).
func IsNewer(have, latest string) bool {
	if have == "" || have == "dev" || latest == "" {
		return false
	}
	a := strings.TrimPrefix(have, "v")
	b := strings.TrimPrefix(latest, "v")
	return cmp(a, b) < 0
}

func cmp(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var av, bv int
		if i < len(as) {
			av = atoi(as[i])
		}
		if i < len(bs) {
			bv = atoi(bs[i])
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}

func atoi(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + int(ch-'0')
	}
	return n
}
