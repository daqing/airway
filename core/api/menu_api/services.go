package menu_api

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
)

func CreateMenu(name, url, place string) (menu *models.Menu, err error) {
	return sql_orm.Insert[models.Menu](
		[]sql_orm.KVPair{
			sql_orm.KV("name", name),
			sql_orm.KV("url", url),
			sql_orm.KV("place", place),
		},
	)
}

func UpdateMenu(id models.IdType, name, url, place string) bool {
	return sql_orm.UpdateFields[models.Menu](
		id,
		[]sql_orm.KVPair{
			sql_orm.KV("name", name),
			sql_orm.KV("url", url),
			sql_orm.KV("place", place),
		},
	)
}

func Menus(fields []string, order string, page, limit int) ([]*models.Menu, error) {
	if page == 0 {
		page = 1
	}

	return sql_orm.FindLimit[models.Menu](
		fields,
		[]sql_orm.KVPair{},
		order,
		(page-1)*limit,
		limit,
	)
}

func MenuPlace(fields []string, place string) ([]*models.Menu, error) {
	return sql_orm.Find[models.Menu](
		fields,
		[]sql_orm.KVPair{
			sql_orm.KV("place", place),
		},
	)
}

func FindBy(field string, value any) (*models.Menu, error) {
	return sql_orm.FindOne[models.Menu](
		[]string{"id", "name", "url", "place"},
		[]sql_orm.KVPair{
			sql_orm.KV(field, value),
		},
	)
}
