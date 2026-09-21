package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"
)

// adminRoles are the roles the generated admin panel understands: viewer is
// read-only, editor reads and writes, admin additionally manages the audit
// log view.
var adminRoles = map[string]bool{
	"admin":  true,
	"editor": true,
	"viewer": true,
}

// runAdminRoot creates the administrator account (role admin) for the
// generated admin panel:
//
//	airway admin:root <username> <password>
//
// It talks to admin_users through the map-based repo API, so it works in any
// project (and in the framework repo) without importing project code.
func runAdminRoot(args []string) error {
	return createAdminPanelAccount(args, "root")
}

// runAdminMember creates a non-admin account for the generated admin panel:
//
//	airway admin:member <username> <password> [--role=editor|viewer]
//
// The default role is editor; creating admins goes through `admin:root`.
func runAdminMember(args []string) error {
	return createAdminPanelAccount(args, "member")
}

func createAdminPanelAccount(args []string, kind string) error {
	usage := "usage: airway admin:root <username> <password>"
	if kind == "member" {
		usage = "usage: airway admin:member <username> <password> [--role=editor|viewer]"
	}

	role := "editor"
	if kind == "root" {
		role = "admin"
	}

	rest := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "--role=") {
			if kind != "member" {
				return fmt.Errorf("%s creates admin accounts; use admin:member for lesser roles", usage)
			}
			role = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--role=")))
			continue
		}
		rest = append(rest, arg)
	}
	if !adminRoles[role] {
		return fmt.Errorf("unknown role %q (use editor or viewer)", role)
	}
	if kind == "root" && role != "admin" {
		return fmt.Errorf("admin:root always creates role admin; use admin:member for lesser roles")
	}
	if kind == "member" && role == "admin" {
		return fmt.Errorf("admin:member cannot create role admin; use admin:root")
	}
	if len(rest) >= 1 && isHelpArg(rest[0]) {
		fmt.Println(usage)
		return nil
	}
	if len(rest) != 2 {
		return fmt.Errorf("%s", usage)
	}

	username := strings.TrimSpace(rest[0])
	password := rest[1]
	if username == "" || strings.ContainsAny(username, " \t\r\n") || len(username) > 255 {
		return fmt.Errorf("invalid admin username %q: use a non-empty name without whitespace", username)
	}
	if password == "" {
		return fmt.Errorf("password must not be empty")
	}

	dsn, err := cliDSN()
	if err != nil {
		return err
	}

	db, err := repo.NewDB(dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	existing, err := repo.Count(db, sql.SelectColumns("count(*)").From("admin_users").Where(sql.Eq("username", username)))
	if err != nil {
		return fmt.Errorf("looking up admin_users failed (did you run db:migrate?): %w", err)
	}
	if existing > 0 {
		return fmt.Errorf("admin user %q already exists", username)
	}

	digest, err := utils.EncryptPassword(password)
	if err != nil {
		return err
	}

	now := time.Now()
	if _, err := repo.InsertMap(db, sql.Insert(sql.H{
		"username":        username,
		"password_digest": digest,
		"role":            role,
		"created_at":      now,
		"updated_at":      now,
	}).Into("admin_users")); err != nil {
		return err
	}

	fmt.Printf("admin user created: %s (role %s)\n", username, role)
	return nil
}
