package setting_api

import (
	"fmt"

	"github.com/daqing/airway/lib/pg_repo"
)

func CreateSetting(key string, val string) (*Setting, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("key cannot be empty")
	}

	if len(val) == 0 {
		return nil, fmt.Errorf("val cannot be empty")
	}

	return pg_repo.Insert[Setting](
		[]pg_repo.KVPair{
			pg_repo.KV("key", key),
			pg_repo.KV("val", val),
		},
	)
}

func UpdateSetting(id int64, key string, val string) bool {
	return pg_repo.UpdateFields[Setting](id, []pg_repo.KVPair{
		pg_repo.KV("key", key),
		pg_repo.KV("val", val),
	})
}
