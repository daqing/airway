package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"github.com/daqing/airway/lib/repo"
	reposql "github.com/daqing/airway/lib/sql"
	mysqlsql "github.com/daqing/airway/lib/sql/mysql"
	pgsql "github.com/daqing/airway/lib/sql/pg"
	sqlitesql "github.com/daqing/airway/lib/sql/sqlite"
)

type replCallable func(args []any) (any, error)

type replNamespace map[string]any

type replEvaluator struct {
	db      *repo.DB
	symbols map[string]any
}

func newREPLEvaluator(db *repo.DB) *replEvaluator {
	evaluator := &replEvaluator{db: db, symbols: map[string]any{}}
	evaluator.symbols["repo"] = evaluator.newRepoNamespace()
	evaluator.symbols["sql"] = newSQLNamespace()
	evaluator.symbols["pg"] = newPGNamespace()
	evaluator.symbols["mysql"] = newMySQLNamespace()
	evaluator.symbols["sqlite"] = newSQLiteNamespace()

	return evaluator
}

func (e *replEvaluator) Eval(input string) (any, error) {
	expr, err := parser.ParseExpr(strings.TrimSpace(input))
	if err != nil {
		return nil, err
	}

	return e.evalExpr(expr)
}

func (e *replEvaluator) evalExpr(expr ast.Expr) (any, error) {
	switch node := expr.(type) {
	case *ast.BasicLit:
		return evalBasicLit(node)
	case *ast.Ident:
		return e.evalIdent(node)
	case *ast.CallExpr:
		return e.evalCall(node)
	case *ast.SelectorExpr:
		return e.evalSelector(node)
	case *ast.CompositeLit:
		return e.evalCompositeLit(node)
	case *ast.UnaryExpr:
		return e.evalUnary(node)
	case *ast.ParenExpr:
		return e.evalExpr(node.X)
	default:
		return nil, fmt.Errorf("unsupported expression: %T", expr)
	}
}

func evalBasicLit(lit *ast.BasicLit) (any, error) {
	switch lit.Kind {
	case token.STRING:
		return strconv.Unquote(lit.Value)
	case token.INT:
		return strconv.ParseInt(lit.Value, 0, 64)
	case token.FLOAT:
		return strconv.ParseFloat(lit.Value, 64)
	case token.CHAR:
		decoded, err := strconv.Unquote(lit.Value)
		if err != nil {
			return nil, err
		}

		runes := []rune(decoded)
		if len(runes) != 1 {
			return nil, fmt.Errorf("invalid rune literal: %s", lit.Value)
		}

		return runes[0], nil
	default:
		return nil, fmt.Errorf("unsupported literal: %s", lit.Value)
	}
}

func (e *replEvaluator) evalIdent(ident *ast.Ident) (any, error) {
	switch ident.Name {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "nil":
		return nil, nil
	}

	value, ok := e.symbols[ident.Name]
	if !ok {
		return nil, fmt.Errorf("unknown identifier: %s", ident.Name)
	}

	return value, nil
}

func (e *replEvaluator) evalCall(call *ast.CallExpr) (any, error) {
	if call.Ellipsis.IsValid() {
		return nil, fmt.Errorf("ellipsis calls are not supported")
	}

	callee, err := e.evalExpr(call.Fun)
	if err != nil {
		return nil, err
	}

	args := make([]any, 0, len(call.Args))
	for _, arg := range call.Args {
		value, evalErr := e.evalExpr(arg)
		if evalErr != nil {
			return nil, evalErr
		}

		args = append(args, value)
	}

	return invokeCallable(callee, args)
}

func (e *replEvaluator) evalSelector(selector *ast.SelectorExpr) (any, error) {
	parent, err := e.evalExpr(selector.X)
	if err != nil {
		return nil, err
	}

	if namespace, ok := parent.(replNamespace); ok {
		value, exists := namespace[selector.Sel.Name]
		if !exists {
			return nil, fmt.Errorf("unknown symbol: %s", selector.Sel.Name)
		}

		return value, nil
	}

	method, ok := lookupMethod(parent, selector.Sel.Name)
	if !ok {
		return nil, fmt.Errorf("%T has no method %s", parent, selector.Sel.Name)
	}

	return method, nil
}

func (e *replEvaluator) evalCompositeLit(lit *ast.CompositeLit) (any, error) {
	switch typed := lit.Type.(type) {
	case *ast.ArrayType:
		items := make([]any, 0, len(lit.Elts))
		for _, elt := range lit.Elts {
			value, err := e.evalExpr(elt)
			if err != nil {
				return nil, err
			}

			items = append(items, value)
		}

		return items, nil
	case *ast.MapType:
		_ = typed
		return e.evalMapLiteral(lit)
	case *ast.SelectorExpr:
		name := typed.Sel.Name
		if name == "H" || name == "NamedArgs" {
			return e.evalMapLiteral(lit)
		}

		return nil, fmt.Errorf("unsupported composite literal type: %s", name)
	default:
		return nil, fmt.Errorf("unsupported composite literal: %T", lit.Type)
	}
}

func (e *replEvaluator) evalMapLiteral(lit *ast.CompositeLit) (map[string]any, error) {
	values := map[string]any{}
	for _, elt := range lit.Elts {
		pair, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil, fmt.Errorf("map literals require key/value elements")
		}

		keyValue, err := e.evalExpr(pair.Key)
		if err != nil {
			return nil, err
		}

		key, err := coerceString(keyValue)
		if err != nil {
			return nil, err
		}

		value, err := e.evalExpr(pair.Value)
		if err != nil {
			return nil, err
		}

		values[key] = value
	}

	return values, nil
}

func (e *replEvaluator) evalUnary(expr *ast.UnaryExpr) (any, error) {
	value, err := e.evalExpr(expr.X)
	if err != nil {
		return nil, err
	}

	switch expr.Op {
	case token.SUB:
		switch typed := value.(type) {
		case int64:
			return -typed, nil
		case float64:
			return -typed, nil
		default:
			return nil, fmt.Errorf("cannot negate %T", value)
		}
	case token.ADD:
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported unary operator: %s", expr.Op.String())
	}
}

func invokeCallable(callee any, args []any) (any, error) {
	switch typed := callee.(type) {
	case replCallable:
		return typed(args)
	case reflect.Value:
		return callReflect(typed, args)
	default:
		return callReflect(reflect.ValueOf(callee), args)
	}
}

func callReflect(fn reflect.Value, args []any) (any, error) {
	if !fn.IsValid() || fn.Kind() != reflect.Func {
		return nil, fmt.Errorf("value is not callable")
	}

	callArgs, err := buildCallArgs(fn.Type(), args)
	if err != nil {
		return nil, err
	}

	results := fn.Call(callArgs)
	if len(results) == 0 {
		return nil, nil
	}

	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if last := results[len(results)-1]; last.Type().Implements(errorType) {
		if !last.IsNil() {
			return nil, last.Interface().(error)
		}

		results = results[:len(results)-1]
	}

	if len(results) == 0 {
		return nil, nil
	}

	if len(results) == 1 {
		return results[0].Interface(), nil
	}

	values := make([]any, 0, len(results))
	for _, result := range results {
		values = append(values, result.Interface())
	}

	return values, nil
}

func buildCallArgs(fnType reflect.Type, args []any) ([]reflect.Value, error) {
	if !fnType.IsVariadic() {
		if len(args) != fnType.NumIn() {
			return nil, fmt.Errorf("expected %d args, got %d", fnType.NumIn(), len(args))
		}

		converted := make([]reflect.Value, 0, len(args))
		for idx, arg := range args {
			value, err := convertArg(arg, fnType.In(idx))
			if err != nil {
				return nil, err
			}

			converted = append(converted, value)
		}

		return converted, nil
	}

	minArgs := fnType.NumIn() - 1
	if len(args) < minArgs {
		return nil, fmt.Errorf("expected at least %d args, got %d", minArgs, len(args))
	}

	converted := make([]reflect.Value, 0, len(args))
	for idx := 0; idx < minArgs; idx++ {
		value, err := convertArg(args[idx], fnType.In(idx))
		if err != nil {
			return nil, err
		}

		converted = append(converted, value)
	}

	variadicType := fnType.In(fnType.NumIn() - 1).Elem()
	for idx := minArgs; idx < len(args); idx++ {
		value, err := convertArg(args[idx], variadicType)
		if err != nil {
			return nil, err
		}

		converted = append(converted, value)
	}

	return converted, nil
}

func convertArg(arg any, target reflect.Type) (reflect.Value, error) {
	if arg == nil {
		switch target.Kind() {
		case reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Func:
			return reflect.Zero(target), nil
		default:
			return reflect.Value{}, fmt.Errorf("cannot use nil as %s", target)
		}
	}

	value := reflect.ValueOf(arg)
	if value.Type().AssignableTo(target) {
		return value, nil
	}

	if target.Kind() == reflect.Interface && value.Type().Implements(target) {
		return value, nil
	}

	if target.Kind() == reflect.Interface && target.NumMethod() == 0 {
		return value, nil
	}

	if converted, err := convertScalarArg(arg, target); err == nil {
		return converted, nil
	}

	if target.Kind() == reflect.Slice {
		return convertSliceArg(arg, target)
	}

	if target.Kind() == reflect.Map {
		return convertMapArg(arg, target)
	}

	if value.Type().ConvertibleTo(target) {
		return value.Convert(target), nil
	}

	return reflect.Value{}, fmt.Errorf("cannot use %T as %s", arg, target)
}

func convertScalarArg(arg any, target reflect.Type) (reflect.Value, error) {
	switch target.Kind() {
	case reflect.String:
		text, err := coerceString(arg)
		if err != nil {
			return reflect.Value{}, err
		}

		return reflect.ValueOf(text).Convert(target), nil
	case reflect.Bool:
		truth, ok := arg.(bool)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected bool, got %T", arg)
		}

		return reflect.ValueOf(truth).Convert(target), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		integer, err := coerceInt64(arg)
		if err != nil {
			return reflect.Value{}, err
		}

		value := reflect.New(target).Elem()
		value.SetInt(integer)
		return value, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		integer, err := coerceInt64(arg)
		if err != nil {
			return reflect.Value{}, err
		}

		if integer < 0 {
			return reflect.Value{}, fmt.Errorf("negative value %d cannot be used as %s", integer, target)
		}

		value := reflect.New(target).Elem()
		value.SetUint(uint64(integer))
		return value, nil
	case reflect.Float32, reflect.Float64:
		floating, err := coerceFloat64(arg)
		if err != nil {
			return reflect.Value{}, err
		}

		value := reflect.New(target).Elem()
		value.SetFloat(floating)
		return value, nil
	default:
		return reflect.Value{}, fmt.Errorf("unsupported scalar conversion to %s", target)
	}
}

func convertSliceArg(arg any, target reflect.Type) (reflect.Value, error) {
	value := reflect.ValueOf(arg)
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return reflect.Value{}, fmt.Errorf("expected slice, got %T", arg)
	}

	converted := reflect.MakeSlice(target, value.Len(), value.Len())
	for idx := 0; idx < value.Len(); idx++ {
		item, err := convertArg(value.Index(idx).Interface(), target.Elem())
		if err != nil {
			return reflect.Value{}, err
		}

		converted.Index(idx).Set(item)
	}

	return converted, nil
}

func convertMapArg(arg any, target reflect.Type) (reflect.Value, error) {
	value := reflect.ValueOf(arg)
	if value.Kind() != reflect.Map {
		return reflect.Value{}, fmt.Errorf("expected map, got %T", arg)
	}

	converted := reflect.MakeMapWithSize(target, value.Len())
	for _, key := range value.MapKeys() {
		mappedKey, err := convertArg(key.Interface(), target.Key())
		if err != nil {
			return reflect.Value{}, err
		}

		mappedValue, err := convertArg(value.MapIndex(key).Interface(), target.Elem())
		if err != nil {
			return reflect.Value{}, err
		}

		converted.SetMapIndex(mappedKey, mappedValue)
	}

	return converted, nil
}

func lookupMethod(receiver any, name string) (any, bool) {
	value := reflect.ValueOf(receiver)
	if !value.IsValid() {
		return nil, false
	}

	if method := value.MethodByName(name); method.IsValid() {
		return method, true
	}

	if value.Kind() != reflect.Pointer {
		pointer := reflect.New(value.Type())
		pointer.Elem().Set(value)
		if method := pointer.MethodByName(name); method.IsValid() {
			return method, true
		}
	}

	return nil, false
}

func (e *replEvaluator) newRepoNamespace() replNamespace {
	return replNamespace{
		"Find":    replCallable(e.callRepoFind),
		"FindOne": replCallable(e.callRepoFindOne),
		"Count":   replCallable(e.callRepoCount),
		"Exists":  replCallable(e.callRepoExists),
		"Insert":  replCallable(e.callRepoInsert),
		"Update":  replCallable(e.callRepoUpdate),
		"Delete":  replCallable(e.callRepoDelete),
		"Preview": replCallable(e.callRepoPreview),
		"SQL":     replCallable(e.callRepoPreview),
		"Tables":  replCallable(e.callRepoTables),
		"Driver":  replCallable(e.callRepoDriver),
	}
}

func newSQLNamespace() replNamespace {
	return newBuilderNamespace(
		any(reposql.Select),
		any(reposql.SelectColumns),
		any(reposql.Insert),
		any(reposql.Update),
		any(reposql.Delete),
		replCallable(func(args []any) (any, error) {
			table, err := coerceTableArg(args)
			if err != nil {
				return nil, err
			}

			return reposql.DeleteFrom(table), nil
		}),
		replCallable(func(args []any) (any, error) {
			table, err := coerceTableArg(args)
			if err != nil {
				return nil, err
			}

			return reposql.UpdateTable(table), nil
		}),
	)
}

func newPGNamespace() replNamespace {
	namespace := newBuilderNamespace(
		any(pgsql.Select),
		any(pgsql.SelectColumns),
		any(pgsql.Insert),
		any(pgsql.Update),
		any(pgsql.Delete),
		replCallable(func(args []any) (any, error) {
			table, err := coerceTableArg(args)
			if err != nil {
				return nil, err
			}

			return pgsql.DeleteFrom(table), nil
		}),
		replCallable(func(args []any) (any, error) {
			table, err := coerceTableArg(args)
			if err != nil {
				return nil, err
			}

			return pgsql.UpdateTable(table), nil
		}),
	)
	namespace["SubQuery"] = any(pgsql.SubQuery)
	namespace["ExistsQuery"] = any(pgsql.ExistsQuery)
	namespace["NotExistsQuery"] = any(pgsql.NotExistsQuery)
	return namespace
}

func newMySQLNamespace() replNamespace {
	namespace := newBuilderNamespace(
		any(mysqlsql.Select),
		any(mysqlsql.SelectColumns),
		any(mysqlsql.Insert),
		any(mysqlsql.Update),
		any(mysqlsql.Delete),
		replCallable(func(args []any) (any, error) {
			table, err := coerceTableArg(args)
			if err != nil {
				return nil, err
			}

			return mysqlsql.DeleteFrom(table), nil
		}),
		replCallable(func(args []any) (any, error) {
			table, err := coerceTableArg(args)
			if err != nil {
				return nil, err
			}

			return mysqlsql.UpdateTable(table), nil
		}),
	)
	namespace["SubQuery"] = any(mysqlsql.SubQuery)
	namespace["ExistsQuery"] = any(mysqlsql.ExistsQuery)
	namespace["NotExistsQuery"] = any(mysqlsql.NotExistsQuery)
	return namespace
}

func newSQLiteNamespace() replNamespace {
	namespace := newBuilderNamespace(
		any(sqlitesql.Select),
		any(sqlitesql.SelectColumns),
		any(sqlitesql.Insert),
		any(sqlitesql.Update),
		any(sqlitesql.Delete),
		replCallable(func(args []any) (any, error) {
			table, err := coerceTableArg(args)
			if err != nil {
				return nil, err
			}

			return sqlitesql.DeleteFrom(table), nil
		}),
		replCallable(func(args []any) (any, error) {
			table, err := coerceTableArg(args)
			if err != nil {
				return nil, err
			}

			return sqlitesql.UpdateTable(table), nil
		}),
	)
	namespace["SubQuery"] = any(sqlitesql.SubQuery)
	namespace["ExistsQuery"] = any(sqlitesql.ExistsQuery)
	namespace["NotExistsQuery"] = any(sqlitesql.NotExistsQuery)
	return namespace
}

func newBuilderNamespace(selectFn any, selectColumnsFn any, insertFn any, updateFn any, deleteFn any, deleteFromFn replCallable, updateTableFn replCallable) replNamespace {
	return replNamespace{
		"Select":        selectFn,
		"SelectColumns": selectColumnsFn,
		"Insert":        insertFn,
		"Update":        updateFn,
		"Delete":        deleteFn,
		"DeleteFrom":    deleteFromFn,
		"UpdateTable":   updateTableFn,
		"TableOf":       any(reposql.TableOf),
		"TableAlias":    any(reposql.TableAlias),
		"Ref":           any(reposql.Ref),
		"RefAs":         any(reposql.RefAs),
		"Field":         any(reposql.Field),
		"Col":           any(reposql.Col),
		"Ident":         any(reposql.Ident),
		"Eq":            any(reposql.Eq),
		"NotEq":         any(reposql.NotEq),
		"Gt":            any(reposql.Gt),
		"Gte":           any(reposql.Gte),
		"Lt":            any(reposql.Lt),
		"Lte":           any(reposql.Lte),
		"Like":          any(reposql.Like),
		"NotLike":       any(reposql.NotLike),
		"ILike":         any(reposql.ILike),
		"NotILike":      any(reposql.NotILike),
		"AllOf":         any(reposql.AllOf),
		"AnyOf":         any(reposql.AnyOf),
		"Not":           any(reposql.Not),
		"IsNull":        any(reposql.IsNull),
		"IsNotNull":     any(reposql.IsNotNull),
		"Between":       any(reposql.Between),
		"NotBetween":    any(reposql.NotBetween),
		"HCond":         any(reposql.HCond),
		"Compare":       any(reposql.Compare),
		"FieldEq":       any(reposql.FieldEq),
		"FieldNotEq":    any(reposql.FieldNotEq),
		"FieldGt":       any(reposql.FieldGt),
		"FieldGte":      any(reposql.FieldGte),
		"FieldLt":       any(reposql.FieldLt),
		"FieldLte":      any(reposql.FieldLte),
		"FieldLike":     any(reposql.FieldLike),
		"FieldILike":    any(reposql.FieldILike),
		"MatchFields":   any(reposql.MatchFields),
		"HCondRef":      any(reposql.HCondRef),
		"HCondTable":    any(reposql.HCondTable),
		"MatchTable":    any(reposql.MatchTable),
		"RawCondition":  any(reposql.RawCondition),
		"In":            replCallable(callInCondition),
		"NotIn":         replCallable(callNotInCondition),
		"Expr":          any(reposql.Expr),
		"ExprNamed":     any(reposql.ExprNamed),
		"Raw":           any(reposql.Raw),
		"Column":        any(reposql.Column),
		"Excluded":      any(reposql.Excluded),
		"Default":       any(reposql.Default),
		"Func":          any(reposql.Func),
		"Op":            any(reposql.Op),
		"Cast":          any(reposql.Cast),
		"Array":         any(reposql.Array),
		"Any":           any(reposql.Any),
		"AllExpr":       any(reposql.AllExpr),
	}
}

func callInCondition(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("In expects 2 args")
	}

	column, err := coerceString(args[0])
	if err != nil {
		return nil, err
	}

	items, err := toAnySlice(args[1])
	if err != nil {
		return nil, err
	}

	return reposql.In(column, items), nil
}

func callNotInCondition(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("NotIn expects 2 args")
	}

	column, err := coerceString(args[0])
	if err != nil {
		return nil, err
	}

	items, err := toAnySlice(args[1])
	if err != nil {
		return nil, err
	}

	return reposql.NotIn(column, items), nil
}

func (e *replEvaluator) callRepoFind(args []any) (any, error) {
	stmt, err := e.buildSelectStmt(args)
	if err != nil {
		return nil, err
	}

	return repo.FindMaps(e.db, stmt)
}

func (e *replEvaluator) callRepoFindOne(args []any) (any, error) {
	stmt, err := e.buildSelectStmt(args)
	if err != nil {
		return nil, err
	}

	return repo.FindOneMap(e.db, stmt)
}

func (e *replEvaluator) callRepoCount(args []any) (any, error) {
	stmt, err := e.buildCountStmt(args)
	if err != nil {
		return nil, err
	}

	return repo.Count(e.db, stmt)
}

func (e *replEvaluator) callRepoExists(args []any) (any, error) {
	stmt, err := e.buildCountStmt(args)
	if err != nil {
		return nil, err
	}

	return repo.Exists(e.db, stmt)
}

func (e *replEvaluator) callRepoInsert(args []any) (any, error) {
	stmt, err := e.buildInsertStmt(args)
	if err != nil {
		return nil, err
	}

	return repo.InsertMap(e.db, stmt)
}

func (e *replEvaluator) callRepoUpdate(args []any) (any, error) {
	stmt, err := e.buildUpdateStmt(args)
	if err != nil {
		return nil, err
	}

	return repo.UpdateAffected(e.db, stmt)
}

func (e *replEvaluator) callRepoDelete(args []any) (any, error) {
	stmt, err := e.buildDeleteStmt(args)
	if err != nil {
		return nil, err
	}

	return repo.DeleteAffected(e.db, stmt)
}

func (e *replEvaluator) callRepoPreview(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("repo.Preview expects exactly 1 stmt argument")
	}

	stmt, err := coerceStmt(args[0])
	if err != nil {
		return nil, err
	}

	query, boundArgs, err := repo.Preview(e.db, stmt)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"query": query,
		"args":  normalizeResultValue(boundArgs),
	}, nil
}

func (e *replEvaluator) callRepoTables(args []any) (any, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("repo.Tables expects no args")
	}

	return repo.ListTables(e.db)
}

func (e *replEvaluator) callRepoDriver(args []any) (any, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("repo.Driver expects no args")
	}

	return string(e.db.Driver()), nil
}

func (e *replEvaluator) buildSelectStmt(args []any) (reposql.Stmt, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("repo.Find and repo.FindOne require at least 1 arg")
	}

	if len(args) == 1 {
		if stmt, err := coerceStmt(args[0]); err == nil {
			return stmt, nil
		}

		table, err := coerceTable(args[0])
		if err != nil {
			return nil, err
		}

		return reposql.SelectColumns(table.AllFields().String()).FromTable(table), nil
	}

	table, err := coerceTable(args[0])
	if err != nil {
		return nil, err
	}

	fields := []string{table.AllFields().String()}
	var cond reposql.CondBuilder

	if len(args) == 2 {
		if candidate, err := coerceCond(args[1]); err == nil {
			cond = candidate
		} else {
			fields, err = coerceFields(table, args[1])
			if err != nil {
				return nil, err
			}
		}
	} else if len(args) == 3 {
		fields, err = coerceFields(table, args[1])
		if err != nil {
			return nil, err
		}

		cond, err = coerceCond(args[2])
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("too many args for repo.Find or repo.FindOne")
	}

	builder := reposql.SelectColumns(fields...).FromTable(table)
	if cond != nil {
		builder.Where(cond)
	}

	return builder, nil
}

func (e *replEvaluator) buildCountStmt(args []any) (reposql.Stmt, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("repo.Count and repo.Exists require at least 1 arg")
	}

	if len(args) == 1 {
		if stmt, err := coerceStmt(args[0]); err == nil {
			return stmt, nil
		}

		table, err := coerceTable(args[0])
		if err != nil {
			return nil, err
		}

		return reposql.SelectColumns("count(*)").FromTable(table), nil
	}

	if len(args) != 2 {
		return nil, fmt.Errorf("repo.Count and repo.Exists accept table+cond or stmt")
	}

	table, err := coerceTable(args[0])
	if err != nil {
		return nil, err
	}

	cond, err := coerceCond(args[1])
	if err != nil {
		return nil, err
	}

	return reposql.SelectColumns("count(*)").FromTable(table).Where(cond), nil
}

func (e *replEvaluator) buildInsertStmt(args []any) (reposql.Stmt, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("repo.Insert requires at least 1 arg")
	}

	if len(args) == 1 {
		return coerceStmt(args[0])
	}

	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("repo.Insert accepts stmt or table, values, [returning]")
	}

	table, err := coerceTable(args[0])
	if err != nil {
		return nil, err
	}

	values, err := coerceH(args[1])
	if err != nil {
		return nil, err
	}

	builder := reposql.Insert(values).IntoTable(table)
	if len(args) == 3 {
		fields, err := coerceFields(table, args[2])
		if err != nil {
			return nil, err
		}

		builder.Returning(fields...)
	}

	return builder, nil
}

func (e *replEvaluator) buildUpdateStmt(args []any) (reposql.Stmt, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("repo.Update requires at least 1 arg")
	}

	if len(args) == 1 {
		return coerceStmt(args[0])
	}

	if len(args) != 3 {
		return nil, fmt.Errorf("repo.Update accepts stmt or table, values, cond|true")
	}

	table, err := coerceTable(args[0])
	if err != nil {
		return nil, err
	}

	values, err := coerceH(args[1])
	if err != nil {
		return nil, err
	}

	builder := reposql.UpdateTable(table).Set(values)
	if allowAll, ok := args[2].(bool); ok {
		if !allowAll {
			return nil, fmt.Errorf("repo.Update full-table update requires true as the last arg")
		}

		return builder, nil
	}

	cond, err := coerceCond(args[2])
	if err != nil {
		return nil, err
	}

	builder.Where(cond)
	return builder, nil
}

func (e *replEvaluator) buildDeleteStmt(args []any) (reposql.Stmt, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("repo.Delete requires at least 1 arg")
	}

	if len(args) == 1 {
		if stmt, err := coerceStmt(args[0]); err == nil {
			return stmt, nil
		}

		return nil, fmt.Errorf("repo.Delete full-table delete requires true as the second arg")
	}

	if len(args) != 2 {
		return nil, fmt.Errorf("repo.Delete accepts stmt or table, cond|true")
	}

	table, err := coerceTable(args[0])
	if err != nil {
		return nil, err
	}

	builder := reposql.DeleteFrom(table)
	if allowAll, ok := args[1].(bool); ok {
		if !allowAll {
			return nil, fmt.Errorf("repo.Delete full-table delete requires true as the second arg")
		}

		return builder, nil
	}

	cond, err := coerceCond(args[1])
	if err != nil {
		return nil, err
	}

	builder.Where(cond)
	return builder, nil
}

func coerceStmt(value any) (reposql.Stmt, error) {
	stmt, ok := value.(reposql.Stmt)
	if !ok {
		return nil, fmt.Errorf("expected stmt, got %T", value)
	}

	return stmt, nil
}

func coerceTableArg(args []any) (reposql.TableName, error) {
	if len(args) != 1 {
		return reposql.TableName{}, fmt.Errorf("expected exactly 1 table arg")
	}

	return coerceTable(args[0])
}

func coerceTable(value any) (reposql.TableName, error) {
	switch typed := value.(type) {
	case reposql.TableName:
		return typed, nil
	case string:
		return reposql.TableOf(typed), nil
	default:
		return reposql.TableName{}, fmt.Errorf("expected table name string or TableName, got %T", value)
	}
}

func coerceCond(value any) (reposql.CondBuilder, error) {
	if value == nil {
		return nil, fmt.Errorf("condition cannot be nil")
	}

	cond, ok := value.(reposql.CondBuilder)
	if !ok {
		return nil, fmt.Errorf("expected condition, got %T", value)
	}

	return cond, nil
}

func coerceH(value any) (reposql.H, error) {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || reflected.Kind() != reflect.Map || reflected.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("expected map[string]any compatible value, got %T", value)
	}

	mapped := reposql.H{}
	for _, key := range reflected.MapKeys() {
		mapped[key.String()] = reflected.MapIndex(key).Interface()
	}

	return mapped, nil
}

func coerceFields(table reposql.TableName, value any) ([]string, error) {
	switch typed := value.(type) {
	case string:
		return normalizeFieldList(table, strings.Split(typed, ","))
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			text, err := coerceString(item)
			if err != nil {
				return nil, err
			}

			parts = append(parts, text)
		}

		return normalizeFieldList(table, parts)
	case []string:
		return normalizeFieldList(table, typed)
	default:
		return nil, fmt.Errorf("expected fields string or slice, got %T", value)
	}
}

func normalizeFieldList(table reposql.TableName, fields []string) ([]string, error) {
	normalized := make([]string, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed == "" {
			continue
		}

		if trimmed == "*" {
			normalized = append(normalized, table.AllFields().String())
			continue
		}

		if strings.Contains(trimmed, ".") {
			normalized = append(normalized, reposql.Field(trimmed).String())
			continue
		}

		normalized = append(normalized, table.Field(trimmed).String())
	}

	if len(normalized) == 0 {
		return nil, fmt.Errorf("fields cannot be empty")
	}

	return normalized, nil
}

func toAnySlice(value any) ([]any, error) {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || (reflected.Kind() != reflect.Slice && reflected.Kind() != reflect.Array) {
		return nil, fmt.Errorf("expected slice, got %T", value)
	}

	items := make([]any, 0, reflected.Len())
	for idx := 0; idx < reflected.Len(); idx++ {
		items = append(items, reflected.Index(idx).Interface())
	}

	return items, nil
}

func coerceString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case fmt.Stringer:
		return typed.String(), nil
	default:
		return "", fmt.Errorf("expected string, got %T", value)
	}
}

func coerceInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case int:
		return int64(typed), nil
	case int8:
		return int64(typed), nil
	case int16:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case int64:
		return typed, nil
	case uint:
		return int64(typed), nil
	case uint8:
		return int64(typed), nil
	case uint16:
		return int64(typed), nil
	case uint32:
		return int64(typed), nil
	case uint64:
		return int64(typed), nil
	default:
		return 0, fmt.Errorf("expected integer, got %T", value)
	}
}

func coerceFloat64(value any) (float64, error) {
	switch typed := value.(type) {
	case float32:
		return float64(typed), nil
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	default:
		return 0, fmt.Errorf("expected float, got %T", value)
	}
}
