package point_api

import "github.com/daqing/airway/app/models"

type PointResp struct {
	Id int64

	UserId int64
	Count  int

	CreatedAt models.Timestamp
	UpdatedAt models.Timestamp
}

func (r PointResp) Fields() []string {
	return []string{"id", "user_id", "count"}
}
