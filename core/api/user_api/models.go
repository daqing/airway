package user_api

import (
	"time"

	"github.com/daqing/airway/core/api/membership_api"
	"github.com/daqing/airway/lib/repo"
)

type User struct {
	Id int64

	Nickname          string
	Username          string
	Phone             string
	Email             string
	Avatar            string
	Role              UserRole
	ApiToken          string
	EncryptedPassword string
	Balance           repo.PriceCent

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "users"

func (u User) TableName() string { return tableName }

type UserRole int

const (
	AllRole UserRole = iota
	RootRole
	AdminRole
	BasicRole
)

const polyType = "user"

func (u *User) PolyType() string { return polyType }

func (u *User) PolyId() int64 { return u.Id }

func (u *User) Membership() (*membership_api.MembershipResp, error) {
	return membership_api.MembershipFor(u.Id)
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
