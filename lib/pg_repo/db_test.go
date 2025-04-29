package pg_repo

import (
	"testing"
	"time"

	"github.com/daqing/airway/lib/sql"
)

const DSN = "postgres://daqing@localhost:5432/vp2"

type Todo struct {
	ID        int64     `db:"id"`
	Title     string    `db:"title"`
	Completed bool      `db:"completed"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func TestInsert(t *testing.T) {
	db, err := NewDB(DSN)
	if err != nil {
		t.Fatal(err)
	}

	var todo *Todo
	b := sql.Insert(sql.H{"title": "test233", "completed": true}).Into("todos")
	todo, err = Insert[Todo](db, b)
	if err != nil {
		t.Fatal(err)
	}

	if todo == nil {
		t.Fatal("todo is nil")
	}
}

func TestSelect(t *testing.T) {
	db, err := NewDB(DSN)
	if err != nil {
		t.Fatal(err)
	}

	var todos []*Todo
	b := sql.Select("*").From("todos")
	todos, err = Find[Todo](db, b)
	if err != nil {
		t.Fatal(err)
	}

	if len(todos) == 0 {
		t.Fatal("todos is empty")
	}
}

func TestUpdate(t *testing.T) {
	db, err := NewDB(DSN)
	if err != nil {
		t.Fatal(err)
	}

	b := sql.Update("todos").Set(sql.H{"title": "test updated", "completed": false})
	err = db.Update(b)
	if err != nil {
		t.Fatalf("failed to update todo")
	}
}

func TestDelete(t *testing.T) {
	db, err := NewDB(DSN)
	if err != nil {
		t.Fatal(err)
	}

	b := sql.Delete().From("todos").OrderBy("id ASC").Limit(1)
	err = db.Delete(b)
	if err != nil {
		t.Fatal(err)
	}
}
