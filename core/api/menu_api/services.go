package menu_api

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/repo"
)

func CreateMenu(name, url, place string) (menu *models.Menu, err error) {
	return repo.Insert[models.Menu](
		[]repo.KVPair{
			repo.KV("name", name),
			repo.KV("url", url),
			repo.KV("place", place),
		},
	)
}

func UpdateMenu(id uint, name, url, place string) bool {
	return repo.UpdateFields[models.Menu](
		id,
		[]repo.KVPair{
			repo.KV("name", name),
			repo.KV("url", url),
			repo.KV("place", place),
		},
	)
}

func Menus(fields []string, order string, page, limit int) ([]*models.Menu, error) {
	if page == 0 {
		page = 1
	}

	return repo.FindLimit[models.Menu](
		fields,
		[]repo.KVPair{},
		order,
		(page-1)*limit,
		limit,
	)
}

func MenuPlace(fields []string, place string) ([]*models.Menu, error) {
	return repo.Find[models.Menu](
		fields,
		[]repo.KVPair{
			repo.KV("place", place),
		},
	)
}

func FindBy(field string, value any) (*models.Menu, error) {
	return repo.FindRow[models.Menu](
		[]string{"id", "name", "url", "place"},
		[]repo.KVPair{
			repo.KV(field, value),
		},
	)
}
