package setting_plugin

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
		[]repo.KeyValueField{
			repo.NewKV("key", key),
			repo.NewKV("val", val),
		},
	)
}

func UpdateSetting(id int64, key string, val string) bool {
	return repo.UpdateFields[Setting](id, []repo.KeyValueField{
		repo.NewKV("key", key),
		repo.NewKV("val", val),
	})
}
