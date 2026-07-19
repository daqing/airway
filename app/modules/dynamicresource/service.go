package dynamicresource

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/daqing/airway/lib/repo"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound     = errors.New("resource definition not found")
	ErrConflict     = errors.New("resource definition already exists")
	ErrInvalidState = errors.New("invalid resource lifecycle transition")
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string { return "resource definition validation failed" }

type Field struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Required   bool   `json:"required"`
	Default    any    `json:"default,omitempty"`
	List       bool   `json:"list"`
	Searchable bool   `json:"searchable"`
	Filterable bool   `json:"filterable"`
	Sortable   bool   `json:"sortable"`
	Input      string `json:"input,omitempty"`
}
type ActionPermissions struct {
	List   string `json:"list"`
	Read   string `json:"read"`
	Create string `json:"create"`
	Update string `json:"update"`
	Delete string `json:"delete"`
}
type Schema struct {
	Fields      []Field           `json:"fields"`
	Permissions ActionPermissions `json:"permissions"`
}
type Definition struct {
	ID              int64  `db:"id" json:"id"`
	Code            string `db:"code" json:"code"`
	Name            string `db:"name" json:"name"`
	TableName       string `db:"table_name" json:"table_name"`
	Status          string `db:"status" json:"status"`
	ActiveVersion   *int64 `db:"active_version" json:"active_version"`
	DraftSchemaJSON string `db:"draft_schema_json" json:"-"`
	Schema          Schema `json:"schema"`
}
type Version struct {
	ID                   int64     `db:"id" json:"id"`
	ResourceDefinitionID int64     `db:"resource_definition_id" json:"resource_definition_id"`
	Version              int64     `db:"version" json:"version"`
	Checksum             string    `db:"checksum" json:"checksum"`
	PublishedAt          time.Time `db:"published_at" json:"published_at"`
	PublishedBy          int64     `db:"published_by" json:"published_by"`
}

type Service struct {
	db          *repo.DB
	provisioned sync.Map
}

func NewService(db *repo.DB) *Service { return &Service{db: db} }

func (s *Service) List(ctx context.Context) ([]Definition, error) {
	items := make([]Definition, 0)
	if err := s.db.Conn().SelectContext(ctx, &items, `SELECT id,code,name,table_name,status,active_version,draft_schema_json FROM resource_definitions ORDER BY id`); err != nil {
		return nil, err
	}
	for i := range items {
		if err := decodeSchema(&items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}
func (s *Service) Get(ctx context.Context, id int64) (Definition, error) {
	var item Definition
	err := s.db.Conn().GetContext(ctx, &item, s.db.Conn().Rebind(`SELECT id,code,name,table_name,status,active_version,draft_schema_json FROM resource_definitions WHERE id=?`), id)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	err = decodeSchema(&item)
	return item, err
}
func (s *Service) Create(ctx context.Context, code, name, tableName string, schema Schema) (Definition, error) {
	code, name, tableName = strings.TrimSpace(code), strings.TrimSpace(name), strings.TrimSpace(tableName)
	data, _ := json.Marshal(schema)
	now := time.Now().UTC()
	res, err := s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind(`INSERT INTO resource_definitions (code,name,table_name,status,active_version,draft_schema_json,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`), code, name, tableName, "draft", nil, string(data), now, now)
	if err != nil {
		return Definition{}, ErrConflict
	}
	id, err := res.LastInsertId()
	if err != nil && s.db.Driver() == repo.DriverPostgres {
		err = s.db.Conn().GetContext(ctx, &id, `SELECT id FROM resource_definitions WHERE code=$1`, code)
	}
	if err != nil {
		return Definition{}, err
	}
	return s.Get(ctx, id)
}
func (s *Service) Update(ctx context.Context, id int64, code, name, tableName string, schema Schema) (Definition, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return Definition{}, err
	}
	if current.Status == "published" {
		return Definition{}, ErrInvalidState
	}
	data, _ := json.Marshal(schema)
	res, err := s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind(`UPDATE resource_definitions SET code=?,name=?,table_name=?,status='draft',draft_schema_json=?,updated_at=? WHERE id=?`), strings.TrimSpace(code), strings.TrimSpace(name), strings.TrimSpace(tableName), string(data), time.Now().UTC(), id)
	if err != nil {
		return Definition{}, ErrConflict
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Definition{}, ErrNotFound
	}
	return s.Get(ctx, id)
}
func (s *Service) Validate(ctx context.Context, id int64) (Definition, error) {
	item, err := s.Get(ctx, id)
	if err != nil {
		return item, err
	}
	if item.Status == "published" {
		return item, ErrInvalidState
	}
	if validationErrors := validate(item); len(validationErrors) > 0 {
		return item, validationErrors
	}
	_, err = s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind(`UPDATE resource_definitions SET status='validated',updated_at=? WHERE id=?`), time.Now().UTC(), id)
	if err != nil {
		return item, err
	}
	return s.Get(ctx, id)
}
func (s *Service) Publish(ctx context.Context, id, actorID int64) (Definition, error) {
	item, err := s.Get(ctx, id)
	if err != nil {
		return item, err
	}
	if item.Status != "validated" {
		return item, ErrInvalidState
	}
	if validationErrors := validate(item); len(validationErrors) > 0 {
		return item, validationErrors
	}
	if err := s.Provision(ctx, item); err != nil {
		return item, err
	}
	s.provisioned.Store(item.Code, true)
	schemaData, _ := json.Marshal(item.Schema)
	sum := sha256.Sum256(schemaData)
	checksum := hex.EncodeToString(sum[:])
	err = repo.Tx(s.db, func(tx *sqlx.Tx) error {
		var version int64
		if err := tx.GetContext(ctx, &version, tx.Rebind(`SELECT COALESCE(MAX(version),0)+1 FROM resource_versions WHERE resource_definition_id=?`), id); err != nil {
			return err
		}
		now := time.Now().UTC()
		_, err := tx.ExecContext(ctx, tx.Rebind(`INSERT INTO resource_versions (resource_definition_id,version,schema_json,checksum,published_at,published_by) VALUES (?,?,?,?,?,?)`), id, version, string(schemaData), checksum, now, actorID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, tx.Rebind(`UPDATE resource_definitions SET status='published',active_version=?,updated_at=? WHERE id=? AND status='validated'`), version, now, id)
		return err
	})
	if err != nil {
		return item, err
	}
	return s.Get(ctx, id)
}
func (s *Service) Deactivate(ctx context.Context, id int64) (Definition, error) {
	item, err := s.Get(ctx, id)
	if err != nil {
		return item, err
	}
	if item.Status != "published" {
		return item, ErrInvalidState
	}
	_, err = s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind(`UPDATE resource_definitions SET status='inactive',updated_at=? WHERE id=?`), time.Now().UTC(), id)
	if err != nil {
		return item, err
	}
	return s.Get(ctx, id)
}
func (s *Service) Versions(ctx context.Context, id int64) ([]Version, error) {
	items := make([]Version, 0)
	err := s.db.Conn().SelectContext(ctx, &items, s.db.Conn().Rebind(`SELECT id,resource_definition_id,version,checksum,published_at,published_by FROM resource_versions WHERE resource_definition_id=? ORDER BY version DESC`), id)
	return items, err
}

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
var permissionPattern = regexp.MustCompile(`^[a-z][a-z0-9_.]*:[a-z][a-z0-9_]*$`)
var fieldTypes = map[string]bool{"string": true, "text": true, "integer": true, "bigint": true, "boolean": true, "datetime": true, "json": true}
var reservedFields = map[string]bool{"id": true, "created_at": true, "updated_at": true, "lock_version": true}

func validate(item Definition) ValidationErrors {
	var result ValidationErrors
	if !identifierPattern.MatchString(item.Code) {
		result = append(result, ValidationError{"code", "must start with a letter and contain only lowercase letters, digits, and underscores"})
	}
	if strings.TrimSpace(item.Name) == "" {
		result = append(result, ValidationError{"name", "is required"})
	}
	if !identifierPattern.MatchString(item.TableName) {
		result = append(result, ValidationError{"table_name", "must be a safe lowercase database identifier"})
	}
	if len(item.Schema.Fields) == 0 {
		result = append(result, ValidationError{"schema.fields", "must contain at least one field"})
	}
	seen := map[string]bool{}
	for i, field := range item.Schema.Fields {
		path := "schema.fields[" + itoa(i) + "]"
		if !identifierPattern.MatchString(field.Code) || reservedFields[field.Code] {
			result = append(result, ValidationError{path + ".code", "is not a valid field identifier"})
		}
		if seen[field.Code] {
			result = append(result, ValidationError{path + ".code", "is duplicated"})
		}
		seen[field.Code] = true
		if strings.TrimSpace(field.Name) == "" {
			result = append(result, ValidationError{path + ".name", "is required"})
		}
		if !fieldTypes[field.Type] {
			result = append(result, ValidationError{path + ".type", "is not supported"})
		}
	}
	permissions := []struct{ name, value string }{{"list", item.Schema.Permissions.List}, {"read", item.Schema.Permissions.Read}, {"create", item.Schema.Permissions.Create}, {"update", item.Schema.Permissions.Update}, {"delete", item.Schema.Permissions.Delete}}
	for _, p := range permissions {
		if !permissionPattern.MatchString(p.value) {
			result = append(result, ValidationError{"schema.permissions." + p.name, "must be a stable permission code such as articles:read"})
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Field < result[j].Field })
	return result
}
func decodeSchema(item *Definition) error {
	return json.Unmarshal([]byte(item.DraftSchemaJSON), &item.Schema)
}
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}
