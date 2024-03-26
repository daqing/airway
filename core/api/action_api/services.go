package action_api

import (
	"fmt"

	"github.com/daqing/airway/lib/pg_repo"
)

// relation 被关注/收藏/点赞的对象
// userId 谁发起了这个动作
func ToggleAction(userId int64, action string, relation pg_repo.PolyModel) (int64, error) {
	var attrs = []pg_repo.KVPair{
		pg_repo.KV("user_id", userId),
		pg_repo.KV("action", action),
		pg_repo.KV("target_id", relation.PolyId()),
		pg_repo.KV("target_type", relation.PolyType()),
	}

	row, err := pg_repo.FindRow[Action]([]string{"id"}, attrs)

	if err != nil {
		return pg_repo.InvalidCount, err
	}

	if row == nil {
		// current  action not found, create a new one
		row, err = pg_repo.Insert[Action](attrs)
		if err != nil {
			return pg_repo.InvalidCount, err
		}

		if row == nil {
			// record not created
			return pg_repo.InvalidCount, fmt.Errorf("action was not created")
		}
	} else {
		// delete current like action
		err = pg_repo.Delete[Action](attrs)
		if err != nil {
			return pg_repo.InvalidCount, err
		}
	}

	count, err := pg_repo.Count[Action]([]pg_repo.KVPair{
		pg_repo.KV("action", action),
		pg_repo.KV("target_id", relation.PolyId()),
		pg_repo.KV("target_type", relation.PolyType()),
	})

	if err != nil {
		return pg_repo.InvalidCount, err
	}

	return count, nil

}
