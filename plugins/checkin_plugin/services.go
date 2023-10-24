package checkin_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/plugins/user_plugin"
)

func CreateCheckin(user *user_plugin.User, when utils.Date) (*Checkin, error) {
	yesterday := when.Yesterday()

	prev, err := repo.FindRow[Checkin](
		[]string{"id", "acc"},
		[]repo.KVPair{
			repo.KV("user_id", user.Id),
			repo.KV("year", yesterday.Year),
			repo.KV("month", yesterday.Month),
			repo.KV("day", yesterday.Day),
		},
	)

	if err != nil {
		return nil, err
	}

	acc := 1
	if prev != nil {
		// user has checked in at yesterday
		acc += prev.Acc
	}

	return repo.Insert[Checkin]([]repo.KVPair{
		repo.KV("user_id", user.Id),
		repo.KV("year", when.Year),
		repo.KV("month", when.Month),
		repo.KV("day", when.Day),
		repo.KV("acc", acc),
	})

}
