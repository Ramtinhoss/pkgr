package tui

import "github.com/ramtinhoss/pkgr/internal/manager"

// SelectionKey uniquely identifies a row across screens: "<pm>/<name>".
type SelectionKey struct {
	PM, Name string
}

func KeyFor(p manager.Package) SelectionKey { return SelectionKey{PM: p.Manager, Name: p.Name} }

// Selection is a set of SelectionKey -> Package (so we can pass full pkgs to ops).
type Selection map[SelectionKey]manager.Package

func NewSelection() Selection { return Selection{} }

func (s Selection) Toggle(p manager.Package) {
	k := KeyFor(p)
	if _, ok := s[k]; ok {
		delete(s, k)
	} else {
		s[k] = p
	}
}

func (s Selection) Has(p manager.Package) bool { _, ok := s[KeyFor(p)]; return ok }

func (s Selection) Clear() {
	for k := range s {
		delete(s, k)
	}
}

func (s Selection) Count() int { return len(s) }

// GroupByPM returns the selection grouped by PM -> []name.
func (s Selection) GroupByPM() map[string][]string {
	out := map[string][]string{}
	for _, p := range s {
		out[p.Manager] = append(out[p.Manager], p.Name)
	}
	return out
}
