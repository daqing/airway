package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	Nickname          string
	Username          string
	Phone             string
	Email             string
	Avatar            string
	Role              UserRole
	APIToken          string
	EncryptedPassword string
	Balance           PriceCent
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

func (u *User) PolyId() uint { return u.ID }

func (u *User) Membership() (*MembershipResp, error) {
	return MembershipFor(u.ID)
}

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
