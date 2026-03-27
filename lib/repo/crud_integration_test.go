package repo

import (
	"testing"

	"github.com/daqing/airway/lib/sql"
)

func TestInsert(t *testing.T) {
	forEachTestDB(t, func(t *testing.T, db *DB) {
		tableName := createTodoTable(t, db)
		todos := sql.TableOf(tableName)

		todo, err := Insert[Todo](db, sql.Insert(sql.H{"title": "test233", "completed": true}).IntoTable(todos))
		if err != nil {
			t.Fatal(err)
		}

		if todo == nil {
			t.Fatal("todo is nil")
		}

		if todo.Title != "test233" || !todo.Completed || todo.ID == 0 {
			t.Fatalf("unexpected inserted todo: %#v", todo)
		}
	})
}

func TestSelect(t *testing.T) {
	forEachTestDB(t, func(t *testing.T, db *DB) {
		tableName := createTodoTable(t, db)
		insertTodoRow(t, db, tableName, "select me", false)

		todosTable := sql.TableOf(tableName)
		todos, err := Find[Todo](db, sql.SelectFields(todosTable.AllFields()).FromTable(todosTable))
		if err != nil {
			t.Fatal(err)
		}

		if len(todos) != 1 || todos[0].Title != "select me" || todos[0].Completed {
			t.Fatalf("unexpected todos: %#v", todos)
		}
	})
}

func TestUpdate(t *testing.T) {
	forEachTestDB(t, func(t *testing.T, db *DB) {
		tableName := createTodoTable(t, db)
		insertTodoRow(t, db, tableName, "before update", true)

		todos := sql.TableOf(tableName)
		err := Update(db, sql.UpdateTable(todos).
			Set(sql.H{"title": "test updated", "completed": false}).
			Where(sql.FieldEq(todos.Field("title"), "before update")))
		if err != nil {
			t.Fatalf("failed to update todo: %v", err)
		}

		updated, err := Find[Todo](db, sql.SelectFields(todos.AllFields()).FromTable(todos))
		if err != nil {
			t.Fatalf("find updated todo: %v", err)
		}

		if len(updated) != 1 || updated[0].Title != "test updated" || updated[0].Completed {
			t.Fatalf("unexpected updated todo: %#v", updated)
		}
	})
}

func TestDelete(t *testing.T) {
	forEachTestDB(t, func(t *testing.T, db *DB) {
		tableName := createTodoTable(t, db)
		firstID := insertTodoRow(t, db, tableName, "first", false)
		insertTodoRow(t, db, tableName, "second", true)

		todosTable := sql.TableOf(tableName)
		err := Delete(db, sql.DeleteFrom(todosTable).OrderBy(todosTable.Field("id").Asc()).Limit(1))
		if err != nil {
			t.Fatal(err)
		}

		remaining, err := Find[Todo](db, sql.SelectFields(todosTable.AllFields()).FromTable(todosTable).OrderBy(todosTable.Field("id").Asc()))
		if err != nil {
			t.Fatalf("find remaining todos: %v", err)
		}

		if count := countRows(t, db, tableName); count != 1 {
			t.Fatalf("expected 1 remaining row, got %d", count)
		}

		if len(remaining) != 1 || remaining[0].ID == firstID || remaining[0].Title != "second" {
			t.Fatalf("unexpected remaining todos: %#v", remaining)
		}
	})
}
