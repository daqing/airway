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

	"github.com/daqing/airway/lib/replreg"
	"github.com/daqing/airway/lib/repo"
	reposql "github.com/daqing/airway/lib/sql"
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

func init() {
	replreg.Register("User", replUser{})
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

	found := executeAndDecode(t, session, `repo.FindOne("users", sql.Eq("id", 1))`)
	foundRow, ok := found.(map[string]any)
	if !ok {
		t.Fatalf("expected row object, got %#v", found)
	}

	if foundRow["email"] != "alice@example.com" {
		t.Fatalf("unexpected row: %#v", foundRow)
	}

	foundByStmt := executeAndDecode(t, session, `repo.Find("users", pg.Select("*").Where(sql.Eq("id", 1)))`)
	foundRows, ok := foundByStmt.([]any)
	if !ok || len(foundRows) != 1 {
		t.Fatalf("expected one row from stmt-based find, got %#v", foundByStmt)
	}

	foundAll := executeAndDecode(t, session, `repo.FindAll("users")`)
	allRows, ok := foundAll.([]any)
	if !ok || len(allRows) != 1 {
		t.Fatalf("expected one row from FindAll, got %#v", foundAll)
	}

	countByStmt := executeAndDecode(t, session, `repo.Count("users", pg.Select("count(*)").Where(sql.Eq("id", 1)))`)
	if countByStmt.(float64) != 1 {
		t.Fatalf("unexpected stmt-based count result: %#v", countByStmt)
	}

	updated := executeAndDecode(t, session, `repo.Update("users", pg.H{"enabled": false}, sql.Eq("email", "alice@example.com"))`)
	if updated.(float64) != 1 {
		t.Fatalf("unexpected update result: %#v", updated)
	}

	count := executeAndDecode(t, session, `repo.Count("users", sql.Eq("enabled", false))`)
	if count.(float64) != 1 {
		t.Fatalf("unexpected count result: %#v", count)
	}

	rows := executeAndDecode(t, session, `repo.Find("users", sql.In("id", []int{1, 2, 3}))`)
	rowList, ok := rows.([]any)
	if !ok || len(rowList) != 1 {
		t.Fatalf("unexpected rows result: %#v", rows)
	}

	deleted := executeAndDecode(t, session, `repo.Delete("users", sql.Eq("email", "alice@example.com"))`)
	if deleted.(float64) != 1 {
		t.Fatalf("unexpected delete result: %#v", deleted)
	}

	exists := executeAndDecode(t, session, `repo.Exists("users", sql.Eq("email", "alice@example.com"))`)
	if exists.(bool) {
		t.Fatalf("expected deleted row to be absent: %#v", exists)
	}
}

func TestRepoREPLPreviewsBuildersAndProtectsFullTableWrites(t *testing.T) {
	session := newTestREPL(t)

	preview := executeAndDecode(t, session, `pg.Select("*").From("users").Where(sql.Eq("id", 1))`)
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

	if _, err := session.execute(`repo.Delete(pg.DeleteFrom("users"))`); err == nil {
		t.Fatal("expected a statement-only delete to be rejected")
	}

	if _, err := session.execute(`repo.Delete("users", sql.Eq("id", 1), true)`); err == nil {
		t.Fatal("expected more than two args to be rejected")
	}

	deleted := executeAndDecode(t, session, `repo.Delete("users", true)`)
	if deleted.(float64) != 0 {
		t.Fatalf("unexpected full-table delete result: %#v", deleted)
	}
}

func TestRepoREPLDeletesRowByIntegerID(t *testing.T) {
	session := newTestREPL(t)

	executeAndDecode(t, session, `repo.Insert("users", pg.H{"email": "alice@example.com"})`)
	executeAndDecode(t, session, `repo.Insert("users", pg.H{"email": "bob@example.com"})`)

	deleted := executeAndDecode(t, session, `repo.Delete("users", 2)`)
	if deleted.(float64) != 1 {
		t.Fatalf("unexpected delete result: %#v", deleted)
	}

	remaining := executeAndDecode(t, session, `repo.FindAll("users")`)
	remainingRows, ok := remaining.([]any)
	if !ok || len(remainingRows) != 1 {
		t.Fatalf("expected one remaining row, got %#v", remaining)
	}

	row, ok := remainingRows[0].(map[string]any)
	if !ok || row["email"] != "alice@example.com" {
		t.Fatalf("expected alice to remain, got %#v", remainingRows[0])
	}

	if _, err := session.execute(`repo.Delete("users", "2")`); err == nil {
		t.Fatal("expected a non-integer id to be rejected")
	}
}

func TestRepoREPLUpdatesRowByIntegerID(t *testing.T) {
	session := newTestREPL(t)

	executeAndDecode(t, session, `repo.Insert("users", pg.H{"email": "alice@example.com"})`)
	executeAndDecode(t, session, `repo.Insert("users", pg.H{"email": "bob@example.com"})`)

	updated := executeAndDecode(t, session, `repo.Update("users", pg.H{"enabled": true}, 2)`)
	if updated.(float64) != 1 {
		t.Fatalf("unexpected update result: %#v", updated)
	}

	enabled := executeAndDecode(t, session, `repo.Find("users", sql.Eq("enabled", true))`)
	enabledRows, ok := enabled.([]any)
	if !ok || len(enabledRows) != 1 {
		t.Fatalf("expected one enabled row, got %#v", enabled)
	}

	row, ok := enabledRows[0].(map[string]any)
	if !ok || row["email"] != "bob@example.com" {
		t.Fatalf("expected bob to be enabled, got %#v", enabledRows[0])
	}

	if _, err := session.execute(`repo.Update("users", pg.H{"enabled": false}, "2")`); err == nil {
		t.Fatal("expected a non-integer id to be rejected")
	}

	if _, err := session.execute(`repo.Update(pg.Update("users").Set(pg.H{"enabled": false}))`); err == nil {
		t.Fatal("expected a statement-only update to be rejected")
	}

	if _, err := session.execute(`repo.Update("users", pg.H{"enabled": false})`); err == nil {
		t.Fatal("expected a two-arg update to be rejected")
	}

	if _, err := session.execute(`repo.Update("users", pg.H{"enabled": false}, sql.Eq("id", 1), true)`); err == nil {
		t.Fatal("expected more than three args to be rejected")
	}
}

func TestRepoREPLUpdateByIntegerIDMatchesExplicitCondition(t *testing.T) {
	session := newTestREPL(t)

	values := map[string]any{"enabled": true}

	byID, err := session.evaluator.buildUpdateStmt([]any{"users", values, int64(2)})
	if err != nil {
		t.Fatalf("buildUpdateStmt by id: %v", err)
	}

	byCond, err := session.evaluator.buildUpdateStmt([]any{"users", values, reposql.FieldEq(reposql.TableOf("users").Field("id"), int64(2))})
	if err != nil {
		t.Fatalf("buildUpdateStmt by cond: %v", err)
	}

	idQuery, idArgs, err := repo.Preview(session.db, byID)
	if err != nil {
		t.Fatalf("preview by id: %v", err)
	}

	condQuery, condArgs, err := repo.Preview(session.db, byCond)
	if err != nil {
		t.Fatalf("preview by cond: %v", err)
	}

	if idQuery != condQuery || !reflect.DeepEqual(idArgs, condArgs) {
		t.Fatalf("update by id = %q %#v, want %q %#v", idQuery, idArgs, condQuery, condArgs)
	}
}

func TestRepoREPLExposesProjectModelsNamespace(t *testing.T) {
	session := newTestREPL(t)

	value, err := session.evaluator.Eval(`models.User`)
	if err != nil {
		t.Fatalf("models.User: %v", err)
	}

	if _, ok := value.(reflect.Type); !ok {
		t.Fatalf("expected models.User to resolve to a type, got %#v", value)
	}
}

func TestRepoREPLExposesProjectModelsAsTopLevelTypes(t *testing.T) {
	session := newTestREPL(t)

	value, err := session.evaluator.Eval(`User`)
	if err != nil {
		t.Fatalf("User: %v", err)
	}

	if _, ok := value.(reflect.Type); !ok {
		t.Fatalf("expected User to resolve to a type, got %#v", value)
	}
}

func TestRepoREPLRejectsGenericRepoCalls(t *testing.T) {
	session := newTestREPL(t)

	if _, err := session.evaluator.Eval(`repo.FindOne[User](sql.Eq("id", 1))`); err == nil {
		t.Fatal("expected generic repo calls to be rejected")
	}
}

func TestOpenREPLDBUsesEnvDSN(t *testing.T) {
	t.Setenv("AIRWAY_DSN", "")
	t.Setenv("DSN", "sqlite://:memory:")

	db, err := openREPLDB("", "")
	if err != nil {
		t.Fatalf("openREPLDB: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	if db.Driver() != "sqlite" {
		t.Fatalf("driver = %q, want sqlite", db.Driver())
	}
}

func newTestREPL(t *testing.T) *repoREPL {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "repo-repl.sqlite")
	setupDB, err := sql.Open("sqlite", dbPath)
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

	repoDB, err := repo.NewDBWithDriver("sqlite", dbPath)
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
