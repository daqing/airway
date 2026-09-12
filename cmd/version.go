package cmd

import "fmt"

// Version is reported by `airway version`. The binary's main package sets it
// from the embedded VERSION file; "dev" is the fallback for binaries built
// without one.
var Version = "dev"

func showVersion(_ []string) {
	fmt.Printf("airway %s\n", Version)
}
