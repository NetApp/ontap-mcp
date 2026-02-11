//nolint:revive
package version

import (
	"fmt"
	"runtime"
)

var (
	VERSION   = "1.0.0"
	Release   = "1"
	Commit    = "HEAD"
	BuildDate = "undefined"
)

// String returns the full version string
func String() string {
	return fmt.Sprintf("harvest-mcp version %s-%s (commit %s) (build date %s) %s/%s\n",
		VERSION,
		Release,
		Commit,
		BuildDate,
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// Info returns the version-release string
func Info() string {
	return fmt.Sprintf("%s-%s", VERSION, Release)
}
