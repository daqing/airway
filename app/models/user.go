package models

import (
	"time"

	"github.com/daqing/airway/lib/replreg"
	airwaysql "github.com/daqing/airway/lib/sql"
)

type User struct {
	ID        airwaysql.IdType `db:"id" json:"id"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

func init() {
	replreg.Register("User", User{})
}
