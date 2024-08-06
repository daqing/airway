package models

import (
	"fmt"
	"time"
)

type IdType int64

type BaseModel struct {
	ID IdType `gorm:"primarykey" json:"id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Disable `DeletedAt` by default
	// DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

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
