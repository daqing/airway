package schema

import "testing"

func TestRegisterChangeBuildsUpAndDownOps(t *testing.T) {
	ResetRegistryForTest()
	t.Cleanup(ResetRegistryForTest)

	RegisterChange("20240101000000", "create_users", func(m *Migrator) {
		m.CreateTable("users", func(tb *Table) {
			tb.ID()
			tb.Timestamps()
		})
	})

	defs := Definitions()
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}

	def := defs[0]
	if def.Version != "20240101000000" || def.Name != "create_users" {
		t.Fatalf("unexpected definition: %#v", def)
	}

	if len(def.UpOps) != 1 {
		t.Fatalf("expected 1 up op, got %#v", def.UpOps)
	}
	if _, ok := def.UpOps[0].(CreateTableOp); !ok {
		t.Fatalf("expected up op to be CreateTableOp, got %T", def.UpOps[0])
	}

	if len(def.DownOps) != 1 {
		t.Fatalf("expected 1 down op, got %#v", def.DownOps)
	}
	if _, ok := def.DownOps[0].(DropTableOp); !ok {
		t.Fatalf("expected down op to be DropTableOp, got %T", def.DownOps[0])
	}
}

func TestStateApplyCreateTable(t *testing.T) {
	state := NewState()

	op := CreateTableOp{
		Table: "users",
		Columns: []Column{
			{Name: "id"},
		},
	}

	if err := state.Apply(op); err != nil {
		t.Fatalf("apply create table: %v", err)
	}

	table, ok := state.Table("users")
	if !ok {
		t.Fatalf("expected table users in state")
	}

	if len(table.Columns) != 1 || table.Columns[0].Name != "id" {
		t.Fatalf("unexpected columns: %#v", table.Columns)
	}
}
