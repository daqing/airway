package models

import (
	"github.com/daqing/airway/app/services"
)

type User struct {
	BaseModel

	Nickname          string
	Username          string
	Phone             string
	Email             string
	Avatar            string
	Role              UserRole
	APIToken          string
	EncryptedPassword string
	Balance           services.PriceCent
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
