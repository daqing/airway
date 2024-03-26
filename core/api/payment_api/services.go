package payment_api

import (
	"strings"
	"time"

	"github.com/daqing/airway/lib/pg_repo"
	"github.com/daqing/airway/lib/utils"
)

const PREFIX = "PMT"

func GenerateUUID() string {
	ts := time.Now().Format("20060102150405")
	rand := strings.ToUpper(utils.RandomHex(10))

	return strings.Join([]string{PREFIX, ts, "H", rand}, "")
}

func BuyGoods(userId int64, goods pg_repo.PolyModel, price pg_repo.PriceCent, action, note string) (*Payment, error) {
	pair := []pg_repo.KVPair{
		pg_repo.KV("uuid", GenerateUUID()),
		pg_repo.KV("user_id", userId),
		pg_repo.KV("goods_type", goods.PolyType()),
		pg_repo.KV("goods_id", goods.PolyId()),
		pg_repo.KV("cent", price),
		pg_repo.KV("action", action),
		pg_repo.KV("note", note),
		pg_repo.KV("status", FreshStatus),
	}

	return pg_repo.Insert[Payment](pair)
}
