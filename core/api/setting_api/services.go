package setting_api

import (
	"fmt"

	"github.com/daqing/airway/lib/repo"
)

func CreateSetting(key string, val string) (*Setting, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("key cannot be empty")
	}

	if len(val) == 0 {
		return nil, fmt.Errorf("val cannot be empty")
	}

	return repo.Insert[Setting](
		[]repo.KVPair{
			repo.KV("key", key),
			repo.KV("val", val),
		},
	)
}

func UpdateSetting(id int64, key string, val string) bool {
	return repo.UpdateFields[Setting](id, []repo.KVPair{
		repo.KV("key", key),
		repo.KV("val", val),
	})
}
