package membership_api

import (
	"fmt"
	"time"

	"github.com/daqing/airway/lib/pg_repo"
)

func AddMembership(userId int64, membershipType MembershipType, expiredAt time.Time) error {
	// check if user already has membership
	exists, err := pg_repo.Exists[Membership](
		[]pg_repo.KVPair{
			pg_repo.KV("user_id", userId),
			pg_repo.KV("name", string(membershipType)),
		})

	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("user already has membership: %v", membershipType)
	}

	_, err = pg_repo.Insert[Membership]([]pg_repo.KVPair{
		pg_repo.KV("user_id", userId),
		pg_repo.KV("name", string(membershipType)),
		pg_repo.KV("expired_at", expiredAt),
	})

	return err
}

func MembershipFor(userId int64) (*MembershipResp, error) {
	row, err := pg_repo.FindRow[Membership](
		[]string{"name", "expired_at"},
		[]pg_repo.KVPair{
			pg_repo.KV("user_id", userId),
		},
	)

	if err != nil {
		return nil, err
	}

	if row == nil {
		return &MembershipResp{}, nil
	}

	item := pg_repo.ItemResp[Membership, MembershipResp](row)

	return item, nil
}
