package pg

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type Driver string

const (
	DriverPostgres Driver = "pgx"
	DriverMySQL    Driver = "mysql"
	DriverSQLite   Driver = "sqlite3"
)

type DB struct {
	driver Driver
	conn   *sqlx.DB
}

func NewDB(dsn string) (*DB, error) {
	return NewDBWithDriver("", dsn)
}

func NewDBWithDriver(driverName string, dsn string) (*DB, error) {
	driver, normalizedDSN, err := resolveDriver(driverName, dsn)
	if err != nil {
		return nil, err
	}

	conn, err := sqlx.Open(string(driver), normalizedDSN)
	if err != nil {
		return nil, err
	}

	configurePool(driver, conn)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	return &DB{driver: driver, conn: conn}, nil
}

func (db *DB) Driver() Driver {
	if db == nil {
		return ""
	}

	return db.driver
}

func (db *DB) Close() error {
	if db == nil || db.conn == nil {
		return nil
	}

	return db.conn.Close()
}

func configurePool(driver Driver, conn *sqlx.DB) {
	switch driver {
	case DriverSQLite:
		conn.SetMaxOpenConns(1)
		conn.SetMaxIdleConns(1)
		conn.SetConnMaxLifetime(0)
		conn.SetConnMaxIdleTime(0)
	default:
		conn.SetMaxOpenConns(20)
		conn.SetMaxIdleConns(5)
		conn.SetConnMaxLifetime(time.Hour)
		conn.SetConnMaxIdleTime(30 * time.Minute)
	}
}

func resolveDriver(driverName string, dsn string) (Driver, string, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return "", "", fmt.Errorf("database dsn must not be empty")
	}

	driver, err := normalizeDriver(driverName)
	if err != nil {
		return "", "", err
	}

	if driver == "" {
		driver = detectDriverFromDSN(dsn)
		if driver == "" {
			return "", "", fmt.Errorf("cannot infer database driver from dsn: %s", dsn)
		}
	}

	return driver, normalizeDSN(driver, dsn), nil
}

func normalizeDriver(driverName string) (Driver, error) {
	switch strings.ToLower(strings.TrimSpace(driverName)) {
	case "", "auto":
		return "", nil
	case "postgres", "postgresql", "pg", "pgx":
		return DriverPostgres, nil
	case "mysql", "mysql8", "mysql8.4":
		return DriverMySQL, nil
	case "sqlite", "sqlite3":
		return DriverSQLite, nil
	default:
		return "", fmt.Errorf("unsupported database driver: %s", driverName)
	}
}

func detectDriverFromDSN(dsn string) Driver {
	lowerDSN := strings.ToLower(strings.TrimSpace(dsn))

	switch {
	case strings.HasPrefix(lowerDSN, "postgres://"), strings.HasPrefix(lowerDSN, "postgresql://"):
		return DriverPostgres
	case strings.HasPrefix(lowerDSN, "mysql://"):
		return DriverMySQL
	case strings.HasPrefix(lowerDSN, "sqlite://"), strings.HasPrefix(lowerDSN, "sqlite3://"):
		return DriverSQLite
	case strings.HasPrefix(lowerDSN, "file:"), lowerDSN == ":memory:":
		return DriverSQLite
	case strings.Contains(lowerDSN, "@tcp("), strings.Contains(lowerDSN, "@unix("), strings.Contains(lowerDSN, ")/"):
		return DriverMySQL
	case strings.HasSuffix(lowerDSN, ".db"), strings.HasSuffix(lowerDSN, ".sqlite"), strings.HasSuffix(lowerDSN, ".sqlite3"):
		return DriverSQLite
	case strings.Contains(lowerDSN, "host="), strings.Contains(lowerDSN, "user="), strings.Contains(lowerDSN, "dbname="):
		return DriverPostgres
	default:
		return ""
	}
}

func normalizeDSN(driver Driver, dsn string) string {
	dsn = strings.TrimSpace(dsn)
	lowerDSN := strings.ToLower(dsn)

	if driver != DriverSQLite {
		if driver == DriverMySQL {
			return normalizeMySQLDSN(dsn)
		}

		return dsn
	}

	switch {
	case strings.HasPrefix(lowerDSN, "sqlite://"):
		return dsn[len("sqlite://"):]
	case strings.HasPrefix(lowerDSN, "sqlite3://"):
		return dsn[len("sqlite3://"):]
	default:
		return dsn
	}
}

func normalizeMySQLDSN(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	lowerDSN := strings.ToLower(dsn)

	if strings.HasPrefix(lowerDSN, "mysql://") {
		parsed, err := url.Parse(dsn)
		if err == nil {
			credentials := ""
			if parsed.User != nil {
				credentials = parsed.User.Username()
				if password, ok := parsed.User.Password(); ok {
					credentials += ":" + password
				}
			}

			host := parsed.Host
			if host == "" {
				host = "127.0.0.1:3306"
			}

			database := strings.TrimPrefix(parsed.Path, "/")
			params := parsed.Query()
			if params.Get("parseTime") == "" {
				params.Set("parseTime", "true")
			}

			encoded := params.Encode()
			if encoded != "" {
				encoded = "?" + encoded
			}

			return fmt.Sprintf("%s@tcp(%s)/%s%s", credentials, host, database, encoded)
		}
	}

	if strings.Contains(lowerDSN, "parsetime=") {
		return dsn
	}

	if strings.Contains(dsn, "?") {
		return dsn + "&parseTime=true"
	}

	return dsn + "?parseTime=true"
}
