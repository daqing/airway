package setting_api

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/repo"
)

func CreateSetting(key string, val string) (*models.Setting, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("key cannot be empty")
	}

	if len(val) == 0 {
		return nil, fmt.Errorf("val cannot be empty")
	}

	return repo.Insert[models.Setting](
		[]repo.KVPair{
			repo.KV("key", key),
			repo.KV("val", val),
		},
	)
}

func UpdateSetting(id uint, key string, val string) bool {
	return repo.UpdateFields[models.Setting](id, []repo.KVPair{
		repo.KV("key", key),
		repo.KV("val", val),
	})
}
