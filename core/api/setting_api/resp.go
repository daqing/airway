package setting_api

import "github.com/daqing/airway/lib/pg_repo"

type SettingResp struct {
	Id int64

	Key string
	Val string

	CreatedAt pg_repo.Timestamp
	UpdatedAt pg_repo.Timestamp
}

func (sr SettingResp) Fields() []string {
	return []string{"id", "key", "val"}
}
