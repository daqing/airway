package repo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/daqing/airway/lib/sql"
)

func TestWithTxCommit(t *testing.T) {
	db := requireSQLiteTestDB(t)
	tableName := createTodoTable(t, db)
	todos := sql.TableOf(tableName)

	err := WithTx(db, func(tx *Tx) error {
		if _, err := InsertWith[Todo](tx.Executor(), sql.Insert(sql.H{"title": "first", "completed": false}).IntoTable(todos)); err != nil {
			return err
		}

		if _, err := InsertWith[Todo](tx.Executor(), sql.Insert(sql.H{"title": "second", "completed": true}).IntoTable(todos)); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Fatalf("commit transaction: %v", err)
	}

	if count := countRows(t, db, tableName); count != 2 {
		t.Fatalf("expected 2 rows after commit, got %d", count)
	}
}

func TestWithTxRollback(t *testing.T) {
	db := requireSQLiteTestDB(t)
	tableName := createTodoTable(t, db)
	todos := sql.TableOf(tableName)

	rollbackErr := errors.New("force rollback")
	err := WithTx(db, func(tx *Tx) error {
		if _, err := InsertWith[Todo](tx.Executor(), sql.Insert(sql.H{"title": "rolled-back", "completed": false}).IntoTable(todos)); err != nil {
			return err
		}

		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("expected rollback error, got %v", err)
	}

	if count := countRows(t, db, tableName); count != 0 {
		t.Fatalf("expected 0 rows after rollback, got %d", count)
	}
}

func TestWithTxPostgresIsolation(t *testing.T) {
	db := requirePostgresTestDB(t)
	tableName := createTodoTable(t, db)
	todos := sql.TableOf(tableName)

	countBuilder := sql.SelectColumns("count(*)").FromTable(todos)

	err := WithTx(db, func(tx *Tx) error {
		if _, err := InsertWith[Todo](tx.Executor(), sql.Insert(sql.H{"title": "uncommitted", "completed": false}).IntoTable(todos)); err != nil {
			return err
		}

		poolCount, err := Count(db, countBuilder)
		if err != nil {
			return err
		}

		if poolCount != 0 {
			return fmt.Errorf("pool-bound read saw %d rows, want 0 (uncommitted)", poolCount)
		}

		txCount, err := tx.Count(countBuilder)
		if err != nil {
			return err
		}

		if txCount != 1 {
			return fmt.Errorf("tx-bound count = %d, want 1", txCount)
		}

		rows, err := FindWith[Todo](tx.Executor(), sql.SelectFields(todos.AllFields()).FromTable(todos))
		if err != nil {
			return err
		}

		if len(rows) != 1 || rows[0].Title != "uncommitted" {
			return fmt.Errorf("tx-bound find returned %#v", rows)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("isolation transaction: %v", err)
	}

	if count := countRows(t, db, tableName); count != 1 {
		t.Fatalf("expected 1 row after commit, got %d", count)
	}
}

func TestWithTxPostgresSkipLocked(t *testing.T) {
	db := requirePostgresTestDB(t)
	tableName := createTodoTable(t, db)
	todos := sql.TableOf(tableName)
	insertTodoRow(t, db, tableName, "Alpha task", false)

	err := WithTx(db, func(tx *Tx) error {
		rows, err := FindWith[Todo](tx.Executor(), sql.SelectFields(todos.AllFields()).FromTable(todos).
			Where(sql.FieldEq(todos.Field("title"), "Alpha task")).
			ForUpdateSkipLocked())
		if err != nil {
			return err
		}

		if len(rows) != 1 || rows[0].Title != "Alpha task" {
			return fmt.Errorf("skip locked find returned %#v", rows)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("skip locked transaction: %v", err)
	}
}

func TestTxExecutorCompileEquivalence(t *testing.T) {
	forEachPortableTestDB(t, func(t *testing.T, db *DB) {
		tableName := createTodoTable(t, db)
		todos := sql.TableOf(tableName)

		builder := sql.SelectFields(todos.AllFields()).FromTable(todos).
			Where(&sql.ConditionGroup{
				Left:  sql.FieldEq(todos.Field("completed"), true),
				Op:    sql.And,
				Right: sql.FieldEq(todos.Field("title"), "alpha"),
			}).
			OrderBy(todos.Field("id").Desc())

		wantQuery, wantArgs, err := Preview(db, builder)
		if err != nil {
			t.Fatalf("preview query: %v", err)
		}

		err = WithTx(db, func(tx *Tx) error {
			gotQuery, gotArgs, err := PreviewWith(tx.Executor(), builder)
			if err != nil {
				return err
			}

			if gotQuery != wantQuery {
				return fmt.Errorf("query mismatch:\n tx: %s\npool: %s", gotQuery, wantQuery)
			}

			if !reflect.DeepEqual(gotArgs, wantArgs) {
				return fmt.Errorf("args mismatch: tx %v, pool %v", gotArgs, wantArgs)
			}

			return nil
		})
		if err != nil {
			t.Fatalf("compare compiled queries: %v", err)
		}
	})
}

func TestWithTxOptimisticLockConcurrency(t *testing.T) {
	forEachTestDB(t, func(t *testing.T, db *DB) {
		tableName := testTableName(t)
		createQuery := fmt.Sprintf(`
CREATE TABLE %s (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title TEXT NOT NULL,
    version BIGINT NOT NULL DEFAULT 0
)`, quoteTestIdentifier(tableName))
		if db.Driver() == DriverSQLite {
			createQuery = fmt.Sprintf(`
CREATE TABLE %s (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    version BIGINT NOT NULL DEFAULT 0
)`, quoteTestIdentifier(tableName))
		}
		if db.Driver() == DriverMySQL {
			createQuery = fmt.Sprintf(`
CREATE TABLE %s (
	id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
	title TEXT NOT NULL,
	version BIGINT NOT NULL DEFAULT 0
)`, quoteTestIdentifierForDriver(db.Driver(), tableName))
		}

		if _, err := db.conn.ExecContext(context.Background(), createQuery); err != nil {
			t.Fatalf("create test table: %v", err)
		}

		t.Cleanup(func() {
			dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteTestIdentifierForDriver(db.Driver(), tableName))
			if _, err := db.conn.ExecContext(context.Background(), dropQuery); err != nil {
				t.Fatalf("drop test table: %v", err)
			}
		})

		items := sql.TableOf(tableName)
		type lockItem struct {
			ID      int64  `db:"id"`
			Title   string `db:"title"`
			Version int64  `db:"version"`
		}

		if _, err := Insert[lockItem](db, sql.Insert(sql.H{"title": "contested", "version": 0}).IntoTable(items)); err != nil {
			t.Fatalf("seed row: %v", err)
		}

		var id int64
		if err := db.conn.QueryRowContext(context.Background(), fmt.Sprintf("SELECT id FROM %s LIMIT 1", quoteTestIdentifierForDriver(db.Driver(), tableName))).Scan(&id); err != nil {
			t.Fatalf("lookup seed row: %v", err)
		}

		type lockResult struct {
			affected int64
			err      error
		}

		results := make(chan lockResult, 2)
		var wg sync.WaitGroup
		for range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := WithTx(db, func(tx *Tx) error {
					affected, err := tx.UpdateAffected(
						sql.UpdateTable(items).
							Set(sql.H{"title": "won", "version": sql.Expr("version + 1")}).
							Where(&sql.ConditionGroup{
								Left:  sql.FieldEq(items.Field("id"), id),
								Op:    sql.And,
								Right: sql.FieldEq(items.Field("version"), 0),
							}),
					)
					if err != nil {
						return err
					}

					results <- lockResult{affected: affected}
					return nil
				})
				if err != nil {
					results <- lockResult{err: err}
				}
			}()
		}
		wg.Wait()
		close(results)

		winners := 0
		for result := range results {
			if result.err != nil {
				t.Fatalf("optimistic lock transaction: %v", result.err)
			}

			if result.affected == 1 {
				winners++
			}
		}

		if winners != 1 {
			t.Fatalf("expected exactly 1 winning update, got %d", winners)
		}

		var version int64
		versionQuery := db.rebind(fmt.Sprintf("SELECT version FROM %s WHERE id = ?", quoteTestIdentifierForDriver(db.Driver(), tableName)))
		if err := db.conn.QueryRowContext(context.Background(), versionQuery, id).Scan(&version); err != nil {
			t.Fatalf("read final version: %v", err)
		}

		if version != 1 {
			t.Fatalf("expected version 1 after one winning update, got %d", version)
		}
	})
}
