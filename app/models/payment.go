package models

import (
	"github.com/daqing/airway/app/services"
	"gorm.io/gorm"
)

type Payment struct {
	gorm.Model

	UserId    int64
	UUID      string
	GoodsType string
	GoodsId   int64
	Cent      services.PriceCent
	Action    string
	Note      string
	Status    PaymentStatus
}

type PaymentStatus string

const FreshStatus PaymentStatus = "fresh"
const PaidStatus PaymentStatus = "paid"

func (p Payment) TableName() string { return "payments" }
