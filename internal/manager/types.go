// Package manager defines the adapter interface and shared types.
package manager

import "context"

type OS string

const (
	Darwin  OS = "darwin"
	Linux   OS = "linux"
	Windows OS = "windows"
)

type Op string

const (
	OpInstall   Op = "install"
	OpUninstall Op = "uninstall"
	OpUpdate    Op = "update"
	OpSearch    Op = "search"
	OpList      Op = "list"
	OpInfo      Op = "info"
	OpOutdated  Op = "outdated"
)

type Scope string

const (
	ScopeSystem       Scope = "system"
	ScopeUserGlobal   Scope = "user-global"
	ScopeProjectLocal Scope = "project-local"
)

type Package struct {
	Name        string            `json:"name"`
	Version     string            `json:"version,omitempty"`
	Latest      string            `json:"latest,omitempty"`
	Manager     string            `json:"manager"`
	Installed   bool              `json:"installed"`
	Description string            `json:"description,omitempty"`
	Homepage    string            `json:"homepage,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
}

type Manager interface {
	ID() string
	DisplayName() string
	OSes() []OS
	Detect() bool
	NeedsSudo(op Op) bool
	Scope() Scope

	List(ctx context.Context) ([]Package, error)
	Outdated(ctx context.Context) ([]Package, error)
	Search(ctx context.Context, q string) ([]Package, error)
	Info(ctx context.Context, name string) (Package, error)
	Install(ctx context.Context, names ...string) error
	Uninstall(ctx context.Context, names ...string) error
	Update(ctx context.Context, names ...string) error
}
