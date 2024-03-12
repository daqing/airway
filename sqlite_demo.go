package main

import (
	"database/sql"
	"fmt"
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
	//

	const insert = "INSERT INTO activities VALUES(NULL, ?, ?)"

	if _, err = db.Exec(insert, time.Now(), "Demo testing"); err != nil {
		panic(err)
	}

	stmt, err := db.Prepare(insert)
	if err != nil {
		panic(err)
	}

	res, err := stmt.Exec(time.Now(), "Demo testing Prepare statement")
	if err != nil {
		panic(err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		panic(err)
	}

	fmt.Println("Last Row Id:", id)
}
