package setting_api

import "github.com/daqing/airway/lib/utils"

type SettingResp struct {
	Id int64

	Key string
	Val string

	CreatedAt utils.Timestamp
	UpdatedAt utils.Timestamp
}

func (sr SettingResp) Fields() []string {
	return []string{"id", "key", "val"}
}
