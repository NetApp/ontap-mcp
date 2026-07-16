package version

import (
	"fmt"
	"runtime"
)

var (
	VERSION   = "1.0.0"
	Commit    = "HEAD"
	BuildDate = "undefined"
)

// String returns the full version string
func String() string {
	return fmt.Sprintf("ontap-mcp version %s (commit %s) (build date %s) %s/%s",
		VERSION,
		Commit,
		BuildDate,
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// Info returns the version string
func Info() string {
	return VERSION
}
