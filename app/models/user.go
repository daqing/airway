package models

import (
	"github.com/daqing/airway/app/services"
)

type User struct {
	BaseModel

	Nickname          string             `json:"nickname"`
	Username          string             `json:"username"`
	Phone             string             `json:"phone"`
	Email             string             `json:"email"`
	Avatar            string             `json:"avatar"`
	Role              UserRole           `json:"role"`
	APIToken          string             `json:"api_token"`
	EncryptedPassword string             `json:"-"`
	Balance           services.PriceCent `json:"balance"`
}

func (u User) TableName() string { return "users" }

type UserRole int

const (
	AllRole UserRole = iota
	RootRole
	AdminRole
	BasicRole
)

const polyType = "user"

func (u *User) PolyType() string { return polyType }
func (u *User) PolyId() IdType   { return u.ID }

func (u *User) IsAdmin() bool { return u.Role == AdminRole || u.Role == RootRole }

func RoleName(role UserRole) string {
	switch role {
	case RootRole:
		return "ROOT"
	case AdminRole:
		return "ADMIN"
	case BasicRole:
		return "BASIC"
	default:
		return "[OTHER]"
	}
}
