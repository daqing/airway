package checkin_api

import (
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/pg_repo"
	"github.com/daqing/airway/lib/utils"
)

func CreateCheckin(user *user_api.User, when utils.Date) (*Checkin, error) {
	yesterday := when.Yesterday()

	prev, err := pg_repo.FindRow[Checkin](
		[]string{"id", "acc"},
		[]pg_repo.KVPair{
			pg_repo.KV("user_id", user.Id),
			pg_repo.KV("year", yesterday.Year),
			pg_repo.KV("month", yesterday.Month),
			pg_repo.KV("day", yesterday.Day),
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

	return pg_repo.Insert[Checkin]([]pg_repo.KVPair{
		pg_repo.KV("user_id", user.Id),
		pg_repo.KV("year", when.Year),
		pg_repo.KV("month", when.Month),
		pg_repo.KV("day", when.Day),
		pg_repo.KV("acc", acc),
	})

}
