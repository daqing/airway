package repo

import (
	"fmt"
	"regexp"
	"strings"

	buildersql "github.com/daqing/airway/lib/sql"
)

var compiledArgPattern = regexp.MustCompile(`@([A-Za-z_][A-Za-z0-9_]*)`)

var sqliteForClausePattern = regexp.MustCompile(`\s+FOR\s+(UPDATE|SHARE)\b`)

var sqliteILikePattern = regexp.MustCompile(`\bILIKE\b`)

var sqliteNotILikePattern = regexp.MustCompile(`\bNOT\s+ILIKE\b`)

var mysqlReturningPattern = regexp.MustCompile(`\s+RETURNING\s+.+$`)

var mysqlConflictDoNothingPattern = regexp.MustCompile(`\s+ON\s+CONFLICT(?:\s+ON\s+CONSTRAINT\s+\S+|\s*\([^)]*\))\s+DO\s+NOTHING`)

var mysqlConflictDoUpdatePattern = regexp.MustCompile(`\s+ON\s+CONFLICT(?:\s+ON\s+CONSTRAINT\s+\S+|\s*\([^)]*\))\s+DO\s+UPDATE\s+SET\s+`)

var mysqlExcludedPattern = regexp.MustCompile(`\bEXCLUDED\.([A-Za-z_][A-Za-z0-9_]*)\b`)

func (db *DB) prepareBuilder(b buildersql.Stmt) (string, []any, error) {
	query, vals := b.ToSQL()
	return db.prepareQuery(query, vals)
}

func (db *DB) prepareInsertBuilder(b buildersql.Stmt) (string, []any, error) {
	query, vals := b.ToSQL()
	if db.driver == DriverMySQL {
		query = stripReturningClause(query)
	}

	return db.prepareQuery(query, vals)
}

func (db *DB) prepareQuery(query string, vals buildersql.NamedArgs) (string, []any, error) {
	transformedQuery, err := transformQueryForDriver(db.driver, query)
	if err != nil {
		return "", nil, err
	}

	compiledQuery, args, err := compileNamedQuery(transformedQuery, vals)
	if err != nil {
		return "", nil, err
	}

	return db.conn.Rebind(compiledQuery), args, nil
}

func compileNamedQuery(query string, vals buildersql.NamedArgs) (string, []any, error) {
	if query == "" {
		return "", nil, nil
	}

	matches := compiledArgPattern.FindAllStringSubmatchIndex(query, -1)
	if len(matches) == 0 {
		return query, nil, nil
	}

	args := make([]any, 0, len(matches))
	var builder strings.Builder
	last := 0

	for _, match := range matches {
		name := query[match[2]:match[3]]
		value, ok := vals[name]
		if !ok {
			return "", nil, fmt.Errorf("missing named arg: %s", name)
		}

		builder.WriteString(query[last:match[0]])
		builder.WriteString("?")
		args = append(args, value)
		last = match[1]
	}

	builder.WriteString(query[last:])
	return builder.String(), args, nil
}

func transformQueryForDriver(driver Driver, query string) (string, error) {
	switch driver {
	case DriverSQLite:
		if strings.Contains(strings.ToUpper(query), " ON CONFLICT ON CONSTRAINT ") {
			return "", fmt.Errorf("sqlite does not support ON CONFLICT ON CONSTRAINT")
		}

		query = sqliteForClausePattern.ReplaceAllString(query, "")
		query = sqliteNotILikePattern.ReplaceAllString(query, "NOT LIKE")
		query = sqliteILikePattern.ReplaceAllString(query, "LIKE")

		return strings.TrimSpace(query), nil
	case DriverMySQL:
		return transformMySQLQuery(query)
	default:
		return query, nil
	}
}

func transformMySQLQuery(query string) (string, error) {
	upperQuery := strings.ToUpper(query)
	if strings.Contains(upperQuery, " ON CONFLICT ON CONSTRAINT ") {
		return "", fmt.Errorf("mysql does not support ON CONFLICT ON CONSTRAINT")
	}

	if strings.Contains(upperQuery, " RETURNING ") {
		return "", fmt.Errorf("mysql does not support RETURNING in this execution path")
	}

	query = strings.ReplaceAll(query, `"`, "`")
	query = sqliteNotILikePattern.ReplaceAllString(query, "NOT LIKE")
	query = sqliteILikePattern.ReplaceAllString(query, "LIKE")
	query = mysqlExcludedPattern.ReplaceAllString(query, "VALUES($1)")

	if mysqlConflictDoNothingPattern.MatchString(query) {
		query = strings.Replace(query, "INSERT INTO", "INSERT IGNORE INTO", 1)
		query = mysqlConflictDoNothingPattern.ReplaceAllString(query, "")
	}

	if mysqlConflictDoUpdatePattern.MatchString(query) {
		query = mysqlConflictDoUpdatePattern.ReplaceAllString(query, " ON DUPLICATE KEY UPDATE ")
	}

	return strings.TrimSpace(query), nil
}

func stripReturningClause(query string) string {
	return strings.TrimSpace(mysqlReturningPattern.ReplaceAllString(query, ""))
}
