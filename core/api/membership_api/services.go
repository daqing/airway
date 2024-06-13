package membership_api

import (
	"fmt"
	"time"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/repo"
)

func AddMembership(userId int64, membershipType models.MembershipType, expiredAt time.Time) error {
	// check if user already has membership
	exists, err := repo.Exists[models.Membership](
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

	_, err = repo.Insert[models.Membership]([]repo.KVPair{
		repo.KV("user_id", userId),
		repo.KV("name", string(membershipType)),
		repo.KV("expired_at", expiredAt),
	})

	return err
}
