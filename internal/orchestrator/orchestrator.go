// Package orchestrator fans out aggregate ops across multiple managers
// and merges/ranks the results.
package orchestrator

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

type Ranking struct {
	Preferred []string
}

type Result struct {
	Pkg  manager.Package
	Rank int // lower is better
}

type Orchestrator struct {
	rank Ranking
}

func New(r Ranking) *Orchestrator { return &Orchestrator{rank: r} }

func (o *Orchestrator) rankFor(pmID, query string, name string) int {
	r := 1000
	for i, p := range o.rank.Preferred {
		if p == pmID {
			r = i
			break
		}
	}
	if strings.EqualFold(name, query) {
		r -= 500 // exact match boost
	}
	return r
}

func (o *Orchestrator) Search(ctx context.Context, mgrs []manager.Manager, q string) ([]Result, []error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var out []Result
	var errs []error

	for _, m := range mgrs {
		wg.Add(1)
		go func(m manager.Manager) {
			defer wg.Done()
			pkgs, err := m.Search(ctx, q)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			for _, p := range pkgs {
				out = append(out, Result{Pkg: p, Rank: o.rankFor(m.ID(), q, p.Name)})
			}
		}(m)
	}
	wg.Wait()

	sort.Slice(out, func(i, j int) bool {
		if out[i].Rank != out[j].Rank {
			return out[i].Rank < out[j].Rank
		}
		return out[i].Pkg.Name < out[j].Pkg.Name
	})
	return out, errs
}

// List fans out List across all managers; results carry their Manager field set.
func (o *Orchestrator) List(ctx context.Context, mgrs []manager.Manager) ([]manager.Package, []error) {
	return fanOutPkgs(ctx, mgrs, func(m manager.Manager) ([]manager.Package, error) { return m.List(ctx) })
}

func (o *Orchestrator) Outdated(ctx context.Context, mgrs []manager.Manager) ([]manager.Package, []error) {
	return fanOutPkgs(ctx, mgrs, func(m manager.Manager) ([]manager.Package, error) { return m.Outdated(ctx) })
}

func fanOutPkgs(ctx context.Context, mgrs []manager.Manager, fn func(manager.Manager) ([]manager.Package, error)) ([]manager.Package, []error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var pkgs []manager.Package
	var errs []error
	for _, m := range mgrs {
		wg.Add(1)
		go func(m manager.Manager) {
			defer wg.Done()
			p, err := fn(m)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			pkgs = append(pkgs, p...)
		}(m)
	}
	wg.Wait()
	return pkgs, errs
}
