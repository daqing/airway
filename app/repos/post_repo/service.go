package post_repo

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
)

func PostUser(p *models.Post) *models.User {
	user, err := sql_orm.FindOne[models.User](
		[]string{"id", "nickname", "username", "avatar"},
		[]sql_orm.KVPair{
			sql_orm.KV("id", p.UserId),
		},
	)

	if err != nil {
		return nil
	}

	return user
}

func PostUserAvatar(p *models.Post) string {
	user := PostUser(p)

	if user == nil {
		return sql_orm.EMPTY_STRING
	}

	return user.Avatar
}

func PostNode(p *models.Post) *models.Node {
	node, err := sql_orm.FindOne[models.Node](
		[]string{"id", "name", "key"},
		[]sql_orm.KVPair{
			sql_orm.KV("id", p.NodeId),
		},
	)

	if err != nil {
		return nil
	}

	return node
}

func PostComments(p *models.Post) ([]*models.Comment, error) {
	return sql_orm.Find[models.Comment](
		[]string{"id", "user_id", "content", "created_at"},
		[]sql_orm.KVPair{
			sql_orm.KV("target_type", p.PolyType()),
			sql_orm.KV("target_id", p.PolyId()),
		},
	)
}
