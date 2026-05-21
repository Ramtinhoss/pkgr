package format

import (
	"encoding/json"
	"io"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

type Envelope struct {
	Packages []manager.Package `json:"packages"`
	Errors   []errorRendered   `json:"errors"`
}

type errorRendered struct {
	Manager string `json:"manager"`
	Op      string `json:"op"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func renderErr(e error) errorRendered {
	if me, ok := e.(*manager.Error); ok {
		return errorRendered{Manager: me.Manager, Op: string(me.Op), Code: string(me.Code), Message: me.Err.Error()}
	}
	return errorRendered{Code: "unknown", Message: e.Error()}
}

func JSONResult(w io.Writer, pkgs []manager.Package, errs []error) error {
	env := Envelope{Packages: pkgs, Errors: make([]errorRendered, 0, len(errs))}
	for _, e := range errs {
		env.Errors = append(env.Errors, renderErr(e))
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}
