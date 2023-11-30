package generator

import (
	"fmt"
	"os"
	"time"

	"github.com/daqing/airway/cli/helper"
)

func GenMigration(xargs []string) {
	if len(xargs) == 0 {
		helper.Help("cli g migration [name]")
	}

	fmt.Println(GenerateMigration(xargs[0]))
}

func GenerateMigration(name string) string {
	ts := time.Now().Format("20060102150405")

	targetPath := fmt.Sprintf("./db/%s_%s.sql", ts, name)
	if _, err := os.Stat(targetPath); err == nil {
		// target file exists
		fmt.Println("target file exists")
		return ""
	}

	if err := os.WriteFile(targetPath, []byte("\n"), 0644); err != nil {
		panic(err)
	}

	return targetPath
}
