package setting_api

import "github.com/daqing/airway/app/models"

type SettingResp struct {
	Id int64

	Key string
	Val string

	CreatedAt models.Timestamp
	UpdatedAt models.Timestamp
}

func (sr SettingResp) Fields() []string {
	return []string{"id", "key", "val"}
}
