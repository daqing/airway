package sql

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var namedPlaceholderPattern = regexp.MustCompile(`@([A-Za-z_][A-Za-z0-9_]*)`)

type SQLExpr struct {
	SQL      string
	Args     NamedArgs
	compiled compiledExpr
}

type functionExpr struct {
	name string
	args []any
}

type binaryExpr struct {
	left  any
	op    string
	right any
}

type wrapperExpr struct {
	prefix string
	value  any
	suffix string
}

type arrayExpr struct {
	values []any
}

func Expr(sql string) SQLExpr {
	return SQLExpr{SQL: sql}
}

func ExprNamed(sql string, args NamedArgs) SQLExpr {
	return SQLExpr{SQL: sql, Args: args}
}

func Raw(sql string) SQLExpr {
	return Expr(sql)
}

func Column(name string) SQLExpr {
	return Expr(name)
}

func Excluded(column string) SQLExpr {
	return Expr("EXCLUDED." + column)
}

func Default() SQLExpr {
	return Expr("DEFAULT")
}

func Func(name string, args ...any) SQLExpr {
	return ExprFrom(functionExpr{name: name, args: args})
}

func Op(left any, operator string, right any) SQLExpr {
	return ExprFrom(binaryExpr{left: left, op: operator, right: right})
}

func Cast(value any, targetType string) SQLExpr {
	return ExprFrom(wrapperExpr{
		prefix: "CAST(",
		value:  Op(value, "AS", Raw(targetType)),
		suffix: ")",
	})
}

func Array(values ...any) SQLExpr {
	return ExprFrom(arrayExpr{values: values})
}

func Any(value any) SQLExpr {
	return ExprFrom(wrapperExpr{prefix: "ANY(", value: value, suffix: ")"})
}

func AllExpr(value any) SQLExpr {
	return ExprFrom(wrapperExpr{prefix: "ALL(", value: value, suffix: ")"})
}

func ExprFrom(expr compiledExpr) SQLExpr {
	if expr == nil {
		return SQLExpr{}
	}

	return SQLExpr{compiled: expr}
}

func SubQuery(query *Builder) SQLExpr {
	if query == nil {
		return SQLExpr{}
	}

	sql, args := query.ToSQL()

	return ExprNamed("("+sql+")", args)
}

type compiledExpr interface {
	buildExpr(*buildState) string
}

func (e SQLExpr) buildExpr(state *buildState) string {
	if e.compiled != nil {
		return e.compiled.buildExpr(state)
	}

	return state.mergeExpr(e.SQL, e.Args)
}

func (e functionExpr) buildExpr(state *buildState) string {
	parts := make([]string, 0, len(e.args))
	for idx, arg := range e.args {
		parts = append(parts, renderValue(state, fmt.Sprintf("%s_%d", e.name, idx), arg))
	}

	return e.name + "(" + strings.Join(parts, ", ") + ")"
}

func (e binaryExpr) buildExpr(state *buildState) string {
	left := renderValue(state, "left", e.left)
	right := renderValue(state, "right", e.right)
	return left + " " + e.op + " " + right
}

func (e wrapperExpr) buildExpr(state *buildState) string {
	return e.prefix + renderValue(state, "expr", e.value) + e.suffix
}

func (e arrayExpr) buildExpr(state *buildState) string {
	parts := make([]string, 0, len(e.values))
	for idx, value := range e.values {
		parts = append(parts, renderValue(state, fmt.Sprintf("array_%d", idx), value))
	}

	return "ARRAY[" + strings.Join(parts, ", ") + "]"
}

type buildState struct {
	args NamedArgs
	seq  map[string]int
}

func newBuildState() *buildState {
	return &buildState{
		args: NamedArgs{},
		seq:  map[string]int{},
	}
}

func (s *buildState) bind(hint string, val any) string {
	key := s.nextKey(hint)
	s.args[key] = val

	return "@" + key
}

func (s *buildState) nextKey(hint string) string {
	base := sanitizeArgName(hint)
	if _, exists := s.args[base]; !exists {
		s.seq[base] = 1
		return base
	}

	for idx := s.seq[base]; ; idx++ {
		candidate := fmt.Sprintf("%s_%d", base, idx)
		if _, exists := s.args[candidate]; !exists {
			s.seq[base] = idx + 1
			return candidate
		}
	}
}

func (s *buildState) mergeExpr(fragment string, args NamedArgs) string {
	if fragment == "" || len(args) == 0 {
		return fragment
	}

	mapping := make(map[string]string, len(args))
	for _, key := range sortedMapKeys(args) {
		newKey := s.nextKey(key)
		s.args[newKey] = args[key]
		mapping[key] = newKey
	}

	return namedPlaceholderPattern.ReplaceAllStringFunc(fragment, func(match string) string {
		name := strings.TrimPrefix(match, "@")
		replacement, ok := mapping[name]
		if !ok {
			return match
		}

		return "@" + replacement
	})
}

func sanitizeArgName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "p"
	}

	var builder strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}

	cleaned := strings.Trim(builder.String(), "_")
	if cleaned == "" {
		return "p"
	}

	first := rune(cleaned[0])
	if unicode.IsDigit(first) {
		return "p_" + cleaned
	}

	return cleaned
}

func renderValue(state *buildState, hint string, value any) string {
	switch v := value.(type) {
	case compiledExpr:
		return v.buildExpr(state)
	case *Builder:
		return SubQuery(v).buildExpr(state)
	default:
		return state.bind(hint, value)
	}
}

func sortedMapKeys[T any](vals map[string]T) []string {
	keys := make([]string, 0, len(vals))
	for key := range vals {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

func normalizeFields(fields []string) []string {
	normalized := make([]string, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	return normalized
}
