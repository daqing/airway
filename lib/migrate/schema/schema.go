package schema

import (
	"fmt"
	"strings"
	"sync"
)

type TypeKind string

const (
	TypeID       TypeKind = "id"
	TypeString   TypeKind = "string"
	TypeText     TypeKind = "text"
	TypeBoolean  TypeKind = "boolean"
	TypeInteger  TypeKind = "integer"
	TypeBigInt   TypeKind = "bigint"
	TypeDateTime TypeKind = "datetime"
	TypeJSON     TypeKind = "json"
)

type Type struct {
	Kind      TypeKind
	Length    int
	Precision int
	Scale     int
}

type Column struct {
	Name          string
	Type          Type
	Null          *bool
	Default       any
	PrimaryKey    bool
	AutoIncrement bool
	Unique        bool
}

type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

type ForeignKey struct {
	Name      string
	Column    string
	RefTable  string
	RefColumn string
	OnDelete  string
	OnUpdate  string
}

type Operation interface {
	Reverse() (Operation, bool)
}

type CreateTableOp struct {
	Table       string
	Columns     []Column
	Indexes     []Index
	ForeignKeys []ForeignKey
}

func (op CreateTableOp) Reverse() (Operation, bool) {
	return DropTableOp{Table: op.Table}, true
}

type DropTableOp struct {
	Table string
}

func (op DropTableOp) Reverse() (Operation, bool) {
	return nil, false
}

type AddColumnOp struct {
	Table  string
	Column Column
}

func (op AddColumnOp) Reverse() (Operation, bool) {
	return RemoveColumnOp{Table: op.Table, ColumnName: op.Column.Name}, true
}

type RemoveColumnOp struct {
	Table      string
	ColumnName string
}

func (op RemoveColumnOp) Reverse() (Operation, bool) {
	return nil, false
}

type ChangeColumnOp struct {
	Table  string
	Column Column
}

func (op ChangeColumnOp) Reverse() (Operation, bool) {
	return nil, false
}

type SetNullOp struct {
	Table      string
	ColumnName string
	Nullable   bool
}

func (op SetNullOp) Reverse() (Operation, bool) {
	return SetNullOp{Table: op.Table, ColumnName: op.ColumnName, Nullable: !op.Nullable}, true
}

type SetDefaultOp struct {
	Table      string
	ColumnName string
	Default    any
	Remove     bool
}

func (op SetDefaultOp) Reverse() (Operation, bool) {
	return nil, false
}

type RenameTableOp struct {
	From string
	To   string
}

func (op RenameTableOp) Reverse() (Operation, bool) {
	return RenameTableOp{From: op.To, To: op.From}, true
}

type RenameColumnOp struct {
	Table string
	From  string
	To    string
}

func (op RenameColumnOp) Reverse() (Operation, bool) {
	return RenameColumnOp{Table: op.Table, From: op.To, To: op.From}, true
}

type AddIndexOp struct {
	Table string
	Index Index
}

func (op AddIndexOp) Reverse() (Operation, bool) {
	return RemoveIndexOp{Table: op.Table, Name: op.Index.Name, Columns: append([]string{}, op.Index.Columns...)}, true
}

type RemoveIndexOp struct {
	Table   string
	Name    string
	Columns []string
}

func (op RemoveIndexOp) Reverse() (Operation, bool) {
	if strings.TrimSpace(op.Name) == "" && len(op.Columns) == 0 {
		return nil, false
	}

	return AddIndexOp{Table: op.Table, Index: Index{Name: op.Name, Columns: append([]string{}, op.Columns...)}}, true
}

type AddForeignKeyOp struct {
	Table      string
	ForeignKey ForeignKey
}

func (op AddForeignKeyOp) Reverse() (Operation, bool) {
	return RemoveForeignKeyOp{
		Table:     op.Table,
		Name:      op.ForeignKey.Name,
		Column:    op.ForeignKey.Column,
		RefTable:  op.ForeignKey.RefTable,
		RefColumn: op.ForeignKey.RefColumn,
	}, true
}

type RemoveForeignKeyOp struct {
	Table     string
	Name      string
	Column    string
	RefTable  string
	RefColumn string
}

func (op RemoveForeignKeyOp) Reverse() (Operation, bool) {
	if strings.TrimSpace(op.Column) == "" || strings.TrimSpace(op.RefTable) == "" {
		return nil, false
	}

	return AddForeignKeyOp{
		Table: op.Table,
		ForeignKey: ForeignKey{
			Name:      op.Name,
			Column:    op.Column,
			RefTable:  op.RefTable,
			RefColumn: op.RefColumn,
		},
	}, true
}

type RawSQLOp struct {
	UpSQL   string
	DownSQL string
}

func (op RawSQLOp) Reverse() (Operation, bool) {
	if strings.TrimSpace(op.DownSQL) == "" {
		return nil, false
	}

	return RawSQLOp{UpSQL: op.DownSQL, DownSQL: op.UpSQL}, true
}

type Definition struct {
	Version string
	Name    string
	UpOps   []Operation
	DownOps []Operation
}

type Migrator struct {
	ops []Operation
}

func (m *Migrator) Ops() []Operation {
	return append([]Operation{}, m.ops...)
}

func (m *Migrator) CreateTable(name string, fn func(*Table)) {
	op := &CreateTableOp{Table: strings.TrimSpace(name)}
	if fn != nil {
		fn(&Table{op: op})
	}

	m.ops = append(m.ops, *op)
}

func (m *Migrator) DropTable(name string) {
	m.ops = append(m.ops, DropTableOp{Table: strings.TrimSpace(name)})
}

func (m *Migrator) RenameTable(from string, to string) {
	m.ops = append(m.ops, RenameTableOp{From: strings.TrimSpace(from), To: strings.TrimSpace(to)})
}

func (m *Migrator) AddColumn(table string, column Column) {
	m.ops = append(m.ops, AddColumnOp{Table: strings.TrimSpace(table), Column: column})
}

func (m *Migrator) RemoveColumn(table string, columnName string) {
	m.ops = append(m.ops, RemoveColumnOp{Table: strings.TrimSpace(table), ColumnName: strings.TrimSpace(columnName)})
}

func (m *Migrator) ChangeColumn(table string, column Column) {
	m.ops = append(m.ops, ChangeColumnOp{Table: strings.TrimSpace(table), Column: column})
}

func (m *Migrator) SetNull(table string, columnName string, nullable bool) {
	m.ops = append(m.ops, SetNullOp{Table: strings.TrimSpace(table), ColumnName: strings.TrimSpace(columnName), Nullable: nullable})
}

func (m *Migrator) SetDefault(table string, columnName string, value any) {
	m.ops = append(m.ops, SetDefaultOp{Table: strings.TrimSpace(table), ColumnName: strings.TrimSpace(columnName), Default: value})
}

func (m *Migrator) RemoveDefault(table string, columnName string) {
	m.ops = append(m.ops, SetDefaultOp{Table: strings.TrimSpace(table), ColumnName: strings.TrimSpace(columnName), Remove: true})
}

func (m *Migrator) RenameColumn(table string, from string, to string) {
	m.ops = append(m.ops, RenameColumnOp{
		Table: strings.TrimSpace(table),
		From:  strings.TrimSpace(from),
		To:    strings.TrimSpace(to),
	})
}

func (m *Migrator) AddIndex(table string, columns ...string) *StandaloneIndexBuilder {
	index := Index{Columns: append([]string{}, columns...)}
	m.ops = append(m.ops, AddIndexOp{Table: strings.TrimSpace(table), Index: index})
	return &StandaloneIndexBuilder{migrator: m, index: len(m.ops) - 1}
}

func (m *Migrator) RemoveIndex(table string, nameOrColumns ...string) {
	op := RemoveIndexOp{Table: strings.TrimSpace(table)}
	if len(nameOrColumns) == 1 {
		op.Name = strings.TrimSpace(nameOrColumns[0])
	} else {
		op.Columns = append([]string{}, nameOrColumns...)
	}

	m.ops = append(m.ops, op)
}

func (m *Migrator) AddForeignKey(table string, column string, refTable string, refColumn ...string) *StandaloneForeignKeyBuilder {
	targetColumn := "id"
	if len(refColumn) > 0 && strings.TrimSpace(refColumn[0]) != "" {
		targetColumn = strings.TrimSpace(refColumn[0])
	}

	op := AddForeignKeyOp{
		Table: strings.TrimSpace(table),
		ForeignKey: ForeignKey{
			Column:    strings.TrimSpace(column),
			RefTable:  strings.TrimSpace(refTable),
			RefColumn: targetColumn,
		},
	}
	m.ops = append(m.ops, op)
	return &StandaloneForeignKeyBuilder{migrator: m, index: len(m.ops) - 1}
}

func (m *Migrator) RemoveForeignKey(table string, column string, refTable string, refColumn ...string) {
	targetColumn := "id"
	if len(refColumn) > 0 && strings.TrimSpace(refColumn[0]) != "" {
		targetColumn = strings.TrimSpace(refColumn[0])
	}

	m.ops = append(m.ops, RemoveForeignKeyOp{
		Table:     strings.TrimSpace(table),
		Column:    strings.TrimSpace(column),
		RefTable:  strings.TrimSpace(refTable),
		RefColumn: targetColumn,
	})
}

func (m *Migrator) Exec(sql string) {
	m.ops = append(m.ops, RawSQLOp{UpSQL: sql})
}

func (m *Migrator) Reversible(up func(*Migrator), down func(*Migrator)) {
	upMigrator := &Migrator{}
	if up != nil {
		up(upMigrator)
	}

	downMigrator := &Migrator{}
	if down != nil {
		down(downMigrator)
	}

	m.ops = append(m.ops, RawSQLOp{
		UpSQL:   joinOpsAsRawSQL(upMigrator.ops),
		DownSQL: joinOpsAsRawSQL(downMigrator.ops),
	})
}

func joinOpsAsRawSQL(ops []Operation) string {
	parts := make([]string, 0, len(ops))
	for _, op := range ops {
		raw, ok := op.(RawSQLOp)
		if !ok {
			panic("Reversible currently supports raw SQL only")
		}

		if strings.TrimSpace(raw.UpSQL) != "" {
			parts = append(parts, raw.UpSQL)
		}
	}

	return strings.Join(parts, ";\n")
}

type Table struct {
	op *CreateTableOp
}

func (t *Table) addColumn(column Column) *ColumnBuilder {
	t.op.Columns = append(t.op.Columns, column)
	return &ColumnBuilder{table: t, index: len(t.op.Columns) - 1}
}

func (t *Table) ID() *ColumnBuilder {
	return t.addColumn(Column{
		Name:          "id",
		Type:          Type{Kind: TypeID},
		PrimaryKey:    true,
		AutoIncrement: true,
	})
}

func (t *Table) String(name string, length ...int) *ColumnBuilder {
	size := 255
	if len(length) > 0 && length[0] > 0 {
		size = length[0]
	}

	return t.addColumn(Column{Name: name, Type: Type{Kind: TypeString, Length: size}})
}

func (t *Table) Text(name string) *ColumnBuilder {
	return t.addColumn(Column{Name: name, Type: Type{Kind: TypeText}})
}

func (t *Table) Boolean(name string) *ColumnBuilder {
	return t.addColumn(Column{Name: name, Type: Type{Kind: TypeBoolean}})
}

func (t *Table) Integer(name string) *ColumnBuilder {
	return t.addColumn(Column{Name: name, Type: Type{Kind: TypeInteger}})
}

func (t *Table) BigInt(name string) *ColumnBuilder {
	return t.addColumn(Column{Name: name, Type: Type{Kind: TypeBigInt}})
}

func (t *Table) JSON(name string) *ColumnBuilder {
	return t.addColumn(Column{Name: name, Type: Type{Kind: TypeJSON}})
}

func (t *Table) DateTime(name string) *ColumnBuilder {
	return t.addColumn(Column{Name: name, Type: Type{Kind: TypeDateTime}})
}

func (t *Table) Timestamps() {
	t.DateTime("created_at").Null(false).Default(CurrentTimestamp)
	t.DateTime("updated_at").Null(false).Default(CurrentTimestamp)
}

func (t *Table) Index(columns ...string) *IndexBuilder {
	t.op.Indexes = append(t.op.Indexes, Index{Columns: append([]string{}, columns...)})
	return &IndexBuilder{table: t, index: len(t.op.Indexes) - 1}
}

func (t *Table) UniqueIndex(columns ...string) {
	t.Index(columns...).Unique()
}

func (t *Table) ForeignKey(column string, refTable string, refColumn ...string) *ForeignKeyBuilder {
	targetColumn := "id"
	if len(refColumn) > 0 && strings.TrimSpace(refColumn[0]) != "" {
		targetColumn = strings.TrimSpace(refColumn[0])
	}

	t.op.ForeignKeys = append(t.op.ForeignKeys, ForeignKey{
		Column:    strings.TrimSpace(column),
		RefTable:  strings.TrimSpace(refTable),
		RefColumn: targetColumn,
	})

	return &ForeignKeyBuilder{table: t, index: len(t.op.ForeignKeys) - 1}
}

func (t *Table) References(name string) *ReferenceBuilder {
	base := singularName(name)
	columnName := base + "_id"
	columnBuilder := t.BigInt(columnName)

	return &ReferenceBuilder{
		table:         t,
		columnBuilder: columnBuilder,
		refTable:      pluralName(base),
		refColumn:     "id",
	}
}

type ColumnBuilder struct {
	table *Table
	index int
}

func (b *ColumnBuilder) column() *Column {
	return &b.table.op.Columns[b.index]
}

func (b *ColumnBuilder) Null(allowed bool) *ColumnBuilder {
	b.column().Null = Bool(allowed)
	return b
}

func (b *ColumnBuilder) Default(value any) *ColumnBuilder {
	b.column().Default = value
	return b
}

func (b *ColumnBuilder) Unique() *ColumnBuilder {
	b.column().Unique = true
	return b
}

type IndexBuilder struct {
	table *Table
	index int
}

func (b *IndexBuilder) idx() *Index {
	return &b.table.op.Indexes[b.index]
}

func (b *IndexBuilder) Unique() *IndexBuilder {
	b.idx().Unique = true
	return b
}

func (b *IndexBuilder) Name(name string) *IndexBuilder {
	b.idx().Name = strings.TrimSpace(name)
	return b
}

type StandaloneIndexBuilder struct {
	migrator *Migrator
	index    int
}

func (b *StandaloneIndexBuilder) op() *AddIndexOp {
	op, ok := b.migrator.ops[b.index].(AddIndexOp)
	if !ok {
		panic("migration operation is not AddIndexOp")
	}

	return &op
}

func (b *StandaloneIndexBuilder) save(op AddIndexOp) {
	b.migrator.ops[b.index] = op
}

func (b *StandaloneIndexBuilder) Unique() *StandaloneIndexBuilder {
	op := b.op()
	op.Index.Unique = true
	b.save(*op)
	return b
}

func (b *StandaloneIndexBuilder) Name(name string) *StandaloneIndexBuilder {
	op := b.op()
	op.Index.Name = strings.TrimSpace(name)
	b.save(*op)
	return b
}

type ForeignKeyBuilder struct {
	table *Table
	index int
}

func (b *ForeignKeyBuilder) fk() *ForeignKey {
	return &b.table.op.ForeignKeys[b.index]
}

func (b *ForeignKeyBuilder) Name(name string) *ForeignKeyBuilder {
	b.fk().Name = strings.TrimSpace(name)
	return b
}

func (b *ForeignKeyBuilder) OnDelete(action string) *ForeignKeyBuilder {
	b.fk().OnDelete = strings.TrimSpace(strings.ToUpper(action))
	return b
}

func (b *ForeignKeyBuilder) OnUpdate(action string) *ForeignKeyBuilder {
	b.fk().OnUpdate = strings.TrimSpace(strings.ToUpper(action))
	return b
}

type StandaloneForeignKeyBuilder struct {
	migrator *Migrator
	index    int
}

func (b *StandaloneForeignKeyBuilder) op() *AddForeignKeyOp {
	op, ok := b.migrator.ops[b.index].(AddForeignKeyOp)
	if !ok {
		panic("migration operation is not AddForeignKeyOp")
	}

	return &op
}

func (b *StandaloneForeignKeyBuilder) save(op AddForeignKeyOp) {
	b.migrator.ops[b.index] = op
}

func (b *StandaloneForeignKeyBuilder) Name(name string) *StandaloneForeignKeyBuilder {
	op := b.op()
	op.ForeignKey.Name = strings.TrimSpace(name)
	b.save(*op)
	return b
}

func (b *StandaloneForeignKeyBuilder) OnDelete(action string) *StandaloneForeignKeyBuilder {
	op := b.op()
	op.ForeignKey.OnDelete = strings.TrimSpace(strings.ToUpper(action))
	b.save(*op)
	return b
}

func (b *StandaloneForeignKeyBuilder) OnUpdate(action string) *StandaloneForeignKeyBuilder {
	op := b.op()
	op.ForeignKey.OnUpdate = strings.TrimSpace(strings.ToUpper(action))
	b.save(*op)
	return b
}

type ReferenceBuilder struct {
	table         *Table
	columnBuilder *ColumnBuilder
	refTable      string
	refColumn     string
}

func (b *ReferenceBuilder) Null(allowed bool) *ReferenceBuilder {
	b.columnBuilder.Null(allowed)
	return b
}

func (b *ReferenceBuilder) Default(value any) *ReferenceBuilder {
	b.columnBuilder.Default(value)
	return b
}

func (b *ReferenceBuilder) Index() *ReferenceBuilder {
	columnName := b.columnBuilder.column().Name
	b.table.Index(columnName)
	return b
}

func (b *ReferenceBuilder) Unique() *ReferenceBuilder {
	columnName := b.columnBuilder.column().Name
	b.table.UniqueIndex(columnName)
	return b
}

func (b *ReferenceBuilder) ForeignKey() *ForeignKeyBuilder {
	return b.table.ForeignKey(b.columnBuilder.column().Name, b.refTable, b.refColumn)
}

func (b *ReferenceBuilder) References(table string, column ...string) *ReferenceBuilder {
	b.refTable = strings.TrimSpace(table)
	if len(column) > 0 && strings.TrimSpace(column[0]) != "" {
		b.refColumn = strings.TrimSpace(column[0])
	}
	return b
}

type DefaultExpr string

const CurrentTimestamp DefaultExpr = "CURRENT_TIMESTAMP"

func Bool(v bool) *bool {
	value := v
	return &value
}

func singularName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, "_id")
	name = strings.TrimSuffix(name, "s")
	return name
}

func pluralName(name string) string {
	name = strings.TrimSpace(name)
	if strings.HasSuffix(name, "s") {
		return name
	}
	return name + "s"
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Definition{}
)

func Register(version string, name string, up func(*Migrator), down func(*Migrator)) {
	def := Definition{
		Version: strings.TrimSpace(version),
		Name:    strings.TrimSpace(name),
		UpOps:   buildOps(up),
		DownOps: buildOps(down),
	}

	register(def)
}

func RegisterChange(version string, name string, change func(*Migrator)) {
	upOps := buildOps(change)
	downOps := reverseOps(upOps)

	register(Definition{
		Version: strings.TrimSpace(version),
		Name:    strings.TrimSpace(name),
		UpOps:   upOps,
		DownOps: downOps,
	})
}

func Definitions() []Definition {
	registryMu.RLock()
	defer registryMu.RUnlock()

	defs := make([]Definition, 0, len(registry))
	for _, def := range registry {
		defs = append(defs, def)
	}

	return defs
}

func ResetRegistryForTest() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]Definition{}
}

func register(def Definition) {
	if def.Version == "" {
		panic("migration version must not be empty")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[def.Version]; exists {
		panic(fmt.Sprintf("migration version %s already registered", def.Version))
	}

	registry[def.Version] = def
}

func buildOps(fn func(*Migrator)) []Operation {
	if fn == nil {
		return nil
	}

	m := &Migrator{}
	fn(m)
	return m.Ops()
}

func reverseOps(ops []Operation) []Operation {
	reversed := make([]Operation, 0, len(ops))
	for i := len(ops) - 1; i >= 0; i-- {
		reverse, ok := ops[i].Reverse()
		if !ok {
			panic(fmt.Sprintf("migration operation %T is not automatically reversible", ops[i]))
		}

		reversed = append(reversed, reverse)
	}

	return reversed
}
