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
		[]repo.KeyValueField{
			repo.NewKV("user_id", user.Id),
			repo.NewKV("year", yesterday.Year),
			repo.NewKV("month", yesterday.Month),
			repo.NewKV("day", yesterday.Day),
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

	return repo.Insert[Checkin]([]repo.KeyValueField{
		repo.NewKV("user_id", user.Id),
		repo.NewKV("year", when.Year),
		repo.NewKV("month", when.Month),
		repo.NewKV("day", when.Day),
		repo.NewKV("acc", acc),
	})

}
