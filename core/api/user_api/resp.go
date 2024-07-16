package user_api

import (
	"github.com/daqing/airway/app/models"
)

type UserResp struct {
	Id        int64
	Nickname  string
	Username  string
	ApiToken  string
	Role      models.UserRole
	CreatedAt models.Timestamp
	UpdatedAt models.Timestamp
}

func (ur UserResp) Fields() []string {
	return []string{"id", "username", "nickname", "role", "api_token"}
}
