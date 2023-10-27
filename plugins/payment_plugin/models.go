package payment_plugin

import (
	"time"

	"github.com/daqing/airway/lib/repo"
)

// type JSON map[string]any

type Payment struct {
	Id int64

	UserId    int64
	UUID      string
	GoodsType string
	GoodsId   int64
	Cent      repo.PriceCent
	Action    string
	Note      string
	Status    PaymentStatus

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "payments"

func (m Payment) TableName() string { return tableName }

type PaymentStatus string

const FreshStatus PaymentStatus = "fresh"
