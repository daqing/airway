package cmd

import "fmt"

const version = "0.4.0-dev"

func showVersion(_ []string) {
	fmt.Printf("airway %s\n", version)
}
