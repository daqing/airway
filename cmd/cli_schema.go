package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/daqing/airway/lib/migrate/schema"
	"github.com/daqing/airway/lib/repo"
)

func runCLISchema(args []string) error {
	if len(args) == 0 {
		return runCLISchemaShow(nil)
	}

	subcommand := strings.ToLower(strings.TrimSpace(args[0]))
	switch subcommand {
	case "show":
		return runCLISchemaShow(args[1:])
	case "dump":
		return runCLISchemaDump(args[1:])
	default:
		return fmt.Errorf("unknown schema command: %s", subcommand)
	}
}

func runCLISchemaShow(_ []string) error {
	state, known, err := currentSchemaState()
	if err != nil {
		return err
	}

	if !known {
		fmt.Println("schema state: unknown")
		fmt.Printf("snapshot file: %s\n", schemaSnapshotPath)
		return nil
	}

	printSchemaState(os.Stdout, state)
	return nil
}

func runCLISchemaDump(_ []string) error {
	state, known, err := currentSchemaState()
	if err != nil {
		return err
	}

	if err := schema.SaveSnapshot(schemaSnapshotPath, state, known); err != nil {
		return err
	}

	fmt.Printf("Schema snapshot written to %s\n", schemaSnapshotPath)
	return nil
}

func currentSchemaState() (*schema.State, bool, error) {
	dsn, dsnErr := cliDSN()
	if dsnErr == nil {
		db, err := repo.NewDB(dsn)
		if err != nil {
			return nil, false, err
		}
		defer db.Close()

		state, err := repo.InspectSchema(db)
		if err != nil {
			return nil, false, err
		}

		return state, true, nil
	}

	if snapshot, err := schema.LoadSnapshot(schemaSnapshotPath); err == nil {
		if snapshot.State == nil {
			snapshot.State = schema.NewState()
		}
		return snapshot.State, snapshot.Known, nil
	}

	return nil, false, dsnErr
}

func printSchemaState(w *os.File, state *schema.State) {
	if state == nil || len(state.Tables) == 0 {
		fmt.Fprintln(w, "schema is empty")
		return
	}

	tableNames := make([]string, 0, len(state.Tables))
	for name := range state.Tables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)

	for _, tableName := range tableNames {
		table := state.Tables[tableName]
		fmt.Fprintf(w, "table %s\n", table.Name)
		for _, column := range table.Columns {
			fmt.Fprintf(w, "  column %s %s", column.Name, describeColumnType(column))
			if column.Null != nil && !*column.Null {
				fmt.Fprint(w, " not_null")
			}
			if column.Default != nil {
				fmt.Fprintf(w, " default=%v", column.Default)
			}
			if column.PrimaryKey {
				fmt.Fprint(w, " primary_key")
			}
			fmt.Fprintln(w)
		}
		for _, index := range table.Indexes {
			name := strings.TrimSpace(index.Name)
			if name == "" {
				name = "idx_" + table.Name + "_" + strings.Join(index.Columns, "_")
			}
			kind := "index"
			if index.Unique {
				kind = "unique_index"
			}
			fmt.Fprintf(w, "  %s %s (%s)\n", kind, name, strings.Join(index.Columns, ", "))
		}
		for _, fk := range table.ForeignKeys {
			name := strings.TrimSpace(fk.Name)
			if name == "" {
				name = "fk_" + table.Name + "_" + fk.Column
			}
			fmt.Fprintf(w, "  foreign_key %s %s -> %s(%s)", name, fk.Column, fk.RefTable, fk.RefColumn)
			if fk.OnDelete != "" {
				fmt.Fprintf(w, " on_delete=%s", strings.ToLower(fk.OnDelete))
			}
			if fk.OnUpdate != "" {
				fmt.Fprintf(w, " on_update=%s", strings.ToLower(fk.OnUpdate))
			}
			fmt.Fprintln(w)
		}
	}
}

func describeColumnType(column schema.Column) string {
	switch column.Type.Kind {
	case schema.TypeString:
		if column.Type.Length > 0 {
			return fmt.Sprintf("string(%d)", column.Type.Length)
		}
		return "string"
	default:
		return string(column.Type.Kind)
	}
}

func encodeSnapshot(state *schema.State, known bool) ([]byte, error) {
	snapshot := schema.Snapshot{Version: 1, Known: known, State: state}
	return json.MarshalIndent(snapshot, "", "  ")
}
