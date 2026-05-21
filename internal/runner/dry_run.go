package runner

import (
	"fmt"
	"strings"
)

// Preview returns the human-readable command line that Run would invoke,
// without actually executing it. Mirrors the dry-run formatting in Runner.Run.
func Preview(c Cmd) string {
	prefix := ""
	if c.Sudo {
		prefix = "sudo "
	}
	return fmt.Sprintf("%s%s %s", prefix, c.Bin, strings.Join(c.Args, " "))
}
