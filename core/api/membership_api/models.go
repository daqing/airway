package membership_api

import (
	"time"
)

type Membership struct {
	Id int64

	UserId    int64
	Name      string
	ExpiredAt time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "memberships"

func (m Membership) TableName() string { return tableName }

type MembershipType string

const Writer MembershipType = "writer"
