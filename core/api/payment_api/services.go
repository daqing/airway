package payment_api

import (
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/models"
)

const PREFIX = "PMT"

func GenerateUUID() string {
	ts := time.Now().Format("20060102150405")
	rand := strings.ToUpper(utils.RandomHex(10))

	return strings.Join([]string{PREFIX, ts, "H", rand}, "")
}

func BuyGoods(userId uint, goods models.PolyModel, price models.PriceCent, action, note string) (*models.Payment, error) {
	pair := []repo.KVPair{
		repo.KV("uuid", GenerateUUID()),
		repo.KV("user_id", userId),
		repo.KV("goods_type", goods.PolyType()),
		repo.KV("goods_id", goods.PolyId()),
		repo.KV("cent", price),
		repo.KV("action", action),
		repo.KV("note", note),
		repo.KV("status", models.FreshStatus),
	}

	return repo.Insert[models.Payment](pair)
}
