package membership_api

const tableName = "memberships"

func (m Membership) TableName() string { return tableName }
