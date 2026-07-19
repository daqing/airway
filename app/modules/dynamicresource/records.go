package dynamicresource

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
)

var ErrRecordNotFound = errors.New("record not found")

func (s *Service) Published(ctx context.Context, code string) (Definition, error) {
	var item Definition
	err := s.db.Conn().GetContext(ctx, &item, s.db.Conn().Rebind(`SELECT id,code,name,table_name,status,active_version,draft_schema_json FROM resource_definitions WHERE code=? AND status='published'`), code)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	err = decodeSchema(&item)
	if err != nil {
		return item, err
	}
	// Definitions published before physical-table provisioning was introduced
	// are repaired lazily once per process. CREATE TABLE IF NOT EXISTS keeps this
	// safe for already-provisioned resources and concurrent first requests.
	if _, ready := s.provisioned.Load(item.Code); !ready {
		if err := s.Provision(ctx, item); err != nil {
			return item, err
		}
		s.provisioned.Store(item.Code, true)
	}
	return item, nil
}

// Provision creates the physical table before a definition becomes active.
func (s *Service) Provision(ctx context.Context, item Definition) error {
	columns := []string{primaryKeyDDL(s.db.Driver())}
	for _, field := range item.Schema.Fields {
		column := quoteIdentifier(s.db.Driver(), field.Code) + " " + fieldDDL(s.db.Driver(), field.Type)
		if field.Required {
			column += " NOT NULL"
		}
		columns = append(columns, column)
	}
	columns = append(columns,
		quoteIdentifier(s.db.Driver(), "lock_version")+" BIGINT NOT NULL DEFAULT 1",
		quoteIdentifier(s.db.Driver(), "created_at")+" "+timeDDL(s.db.Driver())+" NOT NULL",
		quoteIdentifier(s.db.Driver(), "updated_at")+" "+timeDDL(s.db.Driver())+" NOT NULL",
	)
	query := "CREATE TABLE IF NOT EXISTS " + quoteIdentifier(s.db.Driver(), item.TableName) + " (" + strings.Join(columns, ",") + ")"
	_, err := s.db.Conn().ExecContext(ctx, query)
	return err
}

func (s *Service) ListRecords(ctx context.Context, item Definition) ([]map[string]any, error) {
	rows, err := s.db.Conn().QueryxContext(ctx, "SELECT * FROM "+quoteIdentifier(s.db.Driver(), item.TableName)+" ORDER BY "+quoteIdentifier(s.db.Driver(), "id")+" DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]map[string]any, 0)
	for rows.Next() {
		values := map[string]any{}
		if err := rows.MapScan(values); err != nil {
			return nil, err
		}
		normalizeRecord(values, item.Schema)
		result = append(result, values)
	}
	return result, rows.Err()
}

func (s *Service) GetRecord(ctx context.Context, item Definition, id int64) (map[string]any, error) {
	row := s.db.Conn().QueryRowxContext(ctx, s.db.Conn().Rebind("SELECT * FROM "+quoteIdentifier(s.db.Driver(), item.TableName)+" WHERE "+quoteIdentifier(s.db.Driver(), "id")+"=?"), id)
	values := map[string]any{}
	if err := row.MapScan(values); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	normalizeRecord(values, item.Schema)
	return values, nil
}

func (s *Service) CreateRecord(ctx context.Context, item Definition, input map[string]any) (map[string]any, error) {
	values, validation := recordValues(item.Schema, input, true)
	if len(validation) > 0 {
		return nil, validation
	}
	columns := make([]string, 0, len(values)+2)
	args := make([]any, 0, len(values)+2)
	placeholders := make([]string, 0, len(values)+2)
	for _, field := range item.Schema.Fields {
		value, exists := values[field.Code]
		if !exists {
			continue
		}
		columns = append(columns, quoteIdentifier(s.db.Driver(), field.Code))
		placeholders = append(placeholders, "?")
		args = append(args, encodeValue(field.Type, value))
	}
	now := time.Now().UTC()
	columns = append(columns, quoteIdentifier(s.db.Driver(), "created_at"), quoteIdentifier(s.db.Driver(), "updated_at"))
	placeholders = append(placeholders, "?", "?")
	args = append(args, now, now)
	base := "INSERT INTO " + quoteIdentifier(s.db.Driver(), item.TableName) + " (" + strings.Join(columns, ",") + ") VALUES (" + strings.Join(placeholders, ",") + ")"
	var id int64
	if s.db.Driver() == repo.DriverPostgres {
		err := s.db.Conn().QueryRowxContext(ctx, s.db.Conn().Rebind(base+" RETURNING id"), args...).Scan(&id)
		if err != nil {
			return nil, err
		}
	} else {
		res, err := s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind(base), args...)
		if err != nil {
			return nil, err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return nil, err
		}
	}
	return s.GetRecord(ctx, item, id)
}

func (s *Service) UpdateRecord(ctx context.Context, item Definition, id int64, input map[string]any) (map[string]any, error) {
	values, validation := recordValues(item.Schema, input, false)
	if len(validation) > 0 {
		return nil, validation
	}
	versionValue, ok := input["lock_version"]
	if !ok {
		return nil, ValidationErrors{{Field: "lock_version", Message: "is required"}}
	}
	version, ok := asInt64(versionValue)
	if !ok {
		return nil, ValidationErrors{{Field: "lock_version", Message: "must be an integer"}}
	}
	sets := make([]string, 0, len(values)+2)
	args := make([]any, 0, len(values)+3)
	for _, field := range item.Schema.Fields {
		value, exists := values[field.Code]
		if !exists {
			continue
		}
		sets = append(sets, quoteIdentifier(s.db.Driver(), field.Code)+"=?")
		args = append(args, encodeValue(field.Type, value))
	}
	if len(sets) == 0 {
		return nil, ValidationErrors{{Field: "data", Message: "must contain at least one editable field"}}
	}
	sets = append(sets, quoteIdentifier(s.db.Driver(), "lock_version")+"="+quoteIdentifier(s.db.Driver(), "lock_version")+"+1", quoteIdentifier(s.db.Driver(), "updated_at")+"=?")
	args = append(args, time.Now().UTC(), id, version)
	query := "UPDATE " + quoteIdentifier(s.db.Driver(), item.TableName) + " SET " + strings.Join(sets, ",") + " WHERE " + quoteIdentifier(s.db.Driver(), "id") + "=? AND " + quoteIdentifier(s.db.Driver(), "lock_version") + "=?"
	res, err := s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind(query), args...)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, getErr := s.GetRecord(ctx, item, id); errors.Is(getErr, ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, ErrConflict
	}
	return s.GetRecord(ctx, item, id)
}

func (s *Service) DeleteRecord(ctx context.Context, item Definition, id int64) error {
	res, err := s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind("DELETE FROM "+quoteIdentifier(s.db.Driver(), item.TableName)+" WHERE "+quoteIdentifier(s.db.Driver(), "id")+"=?"), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func recordValues(schema Schema, input map[string]any, creating bool) (map[string]any, ValidationErrors) {
	allowed := map[string]Field{}
	for _, field := range schema.Fields {
		allowed[field.Code] = field
	}
	result := map[string]any{}
	var failures ValidationErrors
	for key, value := range input {
		if key == "lock_version" {
			continue
		}
		field, ok := allowed[key]
		if !ok {
			failures = append(failures, ValidationError{Field: key, Message: "is not defined for this resource"})
			continue
		}
		if value == nil {
			if field.Required {
				failures = append(failures, ValidationError{Field: key, Message: "is required"})
			}
			result[key] = nil
			continue
		}
		if !validValue(field.Type, value) {
			failures = append(failures, ValidationError{Field: key, Message: "has an invalid " + field.Type + " value"})
			continue
		}
		result[key] = value
	}
	if creating {
		for _, field := range schema.Fields {
			_, exists := result[field.Code]
			if field.Required && !exists && field.Default == nil {
				failures = append(failures, ValidationError{Field: field.Code, Message: "is required"})
			}
			if !exists && field.Default != nil {
				result[field.Code] = field.Default
			}
		}
	}
	return result, failures
}
func validValue(kind string, value any) bool {
	switch kind {
	case "string", "text", "datetime":
		_, ok := value.(string)
		return ok
	case "integer", "bigint":
		_, ok := asInt64(value)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "json":
		return true
	}
	return false
}
func encodeValue(kind string, value any) any {
	if kind == "json" {
		data, _ := json.Marshal(value)
		return string(data)
	}
	return value
}
func normalizeRecord(values map[string]any, schema Schema) {
	types := map[string]string{}
	for _, field := range schema.Fields {
		types[field.Code] = field.Type
	}
	for key, value := range values {
		if data, ok := value.([]byte); ok {
			if types[key] == "json" {
				var decoded any
				if json.Unmarshal(data, &decoded) == nil {
					values[key] = decoded
					continue
				}
			}
			values[key] = string(data)
		}
	}
}
func asInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), v == float64(int64(v))
	case int:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		n, e := v.Int64()
		return n, e == nil
	case string:
		n, e := strconv.ParseInt(v, 10, 64)
		return n, e == nil
	}
	return 0, false
}
func quoteIdentifier(driver repo.Driver, value string) string {
	if driver == repo.DriverMySQL {
		return "`" + strings.ReplaceAll(value, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
func primaryKeyDDL(driver repo.Driver) string {
	switch driver {
	case repo.DriverPostgres:
		return `"id" BIGSERIAL PRIMARY KEY`
	case repo.DriverMySQL:
		return "`id` BIGINT AUTO_INCREMENT PRIMARY KEY"
	default:
		return `"id" INTEGER PRIMARY KEY AUTOINCREMENT`
	}
}
func fieldDDL(driver repo.Driver, kind string) string {
	switch kind {
	case "string":
		return "VARCHAR(255)"
	case "text":
		return "TEXT"
	case "integer":
		return "INTEGER"
	case "bigint":
		return "BIGINT"
	case "boolean":
		return "BOOLEAN"
	case "datetime":
		return timeDDL(driver)
	case "json":
		if driver == repo.DriverPostgres {
			return "JSONB"
		}
		if driver == repo.DriverMySQL {
			return "JSON"
		}
		return "TEXT"
	}
	return "TEXT"
}
func timeDDL(driver repo.Driver) string {
	if driver == repo.DriverPostgres {
		return "TIMESTAMPTZ"
	}
	return "DATETIME"
}
