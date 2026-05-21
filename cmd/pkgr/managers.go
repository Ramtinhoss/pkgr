package main

import (
	"github.com/ramtinhoss/pkgr/internal/manager/brew"
	"github.com/ramtinhoss/pkgr/internal/manager/npm"
	"github.com/ramtinhoss/pkgr/internal/manager/pip"
	"github.com/ramtinhoss/pkgr/internal/registry"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

// registerAdapters wires every known adapter. Each phase appends here.
func registerAdapters(reg *registry.Registry, r *runner.Runner) {
	reg.Register(brew.New(r))
	reg.Register(npm.New(r))
	reg.Register(pip.New(r))
	// phases 4 & 5 will append the rest.
}
