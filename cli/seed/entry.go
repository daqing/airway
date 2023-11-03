package seed

import (
	"fmt"
	"os"

	"github.com/daqing/airway/api/user_api"
)

func Generate(args []string) {
	if len(args) == 0 {
		fmt.Println("cli seed root [password]")
		fmt.Println("cli seed user [nickname] [username] [password]")

		os.Exit(1)
	}

	thing := args[0]
	switch thing {
	case "root":
		xargs := args[1:]

		if len(xargs) == 0 {
			fmt.Println("cli seed root [password]")
			os.Exit(1)
		}

		SeedRoot(xargs[0])
	case "user":
		xargs := args[1:]
		if len(xargs) < 3 {
			fmt.Println("cli seed user [nickname] [username] [password]")
			os.Exit(1)
		}

		SeedUser(xargs[0], xargs[1], xargs[2])
	}

}

func SeedRoot(password string) {
	_, err := user_api.CreateUser("root", "root", user_api.AdminRole, password)
	if err != nil {
		panic(err)
	}

	fmt.Println("done.")
}

func SeedUser(nickname, username string, password string) {
	_, err := user_api.CreateUser(nickname, username, user_api.BasicRole, password)
	if err != nil {
		panic(err)
	}

	fmt.Println("done.")
}
