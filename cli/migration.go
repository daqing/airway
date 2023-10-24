package main

import (
	"fmt"
	"os"
	"time"
)

func GenerateMigration(name string) {
	now := time.Now()
	ts := now.Format("20060102150405")

	targetPath := fmt.Sprintf("./db/%s_%s.sql", ts, name)
	if _, err := os.Stat(targetPath); err == nil {
		// target file exists
		panic("target file exists")
	}

	if err := os.WriteFile(targetPath, []byte("\n"), 0644); err != nil {
		panic(err)
	}
}
