package dialect

import (
	"fmt"
	"strings"

	"github.com/daqing/airway/lib/migrate/schema"
	"github.com/daqing/airway/lib/repo"
)

type Compiler struct {
	driver repo.Driver
}

func NewCompiler(driver repo.Driver) *Compiler {
	return &Compiler{driver: driver}
}

func (c *Compiler) Compile(op schema.Operation) ([]string, error) {
	switch actual := op.(type) {
	case schema.CreateTableOp:
		return c.compileCreateTable(actual)
	case schema.DropTableOp:
		return []string{fmt.Sprintf("DROP TABLE %s", c.quoteIdent(actual.Table))}, nil
	case schema.RenameTableOp:
		return []string{
			fmt.Sprintf("ALTER TABLE %s RENAME TO %s", c.quoteIdent(actual.From), c.quoteIdent(actual.To)),
		}, nil
	case schema.AddColumnOp:
		stmt, err := c.columnSQL(actual.Column)
		if err != nil {
			return nil, err
		}
		return []string{
			fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", c.quoteIdent(actual.Table), stmt),
		}, nil
	case schema.RemoveColumnOp:
		return []string{
			fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", c.quoteIdent(actual.Table), c.quoteIdent(actual.ColumnName)),
		}, nil
	case schema.ChangeColumnOp:
		return c.changeColumnSQL(actual)
	case schema.SetNullOp:
		return c.setNullSQL(actual)
	case schema.SetDefaultOp:
		return c.setDefaultSQL(actual)
	case schema.RenameColumnOp:
		return []string{
			fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", c.quoteIdent(actual.Table), c.quoteIdent(actual.From), c.quoteIdent(actual.To)),
		}, nil
	case schema.AddIndexOp:
		return []string{c.indexSQL(actual.Table, actual.Index)}, nil
	case schema.RemoveIndexOp:
		return []string{c.removeIndexSQL(actual.Table, actual)}, nil
	case schema.AddForeignKeyOp:
		return c.addForeignKeySQL(actual)
	case schema.RemoveForeignKeyOp:
		return c.removeForeignKeySQL(actual)
	case schema.RawSQLOp:
		return splitSQLStatements(actual.UpSQL), nil
	default:
		return nil, fmt.Errorf("unsupported migration operation %T", op)
	}
}

func (c *Compiler) changeColumnSQL(op schema.ChangeColumnOp) ([]string, error) {
	switch c.driver {
	case repo.DriverSQLite:
		return nil, fmt.Errorf("sqlite does not support changing columns without rebuilding the table")
	case repo.DriverMySQL:
		stmt, err := c.columnSQL(op.Column)
		if err != nil {
			return nil, err
		}
		return []string{
			fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s", c.quoteIdent(op.Table), stmt),
		}, nil
	default:
		statements := []string{}
		typeSQL, err := c.renderType(op.Column.Type)
		if err != nil {
			return nil, err
		}
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", c.quoteIdent(op.Table), c.quoteIdent(op.Column.Name), typeSQL))
		if op.Column.Null != nil && !*op.Column.Null {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", c.quoteIdent(op.Table), c.quoteIdent(op.Column.Name)))
		} else {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", c.quoteIdent(op.Table), c.quoteIdent(op.Column.Name)))
		}
		if op.Column.Default != nil {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s", c.quoteIdent(op.Table), c.quoteIdent(op.Column.Name), c.renderDefault(op.Column.Default)))
		} else {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT", c.quoteIdent(op.Table), c.quoteIdent(op.Column.Name)))
		}
		return statements, nil
	}
}

func (c *Compiler) setNullSQL(op schema.SetNullOp) ([]string, error) {
	switch c.driver {
	case repo.DriverSQLite:
		return nil, fmt.Errorf("sqlite does not support changing nullability without rebuilding the table")
	default:
		action := "DROP NOT NULL"
		if !op.Nullable {
			action = "SET NOT NULL"
		}
		return []string{
			fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s %s", c.quoteIdent(op.Table), c.quoteIdent(op.ColumnName), action),
		}, nil
	}
}

func (c *Compiler) setDefaultSQL(op schema.SetDefaultOp) ([]string, error) {
	switch c.driver {
	case repo.DriverSQLite:
		return nil, fmt.Errorf("sqlite does not support changing defaults without rebuilding the table")
	default:
		action := "DROP DEFAULT"
		if !op.Remove {
			action = "SET DEFAULT " + c.renderDefault(op.Default)
		}
		return []string{
			fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s %s", c.quoteIdent(op.Table), c.quoteIdent(op.ColumnName), action),
		}, nil
	}
}

func (c *Compiler) compileCreateTable(op schema.CreateTableOp) ([]string, error) {
	columnDefs := make([]string, 0, len(op.Columns)+len(op.ForeignKeys))
	indexDefs := make([]string, 0, len(op.Indexes))

	for _, column := range op.Columns {
		stmt, err := c.columnSQL(column)
		if err != nil {
			return nil, err
		}
		columnDefs = append(columnDefs, stmt)
	}

	for _, fk := range op.ForeignKeys {
		columnDefs = append(columnDefs, c.foreignKeySQL(fk))
	}

	for _, idx := range op.Indexes {
		indexDefs = append(indexDefs, c.indexSQL(op.Table, idx))
	}

	statements := []string{
		fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", c.quoteIdent(op.Table), strings.Join(columnDefs, ",\n  ")),
	}
	statements = append(statements, indexDefs...)
	return statements, nil
}

func (c *Compiler) columnSQL(column schema.Column) (string, error) {
	if column.Type.Kind == schema.TypeID {
		return c.idColumnSQL(column), nil
	}

	typeSQL, err := c.renderType(column.Type)
	if err != nil {
		return "", err
	}

	parts := []string{c.quoteIdent(column.Name), typeSQL}
	if column.PrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if column.Null != nil && !*column.Null {
		parts = append(parts, "NOT NULL")
	}
	if column.Unique {
		parts = append(parts, "UNIQUE")
	}
	if column.Default != nil {
		parts = append(parts, "DEFAULT "+c.renderDefault(column.Default))
	}

	return strings.Join(parts, " "), nil
}

func (c *Compiler) foreignKeySQL(fk schema.ForeignKey) string {
	if strings.TrimSpace(fk.Name) != "" && c.driver != repo.DriverSQLite {
		return strings.Join([]string{
			"CONSTRAINT",
			c.quoteIdent(fk.Name),
			c.foreignKeyBodySQL(fk),
		}, " ")
	}

	return c.foreignKeyBodySQL(fk)
}

func (c *Compiler) foreignKeyBodySQL(fk schema.ForeignKey) string {
	parts := []string{
		"FOREIGN KEY",
		fmt.Sprintf("(%s)", c.quoteIdent(fk.Column)),
		"REFERENCES",
		fmt.Sprintf("%s (%s)", c.quoteIdent(fk.RefTable), c.quoteIdent(fk.RefColumn)),
	}
	if fk.OnDelete != "" {
		parts = append(parts, "ON DELETE "+fk.OnDelete)
	}
	if fk.OnUpdate != "" {
		parts = append(parts, "ON UPDATE "+fk.OnUpdate)
	}
	return strings.Join(parts, " ")
}

func (c *Compiler) addForeignKeySQL(op schema.AddForeignKeyOp) ([]string, error) {
	if c.driver == repo.DriverSQLite {
		return nil, fmt.Errorf("sqlite does not support adding foreign keys without rebuilding the table")
	}

	prefix := "ALTER TABLE " + c.quoteIdent(op.Table) + " ADD "
	if strings.TrimSpace(op.ForeignKey.Name) != "" {
		return []string{prefix + "CONSTRAINT " + c.quoteIdent(op.ForeignKey.Name) + " " + c.foreignKeyBodySQL(op.ForeignKey)}, nil
	}

	return []string{prefix + c.foreignKeyBodySQL(op.ForeignKey)}, nil
}

func (c *Compiler) removeForeignKeySQL(op schema.RemoveForeignKeyOp) ([]string, error) {
	switch c.driver {
	case repo.DriverSQLite:
		return nil, fmt.Errorf("sqlite does not support dropping foreign keys without rebuilding the table")
	case repo.DriverMySQL:
		name := strings.TrimSpace(op.Name)
		if name == "" {
			name = defaultForeignKeyName(op.Table, op.Column)
		}
		return []string{
			fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", c.quoteIdent(op.Table), c.quoteIdent(name)),
		}, nil
	default:
		name := strings.TrimSpace(op.Name)
		if name == "" {
			name = defaultForeignKeyName(op.Table, op.Column)
		}
		return []string{
			fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", c.quoteIdent(op.Table), c.quoteIdent(name)),
		}, nil
	}
}

func (c *Compiler) idColumnSQL(column schema.Column) string {
	name := c.quoteIdent(column.Name)
	switch c.driver {
	case repo.DriverPostgres:
		return fmt.Sprintf("%s BIGSERIAL PRIMARY KEY", name)
	case repo.DriverMySQL:
		return fmt.Sprintf("%s BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY", name)
	case repo.DriverSQLite:
		return fmt.Sprintf("%s INTEGER PRIMARY KEY AUTOINCREMENT", name)
	default:
		return fmt.Sprintf("%s BIGINT PRIMARY KEY", name)
	}
}

func (c *Compiler) renderType(t schema.Type) (string, error) {
	switch t.Kind {
	case schema.TypeString:
		if c.driver == repo.DriverSQLite {
			return "TEXT", nil
		}
		if t.Length <= 0 {
			t.Length = 255
		}
		return fmt.Sprintf("VARCHAR(%d)", t.Length), nil
	case schema.TypeText:
		return "TEXT", nil
	case schema.TypeBoolean:
		if c.driver == repo.DriverSQLite {
			return "INTEGER", nil
		}
		return "BOOLEAN", nil
	case schema.TypeInteger:
		return "INTEGER", nil
	case schema.TypeBigInt:
		return "BIGINT", nil
	case schema.TypeDateTime:
		switch c.driver {
		case repo.DriverPostgres:
			return "TIMESTAMP", nil
		case repo.DriverMySQL:
			return "DATETIME", nil
		case repo.DriverSQLite:
			return "DATETIME", nil
		}
	case schema.TypeJSON:
		switch c.driver {
		case repo.DriverPostgres:
			return "JSONB", nil
		case repo.DriverMySQL:
			return "JSON", nil
		case repo.DriverSQLite:
			return "TEXT", nil
		}
	}

	return "", fmt.Errorf("unsupported column type: %s", t.Kind)
}

func (c *Compiler) renderDefault(value any) string {
	switch actual := value.(type) {
	case schema.DefaultExpr:
		return string(actual)
	case string:
		return "'" + strings.ReplaceAll(actual, "'", "''") + "'"
	case bool:
		if actual {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("%v", actual)
	}
}

func (c *Compiler) indexSQL(table string, idx schema.Index) string {
	name := idx.Name
	if strings.TrimSpace(name) == "" {
		name = defaultIndexName(table, idx.Columns)
	}

	unique := ""
	if idx.Unique {
		unique = "UNIQUE "
	}

	cols := make([]string, 0, len(idx.Columns))
	for _, column := range idx.Columns {
		cols = append(cols, c.quoteIdent(column))
	}

	return fmt.Sprintf(
		"CREATE %sINDEX %s ON %s (%s)",
		unique,
		c.quoteIdent(name),
		c.quoteIdent(table),
		strings.Join(cols, ", "),
	)
}

func (c *Compiler) removeIndexSQL(table string, idx schema.RemoveIndexOp) string {
	name := strings.TrimSpace(idx.Name)
	if name == "" {
		name = defaultIndexName(table, idx.Columns)
	}

	switch c.driver {
	case repo.DriverMySQL:
		return fmt.Sprintf("DROP INDEX %s ON %s", c.quoteIdent(name), c.quoteIdent(table))
	default:
		return fmt.Sprintf("DROP INDEX %s", c.quoteIdent(name))
	}
}

func (c *Compiler) quoteIdent(name string) string {
	switch c.driver {
	case repo.DriverMySQL:
		return "`" + strings.TrimSpace(name) + "`"
	default:
		return `"` + strings.TrimSpace(name) + `"`
	}
}

func defaultIndexName(table string, columns []string) string {
	return fmt.Sprintf("idx_%s_%s", table, strings.Join(columns, "_"))
}

func defaultForeignKeyName(table string, column string) string {
	return fmt.Sprintf("fk_%s_%s", table, column)
}

func splitSQLStatements(script string) []string {
	var statements []string
	var current strings.Builder

	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(script); i++ {
		ch := script[i]
		next := byte(0)
		if i+1 < len(script) {
			next = script[i+1]
		}

		if inLineComment {
			current.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}

		if inBlockComment {
			current.WriteByte(ch)
			if ch == '*' && next == '/' {
				current.WriteByte(next)
				i++
				inBlockComment = false
			}
			continue
		}

		if !inSingleQuote && !inDoubleQuote && !inBacktick {
			if ch == '-' && next == '-' {
				current.WriteByte(ch)
				current.WriteByte(next)
				i++
				inLineComment = true
				continue
			}

			if ch == '/' && next == '*' {
				current.WriteByte(ch)
				current.WriteByte(next)
				i++
				inBlockComment = true
				continue
			}
		}

		switch ch {
		case '\'':
			if !inDoubleQuote && !inBacktick && !isEscaped(script, i) {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote && !inBacktick && !isEscaped(script, i) {
				inDoubleQuote = !inDoubleQuote
			}
		case '`':
			if !inSingleQuote && !inDoubleQuote {
				inBacktick = !inBacktick
			}
		case ';':
			if !inSingleQuote && !inDoubleQuote && !inBacktick {
				trimmed := strings.TrimSpace(current.String())
				if trimmed != "" {
					statements = append(statements, trimmed)
				}
				current.Reset()
				continue
			}
		}

		current.WriteByte(ch)
	}

	if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
		statements = append(statements, trimmed)
	}

	return statements
}

func isEscaped(value string, index int) bool {
	backslashes := 0
	for i := index - 1; i >= 0 && value[i] == '\\'; i-- {
		backslashes++
	}

	return backslashes%2 == 1
}
