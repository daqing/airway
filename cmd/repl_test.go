package cmd

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/daqing/airway/lib/repo"
)

type replUser struct {
	ID        int64     `db:"id"`
	Name      *string   `db:"name"`
	Email     string    `db:"email"`
	Enabled   bool      `db:"enabled"`
	CreatedAt time.Time `db:"created_at"`
}

func (replUser) TableName() string {
	return "users"
}

func TestRepoREPLExecutesGoStyleRepoCalls(t *testing.T) {
	session := newTestREPL(t)

	inserted := executeAndDecode(t, session, `repo.Insert("users", pg.H{"email": "alice@example.com", "enabled": true})`)
	insertedRow, ok := inserted.(map[string]any)
	if !ok {
		t.Fatalf("expected inserted row object, got %#v", inserted)
	}

	id, ok := insertedRow["id"].(float64)
	if !ok || id <= 0 {
		t.Fatalf("expected numeric id, got %#v", insertedRow["id"])
	}

	found := executeAndDecode(t, session, `repo.FindOne("users", pg.Eq("id", 1))`)
	foundRow, ok := found.(map[string]any)
	if !ok {
		t.Fatalf("expected row object, got %#v", found)
	}

	if foundRow["email"] != "alice@example.com" {
		t.Fatalf("unexpected row: %#v", foundRow)
	}

	foundByStmt := executeAndDecode(t, session, `repo.Find("users", pg.Select("*").Where(pg.Eq("id", 1)))`)
	foundRows, ok := foundByStmt.([]any)
	if !ok || len(foundRows) != 1 {
		t.Fatalf("expected one row from stmt-based find, got %#v", foundByStmt)
	}

	countByStmt := executeAndDecode(t, session, `repo.Count("users", pg.Select("count(*)").Where(pg.Eq("id", 1)))`)
	if countByStmt.(float64) != 1 {
		t.Fatalf("unexpected stmt-based count result: %#v", countByStmt)
	}

	updated := executeAndDecode(t, session, `repo.Update("users", pg.H{"enabled": false}, pg.Eq("email", "alice@example.com"))`)
	if updated.(float64) != 1 {
		t.Fatalf("unexpected update result: %#v", updated)
	}

	count := executeAndDecode(t, session, `repo.Count("users", pg.Eq("enabled", false))`)
	if count.(float64) != 1 {
		t.Fatalf("unexpected count result: %#v", count)
	}

	rows := executeAndDecode(t, session, `repo.Find("users", pg.In("id", []int{1, 2, 3}))`)
	rowList, ok := rows.([]any)
	if !ok || len(rowList) != 1 {
		t.Fatalf("unexpected rows result: %#v", rows)
	}

	deleted := executeAndDecode(t, session, `repo.Delete("users", pg.Eq("email", "alice@example.com"))`)
	if deleted.(float64) != 1 {
		t.Fatalf("unexpected delete result: %#v", deleted)
	}

	exists := executeAndDecode(t, session, `repo.Exists("users", pg.Eq("email", "alice@example.com"))`)
	if exists.(bool) {
		t.Fatalf("expected deleted row to be absent: %#v", exists)
	}
}

func TestRepoREPLPreviewsBuildersAndProtectsFullTableWrites(t *testing.T) {
	session := newTestREPL(t)

	preview := executeAndDecode(t, session, `pg.Select("*").From("users").Where(pg.Eq("id", 1))`)
	previewMap, ok := preview.(map[string]any)
	if !ok {
		t.Fatalf("expected preview map, got %#v", preview)
	}

	query, _ := previewMap["query"].(string)
	if query == "" {
		t.Fatalf("expected compiled query, got %#v", previewMap)
	}

	tables := executeAndDecode(t, session, `repo.Tables()`)
	tableList, ok := tables.([]any)
	if !ok || len(tableList) == 0 {
		t.Fatalf("expected tables list, got %#v", tables)
	}

	if _, err := session.execute(`repo.Delete("users")`); err == nil {
		t.Fatal("expected full-table delete to require explicit true")
	}

	deleted := executeAndDecode(t, session, `repo.Delete("users", true)`)
	if deleted.(float64) != 0 {
		t.Fatalf("unexpected full-table delete result: %#v", deleted)
	}
}

func TestRepoREPLSupportsGenericRepoFindCalls(t *testing.T) {
	session := newTestREPL(t)
	session.evaluator.symbols["replUser"] = reflect.TypeOf(replUser{})

	executeAndDecode(t, session, `repo.Insert("users", pg.H{"email": "alice@example.com", "enabled": true})`)

	inferred := executeAndDecode(t, session, `repo.FindOne[replUser](pg.Eq("id", 1))`)
	inferredRow, ok := inferred.(map[string]any)
	if !ok {
		t.Fatalf("expected inferred typed row object, got %#v", inferred)
	}

	if inferredRow["Email"] != "alice@example.com" {
		t.Fatalf("unexpected inferred typed row: %#v", inferredRow)
	}

	found := executeAndDecode(t, session, `repo.FindOne[struct{ ID int64 `+"`db:\"id\"`"+`; Email string `+"`db:\"email\"`"+`; Enabled bool `+"`db:\"enabled\"`"+` }]("users", pg.Select("id, email, enabled").Where(pg.Eq("id", 1)))`)
	foundRow, ok := found.(map[string]any)
	if !ok {
		t.Fatalf("expected typed row object, got %#v", found)
	}

	if foundRow["Email"] != "alice@example.com" {
		t.Fatalf("unexpected typed row: %#v", foundRow)
	}

	rows := executeAndDecode(t, session, `repo.Find[struct{ ID int64 `+"`db:\"id\"`"+`; Email string `+"`db:\"email\"`"+` }](pg.Select("id, email").From("users").Where(pg.Eq("enabled", true)))`)
	rowList, ok := rows.([]any)
	if !ok || len(rowList) != 1 {
		t.Fatalf("expected typed rows result, got %#v", rows)
	}
}

func TestRepoREPLSupportsGenericRepoInsertCalls(t *testing.T) {
	session := newTestREPL(t)
	session.evaluator.symbols["replUser"] = reflect.TypeOf(replUser{})

	inserted := executeAndDecode(t, session, `repo.Insert[replUser](pg.H{"id": 1234, "email": "typed@example.com", "enabled": true})`)
	insertedRow, ok := inserted.(map[string]any)
	if !ok {
		t.Fatalf("expected typed inserted row object, got %#v", inserted)
	}

	if insertedRow["ID"] != float64(1234) {
		t.Fatalf("unexpected typed inserted row: %#v", insertedRow)
	}
}

func TestRepoREPLExposesProjectModelsNamespace(t *testing.T) {
	session := newTestREPL(t)

	executeAndDecode(t, session, `repo.Insert("users", pg.H{"email": "alice@example.com", "enabled": true})`)

	found := executeAndDecode(t, session, `repo.FindOne[models.User](pg.Select("id").Where(pg.Eq("id", 1)))`)
	foundRow, ok := found.(map[string]any)
	if !ok {
		t.Fatalf("expected model row object, got %#v", found)
	}

	if foundRow["ID"] != float64(1) {
		t.Fatalf("unexpected model row: %#v", foundRow)
	}
}

func TestRepoREPLExposesProjectModelsAsTopLevelTypes(t *testing.T) {
	session := newTestREPL(t)

	executeAndDecode(t, session, `repo.Insert("users", pg.H{"id": 7, "email": "top@example.com", "enabled": true})`)

	found := executeAndDecode(t, session, `repo.FindOne[User](pg.Eq("id", 7))`)
	foundRow, ok := found.(map[string]any)
	if !ok {
		t.Fatalf("expected top-level model row object, got %#v", found)
	}

	if foundRow["ID"] != float64(7) {
		t.Fatalf("unexpected top-level model row: %#v", foundRow)
	}
}

func TestRepoREPLSupportsGenericRepoDeleteCalls(t *testing.T) {
	session := newTestREPL(t)

	executeAndDecode(t, session, `repo.Insert("users", pg.H{"id": 333, "email": "delete@example.com", "enabled": true})`)

	deleted := executeAndDecode(t, session, `repo.Delete[User](pg.Eq("id", 333))`)
	if deleted.(float64) != 1 {
		t.Fatalf("unexpected typed delete result: %#v", deleted)
	}
}

func newTestREPL(t *testing.T) *repoREPL {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "repo-repl.sqlite3")
	setupDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open setup db: %v", err)
	}

	t.Cleanup(func() {
		_ = setupDB.Close()
	})

	_, err = setupDB.Exec(`
CREATE TABLE users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NULL,
	email TEXT NOT NULL,
	enabled BOOLEAN NOT NULL DEFAULT FALSE,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	repoDB, err := repo.NewDBWithDriver("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open repo db: %v", err)
	}

	t.Cleanup(func() {
		_ = repoDB.Close()
	})

	return newRepoREPL(repoDB, &bytes.Buffer{}, &bytes.Buffer{})
}

func executeAndDecode(t *testing.T, session *repoREPL, expr string) any {
	t.Helper()

	outBuffer, ok := session.out.(*bytes.Buffer)
	if !ok {
		t.Fatalf("expected bytes buffer writer, got %T", session.out)
	}
	errBuffer, ok := session.errOut.(*bytes.Buffer)
	if !ok {
		t.Fatalf("expected bytes buffer error writer, got %T", session.errOut)
	}

	outBuffer.Reset()
	errBuffer.Reset()

	if _, err := session.execute(expr); err != nil {
		t.Fatalf("execute %q: %v", expr, err)
	}

	return decodeJSON(t, outBuffer)
}

func decodeJSON(t *testing.T, writer io.Writer) any {
	t.Helper()

	buffer, ok := writer.(*bytes.Buffer)
	if !ok {
		t.Fatalf("expected bytes buffer writer, got %T", writer)
	}

	payload := buffer.String()
	if payload == "" {
		t.Fatal("expected json output, got empty buffer")
	}

	var decoded any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("decode json %q: %v", payload, err)
	}

	return decoded
}
