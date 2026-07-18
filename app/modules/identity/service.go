package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/jmoiron/sqlx"
)

var (
	ErrAlreadyInitialized = errors.New("application is already initialized")
	ErrInvalidCredentials = errors.New("invalid login or password")
	ErrUnauthenticated    = errors.New("authentication required")
	ErrInactive           = errors.New("administrator is inactive")
)

// Admin represents the public profile of an administrator who can access the admin panel.
type Admin struct {
	ID          int64  `db:"id" json:"id"`
	Login       string `db:"login" json:"login"`
	Email       string `db:"email" json:"email"`
	Status      string `db:"status" json:"status"`
	SuperAdmin  bool   `db:"super_admin" json:"super_admin"`
	AuthVersion int64  `db:"auth_version" json:"-"`
}

// adminRow represents an administrator database record that includes the password digest.
type adminRow struct {
	Admin
	PasswordDigest string `db:"password_digest"`
}

// Service provides super administrator initialization and session authentication operations.
type Service struct {
	db          *repo.DB
	now         func() time.Time
	sessionTTL  time.Duration
	dummyDigest string
}

// NewService creates an identity service with the specified database and session lifetime.
func NewService(db *repo.DB, sessionTTL time.Duration) *Service {
	if sessionTTL <= 0 {
		sessionTTL = 12 * time.Hour
	}
	dummyDigest, _ := utils.EncryptPassword("not-a-real-administrator-password")
	return &Service{db: db, now: time.Now, sessionTTL: sessionTTL, dummyDigest: dummyDigest}
}

// IsInitialized reports whether the super administrator has already been initialized.
func (s *Service) IsInitialized(ctx context.Context) (bool, error) {
	var count int
	err := s.db.Conn().GetContext(ctx, &count, s.db.Conn().Rebind(`SELECT COUNT(*) FROM setup_state WHERE key=?`), "super_admin_initialized")
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Initialize creates the initial super administrator and audit event in a transaction.
func (s *Service) Initialize(ctx context.Context, login, email, password, ip, requestID string) (Admin, error) {
	login, email = strings.TrimSpace(strings.ToLower(login)), strings.TrimSpace(strings.ToLower(email))
	if login == "" || email == "" || len(password) < 12 {
		return Admin{}, fmt.Errorf("login and email are required; password must contain at least 12 characters")
	}
	digest, err := utils.EncryptPassword(password)
	if err != nil {
		return Admin{}, err
	}

	var admin Admin
	err = repo.Tx(s.db, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, tx.Rebind(`INSERT INTO setup_state (key, created_at) VALUES (?, ?)`), "super_admin_initialized", s.now().UTC())
		if err != nil || result == nil {
			return ErrAlreadyInitialized
		}
		query := tx.Rebind(`INSERT INTO admins (login,email,password_digest,status,super_admin,auth_version,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`)
		now := s.now().UTC()
		res, err := tx.ExecContext(ctx, query, login, email, digest, "active", true, 1, now, now)
		if err != nil {
			return err
		}
		admin.ID, err = res.LastInsertId()
		if err != nil && s.db.Driver() == repo.DriverPostgres {
			err = tx.GetContext(ctx, &admin.ID, `SELECT id FROM admins WHERE login=$1`, login)
		}
		if err != nil {
			return err
		}
		admin.Login, admin.Email, admin.Status, admin.SuperAdmin, admin.AuthVersion = login, email, "active", true, 1
		return s.auditTx(ctx, tx, &admin.ID, "auth.super_admin_initialized", "admin", fmt.Sprint(admin.ID), "success", ip, requestID, nil)
	})
	return admin, err
}

// Login validates administrator credentials and creates a revocable server-side session.
func (s *Service) Login(ctx context.Context, login, password, ip, requestID string) (Admin, string, time.Time, error) {
	login = strings.TrimSpace(strings.ToLower(login))
	var row adminRow
	err := s.db.Conn().GetContext(ctx, &row, s.db.Conn().Rebind(`SELECT id,login,email,password_digest,status,super_admin,auth_version FROM admins WHERE login=? OR email=? LIMIT 1`), login, login)
	if errors.Is(err, sql.ErrNoRows) {
		utils.ComparePassword(utils.PasswordDigest(s.dummyDigest), password)
		_ = s.Audit(ctx, nil, "auth.login", "session", "", "failure", ip, requestID, map[string]any{"login": login})
		return Admin{}, "", time.Time{}, ErrInvalidCredentials
	}
	if err != nil {
		return Admin{}, "", time.Time{}, err
	}
	if !utils.ComparePassword(utils.PasswordDigest(row.PasswordDigest), password) {
		_ = s.Audit(ctx, &row.ID, "auth.login", "session", "", "failure", ip, requestID, map[string]any{"login": login})
		return Admin{}, "", time.Time{}, ErrInvalidCredentials
	}
	if row.Status != "active" {
		_ = s.Audit(ctx, &row.ID, "auth.login", "session", "", "failure", ip, requestID, map[string]any{"reason": "inactive"})
		return Admin{}, "", time.Time{}, ErrInactive
	}
	token, err := randomToken(32)
	if err != nil {
		return Admin{}, "", time.Time{}, err
	}
	expires := s.now().UTC().Add(s.sessionTTL)
	now := s.now().UTC()
	err = repo.Tx(s.db, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, tx.Rebind(`INSERT INTO sessions (admin_id,token_digest,auth_version,expires_at,last_seen_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`), row.ID, tokenDigest(token), row.AuthVersion, expires, now, now, now)
		if err != nil {
			return err
		}
		return s.auditTx(ctx, tx, &row.ID, "auth.login", "session", "", "success", ip, requestID, nil)
	})
	return row.Admin, token, expires, err
}

// Current returns the active administrator associated with a session token.
func (s *Service) Current(ctx context.Context, token string) (Admin, error) {
	if token == "" {
		return Admin{}, ErrUnauthenticated
	}
	var admin Admin
	query := s.db.Conn().Rebind(`SELECT a.id,a.login,a.email,a.status,a.super_admin,a.auth_version FROM sessions s JOIN admins a ON a.id=s.admin_id WHERE s.token_digest=? AND s.revoked_at IS NULL AND s.expires_at>? AND s.auth_version=a.auth_version LIMIT 1`)
	if err := s.db.Conn().GetContext(ctx, &admin, query, tokenDigest(token), s.now().UTC()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Admin{}, ErrUnauthenticated
		}
		return Admin{}, err
	}
	if admin.Status != "active" {
		return Admin{}, ErrUnauthenticated
	}
	return admin, nil
}

// Logout revokes the current session and records a logout audit event.
func (s *Service) Logout(ctx context.Context, token, ip, requestID string) error {
	admin, err := s.Current(ctx, token)
	if err != nil {
		return err
	}
	return repo.Tx(s.db, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, tx.Rebind(`UPDATE sessions SET revoked_at=?,updated_at=? WHERE token_digest=? AND revoked_at IS NULL`), s.now().UTC(), s.now().UTC(), tokenDigest(token))
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n != 1 {
			return ErrUnauthenticated
		}
		return s.auditTx(ctx, tx, &admin.ID, "auth.logout", "session", "", "success", ip, requestID, nil)
	})
}

// Audit persists an audit event in a separate transaction.
func (s *Service) Audit(ctx context.Context, actorID *int64, action, targetType, targetID, result, ip, requestID string, metadata map[string]any) error {
	return repo.Tx(s.db, func(tx *sqlx.Tx) error {
		return s.auditTx(ctx, tx, actorID, action, targetType, targetID, result, ip, requestID, metadata)
	})
}

// auditTx persists an audit event using the transaction supplied by the caller.
func (s *Service) auditTx(ctx context.Context, tx *sqlx.Tx, actorID *int64, action, targetType, targetID, result, ip, requestID string, metadata map[string]any) error {
	data, _ := json.Marshal(metadata)
	_, err := tx.ExecContext(ctx, tx.Rebind(`INSERT INTO audit_logs (actor_id,action,target_type,target_id,result,request_id,ip_address,metadata_json,created_at) VALUES (?,?,?,?,?,?,?,?,?)`), actorID, action, targetType, nullable(targetID), result, nullable(requestID), nullable(ip), string(data), s.now().UTC())
	return err
}

// randomToken generates a cryptographically secure token with the requested number of bytes.
func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// tokenDigest computes the SHA-256 digest of a session token for secure storage.
func tokenDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// nullable converts an empty string into a database null value.
func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
