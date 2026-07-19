package rbac

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/daqing/airway/app/modules/identity"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound       = errors.New("record not found")
	ErrConflict       = errors.New("record already exists")
	ErrAdminConflict  = errors.New("login or email already exists")
	ErrLastSuperAdmin = errors.New("the last active super administrator cannot be disabled")
	ErrValidation     = errors.New("invalid input")
)

type Role struct {
	ID      int64  `db:"id" json:"id"`
	Code    string `db:"code" json:"code"`
	Name    string `db:"name" json:"name"`
	System  bool   `db:"system" json:"system"`
	Version int64  `db:"version" json:"version"`
}

type Permission struct {
	ID     int64  `db:"id" json:"id"`
	Code   string `db:"code" json:"code"`
	Name   string `db:"name" json:"name"`
	Source string `db:"source" json:"source"`
}

type AdminDetails struct {
	identity.Admin
	Roles []Role `json:"roles"`
}

type Service struct{ db *repo.DB }

func NewService(db *repo.DB) *Service { return &Service{db: db} }

func (s *Service) Allowed(ctx context.Context, admin identity.Admin, code string) (bool, error) {
	if admin.SuperAdmin {
		return true, nil
	}
	var count int
	q := s.db.Conn().Rebind(`SELECT COUNT(*) FROM admin_roles ar JOIN role_permissions rp ON rp.role_id=ar.role_id JOIN permissions p ON p.id=rp.permission_id WHERE ar.admin_id=? AND p.code=?`)
	err := s.db.Conn().GetContext(ctx, &count, q, admin.ID, code)
	return count > 0, err
}

func (s *Service) ListAdmins(ctx context.Context) ([]AdminDetails, error) {
	admins := make([]AdminDetails, 0)
	if err := s.db.Conn().SelectContext(ctx, &admins, `SELECT id,login,email,status,super_admin,auth_version FROM admins ORDER BY id`); err != nil {
		return nil, err
	}
	for i := range admins {
		roles, err := s.AdminRoles(ctx, admins[i].ID)
		if err != nil {
			return nil, err
		}
		admins[i].Roles = roles
	}
	return admins, nil
}

func (s *Service) CreateAdmin(ctx context.Context, login, email, password string) (AdminDetails, error) {
	login, email = strings.ToLower(strings.TrimSpace(login)), strings.ToLower(strings.TrimSpace(email))
	if login == "" || email == "" || len(password) < 12 {
		return AdminDetails{}, ErrValidation
	}
	digest, err := utils.EncryptPassword(password)
	if err != nil {
		return AdminDetails{}, err
	}
	now := time.Now().UTC()
	res, err := s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind(`INSERT INTO admins (login,email,password_digest,status,super_admin,auth_version,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`), login, email, digest, "active", false, 1, now, now)
	if err != nil {
		return AdminDetails{}, ErrAdminConflict
	}
	id, err := res.LastInsertId()
	if err != nil && s.db.Driver() == repo.DriverPostgres {
		err = s.db.Conn().GetContext(ctx, &id, `SELECT id FROM admins WHERE login=$1`, login)
	}
	if err != nil {
		return AdminDetails{}, err
	}
	return AdminDetails{Admin: identity.Admin{ID: id, Login: login, Email: email, Status: "active", AuthVersion: 1}, Roles: []Role{}}, nil
}

func (s *Service) UpdateAdmin(ctx context.Context, id int64, email, status string) (AdminDetails, error) {
	email, status = strings.ToLower(strings.TrimSpace(email)), strings.TrimSpace(status)
	if email == "" || (status != "active" && status != "disabled") {
		return AdminDetails{}, ErrValidation
	}
	err := repo.Tx(s.db, func(tx *sqlx.Tx) error {
		var target struct {
			Status     string `db:"status"`
			SuperAdmin bool   `db:"super_admin"`
		}
		if err := tx.GetContext(ctx, &target, tx.Rebind(`SELECT status,super_admin FROM admins WHERE id=?`), id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if target.SuperAdmin && target.Status == "active" && status == "disabled" {
			var activeSuperAdminIDs []int64
			query := `SELECT id FROM admins WHERE super_admin=? AND status=? ORDER BY id`
			if s.db.Driver() != repo.DriverSQLite {
				query += ` FOR UPDATE`
			}
			if err := tx.SelectContext(ctx, &activeSuperAdminIDs, tx.Rebind(query), true, "active"); err != nil {
				return err
			}
			if len(activeSuperAdminIDs) <= 1 {
				return ErrLastSuperAdmin
			}
		}
		res, err := tx.ExecContext(ctx, tx.Rebind(`UPDATE admins SET email=?,status=?,auth_version=auth_version+1,updated_at=? WHERE id=?`), email, status, time.Now().UTC(), id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		return nil
	})
	if err != nil {
		return AdminDetails{}, err
	}
	return s.Admin(ctx, id)
}

func (s *Service) Admin(ctx context.Context, id int64) (AdminDetails, error) {
	var a AdminDetails
	err := s.db.Conn().GetContext(ctx, &a, s.db.Conn().Rebind(`SELECT id,login,email,status,super_admin,auth_version FROM admins WHERE id=?`), id)
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrNotFound
	}
	if err != nil {
		return a, err
	}
	a.Roles, err = s.AdminRoles(ctx, id)
	return a, err
}

func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	v := make([]Role, 0)
	err := s.db.Conn().SelectContext(ctx, &v, `SELECT id,code,name,system,version FROM roles ORDER BY id`)
	return v, err
}
func (s *Service) CreateRole(ctx context.Context, code, name string) (Role, error) {
	code, name = strings.TrimSpace(code), strings.TrimSpace(name)
	if code == "" || name == "" {
		return Role{}, ErrValidation
	}
	now := time.Now().UTC()
	res, err := s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind(`INSERT INTO roles (code,name,system,version,created_at,updated_at) VALUES (?,?,?,?,?,?)`), code, name, false, 1, now, now)
	if err != nil {
		return Role{}, ErrConflict
	}
	id, err := res.LastInsertId()
	if err != nil && s.db.Driver() == repo.DriverPostgres {
		err = s.db.Conn().GetContext(ctx, &id, `SELECT id FROM roles WHERE code=$1`, code)
	}
	return Role{ID: id, Code: code, Name: name, Version: 1}, err
}
func (s *Service) UpdateRole(ctx context.Context, id int64, name string) (Role, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Role{}, ErrValidation
	}
	res, err := s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind(`UPDATE roles SET name=?,version=version+1,updated_at=? WHERE id=?`), name, time.Now().UTC(), id)
	if err != nil {
		return Role{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Role{}, ErrNotFound
	}
	var r Role
	err = s.db.Conn().GetContext(ctx, &r, s.db.Conn().Rebind(`SELECT id,code,name,system,version FROM roles WHERE id=?`), id)
	return r, err
}

func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	v := make([]Permission, 0)
	err := s.db.Conn().SelectContext(ctx, &v, `SELECT id,code,name,source FROM permissions ORDER BY id`)
	return v, err
}
func (s *Service) CreatePermission(ctx context.Context, code, name string) (Permission, error) {
	code, name = strings.TrimSpace(code), strings.TrimSpace(name)
	if code == "" || name == "" || !strings.Contains(code, ":") {
		return Permission{}, ErrValidation
	}
	now := time.Now().UTC()
	res, err := s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind(`INSERT INTO permissions (code,name,source,created_at,updated_at) VALUES (?,?,?,?,?)`), code, name, "custom", now, now)
	if err != nil {
		return Permission{}, ErrConflict
	}
	id, err := res.LastInsertId()
	if err != nil && s.db.Driver() == repo.DriverPostgres {
		err = s.db.Conn().GetContext(ctx, &id, `SELECT id FROM permissions WHERE code=$1`, code)
	}
	return Permission{ID: id, Code: code, Name: name, Source: "custom"}, err
}
func (s *Service) UpdatePermission(ctx context.Context, id int64, name string) (Permission, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Permission{}, ErrValidation
	}
	res, err := s.db.Conn().ExecContext(ctx, s.db.Conn().Rebind(`UPDATE permissions SET name=?,updated_at=? WHERE id=?`), name, time.Now().UTC(), id)
	if err != nil {
		return Permission{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Permission{}, ErrNotFound
	}
	var p Permission
	err = s.db.Conn().GetContext(ctx, &p, s.db.Conn().Rebind(`SELECT id,code,name,source FROM permissions WHERE id=?`), id)
	return p, err
}

func (s *Service) AdminRoles(ctx context.Context, id int64) ([]Role, error) {
	v := make([]Role, 0)
	err := s.db.Conn().SelectContext(ctx, &v, s.db.Conn().Rebind(`SELECT r.id,r.code,r.name,r.system,r.version FROM roles r JOIN admin_roles ar ON ar.role_id=r.id WHERE ar.admin_id=? ORDER BY r.id`), id)
	return v, err
}
func (s *Service) RolePermissions(ctx context.Context, id int64) ([]Permission, error) {
	v := make([]Permission, 0)
	err := s.db.Conn().SelectContext(ctx, &v, s.db.Conn().Rebind(`SELECT p.id,p.code,p.name,p.source FROM permissions p JOIN role_permissions rp ON rp.permission_id=p.id WHERE rp.role_id=? ORDER BY p.id`), id)
	return v, err
}

func (s *Service) AssignRoles(ctx context.Context, adminID int64, roleIDs []int64) error {
	return repo.Tx(s.db, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, tx.Rebind(`DELETE FROM admin_roles WHERE admin_id=?`), adminID); err != nil {
			return err
		}
		for _, id := range unique(roleIDs) {
			if _, err := tx.ExecContext(ctx, tx.Rebind(`INSERT INTO admin_roles (admin_id,role_id) VALUES (?,?)`), adminID, id); err != nil {
				return ErrValidation
			}
		}
		return nil
	})
}
func (s *Service) AssignPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	return repo.Tx(s.db, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, tx.Rebind(`DELETE FROM role_permissions WHERE role_id=?`), roleID); err != nil {
			return err
		}
		for _, id := range unique(permissionIDs) {
			if _, err := tx.ExecContext(ctx, tx.Rebind(`INSERT INTO role_permissions (role_id,permission_id) VALUES (?,?)`), roleID, id); err != nil {
				return ErrValidation
			}
		}
		if _, err := tx.ExecContext(ctx, tx.Rebind(`UPDATE roles SET version=version+1,updated_at=? WHERE id=?`), time.Now().UTC(), roleID); err != nil {
			return err
		}
		return nil
	})
}
func unique(ids []int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
