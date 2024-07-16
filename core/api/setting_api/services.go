package setting_api

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
)

func CreateSetting(key string, val string) (*models.Setting, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("key cannot be empty")
	}

	if len(val) == 0 {
		return nil, fmt.Errorf("val cannot be empty")
	}

	return sql_orm.Insert[models.Setting](
		[]sql_orm.KVPair{
			sql_orm.KV("key", key),
			sql_orm.KV("val", val),
		},
	)
}

func UpdateSetting(id models.IdType, key string, val string) bool {
	return sql_orm.UpdateFields[models.Setting](id, []sql_orm.KVPair{
		sql_orm.KV("key", key),
		sql_orm.KV("val", val),
	})
}
