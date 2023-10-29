package point_api

import "github.com/daqing/airway/lib/repo"

type PointResp struct {
	Id int64

	UserId int64
	Count  int

	CreatedAt repo.Timestamp
	UpdatedAt repo.Timestamp
}

func (r PointResp) Fields() []string {
	return []string{"id", "user_id", "count"}
}
