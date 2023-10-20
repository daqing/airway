package user_plugin

import "github.com/daqing/airway/lib/utils"

type UserResp struct {
	Id        int64
	Nickname  string
	Username  string
	ApiToken  string
	Role      UserRole
	CreatedAt utils.Timestamp
	UpdatedAt utils.Timestamp
}

func (ur UserResp) Fields() []string {
	return []string{"id", "username", "nickname", "role", "api_token"}
}
