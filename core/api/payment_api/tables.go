package payment_api

const tableName = "payments"

func (m Payment) TableName() string { return tableName }
