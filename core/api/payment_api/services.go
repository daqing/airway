package payment_api

import (
	"strings"
	"time"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/app/services"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/daqing/airway/lib/utils"
)

const PREFIX = "PMT"

func GenerateUUID() string {
	ts := time.Now().Format("20060102150405")
	rand := strings.ToUpper(utils.RandomHex(10))

	return strings.Join([]string{PREFIX, ts, "H", rand}, "")
}

func BuyGoods(userId models.IdType, goods models.PolyModel, price services.PriceCent, action, note string) (*models.Payment, error) {
	pair := []sql_orm.KVPair{
		sql_orm.KV("uuid", GenerateUUID()),
		sql_orm.KV("user_id", userId),
		sql_orm.KV("goods_type", goods.PolyType()),
		sql_orm.KV("goods_id", goods.PolyId()),
		sql_orm.KV("cent", price),
		sql_orm.KV("action", action),
		sql_orm.KV("note", note),
		sql_orm.KV("status", models.FreshStatus),
	}

	return sql_orm.Insert[models.Payment](pair)
}
