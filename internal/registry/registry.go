package registry

import (
	"sort"
	"sync"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

// Registry manages multiple adapter instances with thread-safe access.
type Registry struct {
	mu       sync.RWMutex
	managers map[string]manager.Manager
	enabled  map[string]bool
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{
		managers: make(map[string]manager.Manager),
		enabled:  make(map[string]bool),
	}
}

// Register stores a manager by its ID.
func (r *Registry) Register(m manager.Manager) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.managers[m.ID()] = m
}

// Get retrieves a manager by ID.
func (r *Registry) Get(id string) (manager.Manager, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.managers[id]
	return m, ok
}

// SetEnabled applies enabled state overrides.
func (r *Registry) SetEnabled(m map[string]bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = m
}

// All returns all registered managers sorted by ID.
func (r *Registry) All() []manager.Manager {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.managers))
	for id := range r.managers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	result := make([]manager.Manager, len(ids))
	for i, id := range ids {
		result[i] = r.managers[id]
	}
	return result
}

// Active returns managers where Detect()==true and enabled doesn't override to false, sorted by ID.
func (r *Registry) Active() []manager.Manager {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var active []manager.Manager
	for _, m := range r.managers {
		if m.Detect() {
			// Check if enabled map explicitly disables this manager
			if enabled, ok := r.enabled[m.ID()]; ok && !enabled {
				continue
			}
			active = append(active, m)
		}
	}

	// Sort by ID
	sort.Slice(active, func(i, j int) bool {
		return active[i].ID() < active[j].ID()
	})

	return active
}
