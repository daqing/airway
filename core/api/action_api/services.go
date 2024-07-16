package action_api

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
)

// relation 被关注/收藏/点赞的对象
// userId 谁发起了这个动作
func ToggleAction(userId models.IdType, action string, relation models.PolyModel) (int64, error) {
	var attrs = []sql_orm.KVPair{
		sql_orm.KV("user_id", userId),
		sql_orm.KV("action", action),
		sql_orm.KV("target_id", relation.PolyId()),
		sql_orm.KV("target_type", relation.PolyType()),
	}

	row, err := sql_orm.FindOne[models.Action]([]string{"id"}, attrs)

	if err != nil {
		return sql_orm.InvalidCount, err
	}

	if row == nil {
		// current  action not found, create a new one
		row, err = sql_orm.Insert[models.Action](attrs)
		if err != nil {
			return sql_orm.InvalidCount, err
		}

		if row == nil {
			// record not created
			return sql_orm.InvalidCount, fmt.Errorf("action was not created")
		}
	} else {
		// delete current like action
		err = sql_orm.Delete[models.Action](attrs)
		if err != nil {
			return sql_orm.InvalidCount, err
		}
	}

	count, err := sql_orm.Count[models.Action]([]sql_orm.KVPair{
		sql_orm.KV("action", action),
		sql_orm.KV("target_id", relation.PolyId()),
		sql_orm.KV("target_type", relation.PolyType()),
	})

	if err != nil {
		return sql_orm.InvalidCount, err
	}

	return count, nil

}
