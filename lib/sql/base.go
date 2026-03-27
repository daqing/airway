package sql

import (
	"fmt"

	"github.com/fatih/structs"
)

// Stmt is the common interface implemented by *Builder and all dialect-specific builders.
// lib/repo functions accept this interface so callers can use any dialect builder directly.
type Stmt interface {
	ToSQL() (string, NamedArgs)
	Kind() string
	TableName() string
	InsertValues() H
	InsertRows() []H
	ConflictTarget() []string
}

type PolyModel interface {
	PolyId() IdType
	PolyType() string
}

type H map[string]any

func ToH(obj any) H {
	return structs.Map(obj)
}

type IdType int64
type PriceCent int64

func ToCent(price float64) PriceCent {
	return PriceCent(price * 100)
}

func (c PriceCent) Yuan() string {
	return fmt.Sprintf("%.2f", float64(c/100.0))
}
