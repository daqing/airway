package menu_api

import (
	"time"
)

type Menu struct {
	Id    int64
	Name  string
	URL   string
	Place string

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "menus"

func (m Menu) TableName() string { return tableName }
