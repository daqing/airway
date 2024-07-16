package models

import (
	"time"
)

type Membership struct {
	BaseModel

	UserId    int64
	Name      string
	ExpiredAt time.Time
}

type MembershipType string

const Writer MembershipType = "writer"

func (m Membership) TableName() string { return "memberships" }

type MembershipResp struct {
	Name      string
	ExpiredAt Timestamp
}

func (r MembershipResp) Fields() []string {
	return []string{"name", "expired_at"}
}

// func MembershipFor(userId IdType) (*MembershipResp, error) {
// 	row, err := sql_orm.FindOne[Membership](
// 		[]string{"name", "expired_at"},
// 		[]sql_orm.KVPair{
// 			sql_orm.KV("user_id", userId),
// 		},
// 	)

// 	if err != nil {
// 		return nil, err
// 	}

// 	if row == nil {
// 		return &MembershipResp{}, nil
// 	}

// 	item := sql_orm.ItemResp[Membership, MembershipResp](row)

// 	return item, nil
// }
