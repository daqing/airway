package checkin_plugin

import (
	"time"

	"github.com/daqing/airway/lib/utils"
)

type CheckinResp struct {
	Id int64

	UserId int64

	Year  int
	Month time.Month
	Day   int

	Acc int // 连续签到次数

	CreatedAt utils.Timestamp
	UpdatedAt utils.Timestamp
}

func (c CheckinResp) Fields() []string {
	return []string{"id", "user_id", "year", "month", "day", "acc"}
}
