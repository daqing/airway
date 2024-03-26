package payment_api

import (
	"time"

	"github.com/daqing/airway/lib/pg_repo"
)

type Payment struct {
	Id int64

	UserId    int64
	UUID      string
	GoodsType string
	GoodsId   int64
	Cent      pg_repo.PriceCent
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
const PaidStatus PaymentStatus = "paid"
