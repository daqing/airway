package user_plugin

import (
	"time"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/plugins/membership_plugin"
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
	BasicRole UserRole = iota
	AdminRole
)

const polyType = "user"

func (u *User) PolyType() string { return polyType }

func (u *User) PolyId() int64 { return u.Id }

func (u *User) Membership() (*membership_plugin.MembershipResp, error) {
	return membership_plugin.MembershipFor(u.Id)
}
