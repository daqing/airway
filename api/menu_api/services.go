package menu_api

import (
	"github.com/daqing/airway/lib/repo"
)

func CreateMenu(name, url, place string) (menu *Menu, err error) {
	return repo.Insert[Menu](
		[]repo.KVPair{
			repo.KV("name", name),
			repo.KV("url", url),
			repo.KV("place", place),
		},
	)
}

func UpdateMenu(id int64, name, url, place string) bool {
	return repo.UpdateFields[Menu](
		id,
		[]repo.KVPair{
			repo.KV("name", name),
			repo.KV("url", url),
			repo.KV("place", place),
		},
	)
}

func Menus(fields []string, order string, page, limit int) ([]*Menu, error) {
	if page == 0 {
		page = 1
	}

	return repo.FindLimit[Menu](
		fields,
		[]repo.KVPair{},
		order,
		(page-1)*limit,
		limit,
	)
}

func MenuPlace(fields []string, place string) ([]*Menu, error) {
	return repo.Find[Menu](
		fields,
		[]repo.KVPair{
			repo.KV("place", place),
		},
	)
}

func FindBy(field string, value any) (*Menu, error) {
	return repo.FindRow[Menu](
		[]string{"id", "name", "url", "place"},
		[]repo.KVPair{
			repo.KV(field, value),
		},
	)
}
