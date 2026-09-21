package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"
)

// runAdminUser creates an admin account for the generated admin panel:
//
//	airway admin:user <email> <password>
//
// It talks to admin_users through the map-based repo API, so it works in any
// project (and in the framework repo) without importing project code.
func runAdminUser(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		fmt.Println("usage: airway admin:user <email> <password>")
		return nil
	}
	if len(args) != 2 {
		return fmt.Errorf("usage: airway admin:user <email> <password>")
	}

	email := strings.TrimSpace(args[0])
	password := args[1]
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("invalid admin email %q", email)
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

	existing, err := repo.Count(db, sql.SelectColumns("count(*)").From("admin_users").Where(sql.Eq("email", email)))
	if err != nil {
		return fmt.Errorf("looking up admin_users failed (did you run db:migrate?): %w", err)
	}
	if existing > 0 {
		return fmt.Errorf("admin user %q already exists", email)
	}

	digest, err := utils.EncryptPassword(password)
	if err != nil {
		return err
	}

	now := time.Now()
	if _, err := repo.InsertMap(db, sql.Insert(sql.H{
		"email":           email,
		"password_digest": digest,
		"created_at":      now,
		"updated_at":      now,
	}).Into("admin_users")); err != nil {
		return err
	}

	fmt.Printf("admin user created: %s\n", email)
	return nil
}
