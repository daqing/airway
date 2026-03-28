package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type State struct {
	Tables map[string]*TableState
}

type TableState struct {
	Name        string
	Columns     []Column
	Indexes     []Index
	ForeignKeys []ForeignKey
}

type Snapshot struct {
	Version int    `json:"version"`
	Known   bool   `json:"known"`
	State   *State `json:"state,omitempty"`
}

func NewState() *State {
	return &State{Tables: map[string]*TableState{}}
}

func (s *State) Clone() *State {
	if s == nil {
		return nil
	}

	cloned := NewState()
	for name, table := range s.Tables {
		cloned.Tables[name] = table.Clone()
	}

	return cloned
}

func (t *TableState) Clone() *TableState {
	if t == nil {
		return nil
	}

	clone := &TableState{
		Name:        t.Name,
		Columns:     append([]Column{}, t.Columns...),
		Indexes:     append([]Index{}, t.Indexes...),
		ForeignKeys: append([]ForeignKey{}, t.ForeignKeys...),
	}

	return clone
}

func (s *State) Table(name string) (*TableState, bool) {
	if s == nil {
		return nil, false
	}

	table, ok := s.Tables[strings.TrimSpace(name)]
	return table, ok
}

func (s *State) ApplyAll(ops []Operation) error {
	for _, op := range ops {
		if err := s.Apply(op); err != nil {
			return err
		}
	}

	return nil
}

func (s *State) Apply(op Operation) error {
	if s == nil {
		return fmt.Errorf("schema state is nil")
	}

	switch actual := op.(type) {
	case CreateTableOp:
		s.Tables[actual.Table] = (&TableState{
			Name:        actual.Table,
			Columns:     append([]Column{}, actual.Columns...),
			Indexes:     append([]Index{}, actual.Indexes...),
			ForeignKeys: append([]ForeignKey{}, actual.ForeignKeys...),
		}).Clone()
		return nil
	case DropTableOp:
		delete(s.Tables, actual.Table)
		return nil
	case RenameTableOp:
		table, ok := s.Tables[actual.From]
		if !ok {
			return fmt.Errorf("table %s does not exist", actual.From)
		}
		delete(s.Tables, actual.From)
		table = table.Clone()
		table.Name = actual.To
		s.Tables[actual.To] = table
		return nil
	case AddColumnOp:
		table, ok := s.Tables[actual.Table]
		if !ok {
			return fmt.Errorf("table %s does not exist", actual.Table)
		}
		table.Columns = append(table.Columns, actual.Column)
		return nil
	case RemoveColumnOp:
		table, ok := s.Tables[actual.Table]
		if !ok {
			return fmt.Errorf("table %s does not exist", actual.Table)
		}
		index := findColumnIndex(table.Columns, actual.ColumnName)
		if index < 0 {
			return fmt.Errorf("column %s does not exist on table %s", actual.ColumnName, actual.Table)
		}
		table.Columns = append(table.Columns[:index], table.Columns[index+1:]...)
		table.Indexes = filterIndexesWithoutColumn(table.Indexes, actual.ColumnName)
		table.ForeignKeys = filterForeignKeysWithoutColumn(table.ForeignKeys, actual.ColumnName)
		return nil
	case ChangeColumnOp:
		table, ok := s.Tables[actual.Table]
		if !ok {
			return fmt.Errorf("table %s does not exist", actual.Table)
		}
		index := findColumnIndex(table.Columns, actual.Column.Name)
		if index < 0 {
			return fmt.Errorf("column %s does not exist on table %s", actual.Column.Name, actual.Table)
		}
		table.Columns[index] = actual.Column
		return nil
	case SetNullOp:
		table, ok := s.Tables[actual.Table]
		if !ok {
			return fmt.Errorf("table %s does not exist", actual.Table)
		}
		index := findColumnIndex(table.Columns, actual.ColumnName)
		if index < 0 {
			return fmt.Errorf("column %s does not exist on table %s", actual.ColumnName, actual.Table)
		}
		table.Columns[index].Null = Bool(actual.Nullable)
		return nil
	case SetDefaultOp:
		table, ok := s.Tables[actual.Table]
		if !ok {
			return fmt.Errorf("table %s does not exist", actual.Table)
		}
		index := findColumnIndex(table.Columns, actual.ColumnName)
		if index < 0 {
			return fmt.Errorf("column %s does not exist on table %s", actual.ColumnName, actual.Table)
		}
		if actual.Remove {
			table.Columns[index].Default = nil
		} else {
			table.Columns[index].Default = actual.Default
		}
		return nil
	case RenameColumnOp:
		table, ok := s.Tables[actual.Table]
		if !ok {
			return fmt.Errorf("table %s does not exist", actual.Table)
		}
		index := findColumnIndex(table.Columns, actual.From)
		if index < 0 {
			return fmt.Errorf("column %s does not exist on table %s", actual.From, actual.Table)
		}
		table.Columns[index].Name = actual.To
		for i := range table.Indexes {
			for j := range table.Indexes[i].Columns {
				if table.Indexes[i].Columns[j] == actual.From {
					table.Indexes[i].Columns[j] = actual.To
				}
			}
		}
		for i := range table.ForeignKeys {
			if table.ForeignKeys[i].Column == actual.From {
				table.ForeignKeys[i].Column = actual.To
			}
		}
		return nil
	case AddIndexOp:
		table, ok := s.Tables[actual.Table]
		if !ok {
			return fmt.Errorf("table %s does not exist", actual.Table)
		}
		table.Indexes = append(table.Indexes, actual.Index)
		return nil
	case RemoveIndexOp:
		table, ok := s.Tables[actual.Table]
		if !ok {
			return fmt.Errorf("table %s does not exist", actual.Table)
		}
		table.Indexes = removeIndex(table.Name, table.Indexes, actual)
		return nil
	case AddForeignKeyOp:
		table, ok := s.Tables[actual.Table]
		if !ok {
			return fmt.Errorf("table %s does not exist", actual.Table)
		}
		table.ForeignKeys = append(table.ForeignKeys, actual.ForeignKey)
		return nil
	case RemoveForeignKeyOp:
		table, ok := s.Tables[actual.Table]
		if !ok {
			return fmt.Errorf("table %s does not exist", actual.Table)
		}
		table.ForeignKeys = removeForeignKey(table.ForeignKeys, actual)
		return nil
	case RawSQLOp:
		return nil
	default:
		return fmt.Errorf("unsupported schema operation %T", op)
	}
}

func (s *State) Column(tableName string, columnName string) (Column, bool) {
	table, ok := s.Table(tableName)
	if !ok {
		return Column{}, false
	}

	index := findColumnIndex(table.Columns, columnName)
	if index < 0 {
		return Column{}, false
	}

	return table.Columns[index], true
}

func LoadSnapshot(path string) (*Snapshot, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var snapshot Snapshot
	if err := json.Unmarshal(content, &snapshot); err != nil {
		return nil, err
	}

	if snapshot.State == nil {
		snapshot.State = NewState()
	}

	return &snapshot, nil
}

func SaveSnapshot(path string, state *State, known bool) error {
	snapshot := Snapshot{
		Version: 1,
		Known:   known,
	}
	if state != nil {
		snapshot.State = state.Clone()
	} else {
		snapshot.State = NewState()
	}

	content, err := json.MarshalIndent(normalizeSnapshot(snapshot), "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, content, 0o644)
}

func normalizeSnapshot(snapshot Snapshot) Snapshot {
	if snapshot.State == nil {
		snapshot.State = NewState()
	}

	names := make([]string, 0, len(snapshot.State.Tables))
	for name := range snapshot.State.Tables {
		names = append(names, name)
	}
	sort.Strings(names)

	normalized := NewState()
	for _, name := range names {
		table := snapshot.State.Tables[name].Clone()
		sort.Slice(table.Columns, func(i, j int) bool { return table.Columns[i].Name < table.Columns[j].Name })
		sort.Slice(table.Indexes, func(i, j int) bool {
			return indexSortKey(table.Name, table.Indexes[i]) < indexSortKey(table.Name, table.Indexes[j])
		})
		sort.Slice(table.ForeignKeys, func(i, j int) bool {
			return foreignKeySortKey(table.ForeignKeys[i]) < foreignKeySortKey(table.ForeignKeys[j])
		})
		normalized.Tables[name] = table
	}

	snapshot.State = normalized
	return snapshot
}

func indexSortKey(table string, idx Index) string {
	name := strings.TrimSpace(idx.Name)
	if name == "" {
		name = fmt.Sprintf("idx_%s_%s", table, strings.Join(idx.Columns, "_"))
	}
	return name
}

func foreignKeySortKey(fk ForeignKey) string {
	name := strings.TrimSpace(fk.Name)
	if name == "" {
		name = fmt.Sprintf("fk_%s_%s_%s", fk.Column, fk.RefTable, fk.RefColumn)
	}
	return name
}

func findColumnIndex(columns []Column, name string) int {
	for i, column := range columns {
		if column.Name == name {
			return i
		}
	}

	return -1
}

func filterIndexesWithoutColumn(indexes []Index, column string) []Index {
	filtered := make([]Index, 0, len(indexes))
	for _, idx := range indexes {
		skip := false
		for _, current := range idx.Columns {
			if current == column {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, idx)
		}
	}
	return filtered
}

func filterForeignKeysWithoutColumn(foreignKeys []ForeignKey, column string) []ForeignKey {
	filtered := make([]ForeignKey, 0, len(foreignKeys))
	for _, fk := range foreignKeys {
		if fk.Column == column {
			continue
		}
		filtered = append(filtered, fk)
	}
	return filtered
}

func removeIndex(table string, indexes []Index, op RemoveIndexOp) []Index {
	targetName := strings.TrimSpace(op.Name)
	if targetName == "" && len(op.Columns) > 0 {
		targetName = fmt.Sprintf("idx_%s_%s", table, strings.Join(op.Columns, "_"))
	}

	filtered := make([]Index, 0, len(indexes))
	for _, idx := range indexes {
		name := strings.TrimSpace(idx.Name)
		if name == "" {
			name = fmt.Sprintf("idx_%s_%s", table, strings.Join(idx.Columns, "_"))
		}
		if name == targetName {
			continue
		}
		filtered = append(filtered, idx)
	}

	return filtered
}

func removeForeignKey(foreignKeys []ForeignKey, op RemoveForeignKeyOp) []ForeignKey {
	filtered := make([]ForeignKey, 0, len(foreignKeys))
	for _, fk := range foreignKeys {
		if strings.TrimSpace(op.Name) != "" && strings.TrimSpace(fk.Name) == strings.TrimSpace(op.Name) {
			continue
		}
		if op.Column != "" && op.RefTable != "" && fk.Column == op.Column && fk.RefTable == op.RefTable && (op.RefColumn == "" || fk.RefColumn == op.RefColumn) {
			continue
		}
		filtered = append(filtered, fk)
	}
	return filtered
}
