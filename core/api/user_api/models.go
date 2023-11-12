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
	Role              userRole
	ApiToken          string
	EncryptedPassword string
	Balance           repo.PriceCent

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "users"

func (u User) TableName() string { return tableName }

type userRole int

const (
	basicRole userRole = iota
	adminRole
)

const polyType = "user"

func (u *User) PolyType() string { return polyType }

func (u *User) PolyId() int64 { return u.Id }

func (u *User) Membership() (*membership_api.MembershipResp, error) {
	return membership_api.MembershipFor(u.Id)
}

func (u *User) IsAdmin() bool { return u.Role == adminRole }
