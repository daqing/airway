package membership_api

import (
	"fmt"
	"time"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
)

func AddMembership(userId int64, membershipType models.MembershipType, expiredAt time.Time) error {
	// check if user already has membership
	exists, err := sql_orm.Exists[models.Membership](
		[]sql_orm.KVPair{
			sql_orm.KV("user_id", userId),
			sql_orm.KV("name", string(membershipType)),
		})

	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("user already has membership: %v", membershipType)
	}

	_, err = sql_orm.Insert[models.Membership]([]sql_orm.KVPair{
		sql_orm.KV("user_id", userId),
		sql_orm.KV("name", string(membershipType)),
		sql_orm.KV("expired_at", expiredAt),
	})

	return err
}
