package models

import (
	"time"

	"github.com/daqing/airway/lib/repo"
	"gorm.io/gorm"
)

type Membership struct {
	gorm.Model

	UserId    int64
	Name      string
	ExpiredAt time.Time
}

type MembershipType string

const Writer MembershipType = "writer"

func (m Membership) TableName() string { return "memberships" }

type MembershipResp struct {
	Name      string
	ExpiredAt repo.Timestamp
}

func (r MembershipResp) Fields() []string {
	return []string{"name", "expired_at"}
}

func MembershipFor(userId uint) (*MembershipResp, error) {
	row, err := repo.FindRow[Membership](
		[]string{"name", "expired_at"},
		[]repo.KVPair{
			repo.KV("user_id", userId),
		},
	)

	if err != nil {
		return nil, err
	}

	if row == nil {
		return &MembershipResp{}, nil
	}

	item := repo.ItemResp[Membership, MembershipResp](row)

	return item, nil
}
