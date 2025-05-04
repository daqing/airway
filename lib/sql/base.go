package sql

import (
	"fmt"

	"github.com/fatih/structs"
)

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
