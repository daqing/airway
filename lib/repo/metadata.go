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

	rows, err := db.conn.QueryContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := []string{}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}
