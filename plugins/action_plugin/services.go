package action_plugin

import (
	"fmt"

	"github.com/daqing/airway/lib/repo"
)

// relation 被关注/收藏/点赞的对象
// userId 谁发起了这个动作
func ToggleAction(relation RelationModel, userId int64, action string) (int64, error) {
	var attrs = []repo.KeyValueField{
		repo.NewKV("user_id", userId),
		repo.NewKV("action", action),
		repo.NewKV("target_id", relation.RelId()),
		repo.NewKV("target_type", relation.RelType()),
	}

	row, err := repo.FindRow[Action]([]string{"id"}, attrs)

	if err != nil {
		return repo.InvalidCount, err
	}

	if row == nil {
		// current  action not found, create a new one
		row, err = repo.Insert[Action](attrs)
		if err != nil {
			return repo.InvalidCount, err
		}

		if row == nil {
			// record not created
			return repo.InvalidCount, fmt.Errorf("action was not created")
		}
	} else {
		// delete current like action
		err = repo.Delete[Action](attrs)
		if err != nil {
			return repo.InvalidCount, err
		}
	}

	count, err := repo.Count[Action]([]repo.KeyValueField{
		repo.NewKV("action", action),
		repo.NewKV("target_id", relation.RelId()),
		repo.NewKV("target_type", relation.RelType()),
	})

	if err != nil {
		return repo.InvalidCount, err
	}

	return count, nil

}
