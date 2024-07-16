package models

type Menu struct {
	BaseModel

	Name  string
	URL   string
	Place string
}

func (m Menu) TableName() string { return "menus" }
