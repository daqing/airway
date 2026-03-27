package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/daqing/airway/lib/repo"
	reposql "github.com/daqing/airway/lib/sql"
)

type repoREPL struct {
	db        *repo.DB
	evaluator *replEvaluator
	out       io.Writer
	errOut    io.Writer
}

func newRepoREPL(db *repo.DB, out io.Writer, errOut io.Writer) *repoREPL {
	return &repoREPL{
		db:        db,
		evaluator: newREPLEvaluator(db),
		out:       out,
		errOut:    errOut,
	}
}

func runRepoREPL(args []string) {
	flags := flag.NewFlagSet("repl", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	driver := flags.String("driver", "", "database driver override")
	dsn := flags.String("dsn", "", "database dsn override")

	if err := flags.Parse(args); err != nil {
		logFatal(err)
	}

	db, err := openREPLDB(*driver, *dsn)
	if err != nil {
		logFatal(err)
	}

	session := newRepoREPL(db, os.Stdout, os.Stderr)
	remaining := flags.Args()
	if len(remaining) > 0 {
		_, execErr := session.execute(strings.TrimSpace(strings.Join(remaining, " ")))
		if execErr != nil {
			logFatal(execErr)
		}

		return
	}

	_, _ = io.WriteString(session.out, fmt.Sprintf("repo repl connected to %s\r\n", db.Driver()))
	_, _ = io.WriteString(session.out, "type \"help\" for commands, \"exit\" to quit\r\n")

	input := newREPLInput(os.Stdin, session.out)

	for {
		line, readErr := input.ReadLine("repo> ")
		if readErr != nil {
			if readErr == io.EOF {
				_, _ = io.WriteString(session.out, "\r\n")
				break
			}

			logFatal(readErr)
		}

		exit, execErr := session.execute(line)
		if execErr != nil {
			_, _ = io.WriteString(session.errOut, fmt.Sprintf("error: %v\r\n", execErr))
			continue
		}

		if exit {
			break
		}
	}
}

func openREPLDB(driver string, dsn string) (*repo.DB, error) {
	if strings.TrimSpace(dsn) != "" {
		return repo.NewDBWithDriver(driver, dsn)
	}

	if db, ok := repo.CurrentDBOK(); ok {
		return db, nil
	}

	return nil, fmt.Errorf("database is not configured; set AIRWAY_DB_DSN or pass --dsn")
}

func (r repoREPL) execute(line string) (bool, error) {
	command, _ := splitCommand(line)
	if command == "" {
		return false, nil
	}

	switch command {
	case "help", "?":
		r.printHelp()
		return false, nil
	case "driver":
		return false, r.writeJSON(string(r.db.Driver()))
	case "tables":
		return false, r.handleTables()
	case "exit", "quit":
		return true, nil
	default:
		value, err := r.evaluator.Eval(strings.TrimSpace(line))
		if err != nil {
			return false, err
		}

		return false, r.writeResult(value)
	}
}

func (r repoREPL) handleTables() error {
	tables, err := repo.ListTables(r.db)
	if err != nil {
		return err
	}

	return r.writeJSON(tables)
}

func (r repoREPL) printHelp() {
	writeDisplay(r.out, `Commands:
  help
  driver
  tables
  exit

Expressions:
  repo.FindOne("users", pg.Eq("id", 1))
  repo.FindOne[models.User](pg.Select("id").Where(pg.Eq("id", 1)))
  repo.Insert[models.User](pg.H{"id": 1234})
  repo.FindOne[User](pg.Eq("id", 1))
  repo.FindOne[struct{ ID int64; Email string }](pg.Select("id AS id, email AS email").From("users").Where(pg.Eq("id", 1)))
	repo.Find("users", pg.Select("*").Where(pg.Eq("id", 1)))
  repo.Find("users", pg.AllOf(pg.Eq("enabled", true), pg.Like("email", "%@example.com")))
  repo.Insert("users", pg.H{"email": "dev@example.com", "enabled": true})
  repo.Update("users", pg.H{"enabled": false}, pg.Eq("id", 1))
  repo.Delete("users", pg.Eq("id", 1))
  repo.Preview(pg.Select("*").From("users").Where(pg.Eq("id", 1)))
  pg.Select("*").From("users").Where(pg.Eq("id", 1))

Available namespaces:
  repo, sql, pg, mysql, sqlite, models

Repo helpers:
  repo.Find(tableOrStmt, [cond])
  repo.FindOne(tableOrStmt, [cond])
  repo.Count(tableOrStmt, [cond])
  repo.Exists(tableOrStmt, [cond])
  repo.Insert(tableOrStmt, [values], [returning])
  repo.Update(tableOrStmt, [values], cond|true)
  repo.Delete(tableOrStmt, cond|true)
  repo.Preview(stmt)
  repo.Tables()
  repo.Driver()

Notes:
  update/delete without a condition require explicit true as the last argument
	repo.Find/FindOne/Count/Exists also accept table + stmt and will bind the table when the stmt has no table yet
  repo.Find[T]/FindOne[T] support anonymous struct type arguments and use struct scan instead of map results
  when T implements TableName(), repo.Find[T]/FindOne[T] can infer the table and omit the first table argument
  app/models types are also available as top-level type names such as User
  entering a builder expression directly prints the compiled SQL and args`)
}

func (r repoREPL) writeResult(value any) error {
	if stmt, ok := value.(reposql.Stmt); ok {
		query, args, err := repo.Preview(r.db, stmt)
		if err != nil {
			return err
		}

		return r.writeJSON(map[string]any{
			"query": query,
			"args":  normalizeResultValue(args),
		})
	}

	if fragment, ok := value.(interface {
		ToSQL() (string, reposql.NamedArgs)
	}); ok {
		sql, args := fragment.ToSQL()
		return r.writeJSON(map[string]any{
			"sql":  sql,
			"args": normalizeResultValue(args),
		})
	}

	return r.writeJSON(value)
}

func (r repoREPL) writeJSON(value any) error {
	encoded, err := json.MarshalIndent(normalizeResultValue(value), "", "  ")
	if err != nil {
		return err
	}

	_, err = io.WriteString(r.out, normalizeDisplayText(string(encoded))+"\r\n")
	return err
}

func splitCommand(line string) (string, string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", ""
	}

	index := strings.IndexAny(trimmed, " \t\n")
	if index < 0 {
		return strings.ToLower(trimmed), ""
	}

	return strings.ToLower(strings.TrimSpace(trimmed[:index])), strings.TrimSpace(trimmed[index+1:])
}

func normalizeResultValue(value any) any {
	if value == nil {
		return nil
	}

	switch typed := value.(type) {
	case []byte:
		return string(typed)
	case json.Number:
		return typed.String()
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Pointer:
		if reflected.IsNil() {
			return nil
		}

		return normalizeResultValue(reflected.Elem().Interface())
	case reflect.Map:
		if reflected.Type().Key().Kind() != reflect.String {
			return value
		}

		normalized := map[string]any{}
		for _, key := range reflected.MapKeys() {
			normalized[key.String()] = normalizeResultValue(reflected.MapIndex(key).Interface())
		}

		return normalized
	case reflect.Slice, reflect.Array:
		if reflected.Type().Elem().Kind() == reflect.Uint8 {
			bytes := make([]byte, reflected.Len())
			reflect.Copy(reflect.ValueOf(bytes), reflected)
			return string(bytes)
		}

		normalized := make([]any, 0, reflected.Len())
		for idx := 0; idx < reflected.Len(); idx++ {
			normalized = append(normalized, normalizeResultValue(reflected.Index(idx).Interface()))
		}

		return normalized
	default:
		return value
	}
}

func logFatal(err error) {
	if err == nil {
		return
	}

	_, _ = io.WriteString(os.Stderr, fmt.Sprintf("%v\r\n", err))
	os.Exit(1)
}

func writeDisplay(out io.Writer, text string) {
	_, _ = io.WriteString(out, normalizeDisplayText(text)+"\r\n")
}

func normalizeDisplayText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\n", "\r\n")
}
