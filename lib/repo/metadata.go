package repo

import "context"

func ListTables(db *DB) ([]string, error) {
	query := ""

	switch db.Driver() {
	case DriverSQLite:
		query = `SELECT name
FROM sqlite_master
WHERE type = 'table'
  AND name NOT LIKE 'sqlite_%'
ORDER BY name`
	case DriverMySQL:
		query = `SELECT table_name
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_type IN ('BASE TABLE', 'SYSTEM TABLE', 'VIEW')
ORDER BY table_name`
	default:
		query = `SELECT table_name
FROM information_schema.tables
WHERE table_schema = CURRENT_SCHEMA()
  AND table_type IN ('BASE TABLE', 'VIEW')
ORDER BY table_name`
	}

	tables := []string{}
	if err := db.conn.SelectContext(context.Background(), &tables, query); err != nil {
		return nil, err
	}

	return tables, nil
}
