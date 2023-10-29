package membership_plugin

import (
	"fmt"
	"time"

	"github.com/daqing/airway/lib/repo"
)

func AddMembership(userId int64, membershipType MembershipType, expiredAt time.Time) error {
	// check if user already has membership
	exists, err := repo.Exists[Membership](
		[]repo.KVPair{
			repo.KV("user_id", userId),
			repo.KV("name", string(membershipType)),
		})

	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("user already has membership: %v", membershipType)
	}

	_, err = repo.Insert[Membership]([]repo.KVPair{
		repo.KV("user_id", userId),
		repo.KV("name", string(membershipType)),
		repo.KV("expired_at", expiredAt),
	})

	return err
}

func MembershipFor(userId int64) (*MembershipResp, error) {
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
