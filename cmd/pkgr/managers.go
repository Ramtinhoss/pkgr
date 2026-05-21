package main

import (
	"github.com/ramtinhoss/pkgr/internal/manager/apt"
	"github.com/ramtinhoss/pkgr/internal/manager/asdf"
	"github.com/ramtinhoss/pkgr/internal/manager/brew"
	"github.com/ramtinhoss/pkgr/internal/manager/bun"
	"github.com/ramtinhoss/pkgr/internal/manager/cargo"
	"github.com/ramtinhoss/pkgr/internal/manager/conda"
	"github.com/ramtinhoss/pkgr/internal/manager/mamba"
	"github.com/ramtinhoss/pkgr/internal/manager/dnf"
	"github.com/ramtinhoss/pkgr/internal/manager/pacman"
	"github.com/ramtinhoss/pkgr/internal/manager/flatpak"
	"github.com/ramtinhoss/pkgr/internal/manager/nix"
	"github.com/ramtinhoss/pkgr/internal/manager/choco"
	"github.com/ramtinhoss/pkgr/internal/manager/mas"
	"github.com/ramtinhoss/pkgr/internal/manager/pnpm"
	"github.com/ramtinhoss/pkgr/internal/manager/winget"
	"github.com/ramtinhoss/pkgr/internal/manager/scoop"
	"github.com/ramtinhoss/pkgr/internal/manager/snap"
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
	reg.Register(apt.New(r))
	reg.Register(dnf.New(r))
	reg.Register(pacman.New(r))
	reg.Register(snap.New(r))
	reg.Register(flatpak.New(r))
	reg.Register(nix.New(r))
	reg.Register(scoop.New(r))
	reg.Register(choco.New(r))
	reg.Register(winget.New(r))
	reg.Register(mas.New(r))
	reg.Register(pnpm.New(r))
	reg.Register(asdf.New(r))
	reg.Register(bun.New(r))
	reg.Register(cargo.New(r))
	reg.Register(conda.New(r))
	reg.Register(mamba.New(r))
}
