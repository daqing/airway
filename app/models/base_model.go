package models

import (
	"fmt"
	"time"
)

type IdType int64

type BaseModel struct {
	ID        IdType `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
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
