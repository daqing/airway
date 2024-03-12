package main

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

const file string = "data/activities.db"

const create string = `
  CREATE TABLE IF NOT EXISTS activities (
  id INTEGER NOT NULL PRIMARY KEY,
  time DATETIME NOT NULL,
  description TEXT
  );`

func main() {
	db, err := sql.Open("sqlite", file)
	if err != nil {
		panic(err)
	}

	// if _, err = db.Exec(create); err != nil {
	//		panic(err)
	// }

	if _, err = db.Exec("INSERT INTO activities VALUES(NULL, ?, ?)", time.Now(), "Demo testing"); err != nil {
		panic(err)
	}
}
