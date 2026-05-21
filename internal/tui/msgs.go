package tui

import (
	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/orchestrator"
)

// Cross-screen messages.

type PushScreenMsg struct{ Screen Screen }
type PopScreenMsg struct{}
type StatusMsg struct{ Text string }

type SearchPartialMsg struct {
	Manager string
	Results []manager.Package
}
type SearchDoneMsg struct {
	All  []orchestrator.Result
	Errs []error
}

type OpStartMsg struct {
	Manager string
	Op      manager.Op
	Cmd     string
}
type OpDoneMsg struct {
	Manager string
	Op      manager.Op
	Err     error
}
