package media_api

import "github.com/daqing/airway/app/models"

type MediaResp struct {
	Id int64

	CreatedAt models.Timestamp
	UpdatedAt models.Timestamp
}

func (r MediaResp) Fields() []string {
	return []string{"id"}
}
