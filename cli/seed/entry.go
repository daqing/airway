package seed

import (
	"fmt"
	"os"

	"github.com/daqing/airway/core/api/user_api"
)

func Generate(args []string) {
	if len(args) == 0 {
		fmt.Println("cli seed root [username] [password]")
		fmt.Println("cli seed user [nickname] [username] [password]")

		os.Exit(1)
	}

	thing := args[0]
	switch thing {
	case "root":
		xargs := args[1:]

		if len(xargs) < 2 {
			fmt.Println("cli seed root [username] [password]")
			os.Exit(1)
		}

		SeedRoot(xargs[0], xargs[1])
	case "user":
		xargs := args[1:]
		if len(xargs) < 3 {
			fmt.Println("cli seed user [nickname] [username] [password]")
			os.Exit(1)
		}

		SeedUser(xargs[0], xargs[1], xargs[2])
	}

}

func SeedRoot(username, password string) {
	_, err := user_api.CreateRootUser(username, password)
	if err != nil {
		panic(err)
	}

	fmt.Println("done.")
}

func SeedUser(nickname, username string, password string) {
	_, err := user_api.CreateBasicUser(nickname, username, password)
	if err != nil {
		panic(err)
	}

	fmt.Println("done.")
}
