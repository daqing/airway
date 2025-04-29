package models

import (
	"fmt"
)

type IdType int64

type PolyModel interface {
	PolyId() IdType
	PolyType() string
}

type PriceCent int64

func ToCent(price float64) PriceCent {
	return PriceCent(price * 100)
}

func (c PriceCent) Yuan() string {
	return fmt.Sprintf("%.2f", float64(c/100.0))
}
