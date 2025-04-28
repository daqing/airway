package models

import (
	"fmt"
)

type IdType int64

type PolyModel interface {
	PolyId() IdType
	PolyType() string
}

// type Timestamp time.Time

// func (ts Timestamp) MarshalJSON() ([]byte, error) {
// 	t := time.Time(ts)
// 	str := fmt.Sprintf("%d", t.Unix())

// 	return []byte(str), nil
// }

type PriceCent int64

func ToCent(price float64) PriceCent {
	return PriceCent(price * 100)
}

func (c PriceCent) Yuan() string {
	return fmt.Sprintf("%.2f", float64(c/100.0))
}
