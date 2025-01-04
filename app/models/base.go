package models

import (
	"fmt"
	"time"
)

type IdType int64

// type BaseModel struct {
// 	ID IdType `gorm:"primarykey" json:"id"`

// 	CreatedAt Timestamp `json:"created_at"`
// 	UpdatedAt Timestamp `json:"updated_at"`

// 	// Disable `DeletedAt` by default
// 	// DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
// }

type PolyModel interface {
	PolyId() IdType
	PolyType() string
}

type Timestamp time.Time

func (ts Timestamp) MarshalJSON() ([]byte, error) {
	t := time.Time(ts)
	str := fmt.Sprintf("%d", t.Unix())

	return []byte(str), nil
}

type PriceCent int64

func ToCent(price float64) PriceCent {
	return PriceCent(price * 100)
}

func (c PriceCent) Yuan() string {
	return fmt.Sprintf("%.2f", float64(c/100.0))
}
