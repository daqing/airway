package point_api

import "github.com/daqing/airway/lib/pg_repo"

type PointResp struct {
	Id int64

	UserId int64
	Count  int

	CreatedAt pg_repo.Timestamp
	UpdatedAt pg_repo.Timestamp
}

func (r PointResp) Fields() []string {
	return []string{"id", "user_id", "count"}
}
