package comment_repo

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
)

func CommentUser(c *models.Comment) *models.User {
	user, err := sql_orm.FindOne[models.User](
		[]string{"id", "nickname", "username", "avatar"},
		[]sql_orm.KVPair{
			sql_orm.KV("id", c.UserId),
		},
	)

	if err != nil {
		return nil
	}

	return user
}
