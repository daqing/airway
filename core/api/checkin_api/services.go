package checkin_api

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/daqing/airway/lib/utils"
)

func CreateCheckin(user *models.User, when utils.Date) (*models.Checkin, error) {
	yesterday := when.Yesterday()

	prev, err := sql_orm.FindOne[models.Checkin](
		[]string{"id", "acc"},
		[]sql_orm.KVPair{
			sql_orm.KV("user_id", user.ID),
			sql_orm.KV("year", yesterday.Year),
			sql_orm.KV("month", yesterday.Month),
			sql_orm.KV("day", yesterday.Day),
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

	return sql_orm.Insert[models.Checkin]([]sql_orm.KVPair{
		sql_orm.KV("user_id", user.ID),
		sql_orm.KV("year", when.Year),
		sql_orm.KV("month", when.Month),
		sql_orm.KV("day", when.Day),
		sql_orm.KV("acc", acc),
	})

}
