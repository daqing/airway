package menu_api

import "github.com/daqing/airway/lib/repo"

func CreateMenu(name, url, place string) (menu *Menu, err error) {
	return repo.Insert[Menu](
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
