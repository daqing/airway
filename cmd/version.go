package cmd

import "fmt"

const version = "0.3.2-dev"

func showVersion(_ []string) {
	fmt.Printf("airway %s\n", version)
}
