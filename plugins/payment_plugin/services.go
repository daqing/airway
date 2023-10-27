package payment_plugin

import (
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
)

const PREFIX = "PMT"

func GenerateUUID() string {
	ts := time.Now().Format("20060102150405")
	rand := strings.ToUpper(utils.RandomHex(10))

	return strings.Join([]string{PREFIX, ts, "H", rand}, "")
}

func BuyGoods(userId int64, goods repo.PolyModel, priceCent int, action, note string) (*Payment, error) {
	return repo.Insert[Payment](
		[]repo.KVPair{
			repo.KV("uuid", GenerateUUID()),
			repo.KV("user_id", userId),
			repo.KV("goods_type", goods.PolyType()),
			repo.KV("goods_id", goods.PolyId()),
			repo.KV("cent", priceCent),
			repo.KV("action", action),
			repo.KV("note", note),
			repo.KV("status", FreshStatus),
		},
	)
}
