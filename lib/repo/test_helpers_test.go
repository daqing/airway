package repo

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

const testDSNEnv = "AIRWAY_PG_TEST_DSN"
const testMySQLDSNEnv = "AIRWAY_MYSQL_TEST_DSN"

type Todo struct {
	ID        int64     `db:"id"`
	Title     string    `db:"title"`
	Completed bool      `db:"completed"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func quoteTestIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func quoteTestIdentifierForDriver(driver Driver, name string) string {
	if driver == DriverMySQL {
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}

	return quoteTestIdentifier(name)
}

func testTableName(t *testing.T) string {
	t.Helper()

	name := strings.ToLower(t.Name())
	name = strings.NewReplacer("/", "_", " ", "_", "-", "_").Replace(name)
	return fmt.Sprintf("airway_%s_%d", name, time.Now().UnixNano())
}

func createTodoTable(t *testing.T, db *DB) string {
	t.Helper()
	return createTodoTableWithOptions(t, db, false)
}

func createTodoTableWithUniqueTitle(t *testing.T, db *DB) string {
	t.Helper()
	return createTodoTableWithOptions(t, db, true)
}

func createTodoTableWithOptions(t *testing.T, db *DB, uniqueTitle bool) string {
	t.Helper()

	tableName := testTableName(t)
	var query string
	uniqueClause := ""
	if uniqueTitle {
		uniqueClause = ",\n    UNIQUE(title)"
	}

	switch db.Driver() {
	case DriverSQLite:
		query = fmt.Sprintf(`
CREATE TABLE %s (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
%s)`, quoteTestIdentifierForDriver(db.Driver(), tableName), uniqueClause)
	case DriverMySQL:
		mysqlUnique := ""
		if uniqueTitle {
			mysqlUnique = ",\n\tUNIQUE KEY airway_title_unique (title)"
		}
		query = fmt.Sprintf(`
CREATE TABLE %s (
	id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
	title TEXT NOT NULL,
	completed BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP%s
)`, quoteTestIdentifierForDriver(db.Driver(), tableName), mysqlUnique)
	default:
		query = fmt.Sprintf(`
CREATE TABLE %s (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
%s)`, quoteTestIdentifierForDriver(db.Driver(), tableName), uniqueClause)
	}

	if _, err := db.conn.ExecContext(context.Background(), query); err != nil {
		t.Fatalf("create test table: %v", err)
	}

	t.Cleanup(func() {
		dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteTestIdentifierForDriver(db.Driver(), tableName))
		if _, err := db.conn.ExecContext(context.Background(), dropQuery); err != nil {
			t.Fatalf("drop test table: %v", err)
		}
	})

	return tableName
}

func insertTodoRow(t *testing.T, db *DB, tableName string, title string, completed bool) int64 {
	t.Helper()

	query := fmt.Sprintf("INSERT INTO %s (title, completed) VALUES (?, ?) RETURNING id", quoteTestIdentifierForDriver(db.Driver(), tableName))
	if db.Driver() == DriverPostgres {
		query = fmt.Sprintf("INSERT INTO %s (title, completed) VALUES ($1, $2) RETURNING id", quoteTestIdentifier(tableName))
	}
	if db.Driver() == DriverMySQL {
		result, err := db.conn.ExecContext(context.Background(), fmt.Sprintf("INSERT INTO %s (title, completed) VALUES (?, ?)", quoteTestIdentifierForDriver(db.Driver(), tableName)), title, completed)
		if err != nil {
			t.Fatalf("insert seed row: %v", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("mysql last insert id: %v", err)
		}

		return id
	}

	var id int64
	if err := db.conn.QueryRowxContext(context.Background(), query, title, completed).Scan(&id); err != nil {
		t.Fatalf("insert seed row: %v", err)
	}

	return id
}

func countRows(t *testing.T, db *DB, tableName string) int {
	t.Helper()

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteTestIdentifierForDriver(db.Driver(), tableName))
	var count int
	if err := db.conn.QueryRowxContext(context.Background(), query).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}

	return count
}

func requirePostgresTestDB(t *testing.T) *DB {
	t.Helper()

	dsn := os.Getenv(testDSNEnv)
	if dsn == "" {
		t.Skipf("set %s to run PostgreSQL integration tests", testDSNEnv)
	}

	db, err := NewDBWithDriver("postgres", dsn)
	if err != nil {
		t.Skipf("skip PostgreSQL integration tests: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.conn.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Skipf("skip PostgreSQL integration tests: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func requireSQLiteTestDB(t *testing.T) *DB {
	t.Helper()

	db, err := NewDBWithDriver("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite test db: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func requireMySQLTestDB(t *testing.T) *DB {
	t.Helper()

	dsn := os.Getenv(testMySQLDSNEnv)
	if dsn == "" {
		t.Skipf("set %s to run MySQL integration tests", testMySQLDSNEnv)
	}

	db, err := NewDBWithDriver("mysql", dsn)
	if err != nil {
		t.Skipf("skip MySQL integration tests: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.conn.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Skipf("skip MySQL integration tests: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func forEachTestDB(t *testing.T, fn func(t *testing.T, db *DB)) {
	t.Helper()

	t.Run("sqlite", func(t *testing.T) {
		fn(t, requireSQLiteTestDB(t))
	})

	if strings.TrimSpace(os.Getenv(testDSNEnv)) != "" {
		t.Run("postgres", func(t *testing.T) {
			fn(t, requirePostgresTestDB(t))
		})
	}

	if strings.TrimSpace(os.Getenv(testMySQLDSNEnv)) != "" {
		t.Run("mysql", func(t *testing.T) {
			fn(t, requireMySQLTestDB(t))
		})
	}
}

func forEachPortableTestDB(t *testing.T, fn func(t *testing.T, db *DB)) {
	t.Helper()

	t.Run("sqlite", func(t *testing.T) {
		fn(t, requireSQLiteTestDB(t))
	})

	if strings.TrimSpace(os.Getenv(testMySQLDSNEnv)) != "" {
		t.Run("mysql", func(t *testing.T) {
			fn(t, requireMySQLTestDB(t))
		})
	}
}
