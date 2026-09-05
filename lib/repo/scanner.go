package repo

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// structScanner scans rows from the wrapped *sql.Rows into a struct.
// Each result column is matched to a struct field by its `db` tag (falling
// back to the snake_case form of the Go field name) and scanned via
// rows.Scan. Columns that have no matching field are treated as an error.
type structScanner struct {
	rows    *sql.Rows
	columns []string
	fields  []int // struct field index per column
	dstType reflect.Type
	ready   bool
}

func newStructScanner(rows *sql.Rows) *structScanner {
	return &structScanner{rows: rows}
}

// Scan maps the current row into dst, which must be a non-nil pointer to a
// struct. The column-to-field mapping is computed once and cached for the
// lifetime of the scanner, so every row must be scanned into the same type.
func (s *structScanner) Scan(dst any) error {
	dstValue := reflect.ValueOf(dst)
	if !dstValue.IsValid() || dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return fmt.Errorf("scan destination must be a non-nil pointer, got %T", dst)
	}

	structValue := dstValue.Elem()
	if structValue.Kind() != reflect.Struct {
		return fmt.Errorf("scan destination must be a pointer to struct, got %T", dst)
	}

	if !s.ready {
		if err := s.init(structValue.Type()); err != nil {
			return err
		}
	}

	if structValue.Type() != s.dstType {
		return fmt.Errorf("struct scanner destination type changed from %s to %s", s.dstType, structValue.Type())
	}

	dest := make([]any, len(s.fields))
	for i, fieldIndex := range s.fields {
		dest[i] = structValue.Field(fieldIndex).Addr().Interface()
	}

	return s.rows.Scan(dest...)
}

func (s *structScanner) init(structType reflect.Type) error {
	columns, err := s.rows.Columns()
	if err != nil {
		return err
	}

	fields := make([]int, len(columns))
	for i, column := range columns {
		fieldIndex := structFieldIndexForColumn(structType, column)
		if fieldIndex < 0 {
			return fmt.Errorf("missing destination name %s in %s", column, structType)
		}
		fields[i] = fieldIndex
	}

	s.columns = columns
	s.fields = fields
	s.dstType = structType
	s.ready = true
	return nil
}

// selectStructs queries multiple rows and scans them into a slice of structs
// (or a slice of pointers to structs). dst must be a pointer to such a slice.
func selectStructs(ctx context.Context, conn *sql.DB, dst any, query string, args ...any) error {
	sliceValue := reflect.ValueOf(dst)
	if !sliceValue.IsValid() || sliceValue.Kind() != reflect.Ptr || sliceValue.IsNil() {
		return fmt.Errorf("select destination must be a non-nil pointer, got %T", dst)
	}

	slice := sliceValue.Elem()
	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("select destination must be a pointer to a slice, got %T", dst)
	}

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	slice.Set(slice.Slice(0, 0))
	elementType := slice.Type().Elem()
	byPtr := elementType.Kind() == reflect.Ptr
	baseType := elementType
	if byPtr {
		baseType = elementType.Elem()
	}
	if baseType.Kind() != reflect.Struct {
		return fmt.Errorf("select destination element must be a struct, got %s", elementType)
	}

	scanner := newStructScanner(rows)
	for rows.Next() {
		record := reflect.New(baseType)
		if err := scanner.Scan(record.Interface()); err != nil {
			return err
		}

		if byPtr {
			slice.Set(reflect.Append(slice, record))
		} else {
			slice.Set(reflect.Append(slice, record.Elem()))
		}
	}

	return rows.Err()
}

// getStruct queries a single row and scans it into dst, a pointer to a struct.
// It returns sql.ErrNoRows when the query yields no rows. Additional rows, if
// any, are ignored.
func getStruct(ctx context.Context, conn *sql.DB, dst any, query string, args ...any) error {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	return newStructScanner(rows).Scan(dst)
}

// structFieldIndexForColumn returns the index of the struct field that the
// given column should be scanned into, or -1 if no field matches.
func structFieldIndexForColumn(structType reflect.Type, column string) int {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" {
			continue // unexported field
		}

		tag, tagged := field.Tag.Lookup("db")
		if tagged && tag == "-" {
			continue
		}

		if tagged {
			if tag == column {
				return i
			}
			continue
		}

		if snakeCaseFieldName(field.Name) == column || strings.EqualFold(field.Name, column) {
			return i
		}
	}

	return -1
}

// snakeCaseFieldName converts a Go field name to its snake_case form, e.g.
// "CreatedAt" -> "created_at". Acronyms such as "ID" collapse to "id".
func snakeCaseFieldName(name string) string {
	var builder strings.Builder
	runes := []rune(name)
	for i, r := range runes {
		if r >= 'A' && r <= 'Z' {
			nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
			prevLowerOrDigit := i > 0 && ((runes[i-1] >= 'a' && runes[i-1] <= 'z') || (runes[i-1] >= '0' && runes[i-1] <= '9'))
			if i > 0 && (nextLower || prevLowerOrDigit) {
				builder.WriteByte('_')
			}
			builder.WriteRune(r + ('a' - 'A'))
		} else {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
