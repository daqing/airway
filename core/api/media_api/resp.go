package media_api

import "github.com/daqing/airway/lib/pg_repo"

type MediaResp struct {
	Id int64

	CreatedAt pg_repo.Timestamp
	UpdatedAt pg_repo.Timestamp
}

func (r MediaResp) Fields() []string {
	return []string{"id"}
}
