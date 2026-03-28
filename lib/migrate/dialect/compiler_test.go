package dialect

import (
	"strings"
	"testing"

	"github.com/daqing/airway/lib/migrate/schema"
	"github.com/daqing/airway/lib/repo"
)

func TestCompilerCreatesForeignKeyAndIndexSQLForSQLite(t *testing.T) {
	compiler := NewCompiler(repo.DriverSQLite)

	op := schema.CreateTableOp{
		Table: "users",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Type{Kind: schema.TypeID}, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.Type{Kind: schema.TypeString, Length: 100}, Null: schema.Bool(false)},
			{Name: "account_id", Type: schema.Type{Kind: schema.TypeBigInt}, Null: schema.Bool(false)},
		},
		Indexes: []schema.Index{
			{Columns: []string{"account_id"}},
		},
		ForeignKeys: []schema.ForeignKey{
			{Column: "account_id", RefTable: "accounts", RefColumn: "id", OnDelete: "CASCADE"},
		},
	}

	statements, err := compiler.Compile(op)
	if err != nil {
		t.Fatalf("compile create table: %v", err)
	}

	joined := strings.Join(statements, "\n")
	if !strings.Contains(joined, `FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE`) {
		t.Fatalf("expected foreign key in SQL, got:\n%s", joined)
	}

	if !strings.Contains(joined, `CREATE INDEX "idx_users_account_id" ON "users" ("account_id")`) {
		t.Fatalf("expected index SQL, got:\n%s", joined)
	}
}

func TestCompilerSupportsRenameAndRemoveIndexOnSQLite(t *testing.T) {
	compiler := NewCompiler(repo.DriverSQLite)

	statements, err := compiler.Compile(schema.RenameTableOp{From: "users", To: "members"})
	if err != nil {
		t.Fatalf("compile rename table: %v", err)
	}
	if len(statements) != 1 || statements[0] != `ALTER TABLE "users" RENAME TO "members"` {
		t.Fatalf("unexpected rename table SQL: %#v", statements)
	}

	statements, err = compiler.Compile(schema.RenameColumnOp{Table: "members", From: "name", To: "full_name"})
	if err != nil {
		t.Fatalf("compile rename column: %v", err)
	}
	if len(statements) != 1 || statements[0] != `ALTER TABLE "members" RENAME COLUMN "name" TO "full_name"` {
		t.Fatalf("unexpected rename column SQL: %#v", statements)
	}

	statements, err = compiler.Compile(schema.RemoveIndexOp{Table: "members", Name: "members_email_unique_idx"})
	if err != nil {
		t.Fatalf("compile remove index: %v", err)
	}
	if len(statements) != 1 || statements[0] != `DROP INDEX "members_email_unique_idx"` {
		t.Fatalf("unexpected drop index SQL: %#v", statements)
	}
}
