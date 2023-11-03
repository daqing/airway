package generator

import (
	"fmt"
	"os"
	"time"
)

func GenMigration(xargs []string) {
	if len(xargs) == 0 {
		fmt.Println("cli g migration [name]")
		return
	}

	GenerateMigration(xargs[0])
}

func GenerateMigration(name string) {
	ts := time.Now().Format("20060102150405")

	targetPath := fmt.Sprintf("./db/%s_%s.sql", ts, name)
	if _, err := os.Stat(targetPath); err == nil {
		// target file exists
		panic("target file exists")
	}

	if err := os.WriteFile(targetPath, []byte("\n"), 0644); err != nil {
		panic(err)
	}
}
