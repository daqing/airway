package menu_api

import "github.com/daqing/airway/lib/pg_repo"

func CreateMenu(name, url, place string) (menu *Menu, err error) {
	return pg_repo.Insert[Menu](
		[]pg_repo.KVPair{
			pg_repo.KV("name", name),
			pg_repo.KV("url", url),
			pg_repo.KV("place", place),
		},
	)
}

func UpdateMenu(id int64, name, url, place string) bool {
	return pg_repo.UpdateFields[Menu](
		id,
		[]pg_repo.KVPair{
			pg_repo.KV("name", name),
			pg_repo.KV("url", url),
			pg_repo.KV("place", place),
		},
	)
}

func Menus(fields []string, order string, page, limit int) ([]*Menu, error) {
	if page == 0 {
		page = 1
	}

	return pg_repo.FindLimit[Menu](
		fields,
		[]pg_repo.KVPair{},
		order,
		(page-1)*limit,
		limit,
	)
}

func MenuPlace(fields []string, place string) ([]*Menu, error) {
	return pg_repo.Find[Menu](
		fields,
		[]pg_repo.KVPair{
			pg_repo.KV("place", place),
		},
	)
}

func FindBy(field string, value any) (*Menu, error) {
	return pg_repo.FindRow[Menu](
		[]string{"id", "name", "url", "place"},
		[]pg_repo.KVPair{
			pg_repo.KV(field, value),
		},
	)
}
