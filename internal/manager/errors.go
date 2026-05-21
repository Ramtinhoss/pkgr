package manager

import "fmt"

type Code string

const (
	CodeNotFound         Code = "not_found"
	CodeNotDetected      Code = "not_detected"
	CodeNeedsSudo        Code = "needs_sudo"
	CodeNetworkFailure   Code = "network_failure"
	CodeParseError       Code = "parse_error"
	CodeConflict         Code = "conflict"
	CodePermissionDenied Code = "permission_denied"
	CodeCancelled        Code = "cancelled"
	CodeUnknown          Code = "unknown"
)

type Error struct {
	Manager string
	Op      Op
	Code    Code
	Err     error
	Cmd     string
	Stderr  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("manager=%s op=%s code=%s err=%v",
		e.Manager, e.Op, e.Code, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }
