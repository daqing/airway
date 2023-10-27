package payment_plugin

import (
	"time"
)

// type JSON map[string]any

type Payment struct {
	Id int64

	UUID      string
	UserId    string
	GoodsType string
	GoodsId   int64
	Cent      int
	Action    string
	Note      string
	Status    string

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "payments"

func (m Payment) TableName() string { return tableName }
