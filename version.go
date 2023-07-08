package dragoman

import _ "embed"

//go:embed version.txt
var version string

// Version returns the current version of the dragoman CLI.
func Version() string {
	return version
}
